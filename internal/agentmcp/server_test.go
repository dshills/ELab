package agentmcp

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dshills/elab/transport"
	"github.com/dshills/elab/workbench"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type reply struct {
	Version   int                `json:"version"`
	Result    json.RawMessage    `json:"result"`
	Error     *workbench.Failure `json:"error"`
	Duplicate bool               `json:"duplicate"`
}

func connected(t *testing.T, d Dispatch, version string) *mcp.ClientSession {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	a, b := mcp.NewInMemoryTransports()
	done := make(chan error, 1)
	go func() { done <- NewServer(d).Run(ctx, a) }()
	c, err := mcp.NewClient(&mcp.Implementation{Name: "ELab-reference-client", Version: "1"}, nil).Connect(ctx, b, &mcp.ClientSessionOptions{ProtocolVersion: version})
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = c.Close()
		cancel()
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Error(err)
			}
		case <-time.After(3 * time.Second):
			t.Error("MCP server did not close")
		}
	})
	return c
}
func tool(t *testing.T, c *mcp.ClientSession, name string, args any) reply {
	t.Helper()
	r, err := c.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Content) != 1 {
		t.Fatal("missing JSON fallback", r)
	}
	text, ok := r.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("non-text result")
	}
	var out reply
	if err = json.Unmarshal([]byte(text.Text), &out); err != nil {
		t.Fatal(err)
	}
	if out.Version != 1 || r.IsError != (out.Error != nil) {
		t.Fatal("structured error lost", r, out.Error)
	}
	// Structured data and fallback must encode the exact same response.
	raw, err := json.Marshal(r.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var structured, unstructured any
	if json.Unmarshal(raw, &structured) != nil || json.Unmarshal([]byte(text.Text), &unstructured) != nil || !reflect.DeepEqual(structured, unstructured) {
		t.Fatal("structured and fallback results differ")
	}
	return out
}
func snapshot(t *testing.T, r reply) workbench.Snapshot {
	t.Helper()
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	var s workbench.Snapshot
	if err := json.Unmarshal(r.Result, &s); err != nil {
		t.Fatal(err)
	}
	return s
}
func params(v any) json.RawMessage { raw, _ := json.Marshal(v); return raw }

func TestMCPSameCatalogCommandsGuardsAndRetries(t *testing.T) {
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, remote := range []bool{false, true} {
			name := version + "/local"
			if remote {
				name = version + "/socket"
			}
			t.Run(name, func(t *testing.T) {
				s := workbench.New()
				t.Cleanup(s.Close)
				dispatch := Local(s)
				if remote {
					path := filepath.Join(t.TempDir(), "endpoint.json")
					host, err := transport.Listen(s, path)
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(host.Close)
					r, err := transport.NewRemoteClient(path)
					if err != nil {
						t.Fatal(err)
					}
					dispatch = r.Send
				}
				c := connected(t, dispatch, version)
				tools, err := c.ListTools(t.Context(), nil)
				if err != nil || len(tools.Tools) != 3 {
					t.Fatal("bad tool catalog", err)
				}
				catalog := tool(t, c, "elab_discover", map[string]any{})
				var found struct {
					Capabilities []workbench.Capability `json:"capabilities"`
				}
				if json.Unmarshal(catalog.Result, &found) != nil || len(found.Capabilities) != len(workbench.Catalog()) {
					t.Fatal("incomplete public catalog")
				}
				detail := tool(t, c, "elab_discover", map[string]any{"command": "component.set"})
				if json.Unmarshal(detail.Result, &found) != nil || len(found.Capabilities) != 1 || found.Capabilities[0].ID != "component.set" {
					t.Fatal("missing command schema")
				}
				before := snapshot(t, tool(t, c, "elab_inspect", map[string]any{}))
				invalid := tool(t, c, "elab_command", workbench.Request{Version: 1, ID: "unguarded", Command: "example.load", Params: params(workbench.ExampleLoadParams{ID: "led"})})
				if invalid.Error == nil || invalid.Error.Code != "invalid-request" || !reflect.DeepEqual(before.Components, s.Snapshot().Components) {
					t.Fatal("adapter supplied missing guards")
				}
				serial := 0
				mutate := func(command string, p any) (workbench.Request, reply) {
					t.Helper()
					serial++
					at := s.Snapshot()
					req := workbench.Request{Version: 1, ID: strings.Repeat("x", serial), Key: strings.Repeat("k", serial), Command: command, Params: params(p), ExpectedCommandRevision: &at.CommandRevision, ExpectedDocumentGeneration: &at.Document.Generation}
					r := tool(t, c, "elab_command", req)
					if r.Error != nil {
						t.Fatal(command, r.Error)
					}
					return req, r
				}
				mutate("example.load", workbench.ExampleLoadParams{ID: "led"})
				mutate("simulation.pause", struct{}{})
				original, _ := mutate("component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
				mutate("component.set", workbench.ValueParams{ID: "R1", Value: "3k"})
				current := s.Snapshot()
				cached := tool(t, c, "elab_command", original)
				if !cached.Duplicate || !reflect.DeepEqual(current, s.Snapshot()) {
					t.Fatal("retry re-executed a retired edit")
				}
				old := original
				old.ID = "stale"
				old.Key = "fresh-stale-key"
				refused := tool(t, c, "elab_command", old)
				if refused.Error == nil || refused.Error.Code != "conflict" || !reflect.DeepEqual(current, s.Snapshot()) {
					t.Fatal("stale revision was refreshed")
				}
				mutate("example.load", workbench.ExampleLoadParams{ID: "rc", DiscardChanges: true})
				mutate("simulation.pause", struct{}{})
				replaced, err := s.Wait(t.Context())
				if err != nil {
					t.Fatal(err)
				}
				old.Key = "old-document"
				old.ExpectedCommandRevision = &replaced.CommandRevision
				refused = tool(t, c, "elab_command", old)
				if refused.Error == nil || refused.Error.Code != "conflict" || !reflect.DeepEqual(replaced, s.Snapshot()) {
					t.Fatal("document guard was lost")
				}
				r := tool(t, c, "elab_command", json.RawMessage(`{"version":1,"id":"bad","command":"project.inspect","params":{},"params":{}}`))
				if r.Error == nil || r.Error.Code != "invalid-json" {
					t.Fatal("duplicate fields were normalized")
				}
				if remote {
					_ = c.Close()
					if s.Snapshot().Closed {
						t.Fatal("MCP disconnect closed its host")
					}
				}
			})
		}
	}
}

