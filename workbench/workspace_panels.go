package workbench

import "fmt"

const (
	MinPartsPanelWidth     = 224
	MaxPartsPanelWidth     = 360
	MinInspectorPanelWidth = 280
	MaxInspectorPanelWidth = 560
)

// WorkspacePanels contains session-only preferred logical widths. Native layout
// may temporarily narrow these widths to preserve the drawing on a small window.
type WorkspacePanels struct {
	PartsVisible     bool    `json:"parts_visible"`
	InspectorVisible bool    `json:"inspector_visible"`
	PartsWidth       float64 `json:"parts_width"`
	InspectorWidth   float64 `json:"inspector_width"`
}

func DefaultWorkspacePanels() WorkspacePanels {
	return WorkspacePanels{PartsVisible: true, InspectorVisible: true, PartsWidth: 240, InspectorWidth: MinInspectorPanelWidth}
}

func (s *Service) setWorkspacePanelsLocked(p WorkspacePanels) error {
	if !finite(p.PartsWidth) || p.PartsWidth < MinPartsPanelWidth || p.PartsWidth > MaxPartsPanelWidth || !finite(p.InspectorWidth) || p.InspectorWidth < MinInspectorPanelWidth || p.InspectorWidth > MaxInspectorPanelWidth {
		return fmt.Errorf("parts_width must be 224–360 and inspector_width 280–560 logical units")
	}
	s.workspacePanels = p
	return nil
}

type SimulationToolParams struct {
	Tool string `json:"tool"`
}

func (s *Service) setSimulationToolLocked(p SimulationToolParams) error {
	switch p.Tool {
	case "live", "recordings", "spectrum", "sweeps", "tolerances", "circuit":
		s.simulationTool = p.Tool
		return nil
	default:
		return fmt.Errorf("tool must be live, recordings, spectrum, sweeps, tolerances or circuit")
	}
}

func addWorkspacePanelDetails(id string, schema map[string]any) bool {
	scope, effect, observe := "", "", ""
	switch id {
	case "workspace.panels.set":
		scope, effect, observe = "workspace-panels", "Set both side-panel visibility flags and preferred logical widths; hidden panels retain their widths and current tasks. Native allocation may narrow panels to preserve drawing space.", "project.inspect.workspace_panels"
		addWorkspacePanelBounds(schema)
	case "workspace.simulation.set":
		scope, effect, observe = "workspace-simulation", "Choose one Simulate task. Does not launch analysis, change inspector/panel visibility, stop live simulation, or discard any tool's accepted settings/results.", "project.inspect.simulation_tool"
		inputProperty(schema, "tool")["enum"] = []string{"live", "recordings", "spectrum", "sweeps", "tolerances", "circuit"}
	default:
		return false
	}
	schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: scope,
		Preconditions: []string{"Ordinary command/document revision guards and retry identities apply"},
		Effects:       []string{effect, "Session presentation only; survives document replacement but is not saved in schematic files", "Preserves circuit, electrical run/time/charge, history, dirty flag, probes, cursors, recordings and other tool settings/results"},
		Errors:        []commandFailureDetail{{"invalid-params", "admission", "Missing/null/unknown/malformed parameters or values outside the declared bounds"}, {"conflict", "admission", "Command revision or document generation changed"}},
		Completion:    &commandCompletionDetail{Mode: "immediate shared presentation state; native layout follows asynchronously", Observe: []string{observe, "events.poll"}}}
	return true
}

func addWorkspacePanelBounds(schema map[string]any) {
	for _, p := range []struct {
		name          string
		min, max, def float64
	}{{"parts_width", MinPartsPanelWidth, MaxPartsPanelWidth, MinPartsPanelWidth}, {"inspector_width", MinInspectorPanelWidth, MaxInspectorPanelWidth, MinInspectorPanelWidth}} {
		property := inputProperty(schema, p.name)
		property["minimum"], property["maximum"], property["default"] = p.min, p.max, p.def
	}
}
