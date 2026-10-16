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
