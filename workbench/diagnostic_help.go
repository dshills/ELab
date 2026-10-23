package workbench

import (
	"encoding/json"
	"fmt"
	"strings"
)

type DiagnosticHelp struct {
	Source    string   `json:"source"`
	Code      string   `json:"code"`
	Severity  string   `json:"severity"`
	Summary   string   `json:"summary"`
	Action    string   `json:"action"`
	Details   string   `json:"details"`
	Objects   []string `json:"object_ids"`
	Terminals []string `json:"terminal_refs"`
	Wires     []string `json:"wire_ids"`
}
type DiagnosticPresentation struct {
	DetailsVisible bool             `json:"details_visible"`
	Items          []DiagnosticHelp `json:"items"`
}
type DiagnosticDetailsParams struct {
	Visible bool `json:"visible"`
}

func cloneDiagnosticHelp(in *DiagnosticHelp) *DiagnosticHelp {
	if in == nil {
		return nil
	}
	out := *in
	out.Objects = append([]string{}, in.Objects...)
	out.Terminals = append([]string{}, in.Terminals...)
	out.Wires = append([]string{}, in.Wires...)
	return &out
}
func cloneDiagnosticPresentation(in DiagnosticPresentation) DiagnosticPresentation {
	out := in
	out.Items = append([]DiagnosticHelp{}, in.Items...)
	for i := range out.Items {
		out.Items[i] = *cloneDiagnosticHelp(&out.Items[i])
	}
	return out
}
func makeDiagnosticHelp(g Graph, c Compiled, d Diagnostic, source string, refs []string) DiagnosticHelp {
	out := DiagnosticHelp{Source: source, Code: d.Code, Severity: "note", Summary: d.Message, Details: d.Message, Objects: []string{}, Terminals: []string{}, Wires: []string{}}
	switch d.Code {
	case "missing-ground", "reference-disconnected":
		out.Action = "Add ground and connect the source's negative terminal to it."
	case "missing-parts":
		out.Action = "Place the missing parts and connect their named terminals; the drawing stays editable."
	case "open-terminal", "unconnected-primitive":
		out.Action = "Connect the highlighted terminal, or undo an accidental deletion."
	case "shorted-component":
		out.Severity = "error"
		out.Action = "Remove the unintended connection across the highlighted component."
	case "invalid-terminal", "invalid-net-label":
		out.Severity = "error"
		out.Action = "Repair the highlighted wire/label using existing named terminals and valid labels."
	case "floating-island":
		out.Action = "Connect the highlighted island to the circuit's ground reference."
	case "unsupported-model", "unsupported-topology", "mixed-limit", "ideal-source-capacitor", "ideal-constraint-loop":
		out.Severity = "warning"
		out.Action = "Use the supported topology/model limits described in the details; your drawing is retained."
	case "source-sampling":
		out.Severity = "warning"
		out.Action = "Use Fit waveform sampling or reduce the output sample interval."
	case "integration-attempt-limit":
		out.Severity = "error"
		out.Summary = "This sample needed more calculation attempts than allowed."
		out.Action = "Reduce the output sample interval. Restart a stopped live run, or retry a paused Step; committed time and charge were preserved."
	case "integration-time-limit":
		out.Severity = "error"
		out.Summary = "This sample took longer than the calculation time limit."
		out.Action = "Reduce the output sample interval or retry when the computer is less busy. Restart a stopped live run; committed time and charge were preserved."
	case "solve-failed", "mixed-initialize-failed":
		out.Severity = "error"
		out.Summary = "Simulation could not complete this calculation."
		out.Action = "Check connections, supported model limits and startup conditions, then restart. The technical reason is available below."
	case "mixed-integration-failed":
		out.Severity = "error"
		out.Summary = "Simulation stopped at the last accepted sample."
		out.Action = "Reduce the sample interval or review the circuit/startup conditions, then restart. Previous readings stay identified as stale."
	default:
		out.Action = "Review the explanation and supported values; the circuit remains editable."
	}
	if source == "request" {
		out.Severity = "warning"
		out.Summary = "That request could not be applied."
		out.Action = "Check the requested values and related highlighted parts/terminals. Existing circuit state was preserved."
		if d.Code == "file-error" || d.Code == "invalid-document" || d.Code == "unsupported-format" {
			out.Action = "Choose a valid readable local schematic or save to a writable location. Existing circuit state was preserved."
		}
	}
	appendRef := func(ref string) {
		if part, _, ok := g.Terminal(ref); ok {
			out.Objects = appendUnique(out.Objects, part.ID)
			out.Terminals = appendUnique(out.Terminals, ref)
			return
		}
		if _, ok := g.Component(ref); ok {
			out.Objects = appendUnique(out.Objects, ref)
			return
		}
		for _, wire := range g.Wires {
			if wire.ID == ref {
				out.Wires = appendUnique(out.Wires, ref)
				return
			}
		}
	}
	refs = append(append([]string{}, refs...), d.ObjectID)
	for _, ref := range refs {
		if ref == "" {
			continue
		}
		if strings.HasPrefix(ref, "net:") {
			for _, net := range c.Nets {
				if net.ID == ref {
					for _, terminal := range net.Terminals {
						appendRef(terminal)
					}
					for _, wire := range g.Wires {
						if wire.Connected && c.TerminalNet[wire.From] == ref && c.TerminalNet[wire.To] == ref {
							appendRef(wire.ID)
						}
					}
				}
			}
		} else {
			appendRef(ref)
		}
	}
	return out
}
func (s *Service) diagnosticPresentationLocked(c Compiled) DiagnosticPresentation {
	out := DiagnosticPresentation{DetailsVisible: s.diagnosticDetails, Items: []DiagnosticHelp{}}
	for _, d := range s.diagnostics {
		out.Items = append(out.Items, makeDiagnosticHelp(s.graph, c, d, "simulation", nil))
	}
	if s.stepJob != nil && s.stepJob.work.Status == "failed" && s.stepJob.work.Error != nil {
		work := s.stepJob.work
		d := Diagnostic{Code: work.Error.Code, Message: work.Error.Message}
		help := makeDiagnosticHelp(s.graph, c, d, "step", nil)
		if work.Error.Code == "integration-attempt-limit" || work.Error.Code == "integration-time-limit" {
			help.Summary = "Manual step could not finish. " + help.Summary
		} else {
			help.Summary = "Manual step failed; committed time and charge were preserved."
		}
		if work.RunID != s.run || work.ElectricalRevision != s.electrical || work.ActuationRevision != s.actuationRevision {
			help.Summary = "A previous manual step failed; its committed state was preserved."
		}
		out.Items = append(out.Items, help)
	}
	for _, advice := range sourceSamplingAdvice(s.graph, s.transientSettings) {
		if advice.Unresolved {
			out.Items = append(out.Items, sourceSamplingHelp(s.graph, c, advice, s.transientSettings.StepSeconds))
		}
	}
	if help := lcSamplingHelp(s.graph, c, lcSamplingAdvice(s.graph, s.transientSettings)); help != nil {
		out.Items = append(out.Items, *help)
	}
	if s.inputJob != nil && s.inputJob.work.Status == "failed" && s.inputJob.work.Error != nil {
		work := s.inputJob.work
		help := makeDiagnosticHelp(s.graph, c, Diagnostic{Code: "solve-failed", Message: work.Error.Message}, "input", nil)
		help.Summary = "Input calculation failed; committed circuit, time and charge were preserved."
		if work.RunID != s.run || work.ElectricalRevision != s.electrical || work.ActuationRevision != s.actuationRevision {
			help.Summary = "A previous input calculation failed; its committed state was preserved."
		}
		out.Items = append(out.Items, help)
	}
	return out
}
func (s *Service) requestDiagnosticLocked(r Request, code, message string) *DiagnosticHelp {
	if code != "invalid-params" && code != "not-found" && code != "file-error" && code != "invalid-document" {
		return nil
	}
	refs := []string{}
	// Related request targets are context, never guessed solver causes. Parsing
	// known intent fields cannot change state or make an unknown ID a marker.
	var fields map[string]json.RawMessage
	targeted := strings.HasPrefix(r.Command, "component.") || strings.HasPrefix(r.Command, "wire.") || strings.HasPrefix(r.Command, "source.") || strings.HasPrefix(r.Command, "capacitor.") || strings.HasPrefix(r.Command, "potentiometer.") || r.Command == "switch.set" || r.Command == "object.delete" || r.Command == "simulation.pulse.timing.set"
	if targeted && json.Unmarshal(r.Params, &fields) == nil {
		for _, key := range []string{"id", "from", "to", "terminal"} {
			var ref string
			if json.Unmarshal(fields[key], &ref) == nil {
				refs = append(refs, ref)
			}
		}
	}
	c, _, _ := s.graph.Compile()
	help := makeDiagnosticHelp(s.graph, c, Diagnostic{Code: code, Message: message}, "request", refs)
	if code == "invalid-params" && r.Command == "component.set" {
		componentValueRequestHelp(s.graph, r.Params, &help)
	}
	if code == "invalid-params" && r.Command == "recording.seek" {
		if record, err := s.selectedRecordLocked(); err == nil {
			help.Summary = "Choose a valid position in the recording."
			help.Action = fmt.Sprintf("Use a whole sample index from 0 to %d, or a time from 0 s to %g s. Time units such as ms and us are accepted. Your circuit is unchanged.", len(record.rows)-1, record.metadata.Duration)
			if record.metadata.Duration == 0 {
				help.Summary = "This recording has one sample."
				help.Action = "Use sample index 0 or time 0 s. Your circuit is unchanged."
			}
		}
	}
	return &help
}
