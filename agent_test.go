package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dshills/elab/internal/webagent"
	"github.com/dshills/elab/workbench"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func browserAgentFixture(t *testing.T) (*webFixture, string) {
	t.Helper()
	f := fixture(t)
	host := httptest.NewServer(f.handler)
	f.server.host = &hostRuntime{info: hostInfo{URL: host.URL + "/"}}
	t.Cleanup(host.Close)
	response := f.request("POST", "/api/agent", []byte(`{"enabled":true}`), "")
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	path := filepath.Join(f.server.root, "workspaces", f.cookie.Value, workspaceAgentFile)
	var endpoint webagent.Endpoint
	if data, err := os.ReadFile(path); err != nil || json.Unmarshal(data, &endpoint) != nil {
		t.Fatal("missing private workspace descriptor", err)
	}
	if strings.Contains(response.Body.String(), endpoint.Token) {
		t.Fatal("browser configuration exposed a credential")
	}
	return f, path
}

func agentRequest(t *testing.T, f *webFixture, id, value string) workbench.Request {
	t.Helper()
	s := f.state()
	params, _ := json.Marshal(workbench.ValueParams{ID: "R1", Value: value})
	return workbench.Request{Version: 1, ID: id, Key: id, Command: "component.set", Params: params,
		ExpectedCommandRevision: &s.CommandRevision, ExpectedDocumentGeneration: &s.Document.Generation}
}

func sendAgent(t *testing.T, client *webagent.Client, request workbench.Request) workbench.Response {
	t.Helper()
	data, _ := json.Marshal(request)
	reply, err := client.Send(t.Context(), data)
	if err != nil {
		t.Fatal(err)
	}
	var result workbench.Response
	if err = json.Unmarshal(reply, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestBrowserAgentEditsAreSharedCheckpointedGuardedAndRevocable(t *testing.T) {
	f, path := browserAgentFixture(t)
	client, err := webagent.New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	request := agentRequest(t, f, "agent-edit", "22k")
	if reply := sendAgent(t, client, request); reply.Error != nil {
		t.Fatal(reply.Error)
	}
	if f.state().Components[1].ID == "" {
		t.Fatal("missing browser circuit")
	}
	value := func() float64 {
		for _, p := range f.state().Components {
			if p.ID == "R1" {
				return p.Value
			}
		}
		return 0
	}
	if value() != 22000 {
		t.Fatal("agent edit is absent in browser state")
	}
	checkpoint, err := readCheckpoint(filepath.Join(filepath.Dir(path), "workspace.json"))
	if err != nil || !strings.Contains(string(checkpoint.Document), "22000") {
		t.Fatal("agent edit was not checkpointed", err)
	}
	request.ID, request.Key = "stale", "stale"
	if r := sendAgent(t, client, request); r.Error == nil || r.Error.Code != "conflict" {
		t.Fatal("stale edit accepted", r.Error)
	}
	missing := agentRequest(t, f, "no-generation", "3k")
	missing.ExpectedDocumentGeneration = nil
	if r := sendAgent(t, client, missing); r.Error == nil || r.Error.Code != "invalid-request" {
		t.Fatal("missing generation admitted", r.Error)
	}
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "33k"})
	request.ID, request.Key = "agent-edit", "agent-edit"
	if r := sendAgent(t, client, request); !r.Duplicate || value() != 33000 {
		t.Fatal("retry repeated an edit", r.Error)
	}
	for _, name := range []string{"app.close", "project.save", "recovery.configure", "print.submit"} {
		r := agentRequest(t, f, "blocked-"+name, "4k")
		r.Command = name
		if reply := sendAgent(t, client, r); reply.Error == nil {
			t.Fatal("host/file command admitted", name)
		}
	}
	response := f.request("POST", "/api/agent", []byte(`{"enabled":false}`), "")
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("revoked descriptor remains")
	}
	if r := sendAgent(t, client, agentRequest(t, f, "after-revoke", "4k")); r.Error == nil || r.Error.Code != "agent-disconnected" {
		t.Fatal("revoked client admitted", r.Error)
	}
	f.request("POST", "/api/agent", []byte(`{"enabled":true}`), "")
	if r := sendAgent(t, client, agentRequest(t, f, "old-token", "4k")); r.Error == nil {
		t.Fatal("old bridge silently reattached")
	}
	if value() != 33000 || f.state().ReadyToExit() {
		t.Fatal("revocation changed or closed the circuit")
	}
}

