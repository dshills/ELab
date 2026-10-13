package main

import (
	"fmt"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) verificationInspector(state workbench.Snapshot) []ui.Node {
	cfg := state.VerificationSettings
	transient := workbench.IsTransientAnalysis(state.Analysis)
	active := false
	for _, operation := range state.Operations {
		if operation.Status == "queued" || operation.Status == "running" || operation.Status == "canceling" {
			active = true
		}
	}
	nodes := []ui.Node{section("verification-title", "Verify from startup"), quiet("verification-help", "Records a separate run from captured startup conditions. Your live circuit and time stay unchanged.")}
	if transient {
		for _, field := range []struct {
			id, label  string
			value, max float64
		}{{"duration", "Duration · ms", cfg.Duration, 10}, {"step", "Sample step · ms (0 = project)", cfg.Step, 1}} {
			field := field
			nodes = append(nodes, quiet("verify-label-"+field.id, field.label), ui.Node{ID: "verify-" + field.id, Kind: ui.NumberField, Number: float32(field.value * 1000), Numeric: ui.Numeric{Min: 0, Max: float32(field.max * 1000), Step: .001}, Accessibility: ui.Accessibility{Name: field.label}, Change: func(e ui.ChangeEvent) {
				next := cfg
				if field.id == "duration" {
					next.Duration = float64(e.Number) / 1000
				} else {
					next.Step = float64(e.Number) / 1000
				}
				s.invoke("verification.settings.set", next)
			}})
		}
		modes := []ui.Option{{ID: "auto", Label: "Auto"}, {ID: "charge", Label: "RC · Charge"}, {ID: "discharge", Label: "RC · Discharge"}, {ID: "high", Label: "NPN / mixed · High"}, {ID: "low", Label: "NPN / mixed · Low"}}
		nodes = append(nodes, ui.Node{ID: "verify-mode", Kind: ui.Selector, Options: modes, SelectedID: cfg.InputMode, Accessibility: ui.Accessibility{Name: "Verification input mode"}, Change: func(e ui.ChangeEvent) {
			next := cfg
			next.InputMode = e.SelectedID
			s.invoke("verification.settings.set", next)
		}})
		initial := state.TransientSettings.InitialVoltage
		if cfg.InitialVoltage != nil {
			initial = *cfg.InitialVoltage
		}
		nodes = append(nodes, quiet("verify-initial-label", "Default initial voltage · V"), ui.Node{ID: "verify-initial", Kind: ui.NumberField, Number: float32(initial), Numeric: ui.Numeric{Min: 0, Max: 12, Step: .01}, Accessibility: ui.Accessibility{Name: "Verification default initial voltage"}, Change: func(e ui.ChangeEvent) {
			next := cfg
			v := float64(e.Number)
			next.InitialVoltage = &v
			s.invoke("verification.settings.set", next)
		}})
		graph := workbench.Graph{Components: state.Components, Wires: state.Wires}
		compiled, _, diagnostic := graph.Compile()
		for _, part := range state.Components {
			if part.Kind != "capacitor" {
				continue
			}
			part := part
			v := initial
			if diagnostic == nil && compiled.Analysis == "RC transient" && compiled.TerminalNet[part.ID+".1"] != compiled.Output {
				v = -v
			}
			if part.InitialVoltage != nil {
				v = *part.InitialVoltage
			}
			overridden := false
			for _, condition := range cfg.Capacitors {
				if condition.ID == part.ID {
					v = condition.Voltage
					overridden = true
				}
			}
			text := part.ID + " startup · pin 1 minus 2 · V"
			if overridden {
				text += " · override"
			}
			nodes = append(nodes, quiet("verify-cap-label-"+part.ID, text), ui.Node{ID: "verify-cap-" + part.ID, Kind: ui.NumberField, Number: float32(v), Numeric: ui.Numeric{Min: -12, Max: 12, Step: .01}, Accessibility: ui.Accessibility{Name: "Verify " + part.ID + " initial voltage"}, Change: func(e ui.ChangeEvent) {
				next := cfg
				next.Capacitors = append([]workbench.CapacitorInitial{}, cfg.Capacitors...)
				found := false
				for i := range next.Capacitors {
					if next.Capacitors[i].ID == part.ID {
						next.Capacitors[i].Voltage = float64(e.Number)
						found = true
					}
				}
				if !found {
					next.Capacitors = append(next.Capacitors, workbench.CapacitorInitial{ID: part.ID, Voltage: float64(e.Number)})
				}
				s.invoke("verification.settings.set", next)
			}})
		}
		reset := cfg
		reset.InitialVoltage = nil
		reset.Capacitors = []workbench.CapacitorInitial{}
		reset.InputMode = "auto"
		reset.Step = 0
		nodes = append(nodes, s.action("verify-project-startup", "Use project startup / sampling", "verification.settings.set", reset))
	}
	duration := cfg.Duration
	if !transient {
		duration = 0
	}
	button := s.action("verify-start", "Verify", "simulation.verify", workbench.VerifyParams{Duration: &duration})
	graph := workbench.Graph{Components: state.Components, Wires: state.Wires}
	_, _, diagnostic := graph.Compile()
	button.Disabled = active || diagnostic != nil
	nodes = append(nodes, button)
	if !transient {
		nodes = append(nodes, quiet("verify-dc-help", "DC verification records one operating point."))
	}
	options := []ui.Option{{ID: "__none", Label: "Choose a recorded run…"}}
	var selected *workbench.OperationSummary
	for i := range state.Operations {
		operation := state.Operations[i]
		options = append(options, ui.Option{ID: operation.ID, Label: operation.ID + " · " + operation.Status})
		if operation.ID == state.SelectedOperation {
			copy := operation
			selected = &copy
		}
	}
	value := state.SelectedOperation
	if value == "" {
		value = "__none"
	}
	nodes = append(nodes, ui.Node{ID: "verify-operation", Kind: ui.Selector, Options: options, SelectedID: value, Accessibility: ui.Accessibility{Name: "Selected verification operation"}, Change: func(e ui.ChangeEvent) {
		id := e.SelectedID
		if id == "__none" {
			id = ""
		}
		s.invoke("operation.select", workbench.IDParams{ID: id})
	}})
	if selected != nil {
		operation := *selected
		caption := fmt.Sprintf("%s · %.0f%% · %.4g / %.4g s", operation.Status, operation.Progress*100, operation.SimulatedTime, operation.RequestedDuration)
		nodes = append(nodes, quiet("verify-progress", caption), quiet("verify-identity", fmt.Sprintf("Captured electrical revision %d · %s", operation.ElectricalRevision, operation.Actor)))
		if !operation.SourceUnchanged {
			nodes = append(nodes, quiet("verify-source-changed", "Recorded source differs from the current circuit."))
		}
		cancel := s.action("verify-cancel", "Cancel verification", "operation.cancel", workbench.IDParams{ID: operation.ID})
		cancel.Disabled = operation.Status != "queued" && operation.Status != "running"
		nodes = append(nodes, cancel)
		if operation.Error != nil {
			nodes = append(nodes, quiet("verify-error", operation.Error.Message))
		}
	}
	if preview := state.Verification; preview != nil {
		for _, format := range []string{"json", "csv"} {
			format := format
			nodes = append(nodes, ui.Node{ID: "verify-export-" + format, Kind: ui.Button, Text: "Export recorded " + strings.ToUpper(format), Accessibility: ui.Accessibility{Name: "Export verification " + format}, Activate: func() {
				s.beginFile("verification.export", true, workbench.VerificationExportParams{ID: preview.ID, Format: format}, nil)
			}})
		}
		nodes = append(nodes, quiet("verify-recorded", fmt.Sprintf("Recorded · %d samples · %.4g s\n%s · %s", preview.SampleCount, preview.Duration, preview.Engine, preview.InputMode)))
		for _, reading := range preview.Readings {
			scale, unit := 1., reading.Probe.Unit
			if unit == "A" {
				scale, unit = 1000, "mA"
			}
			nodes = append(nodes, quiet("verify-reading-"+reading.Probe.ID, fmt.Sprintf("%s\nFinal %.5g %s · range %.5g…%.5g", strings.TrimSuffix(reading.Probe.Label, " / ground"), reading.Value*scale, unit, reading.Minimum*scale, reading.Maximum*scale)))
		}
	}
	return nodes
}
