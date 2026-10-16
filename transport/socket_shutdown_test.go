package transport

import (
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func attachShutdownPipe(t *testing.T, service *workbench.Service) (*Server, net.Conn) {
	t.Helper()
	server, err := Listen(service, filepath.Join(t.TempDir(), "endpoint.json"))
	if err != nil {
		t.Fatal(err)
	}
	client, connection := net.Pipe()
	t.Cleanup(func() { client.Close(); server.Close() })
	server.mu.Lock()
	server.clients[connection] = true
	server.mu.Unlock()
	server.wg.Add(1)
	go func() {
		defer server.wg.Done()
		defer connection.Close()
		defer func() { server.mu.Lock(); delete(server.clients, connection); server.mu.Unlock() }()
		server.serveConnection(t.Context(), service, "owned-test-token", connection)
	}()
	client.SetDeadline(time.Now().Add(2 * time.Second))
	return server, client
}

func TestSocketShutdownDrainsAdmittedReplyBeforeClosingConnections(t *testing.T) {
	s := workbench.New()
	defer s.Close()
	server, client := attachShutdownPipe(t, s)
	revision := s.Snapshot().CommandRevision
	request := workbench.Request{Version: 1, ID: "close-reply", Command: "app.close",
		Key: "close-reply", ExpectedCommandRevision: &revision, Params: json.RawMessage(`{}`)}
	if _, err := fmt.Fprintln(client, "owned-test-token"); err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(client).Encode(request); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(2 * time.Second)
	for !s.Snapshot().ReadyToExit() {
		select {
		case <-deadline:
			t.Fatal("owned shutdown did not complete")
		case <-time.After(time.Millisecond):
		}
	}
	// net.Pipe has no response buffer: the production writer cannot finish
	// until this test reads. Domain readiness alone must not truncate it.
	closed := make(chan struct{})
	go func() { server.Close(); close(closed) }()
	select {
	case <-closed:
		t.Fatal("socket closed before the admitted shutdown reply was read")
	case <-time.After(20 * time.Millisecond):
	}
	var reply struct {
		ID     string
		Result workbench.Snapshot
		Error  *workbench.Failure
	}
	if err := json.NewDecoder(client).Decode(&reply); err != nil || reply.Error != nil || reply.ID != request.ID || !reply.Result.ReadyToExit() {
		t.Fatal("shutdown reply was truncated or canceled", err, reply.Error, reply.ID)
	}
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("socket did not finish cleanup after flushing the reply")
	}
}

func TestSocketForcedCloseStillReleasesIdleConnections(t *testing.T) {
	s := workbench.New()
	defer s.Close()
	server, _ := attachShutdownPipe(t, s)
	closed := make(chan struct{})
	go func() { server.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("ordinary forced cleanup waited for an unauthenticated client")
	}
}
