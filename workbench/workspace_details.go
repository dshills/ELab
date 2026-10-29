package workbench

import "fmt"

// WorkspaceDetails is session-only progressive disclosure, independent of files.
type WorkspaceDetails struct {
	Component   bool `json:"component"`
	Selection   bool `json:"selection"`
	Measurement bool `json:"measurement"`
	Pins        bool `json:"pins"`
	Live        bool `json:"live"`
	Recordings  bool `json:"recordings"`
	Spectrum    bool `json:"spectrum"`
	Sweeps      bool `json:"sweeps"`
	Tolerances  bool `json:"tolerances"`
	Circuit     bool `json:"circuit"`
}

type WorkspaceDetailsParams struct {
	Section string `json:"section"`
	Visible bool   `json:"visible"`
}

func (d *WorkspaceDetails) field(section string) *bool {
	switch section {
	case "component":
		return &d.Component
	case "selection":
		return &d.Selection
	case "measurement":
		return &d.Measurement
	case "pins":
		return &d.Pins
	case "live":
		return &d.Live
	case "recordings":
		return &d.Recordings
	case "spectrum":
		return &d.Spectrum
	case "sweeps":
		return &d.Sweeps
	case "tolerances":
		return &d.Tolerances
	case "circuit":
		return &d.Circuit
	}
	return nil
}

func (d WorkspaceDetails) Expanded(section string) bool {
	if p := d.field(section); p != nil {
		return *p
	}
	return false
}

func (s *Service) setWorkspaceDetailsLocked(p WorkspaceDetailsParams) error {
	field := s.workspaceDetails.field(p.Section)
	if field == nil {
		return fmt.Errorf("section must identify component, selection, measurement, pins, live, recordings, spectrum, sweeps, tolerances or circuit details")
	}
	*field = p.Visible
	return nil
}

func workspaceDetailSections() []string {
	return []string{"component", "selection", "measurement", "pins", "live", "recordings", "spectrum", "sweeps", "tolerances", "circuit"}
}

func addWorkspaceDetailsCapability(id string, schema map[string]any) bool {
	if id != "workspace.details.set" {
		return false
	}
	inputProperty(schema, "section")["enum"] = workspaceDetailSections()
	schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "workspace-details", Preconditions: []string{"Ordinary command/document guards and retry identities apply"}, Effects: []string{"Show/hide the named Details/Advanced section; default closed", "Session-only; survives document replacement but is not saved in project files", "Preserves circuit, accepted analysis settings/results, electrical run/time, probes, recordings, history and dirty state"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Missing/null/unknown fields or unknown section"}, {"conflict", "admission", "Command revision or document generation changed"}}, Completion: &commandCompletionDetail{Mode: "immediate shared presentation; native presentation follows asynchronously", Observe: []string{"project.inspect.workspace_details", "events.poll"}}}
	return true
}
