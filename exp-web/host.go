package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const hostFileName = ".host.json"
const hostKind = "elab-web-host"

type hostInfo struct {
	Version  int    `json:"version"`
	Kind     string `json:"kind"`
	URL      string `json:"url"`
	Root     string `json:"root"`
	Instance string `json:"instance"`
	Token    string `json:"token"`
}
type launchInfo struct {
	hostInfo
	Owned bool `json:"owned"`
}
type hostRuntime struct {
	info   hostInfo
	cancel context.CancelFunc
}
type hostOptions struct {
	address, root, startupFile string
	autoPort, dev              bool
}

func validateHostInfo(info hostInfo) error {
	u, err := url.Parse(info.URL)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.Path != "/" || u.RawPath != "" {
		return fmt.Errorf("invalid local ELab address")
	}
	if u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "::1" {
		return fmt.Errorf("ELab host must use loopback")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 || info.Version != 1 || info.Kind != hostKind || !managedID.MatchString(info.Token) || !managedID.MatchString(info.Instance) || !filepath.IsAbs(info.Root) {
		return fmt.Errorf("invalid ELab host information")
	}
	return nil
}
func readHostInfo(path string) (hostInfo, error) {
	data, err := readBounded(path, 16384)
	if err != nil {
		return hostInfo{}, err
	}
	var info hostInfo
	if err = json.Unmarshal(data, &info); err != nil {
		return info, err
	}
	return info, validateHostInfo(info)
}
func writeLaunchInfo(path string, info hostInfo, owned bool) error {
	if path == "" {
		return nil
	}
	data, err := json.Marshal(launchInfo{hostInfo: info, Owned: owned})
	if err != nil {
		return err
	}
	// Never follow/replace a pre-existing launch reply or link.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(append(data, '\n'))
	syncErr := file.Sync()
	closeErr := file.Close()
	return errors.Join(writeErr, syncErr, closeErr)
}
func hostRequest(ctx context.Context, info hostInfo, method, action string) error {
	if err := validateHostInfo(info); err != nil {
		return err
	}
	// A control token must never be sent to an environment proxy or redirect.
	transport := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 2 * time.Second}).DialContext}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(ctx, method, info.URL+"api/host/"+action, bytes.NewReader(nil))
	if err != nil {
		return err
	}
	req.Header.Set("X-ELab-Host-Token", info.Token)
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusAccepted {
		var failure struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(http.MaxBytesReader(nil, response.Body, 4096)).Decode(&failure)
		return fmt.Errorf("ELab refused %s (HTTP %d): %s", action, response.StatusCode, failure.Error)
	}
	if action == "status" {
		var status struct {
			Kind     string `json:"kind"`
			Instance string `json:"instance"`
			Root     string `json:"root"`
		}
		if err = json.NewDecoder(http.MaxBytesReader(nil, response.Body, 16384)).Decode(&status); err != nil {
			return err
		}
		if status.Kind != hostKind || status.Instance != info.Instance || status.Root != info.Root {
			return fmt.Errorf("ELab host identity changed")
		}
	}
	return nil
}
func stopHost(ctx context.Context, info hostInfo) error {
	if err := hostRequest(ctx, info, http.MethodGet, "status"); err != nil {
		// A dead host may leave metadata behind. Verify the library is no longer
		// locked before treating this as an already-completed stop.
		if unlock, lockErr := lockStore(info.Root); lockErr == nil {
			unlock()
			return nil
		}
		return err
	}
	if err := hostRequest(ctx, info, http.MethodPost, "stop"); err != nil {
		return err
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		current, err := readHostInfo(filepath.Join(info.Root, hostFileName))
		if os.IsNotExist(err) || err == nil && current.Instance != info.Instance {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("ELab is still shutting down: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}
func (s *webServer) hostControl(w http.ResponseWriter, r *http.Request) {
	host := s.host
	if host == nil || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-ELab-Host-Token")), []byte(host.info.Token)) != 1 {
		webError(w, 404, "Not found")
		return
	}
	switch {
	case r.URL.Path == "/api/host/status" && r.Method == http.MethodGet:
		jsonReply(w, 200, map[string]any{"kind": hostKind, "version": 1, "instance": host.info.Instance, "root": host.info.Root})
	case r.URL.Path == "/api/host/stop" && r.Method == http.MethodPost:
		if err := s.checkpointForStop(); err != nil {
			webError(w, 409, "ELab could not save its projects. Keep it open and check storage: "+err.Error())
			return
		}
		jsonReply(w, 202, map[string]string{"status": "stopping"})
		host.cancel()
	default:
		webError(w, 405, "Unsupported host action")
	}
}
func (s *webServer) checkpointForStop() error {
	// Drain commands and prevent new requests from crossing the final checkpoint.
	s.lifecycle.Lock()
	defer s.lifecycle.Unlock()
	if s.stopping {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ws := range s.sessions {
		ws.files.Lock()
		err := ws.persist()
		ws.files.Unlock()
		if err != nil {
			return err
		}
	}
	s.stopping = true
	return nil
}
func resolveHostRoot(root string) (string, error) {
	if root == "" {
		config, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(config, "ELab", "web")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(absolute, 0700); err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}
func listenHost(address string, autoPort bool) (net.Listener, string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, "", err
	}
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return nil, "", fmt.Errorf("ELab Web requires localhost, 127.0.0.1 or ::1")
	}
	number, parseErr := strconv.Atoi(port)
	if parseErr != nil || number < 0 || number > 65535 {
		return nil, "", fmt.Errorf("ELab requires a numeric port from 0 to 65535")
	}
	listener, err := net.Listen("tcp", address)
	if err != nil && autoPort {
		var operation *net.OpError
		// Fallback is safe for a busy preferred port; invalid addresses still fail.
		if _, parseErr := strconv.Atoi(port); parseErr == nil && errors.As(err, &operation) {
			listener, err = net.Listen("tcp", net.JoinHostPort(host, "0"))
		}
	}
	if err != nil {
		return nil, "", fmt.Errorf("ELab could not open local address %s: %w", address, err)
	}
	actualPort := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	return listener, "http://" + net.JoinHostPort(host, actualPort) + "/", nil
}
func runWebHost(ctx context.Context, options hostOptions) error {
	root, err := resolveHostRoot(options.root)
	if err != nil {
		return err
	}
	unlock, err := lockStore(root)
	if err != nil {
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) && !os.IsExist(err) {
			return fmt.Errorf("ELab could not access project storage: %w", err)
		}
		if options.startupFile != "" {
			if info, readErr := readHostInfo(filepath.Join(root, hostFileName)); readErr == nil && info.Root == root {
				probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				probeErr := hostRequest(probeCtx, info, http.MethodGet, "status")
				cancel()
				if probeErr == nil {
					return writeLaunchInfo(options.startupFile, info, false)
				}
			}
		}
		return fmt.Errorf("another ELab server is using this project library; close that server before trying again: %w", err)
	}
	defer unlock()
	listener, address, err := listenHost(options.address, options.autoPort)
	if err != nil {
		return err
	}
	defer listener.Close()
	web := newWebServer(root)
	// Stop native workers before the final capture; no late result is published
	// after the host's last persistence attempt.
	if options.dev {
		web.assets = os.DirFS("exp-web/static")
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	info := hostInfo{Version: 1, Kind: hostKind, URL: address, Root: root, Instance: web.instance, Token: token()}
	web.host = &hostRuntime{info: info, cancel: cancel}
	server := &http.Server{Handler: web.handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 90 * time.Second}
	if err = writeJSON(filepath.Join(root, hostFileName), info); err != nil {
		return err
	}
	var workers sync.WaitGroup
	defer func() {
		cancel()
		workers.Wait()
		web.close()
		if got, e := readHostInfo(filepath.Join(root, hostFileName)); e == nil && got.Instance == info.Instance {
			_ = os.Remove(filepath.Join(root, hostFileName))
		}
	}()
	if err = writeLaunchInfo(options.startupFile, info, true); err != nil {
		return err
	}
	workers.Add(2)
	go func() { defer workers.Done(); web.expire(runCtx) }()
	go func() { defer workers.Done(); web.autosave(runCtx) }()
	serving := make(chan error, 1)
	go func() { serving <- server.Serve(listener) }()
	fmt.Printf("ELab Web → %s\nProjects & recovery → %s\n", address, root)
	select {
	case err = <-serving:
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
	case <-runCtx.Done():
	}
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	shutdownErr := server.Shutdown(shutdown)
	cancelShutdown()
	if shutdownErr != nil {
		_ = server.Close()
	}
	return errors.Join(err, shutdownErr)
}