func TestMCPInvalidReadonlyAndOversizeNeverDispatch(t *testing.T) {
	calls := 0
	c := connected(t, func(context.Context, []byte) ([]byte, error) {
		calls++
		return []byte(`{"version":1,"id":"ok","revision":1}`), nil
	}, "2025-11-25")
	for _, name := range []string{"elab_discover", "elab_inspect"} {
		r := tool(t, c, name, map[string]any{"unexpected": true})
		if r.Error == nil || r.Error.Code != "invalid-params" {
			t.Fatal("unknown readonly parameters accepted")
		}
	}
	r := tool(t, c, "elab_command", map[string]any{"payload": strings.Repeat("x", transport.MaxRequestBytes+1)})
	if r.Error == nil || r.Error.Code != "request-too-large" || calls != 0 {
		t.Fatal("oversized or invalid query reached service")
	}
}

func TestMCPBackendResponsesRemainBoundedAndErrorsObservable(t *testing.T) {
	for _, raw := range [][]byte{[]byte(`not JSON`), []byte(`{"version":2}`), []byte(strings.Repeat("x", transport.MaxResponseBytes+1)), []byte(`{"version":1,"error":{"code":"conflict","message":"keep original intent"}}`)} {
		t.Run(string(raw[:min(len(raw), 20)]), func(t *testing.T) {
			c := connected(t, func(context.Context, []byte) ([]byte, error) { return raw, nil }, "2025-11-25")
			r := tool(t, c, "elab_inspect", map[string]any{})
			if r.Error == nil {
				t.Fatal("bad backend response reported successful")
			}
		})
	}
}

func TestMCPConcurrentAdmissionBound(t *testing.T) {
	ready := make(chan struct{}, 8)
	release := make(chan struct{})
	closed := false
	defer func() {
		if !closed {
			close(release)
		}
	}()
	c := connected(t, func(ctx context.Context, line []byte) ([]byte, error) {
		ready <- struct{}{}
		select {
		case <-release:
			return []byte(`{"version":1,"id":"ok","revision":1}`), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}, "2025-11-25")
	results := make(chan error, 8)
	for range 8 {
		go func() {
			_, err := c.CallTool(t.Context(), &mcp.CallToolParams{Name: "elab_inspect", Arguments: map[string]any{}})
			results <- err
		}()
	}
	for range 8 {
		select {
		case <-ready:
		case <-time.After(3 * time.Second):
			t.Fatal("eight in-flight calls did not start")
		}
	}
	refused := tool(t, c, "elab_inspect", map[string]any{})
	if refused.Error == nil || refused.Error.Code != "adapter-busy" {
		t.Fatal("adapter admitted a ninth in-flight command")
	}
	close(release)
	closed = true
	for range 8 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
}

func TestMCPTransportFailureDoesNotRetryOrAssumeNoMutation(t *testing.T) {
	var calls atomic.Int32
	c := connected(t, func(context.Context, []byte) ([]byte, error) {
		calls.Add(1)
		return nil, errors.New("controlled unavailable host")
	}, "2025-11-25")
	r := tool(t, c, "elab_command", map[string]any{"version": 1, "id": "uncertain", "command": "project.inspect", "params": map[string]any{}})
	if r.Error == nil || r.Error.Code != "transport-error" || !strings.Contains(r.Error.Message, "outcome may be unknown") || calls.Load() != 1 {
		t.Fatal("transport failure was hidden or retried", r.Error, calls.Load())
	}
}
