package main

import (
	"fmt"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) verificationInspector(state workbench.Snapshot) []ui.Node {
	s.pruneVerificationRejections(state)
	cfg := state.VerificationSettings
	transient := workbench.IsTransientAnalysis(state.Analysis)
	active := false
	for _, operation := range state.Operations {
		if operation.Status == "queued" || operation.Status == "running" || operation.Status == "canceling" {
			active = true
		}
	}
	nodes := []ui.Node{section("verification-title", "Verify from startup"), quiet("verification-help", "Records a separate run from captured startup conditions. Your live circuit and time stay unchanged.")}
	nodes = append(nodes, ui.Node{ID: "recording-import", Kind: ui.Button, Text: "Open recorded run…", Accessibility: ui.Accessibility{Name: "Open recorded run"}, Disabled: s.fileBusy || s.pendingCommand != "", Activate: func() { s.beginFile("verification.import", true, workbench.RecordingImportParams{}, nil) }})
	if transient {
		for _, field := range []struct {
			id, label string
			value     float64
		}{{"duration", "Duration", cfg.Duration}, {"step", "Sample step (0 = project)", cfg.Step}} {
			field := field
			name := "duration_s"
			if field.id == "step" {
				name = "sample_step_s"
			}
			entry := timeField("verify-"+field.id, field.label, field.value, false, func(text string) {
				s.changeVerificationTime(state, "verify-"+field.id, name, text)
			})
			// Keep the refused entry distinct from the accepted value so a
			// fresh correction back to that value still emits Change.
			if rejected, ok := s.verificationRejected[entry.ID]; ok {
				entry.Text = rejected.text
			}
			nodes = append(nodes, quiet("verify-label-"+field.id, field.label), entry)
		}
		modes := []ui.Option{}
		labels := map[string]string{"auto": "Auto", "charge": "Charge", "discharge": "Discharge", "high": "High", "low": "Low"}
		allowed := state.VerificationInputModes
		if len(allowed) == 0 {
			allowed = workbench.VerificationInputModes(state.Analysis)
		}
		found := false
		for _, mode := range allowed {
			modes = append(modes, ui.Option{ID: mode, Label: labels[mode]})
			found = found || mode == cfg.InputMode
		}
		if !found {
			modes = append(modes, ui.Option{ID: cfg.InputMode, Label: "Unavailable: " + cfg.InputMode, Disabled: true})
		}

		nodes = append(nodes, ui.Node{ID: "verify-mode", Kind: ui.Selector, Options: modes, SelectedID: cfg.InputMode, Accessibility: ui.Accessibility{Name: "Verification input mode"}, Change: func(e ui.ChangeEvent) {
			next := cfg
			next.InputMode = e.SelectedID
			s.changeVerificationSettings(state, "verify-mode", next)
		}})
		initial := state.TransientSettings.InitialVoltage
		if cfg.InitialVoltage != nil {
			initial = *cfg.InitialVoltage
		}
		nodes = append(nodes, quiet("verify-initial-label", "Default initial voltage · V"), ui.Node{ID: "verify-initial", Kind: ui.NumberField, Number: float32(initial), Numeric: ui.Numeric{Min: 0, Max: 12, Step: .01}, Accessibility: ui.Accessibility{Name: "Verification default initial voltage"}, Change: func(e ui.ChangeEvent) {
			next := cfg
			v := float64(e.Number)
			next.InitialVoltage = &v
			s.changeVerificationSettings(state, "verify-initial", next)
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
				s.changeVerificationSettings(state, "verify-cap-"+part.ID, next)
			}})
		}
	}
	resetButton := s.verificationAction(state, true)
	resetButton.Accessibility.Name = "Use project startup and sampling"
	resetButton.Accessibility.Help = "Clear verification input mode, sampling and startup overrides; preserve duration and the live circuit."
	nodes = append(nodes, resetButton)

	button := s.verificationAction(state, false)
	graph := workbench.Graph{Components: state.Components, Wires: state.Wires}
	_, _, diagnostic := graph.Compile()
	button.Disabled = button.Disabled || active || diagnostic != nil
	nodes = append(nodes, button)
	if !transient {
		nodes = append(nodes, quiet("verify-dc-help", "DC records one operating point. Use project startup to clear transient overrides."))
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
		if operation.Status == "completed" {
			keepLabel := "Include in project"
			if operation.KeptInProject {
				keepLabel = "Exclude from project"
			}
			nodes = append(nodes, s.action("recording-keep", keepLabel, "recording.keep.set", workbench.RecordingKeepParams{ID: operation.ID, Keep: !operation.KeptInProject}), quiet("recording-keep-help", "Kept recordings are saved with this project and protected from eviction."))
		}
		nodes = append(nodes, quiet("verify-progress", caption), quiet("verify-identity", fmt.Sprintf("Captured electrical revision %d · %s", operation.ElectricalRevision, operation.Actor)))
		if !operation.SourceUnchanged {
			message := "Captured experiment differs from the current circuit."
			if operation.Kind != "verify" {
				message = "Imported data has no association with the live circuit."
			}
			nodes = append(nodes, quiet("verify-source-changed", message))
		}
		cancel := s.action("verify-cancel", "Cancel verification", "operation.cancel", workbench.IDParams{ID: operation.ID})
		cancel.Disabled = operation.Status != "queued" && operation.Status != "running"
		nodes = append(nodes, cancel)
		if operation.Error != nil {
			if help := operation.FailureHelp; help != nil {
				nodes = append(nodes, quiet("verify-failure-summary", help.Summary), quiet("verify-failure-action", help.Action), ui.Node{ID: "verify-failure-details", Kind: ui.Disclosure, Text: "Technical details", Checked: state.DiagnosticPresentation.DetailsVisible, Accessibility: ui.Accessibility{Name: "Show verification failure technical details"}, Children: []ui.Node{quiet("verify-error", help.Code+"\n"+help.Details)}, Change: func(e ui.ChangeEvent) {
					s.invoke("diagnostics.details.set", workbench.DiagnosticDetailsParams{Visible: e.Checked})
				}})
			} else {
				nodes = append(nodes, quiet("verify-error", operation.Error.Message))
			}
		}
	}
	if preview := state.Verification; preview != nil {
		nodes = append(nodes, s.action("recording-show", "View recorded plot", "recording.view.set", workbench.RecordingViewParams{Mode: "recorded"}))
		nodes = append(nodes, s.recordingInspector(state)...)

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
