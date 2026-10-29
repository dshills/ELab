package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func TestRemoteClientPinsExplicitHostAndDoesNotReconnect(t *testing.T) {
	path := filepath.Join(t.TempDir(), "host.json")
	s := workbench.New()
	defer s.Close()
	first, err := Listen(s, path)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	client, err := NewRemoteClient(path)
	if err != nil {
		t.Fatal(err)
	}
	line := []byte(`{"version":1,"id":"inspect","command":"project.inspect","params":{}}`)
	if _, err = client.Send(t.Context(), line); err != nil {
		t.Fatal(err)
	}
	first.Close()
	replacement := workbench.New()
	defer replacement.Close()
	second, err := Listen(replacement, path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if _, err = client.Send(t.Context(), line); err == nil {
		t.Fatal("retired client borrowed a replacement host")
	}
	fresh, err := NewRemoteClient(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fresh.Send(t.Context(), line); err != nil {
		t.Fatal(err)
	}
	if _, err = fresh.Send(t.Context(), append(line, '\n')); err == nil {
		t.Fatal("embedded line separator accepted")
	}
}

func TestRemoteClientCancellationInterruptsBlockedResponse(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "elab-remote-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "api.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	data, _ := json.Marshal(Endpoint{Version: 1, Socket: socket, Token: strings.Repeat("a", 64)})
	descriptor := filepath.Join(dir, "endpoint.json")
	if err = os.WriteFile(descriptor, data, 0600); err != nil {
		t.Fatal(err)
	}
	client, err := NewRemoteClient(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	ready := make(chan struct{})
	release := make(chan struct{})
	serverDone := make(chan struct{})
	defer func() {
		close(release)
		_ = listener.Close()
		select {
		case <-serverDone:
		case <-time.After(3 * time.Second):
			t.Error("owned socket fixture did not close")
		}
	}()
	go func() {
		defer close(serverDone)
		c, e := listener.Accept()
		if e != nil {
			return
		}
		defer c.Close()
		r := bufio.NewReader(c)
		if _, e = r.ReadString('\n'); e != nil {
			return
		}
		if _, e = r.ReadString('\n'); e != nil {
			return
		}
		close(ready)
		<-release
	}()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, e := client.Send(ctx, []byte(`{}`)); done <- e }()
	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		t.Fatal("request did not arrive")
	}
	cancel()
	select {
	case err = <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal("lost cancellation", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("canceled read remained blocked")
	}
}

func TestRemoteClientDescriptorIsBounded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "endpoint.json")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", MaxRequestBytes+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewRemoteClient(path); err == nil {
		t.Fatal("unbounded endpoint accepted")
	}
}
