// ELab Web is a local browser host for the shared ELab workbench.
package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dshills/elab/workbench"
)

//go:embed static/*
var assets embed.FS

type workspace struct {
	service   *workbench.Service
	directory string
	library   string
	touched   time.Time
	files     sync.Mutex
	info      workspaceInfo
	persisted string
	restored  bool
}
type webServer struct {
	storeMu  sync.Mutex
	assets   fs.FS
	mu       sync.Mutex
	sessions map[string]*workspace
	commands map[string]workbench.Capability
	root     string
	instance string
}

func newWebServer(root string) *webServer {
	commands := make(map[string]workbench.Capability)
	for _, c := range workbench.Catalog() {
		commands[c.ID] = c
	}
	static, _ := fs.Sub(assets, "static")
	return &webServer{assets: static, sessions: make(map[string]*workspace), commands: commands, root: root, instance: token()}
}
func token() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func jsonReply(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
func webError(w http.ResponseWriter, code int, message string) {
	jsonReply(w, code, map[string]string{"error": message})
}
func (s *webServer) workspace(w http.ResponseWriter, r *http.Request) (*workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := ""
	if cookie, err := r.Cookie("elab-workspace"); err == nil && managedID.MatchString(cookie.Value) {
		id = cookie.Value
		if session := s.sessions[id]; session != nil {
			session.touched = time.Now()
			return session, nil
		}
	}
	if len(s.sessions) >= 16 {
		return nil, fmt.Errorf("too many open workspaces")
	}
	if id == "" {
		id = token()
	}
	dir := filepath.Join(s.root, "workspaces", id)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	session := &workspace{service: workbench.New(), directory: dir, library: s.root, touched: time.Now()}
	s.storeMu.Lock()
	migrationErr := session.migrateLibrary()
	s.storeMu.Unlock()
	if err := migrationErr; err != nil {
		session.service.Close()
		return nil, err
	}
	if err := session.restore(r.Context()); err != nil {
		session.service.Close()
		return nil, err
	}
	if !session.restored {
		state := session.service.Snapshot()
		params, _ := json.Marshal(workbench.ExampleLoadParams{ID: "rc"})
		response := session.service.Dispatch(r.Context(), workbench.Request{Version: 1, ID: "welcome", Key: "welcome", Command: "example.load", Params: params, ExpectedCommandRevision: &state.CommandRevision, ExpectedDocumentGeneration: &state.Document.Generation}, "web")
		if response.Error != nil {
			session.service.Close()
			return nil, fmt.Errorf("%s", response.Error.Message)
		}
	}
	s.sessions[id] = session
	_ = session.persist()
	http.SetCookie(w, &http.Cookie{Name: "elab-workspace", Value: id, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 365 * 24 * 60 * 60})
	return session, nil
}
func (s *webServer) close() {
	s.mu.Lock()
	sessions := s.sessions
	s.sessions = make(map[string]*workspace)
	s.mu.Unlock()
	for _, session := range sessions {
		session.files.Lock()
		_ = session.persist()
		session.service.Close()
		session.files.Unlock()
	}
}
func (s *webServer) expire(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			var expired []*workspace
			for id, ws := range s.sessions {
				if time.Since(ws.touched) > 2*time.Hour {
					delete(s.sessions, id)
					expired = append(expired, ws)
				}
			}
			s.mu.Unlock()
			for _, ws := range expired {
				ws.files.Lock()
				_ = ws.persist()
				ws.service.Close()
				ws.files.Unlock()
			}
		}
	}
}
func permitted(command string) bool {
	// These commands control the native host's presentation. Browser gestures
	// and panels have their own equivalents and must not advertise inert forms.
	for _, prefix := range []string{"drawing.", "view.", "workspace."} {
		if strings.HasPrefix(command, prefix) {
			return false
		}
	}
	switch command {
	case "parts.category.set", "parts.search.set", "examples.visibility.set", "examples.category.set", "examples.search.set", "examples.select", "editor.value.set", "wire.begin", "wire.terminals.set", "scope.view.set", "scope.visibility.set", "sweep.visibility.set", "scope.cursor.select", "recording.cursor.select":
		return false
	}

	if command == "audio.render" || command == "audio.read" || command == "audio.stop" || command == "export.page.set" {
		return true
	}
	for _, prefix := range []string{"recovery.", "appearance.", "audio.", "print.", "app.close", "export.", "project.load", "project.save", "verification.import", "sweep.import"} {
		if strings.HasPrefix(command, prefix) {
			return false
		}
	}
	switch command {
	case "project.load", "project.save", "project.discard", "verification.import", "verification.export", "sweep.import", "sweep.export", "spectrum.export", "tolerance.export":
		return false
	}
	return true
}
func sameOrigin(r *http.Request) bool {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host != r.Host || u.Scheme != "http" {
			return false
		}
	}
	// Opening the app from a link is a normal cross-site top-level navigation.
	// API requests still require the local page as their origin.
	return r.Header.Get("Sec-Fetch-Site") != "cross-site" ||
		(r.Method == http.MethodGet && !strings.HasPrefix(r.URL.Path, "/api/") && r.Header.Get("Sec-Fetch-Mode") == "navigate")
}
func (s *webServer) handler() http.Handler {
	files := http.FileServer(http.FS(s.assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; media-src 'self' data: blob:; connect-src 'self'; object-src 'none'; frame-ancestors 'none'; base-uri 'self'")
		if !sameOrigin(r) {
			webError(w, 403, "Use the local ELab Web address to access this workspace.")
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
			files.ServeHTTP(w, r)
			return
		}
		session, err := s.workspace(w, r)
		if err != nil {
			webError(w, 503, err.Error())
			return
		}
		switch r.URL.Path {
		case "/api/catalog":
			if r.Method != http.MethodGet {
				webError(w, 405, "GET required")
				return
			}
			symbols := map[string][]string{}
			for _, part := range workbench.Library() {
				symbols[part.Kind] = symbolPaths(workbench.Component{Kind: part.Kind})
			}
			catalog := []workbench.Capability{}
			for _, c := range workbench.Catalog() {
				if permitted(c.ID) {
					catalog = append(catalog, c)
				}
			}
			jsonReply(w, 200, map[string]any{"parts": workbench.Library(), "examples": workbench.ExampleCatalog(), "categories": workbench.ExampleCategories(), "commands": catalog, "symbols": symbols, "schemas": workbench.SchemaRegistry(), "model_defaults": map[string]any{"bipolar": workbench.DefaultTransistorProfile(), "thermal": map[string]any{"resistor": workbench.DefaultThermalConfig("resistor"), "diode": workbench.DefaultThermalConfig("diode"), "npn": workbench.DefaultThermalConfig("npn"), "pnp": workbench.DefaultThermalConfig("pnp")}}})
		case "/api/state":
			if r.Method != http.MethodGet {
				webError(w, 405, "GET required")
				return
			}
			session.files.Lock()
			var state workbench.Snapshot
			if after, err := strconv.ParseUint(r.URL.Query().Get("after"), 10, 64); err == nil {
				var changed bool
				state, changed = session.service.SnapshotIfChanged(after)
				metadataChanged := r.URL.Query().Has("instance") && (r.URL.Query().Get("instance") != s.instance || r.URL.Query().Get("checkpoint") != session.info.AutosavedAt || r.URL.Query().Get("storage_error") != session.info.Error)
				if !changed && metadataChanged {
					state = session.service.Snapshot()
					changed = true
				}
				if !changed {
					w.WriteHeader(http.StatusNotModified)
					session.files.Unlock()
					return
				}
			} else {
				state = session.service.Snapshot()
			}
			payload := session.presentation(state)
			payload["instance_id"] = s.instance
			session.files.Unlock()
			jsonReply(w, 200, payload)
		case "/api/import-result":
			s.importResult(w, r, session)
		case "/api/projects":
			s.projects(w, r, session)
		case "/api/command":
			s.command(w, r, session)
		case "/api/upload":
			s.upload(w, r, session)
		case "/api/download":
			s.download(w, r, session)
		default:
			webError(w, 404, "Unknown endpoint")
		}
	})
}
func (s *webServer) command(w http.ResponseWriter, r *http.Request, ws *workspace) {
	if r.Method != http.MethodPost || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		webError(w, 405, "POST JSON required")
		return
	}
	var request workbench.Request
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		webError(w, 400, "Invalid command JSON")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		webError(w, 400, "One command is required")
		return
	}
	if _, ok := s.commands[request.Command]; !ok || !permitted(request.Command) {
		webError(w, 400, "This command is unavailable in the browser. Use Open, Download, or Export for files.")
		return
	}
	replacing := s.commands[request.Command].Mutation && (request.Command == "project.clear" || strings.HasPrefix(request.Command, "example."))
	if replacing {
		s.storeMu.Lock()
		defer s.storeMu.Unlock()
	}
	ws.files.Lock()
	defer ws.files.Unlock()
	if replacing {
		var params struct {
			Discard bool `json:"discard_changes"`
		}
		_ = json.Unmarshal(request.Params, &params)
		if ws.info.Recovered && !params.Discard {
			webError(w, 409, "Save or confirm replacing your recovered circuit first.")
			return
		}
		if err := ws.archive(); err != nil {
			webError(w, 500, err.Error())
			return
		}
	}
	response := ws.service.Dispatch(r.Context(), request, "web")
	if response.Error == nil && !response.Duplicate {
		if replacing {
			ws.info = workspaceInfo{}
		}
		if s.commands[request.Command].Mutation {
			_ = ws.persist()
		}
	}
	jsonReply(w, 200, response)
}
func fileCommand(ctx context.Context, ws *workspace, command string, params any, revision, generation uint64) workbench.Response {
	raw, _ := json.Marshal(params)
	return ws.service.Dispatch(ctx, workbench.Request{Version: 1, ID: token(), Key: token(), Command: command, Params: raw, ExpectedCommandRevision: &revision, ExpectedDocumentGeneration: &generation}, "web.files")
}
func guards(r *http.Request) (uint64, uint64, error) {
	revision, a := strconv.ParseUint(r.URL.Query().Get("revision"), 10, 64)
	generation, b := strconv.ParseUint(r.URL.Query().Get("generation"), 10, 64)
	if a != nil || b != nil {
		return 0, 0, fmt.Errorf("workspace guards are required")
	}
	return revision, generation, nil
}
func (s *webServer) upload(w http.ResponseWriter, r *http.Request, ws *workspace) {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	if r.Method != http.MethodPost {
		webError(w, 405, "POST required")
		return
	}
	rev, gen, err := guards(r)
	if err != nil {
		webError(w, 400, err.Error())
		return
	}
	ws.files.Lock()
	defer ws.files.Unlock()
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8<<20))
	if err != nil {
		webError(w, 413, "Projects must be smaller than 8 MiB")
		return
	}
	if ws.info.Recovered && r.URL.Query().Get("discard") != "true" {
		webError(w, 409, "Save or confirm replacing your recovered circuit first.")
		return
	}
	path := filepath.Join(ws.directory, token()+".elab")
	defer os.Remove(path)
	if err := os.WriteFile(path, data, 0600); err != nil {
		webError(w, 500, err.Error())
		return
	}
	if err := ws.archive(); err != nil {
		webError(w, 500, err.Error())
		return
	}
	response := fileCommand(r.Context(), ws, "project.load", workbench.LoadParams{Path: path, DiscardChanges: r.URL.Query().Get("discard") == "true"}, rev, gen)
	if response.Error != nil {
		_ = os.Remove(path)
	} else {
		ws.info = workspaceInfo{Name: strings.TrimSuffix(r.URL.Query().Get("name"), ".elab")}
		_ = ws.persist()
	}
	jsonReply(w, 200, response)
}
func (s *webServer) download(w http.ResponseWriter, r *http.Request, ws *workspace) {
	if r.Method != http.MethodPost {
		webError(w, 405, "POST required")
		return
	}
	rev, gen, err := guards(r)
	if err != nil {
		webError(w, 400, err.Error())
		return
	}
	ws.files.Lock()
	defer ws.files.Unlock()
	kind := r.URL.Query().Get("kind")
	if kind == "project" {
		state := ws.service.Snapshot()
		if state.CommandRevision != rev || state.Document.Generation != gen {
			webError(w, 409, "Workspace changed before download")
			return
		}
		capture, e := ws.service.CaptureProject()
		if e != nil {
			webError(w, 500, e.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="ELab-project.elab"`)
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(capture.Data)
		return
	}
	if s.transfer(w, r, ws, rev, gen, kind) {
		return
	}
	path := filepath.Join(ws.directory, "download.elab")
	command := "project.save"
	var params any = workbench.SaveParams{Path: path, Overwrite: true}
	mime := "application/json"
	name := "ELab-project.elab"
	switch kind {
	case "project":
	case "svg":
		path = filepath.Join(ws.directory, "schematic.svg")
		command = "export.schematic.svg"
		params = workbench.ExportParams{Path: path, Overwrite: true}
		mime = "image/svg+xml"
		name = "ELab-schematic.svg"
	case "csv":
		path = filepath.Join(ws.directory, "scope.csv")
		command = "export.scope.csv"
		params = workbench.ExportParams{Path: path, Overwrite: true}
		mime = "text/csv"
		name = "ELab-scope.csv"
	default:
		webError(w, 400, "Choose project, svg or csv")
		return
	}
	response := fileCommand(r.Context(), ws, command, params, rev, gen)
	if response.Error != nil {
		jsonReply(w, 409, response)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		webError(w, 500, err.Error())
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}
func main() {
	address := flag.String("addr", "localhost:8788", "local web address")
	dataDir := flag.String("data-dir", "", "persistent workspace directory (default: user config/ELab/web)")
	dev := flag.Bool("dev", false, "serve frontend files from exp-web/static for development")
	flag.Parse()
	host, _, err := net.SplitHostPort(*address)
	if err != nil {
		log.Fatal(err)
	}
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		log.Fatal("ELab Web requires a loopback address")
	}
	config, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	root := *dataDir
	if root == "" {
		root = filepath.Join(config, "ELab", "web")
	}
	err = os.MkdirAll(root, 0700)
	if err != nil {
		log.Fatal(err)
	}
	unlock, err := lockStore(root)
	if err != nil {
		log.Fatal(err)
	}
	defer unlock()
	web := newWebServer(root)
	if *dev {
		web.assets = os.DirFS("exp-web/static")
	}
	defer web.close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go web.expire(ctx)
	go web.autosave(ctx)
	server := &http.Server{Addr: *address, Handler: web.handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 90 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	fmt.Printf("ELab Web → http://%s\nProjects & recovery → %s\n", *address, root)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
