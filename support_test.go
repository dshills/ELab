package main

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestApplicationBuildOnlyUsesVCSIdentity(t *testing.T) {
	commit := strings.Repeat("a", 40)
	info := applicationBuild(&debug.BuildInfo{Main: debug.Module{Path: "/Users/owner-secret"},
		Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: commit}, {Key: "vcs.modified", Value: "true"},
			{Key: "CGO_CFLAGS", Value: "owner-secret"}}})
	if info.SourceCommit != commit || !info.SourceModified || info.Version != releaseVersion {
		t.Fatalf("wrong build identity: %+v", info)
	}
	encoded, _ := json.Marshal(info)
	if strings.Contains(string(encoded), "owner-secret") {
		t.Fatal("build metadata disclosed toolchain paths")
	}
	if applicationBuild(nil).SourceCommit != "unknown" || applicationBuild(&debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "owner-secret"}}}).SourceCommit != "unknown" {
		t.Fatal("missing or invalid revision must be explicit")
	}
}

func TestDiagnosticReportProjectsKnownScalarsAndExcludesPrivateState(t *testing.T) {
	secret := "owner-secret"
	state := workbench.Snapshot{Status: secret, Analysis: secret,
		Components:  []workbench.Component{{ID: secret, Name: secret, Value: 901234.567}},
		Diagnostics: []workbench.Diagnostic{{Code: secret, Message: secret, ObjectID: secret}},
		DiagnosticPresentation: workbench.DiagnosticPresentation{Items: []workbench.DiagnosticHelp{
			{Code: "missing-ground", Summary: secret, Details: secret, Objects: []string{secret}}, {Code: secret}}},
		Operations: []workbench.OperationSummary{{ID: secret, SourcePath: "/Users/" + secret, Actor: secret,
			Status: "failed", Error: &workbench.Failure{Code: secret, Message: secret}}, {Status: secret}}}
	report := diagnosticReport(state, true, true)
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), secret) || strings.Contains(string(encoded), "901234.567") || len(encoded) > 8192 {
		t.Fatalf("private or unbounded report: %s", encoded)
	}
	if report.Engine.Status != "other" || !report.Engine.UnsavedChanges || !report.Engine.StorageError ||
		report.Engine.Checks["missing-ground"] != 1 || report.Engine.Checks["other"] != 1 ||
		report.Engine.Operations["failed"] != 1 || report.Engine.Operations["other"] != 1 || report.Engine.Components != 1 {
		t.Fatalf("lost safe diagnostic context: %+v", report.Engine)
	}
}

func TestApplicationAboutDoesNotAllocateWorkspaceAndSupportReportIsReadOnly(t *testing.T) {
	server := newWebServer(t.TempDir())
	defer server.close()
	f := &webFixture{server: server, handler: server.handler(), t: t}
	about := f.request("GET", "/api/about", nil, "")
	if about.Code != http.StatusOK || len(server.sessions) != 0 || f.cookie != nil {
		t.Fatal("build information must not allocate a project session")
	}
	if f.request("POST", "/api/about", nil, "").Code != http.StatusMethodNotAllowed ||
		f.request("GET", "/api/about", nil, "https://example.invalid").Code != http.StatusForbidden {
		t.Fatal("build endpoint bypassed method/origin rules")
	}
	f.request("GET", "/api/state", nil, "")
	f.command("simulation.pause", struct{}{})
	before := f.state()
	for _, ws := range server.sessions {
		ws.files.Lock()
		ws.info.Name = "owner-secret"
		ws.info.Error = "/Users/owner-secret: permission denied"
		ws.agentToken = "owner-secret"
		ws.info.Recovered = true
		ws.files.Unlock()
	}
	response := f.request("GET", "/api/diagnostic-report", nil, "")
	var report supportReport
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &report) != nil {
		t.Fatal(response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" || strings.Contains(response.Body.String(), "owner-secret") ||
		strings.Contains(response.Body.String(), f.cookie.Value) || strings.Contains(response.Body.String(), server.instance) ||
		!report.Engine.StorageError || !report.Engine.UnsavedChanges {
		t.Fatal("private workspace metadata leaked or safe status lost")
	}
	after := f.state()
	if before.CommandRevision != after.CommandRevision || before.Document.Generation != after.Document.Generation {
		t.Fatal("report changed the circuit")
	}
	if f.request("POST", "/api/diagnostic-report", nil, "").Code != http.StatusMethodNotAllowed ||
		f.request("GET", "/api/diagnostic-report", nil, "https://example.invalid").Code != http.StatusForbidden {
		t.Fatal("report endpoint bypassed method/origin rules")
	}
}
