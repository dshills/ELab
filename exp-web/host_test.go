package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func runningHost(t *testing.T, options hostOptions) (launchInfo, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runWebHost(ctx, options); close(done) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("owned host failed to stop")
		}
	})
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(options.startupFile); err == nil {
			var reply launchInfo
			if json.Unmarshal(data, &reply) == nil && reply.URL != "" {
				if err = hostRequest(context.Background(), reply.hostInfo, http.MethodGet, "status"); err != nil {
					t.Fatal(err)
				}
				return reply, done
			}
		}
		select {
		case err := <-done:
			t.Fatalf("host exited during startup: %v", err)
		default:
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("host did not become ready")
	return launchInfo{}, done
}
func hostState(t *testing.T, client *http.Client, info hostInfo) workbench.Snapshot {
	t.Helper()
	response, err := client.Get(info.URL + "api/state")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var data struct{ State workbench.Snapshot }
	if response.StatusCode != 200 || json.NewDecoder(response.Body).Decode(&data) != nil {
		t.Fatal("state unavailable", response.StatusCode)
	}
	return data.State
}
func TestHostPortConflictReuseStopAndRecovery(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	root := t.TempDir()
	replyPath := filepath.Join(t.TempDir(), "launch.json")
	info, done := runningHost(t, hostOptions{address: busy.Addr().String(), root: root, autoPort: true, startupFile: replyPath})
	if !info.Owned || strings.Contains(info.URL, busy.Addr().String()) {
		t.Fatal("did not choose an unused port", info.URL)
	}
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}
	before := hostState(t, client, info.hostInfo)
	params, _ := json.Marshal(workbench.ValueParams{ID: "R1", Value: "22k"})
	body, _ := json.Marshal(workbench.Request{Version: 1, ID: token(), Key: token(), Command: "component.set", Params: params, ExpectedCommandRevision: &before.CommandRevision, ExpectedDocumentGeneration: &before.Document.Generation})
	response, err := client.Post(info.URL+"api/command", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal("edit failed", response.StatusCode)
	}
	second := filepath.Join(t.TempDir(), "second.json")
	if err = runWebHost(context.Background(), hostOptions{address: busy.Addr().String(), root: root, autoPort: true, startupFile: second}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(second)
	var reused launchInfo
	if json.Unmarshal(data, &reused) != nil || reused.Owned || reused.Instance != info.Instance || reused.URL != info.URL {
		t.Fatal("duplicate launch did not reuse the owner")
	}
	if mode, err := os.Stat(second); err != nil || mode.Mode().Perm() != 0600 {
		t.Fatal("launch credentials are not private")
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = stopHost(stopCtx, reused.hostInfo); err != nil {
		t.Fatal(err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, hostFileName)); !os.IsNotExist(err) {
		t.Fatal("stale owned metadata remains", err)
	}
	// A second Quit is harmless when the previous owner is already gone.
	if err = stopHost(stopCtx, reused.hostInfo); err != nil {
		t.Fatal("already-stopped host was not handled", err)
	}
	next, _ := runningHost(t, hostOptions{address: busy.Addr().String(), root: root, autoPort: true, startupFile: filepath.Join(t.TempDir(), "restart.json")})
	restored := hostState(t, client, next.hostInfo)
	found := false
	for _, p := range restored.Components {
		if p.ID == "R1" && p.Value == 22000 {
			found = true
		}
	}
	if !found || !restored.Document.Dirty {
		t.Fatal("normal quit/restart lost acknowledged edits")
	}
}
func TestHostStopRefusesFailedCheckpointAndRequiresPrivateToken(t *testing.T) {
	f := fixture(t)
	cancelled := false
	info := hostInfo{Version: 1, Kind: hostKind, URL: "http://127.0.0.1:8787/", Root: f.server.root, Instance: token(), Token: token()}
	f.server.host = &hostRuntime{info: info, cancel: func() { cancelled = true }}
	call := func(method, action, secret string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, info.URL+"api/host/"+action, nil)
		r.Header.Set("X-ELab-Host-Token", secret)
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		return w
	}
	if w := call("POST", "stop", ""); w.Code != 404 || cancelled {
		t.Fatal("unprivileged stop accepted")
	}
	if w := call("GET", "status", info.Token); w.Code != 200 || strings.Contains(w.Body.String(), info.Token) {
		t.Fatal("status leaked credentials or failed")
	}
	f.server.mu.Lock()
	ws := f.server.sessions[f.cookie.Value]
	f.server.mu.Unlock()
	old := ws.directory
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(blocked, []byte("block"), 0600); err != nil {
		t.Fatal(err)
	}
	ws.directory = filepath.Join(blocked, "missing")
	ws.persisted = ""
	if w := call("POST", "stop", info.Token); w.Code != 409 || cancelled || f.server.stopping {
		t.Fatal("quit discarded a failed checkpoint", w.Body.String())
	}
	if w := f.request("GET", "/api/state", nil, ""); w.Code != 200 {
		t.Fatal("failed stop disabled the editor")
	}
	ws.directory = old
	if w := call("POST", "stop", info.Token); w.Code != 202 || !cancelled {
		t.Fatal("repaired stop did not complete", w.Body.String())
	}
	if w := f.request("GET", "/api/state", nil, ""); w.Code != 503 {
		t.Fatal("command gate stayed open after final checkpoint")
	}
}
func TestHostCredentialsRejectUnsafeAddressesLinksAndRedirects(t *testing.T) {
	info := hostInfo{Version: 1, Kind: hostKind, URL: "http://127.0.0.1:12345/", Root: t.TempDir(), Instance: token(), Token: token()}
	for _, address := range []string{"http://example.com/", "https://localhost:12345/", "http://user@localhost:12345/", "http://localhost:12345/?token=x", "http://localhost:12345/path", "http://localhost:12345/#fragment"} {
		changed := info
		changed.URL = address
		if validateHostInfo(changed) == nil {
			t.Error("unsafe control address accepted", address)
		}
	}
	var reached atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached.Add(1) }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer redirect.Close()
	info.URL = redirect.URL + "/"
	if hostRequest(context.Background(), info, "GET", "status") == nil || reached.Load() != 0 {
		t.Fatal("private control followed a redirect")
	}
	directory := t.TempDir()
	victim := filepath.Join(directory, "victim")
	link := filepath.Join(directory, "reply")
	if err := os.WriteFile(victim, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, link); err != nil {
		t.Fatal(err)
	}
	if writeLaunchInfo(link, info, true) == nil {
		t.Fatal("launch reply followed existing link")
	}
	data, _ := os.ReadFile(victim)
	if string(data) != "preserve" {
		t.Fatal("victim changed")
	}
}

func TestHostStorageFailureDoesNotClaimAnotherServer(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".lock"), 0700); err != nil {
		t.Fatal(err)
	}
	err := runWebHost(context.Background(), hostOptions{root: root, address: "127.0.0.1:0"})
	if err == nil || !strings.Contains(err.Error(), "could not access project storage") || strings.Contains(err.Error(), "another ELab") {
		t.Fatal("misleading storage failure", err)
	}
}
