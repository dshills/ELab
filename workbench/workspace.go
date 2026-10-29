package workbench

import "fmt"

// WorkspaceLayout is session presentation, independent of schematic files.
type WorkspaceLayout struct {
	SchematicFraction float64 `json:"schematic_fraction"`
}

func DefaultWorkspaceLayout() WorkspaceLayout {
	return WorkspaceLayout{SchematicFraction: .65}
}

func (s *Service) setWorkspaceLayoutLocked(p WorkspaceLayout) error {
	if !finite(p.SchematicFraction) || p.SchematicFraction < .35 || p.SchematicFraction > .85 {
		return fmt.Errorf("schematic_fraction must be between 0.35 and 0.85")
	}
	s.workspaceLayout = p
	return nil
}

func addWorkspaceCapabilityDetails(id string, schema map[string]any) bool {
	if addWorkspacePanelDetails(id, schema) || addWorkspaceDetailsCapability(id, schema) {
		return true
	}
	if id == "workspace.inspector.set" {
		inputProperty(schema, "section")["enum"] = []string{"component", "simulation", "document"}
		schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "workspace-inspector", Preconditions: []string{"Ordinary revision/document guards and retries apply"}, Effects: []string{"Select Component, Simulation or Document inspector; default component", "Session only, survives document replacement; does not change circuit, simulation, history, dirty state or saved files"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Missing/unknown section, malformed or extra fields"}, {"conflict", "admission", "Revision/document guard changed"}}, Completion: &commandCompletionDetail{Mode: "immediate shared inspector selection", Observe: []string{"project.inspect.inspector_section", "events.poll"}}}
		return true
	}
	if id != "workspace.layout.set" {
		return false
	}
	schema["x-command-contract"] = commandContractDetail{
		Version: 1, Scope: "workspace-layout",
		Preconditions: []string{"Ordinary revision/document guards and retry retention apply"},
		Effects: []string{
			"Set the schematic's fraction of the vertical schematic/scope split, from 0.35 to 0.85; default 0.65",
			"Session only; preserved through example load/reset, New, file open and recovery; not saved in schematic files",
			"Preserves graph, electrical state/run/time/charge, history, dirty state, scope visibility, probes, cursors and recorded playback",
			"Native divider drag/keyboard/accessibility and agents use this command; collapsed/unavailable scope and output/sweep panes hide the split but retain its fraction",
			"Native layout completes on a later frame; view.viewport.set reports the resulting drawable area and refits only when following is enabled; this command itself preserves the view",
			"Scope contents scroll when their allotted area is smaller than their controls; fraction is a preference, not a guarantee of minimum pixel size or OS window resizing",
		},
		Errors:     []commandFailureDetail{{"invalid-params", "admission", "Missing/null/nonfinite/out-of-range fraction, unknown fields or malformed parameters"}, {"conflict", "admission", "Command revision or document generation changed"}},
		Completion: &commandCompletionDetail{Mode: "immediate shared layout preference; native allocation follows asynchronously", Observe: []string{"project.inspect.workspace_layout", "project.inspect.viewport", "events.poll"}},
	}
	return true
}

// InspectorParams selects a presentation section without changing the circuit.
type InspectorParams struct {
	Section string `json:"section"`
}

func (s *Service) setInspectorSectionLocked(p InspectorParams) error {
	switch p.Section {
	case "component", "simulation", "document":
		s.inspectorSection = p.Section
		return nil
	default:
		return fmt.Errorf("section must be component, simulation or document")
	}
}