func TestBrowserAgentCannotSelectAnotherWorkspaceAndFiltersDiscovery(t *testing.T) {
	f, path := browserAgentFixture(t)
	data, _ := os.ReadFile(path)
	var endpoint webagent.Endpoint
	_ = json.Unmarshal(data, &endpoint)
	other := &webFixture{server: f.server, handler: f.handler, t: t}
	other.request("GET", "/api/state", nil, "")
	endpoint.Workspace = other.cookie.Value
	wrong := filepath.Join(t.TempDir(), "wrong.json")
	if err := writeJSON(wrong, endpoint); err != nil {
		t.Fatal(err)
	}
	client, err := webagent.New(wrong)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if r := sendAgent(t, client, agentRequest(t, other, "wrong-workspace", "4k")); r.Error == nil {
		t.Fatal("credential crossed workspace boundary")
	}
	client, err = webagent.New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	r := sendAgent(t, client, workbench.Request{Version: 1, ID: "catalog", Command: "capabilities.list", Params: json.RawMessage(`{}`)})
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	encoded, _ := json.Marshal(r.Result)
	var result struct {
		Capabilities []workbench.Capability `json:"capabilities"`
	}
	if json.Unmarshal(encoded, &result) != nil || len(result.Capabilities) == 0 {
		t.Fatal("empty discovery")
	}
	for _, capability := range result.Capabilities {
		if !permitted(capability.ID) {
			t.Fatal("unavailable command advertised", capability.ID)
		}
	}
}

func TestBrowserAgentDisconnectDoesNotAdmitQueuedEdits(t *testing.T) {
	f, path := browserAgentFixture(t)
	client, err := webagent.New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	req := agentRequest(t, f, "concurrent", "44k")
	data, _ := json.Marshal(req)
	var workers sync.WaitGroup
	for i := 0; i < 12; i++ {
		workers.Add(1)
		go func() { defer workers.Done(); _, _ = client.Send(t.Context(), data) }()
	}
	if r := f.request("POST", "/api/agent", []byte(`{"enabled":false}`), ""); r.Code != 200 {
		t.Fatal(r.Body.String())
	}
	before := f.state().CommandRevision
	workers.Wait()
	if after := f.state().CommandRevision; after != before {
		t.Fatal("an edit crossed completed revocation", before, after)
	}
}

func TestBrowserAgentRevokesEvenIfDescriptorWasReplaced(t *testing.T) {
	f, path := browserAgentFixture(t)
	client, err := webagent.New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err = os.WriteFile(path, []byte(`{"unrelated":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	r := f.request("POST", "/api/agent", []byte(`{"enabled":false}`), "")
	if r.Code != 409 {
		t.Fatal("cleanup conflict missing", r.Code)
	}
	if r := sendAgent(t, client, agentRequest(t, f, "revoked-despite-cleanup", "3k")); r.Error == nil || r.Error.Code != "agent-disconnected" {
		t.Fatal("cleanup failure retained authority", r.Error)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != `{"unrelated":true}` {
		t.Fatal("unrelated file was removed", err)
	}
}

func TestBrowserAgentReportsAppliedButUncheckpointedEditAndCanRetryCheckpoint(t *testing.T) {
	f, path := browserAgentFixture(t)
	client, err := webagent.New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	checkpoint := filepath.Join(filepath.Dir(path), "workspace.json")
	if err = os.Remove(checkpoint); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(checkpoint, 0700); err != nil {
		t.Fatal(err)
	}
	req := agentRequest(t, f, "checkpoint-retry", "22k")
	if r := sendAgent(t, client, req); r.Error == nil || r.Error.Code != "checkpoint-failed" {
		t.Fatal("uncheckpointed edit reported successful", r.Error)
	}
	if err = os.Remove(checkpoint); err != nil {
		t.Fatal(err)
	}
	if r := sendAgent(t, client, req); r.Error != nil || !r.Duplicate {
		t.Fatal("retry did not checkpoint without reapplying", r.Error)
	}
	if _, err = readCheckpoint(checkpoint); err != nil {
		t.Fatal("retry checkpoint is invalid", err)
	}
}

func TestPackagedHostMCPModeUsesTheVisibleWorkspaceAndLeavesItRunning(t *testing.T) {
	f, path := browserAgentFixture(t)
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "elab-web")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build embedded bridge: %v\n%s", err, output)
	}
	process := exec.CommandContext(ctx, binary, "--mcp-workspace", path)
	client, err := mcp.NewClient(&mcp.Implementation{Name: "web-reference", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: process}, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := agentRequest(t, f, "stdio-shared", "47k")
	result, err := client.CallTool(ctx, &mcp.CallToolParams{Name: "elab_command", Arguments: req})
	if err != nil || result.IsError {
		t.Fatal("embedded MCP edit failed", err, result)
	}
	if err = client.Close(); err != nil {
		t.Fatal(err)
	}
	tip := f.state().UndoEntry
	if tip == nil || tip.Actor != "agent.web" {
		t.Fatal("agent history attribution lost")
	}
	f.command("edit.undo", workbench.HistoryParams{ExpectedEntryRevision: &tip.CommandRevision, AllowOtherActor: true})
	for _, part := range f.state().Components {
		if part.ID == "R1" && part.Value != 10000 {
			t.Fatal("browser undo did not see the agent's edit", part.Value)
		}
	}
	if f.state().ReadyToExit() {
		t.Fatal("bridge close stopped borrowed workspace")
	}
	bad := exec.CommandContext(ctx, binary, "--mcp-workspace", filepath.Join(t.TempDir(), "missing"))
	if err = bad.Run(); err == nil {
		t.Fatal("missing descriptor fell back to a host")
	}
}
