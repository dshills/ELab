package main

import (
	"net/http"
	"regexp"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/dshills/elab/workbench"
)

// Release packagers set these values in the backend itself, not from a file beside it.
var releaseVersion = "development"
var releaseBuild = ""

var sourceRevisionPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

type applicationInfo struct {
	Name           string `json:"name"`
	Interface      string `json:"interface"`
	Version        string `json:"version"`
	Build          string `json:"build,omitempty"`
	SourceCommit   string `json:"source_commit"`
	SourceModified bool   `json:"source_modified"`
	GoVersion      string `json:"go_version"`
	OS             string `json:"os"`
	Architecture   string `json:"architecture"`
}

func applicationBuild(info *debug.BuildInfo) applicationInfo {
	out := applicationInfo{Name: "ELab", Interface: "web", Version: releaseVersion, Build: releaseBuild,
		SourceCommit: "unknown", GoVersion: runtime.Version(), OS: runtime.GOOS, Architecture: runtime.GOARCH}
	if info != nil {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if sourceRevisionPattern.MatchString(setting.Value) {
					out.SourceCommit = setting.Value
				}
			case "vcs.modified":
				out.SourceModified = setting.Value == "true"
			}
		}
	}
	return out
}

func currentApplication() applicationInfo {
	info, _ := debug.ReadBuildInfo()
	return applicationBuild(info)
}

type supportEngine struct {
	Status          string         `json:"status"`
	Paused          bool           `json:"paused"`
	Revision        uint64         `json:"revision"`
	CommandRevision uint64         `json:"command_revision"`
	UnsavedChanges  bool           `json:"unsaved_changes"`
	Components      int            `json:"drawing_objects"`
	Simulated       int            `json:"simulated_components"`
	Wires           int            `json:"wires"`
	Checks          map[string]int `json:"circuit_checks"`
	Operations      map[string]int `json:"operations"`
	StorageError    bool           `json:"storage_needs_attention"`
}

type supportReport struct {
	Schema      int                     `json:"schema"`
	CreatedAt   string                  `json:"created_at"`
	Application applicationInfo         `json:"application"`
	Engine      supportEngine           `json:"engine"`
	Limits      workbench.CircuitLimits `json:"limits"`
}

// Reports deliberately project scalar state and known codes. Never serialize a
// Snapshot, error text, object reference, workspace metadata or agent descriptor.
func diagnosticReport(state workbench.Snapshot, storageError, recovered bool) supportReport {
	engine := supportEngine{Status: "other", Paused: state.Paused, Revision: state.Revision,
		CommandRevision: state.CommandRevision, UnsavedChanges: state.Document.Dirty || recovered,
		Components: len(state.Components), Simulated: state.SimulatedComponents, Wires: len(state.Wires),
		Checks: map[string]int{}, Operations: map[string]int{}, StorageError: storageError}
	switch state.Status {
	case "running", "paused", "updating", "error", "closed":
		engine.Status = state.Status
	}
	for _, help := range state.DiagnosticPresentation.Items {
		code := "other"
		switch help.Code {
		case "missing-ground", "reference-disconnected", "missing-parts", "open-terminal", "unconnected-primitive",
			"shorted-component", "invalid-terminal", "invalid-net-label", "floating-island", "unsupported-model",
			"unsupported-topology", "mixed-limit", "ideal-source-capacitor", "ideal-constraint-loop", "source-sampling",
			"integration-attempt-limit", "integration-time-limit", "solve-failed", "mixed-initialize-failed", "mixed-integration-failed":
			code = help.Code
		}
		engine.Checks[code]++
	}
	for _, operation := range state.Operations {
		status := "other"
		switch operation.Status {
		case "queued", "running", "canceling", "completed", "failed", "canceled":
			status = operation.Status
		}
		engine.Operations[status]++
	}
	return supportReport{Schema: 1, CreatedAt: time.Now().UTC().Format(time.RFC3339), Application: currentApplication(),
		Engine: engine, Limits: workbench.Limits()}
}

func (s *webServer) supportReport(w http.ResponseWriter, r *http.Request, ws *workspace) {
	if r.Method != http.MethodGet {
		webError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	ws.files.Lock()
	report := diagnosticReport(ws.service.Snapshot(), ws.info.Error != "", ws.info.Recovered)
	ws.files.Unlock()
	jsonReply(w, http.StatusOK, report)
}
