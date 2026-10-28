package workbench

import (
	"encoding/json"
	"fmt"
)

type MeasurementSelectParams struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}
type MeasurementQueryParams struct {
	Target *MeasurementSelectParams `json:"target,omitempty"`
}
type MeasurementTarget struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Anchor string `json:"anchor,omitempty"`
}
type MeasurementInspection struct {
	Target     *MeasurementTarget   `json:"target,omitempty"`
	ResultKind string               `json:"result_kind"`
	Available  bool                 `json:"available"`
	Reason     string               `json:"reason,omitempty"`
	Objects    []string             `json:"object_ids"`
	Terminals  []string             `json:"terminal_refs"`
	Wires      []string             `json:"wire_ids"`
	Nets       []Net                `json:"nets"`
	Readings   []MeasurementReading `json:"readings"`
}

func cloneMeasurementInspection(in MeasurementInspection) MeasurementInspection {
	out := in
	if in.Target != nil {
		v := *in.Target
		out.Target = &v
	}
	out.Objects = append([]string{}, in.Objects...)
	out.Terminals = append([]string{}, in.Terminals...)
	out.Wires = append([]string{}, in.Wires...)
	out.Nets = append([]Net{}, in.Nets...)
	for i := range out.Nets {
		out.Nets[i].Terminals = append([]string{}, out.Nets[i].Terminals...)
		out.Nets[i].Labels = append([]string{}, out.Nets[i].Labels...)
	}
	out.Readings = cloneMeasurementReadings(in.Readings)
	return out
}
func (s *Service) measurementTargetLocked(p MeasurementSelectParams) (*MeasurementTarget, error) {
	if p.Kind == "" && p.ID == "" {
		return nil, nil
	}
	t := &MeasurementTarget{Kind: p.Kind, ID: p.ID}
	switch p.Kind {
	case "object":
		if s.graph.HasObject(p.ID) {
			return t, nil
		}
	case "terminal":
		if _, _, ok := s.graph.Terminal(p.ID); ok {
			t.Anchor = p.ID
			return t, nil
		}
	case "net":
		c, _, _ := s.compileGraphLocked()
		for _, net := range c.Nets {
			if net.ID == p.ID && len(net.Terminals) > 0 {
				// Net IDs can change when wiring changes. Follow a terminal identity
				// so splitting or relabeling cannot select an unrelated net.
				t.Anchor = net.ID[len("net:"):]
				return t, nil
			}
		}
	case "probe":
		for _, probe := range probeInventory(s.graph) {
			if probe.ID == p.ID {
				return t, nil
			}
		}
	default:
		return nil, fmt.Errorf("measurement kind must be object, terminal, net or probe; empty kind/ID clears focus")
	}
	return nil, &actionError{"not-found", "Measurement source not found in the current schematic"}
}
func (s *Service) selectMeasurementLocked(raw json.RawMessage) error {
	var p MeasurementSelectParams
	if err := params(raw, &p, "kind", "id"); err != nil {
		return err
	}
	target, err := s.measurementTargetLocked(p)
	if err == nil {
		s.measurementTarget = target
	}
	return err
}
func (s *Service) currentMeasurementTargetLocked() *MeasurementTarget {
	if s.measurementTarget != nil {
		v := *s.measurementTarget
		return &v
	}
	if s.selected != "" {
		return &MeasurementTarget{Kind: "object", ID: s.selected}
	}
	return nil
}
func (s *Service) measurementQueryLocked(raw json.RawMessage) (MeasurementInspection, error) {
	var p MeasurementQueryParams
	if err := params(raw, &p); err != nil {
		return MeasurementInspection{}, err
	}
	target := s.currentMeasurementTargetLocked()
	if p.Target != nil {
		if err := paramsNested(raw, "target", &MeasurementSelectParams{}, "kind", "id"); err != nil {
			return MeasurementInspection{}, err
		}
		var err error
		target, err = s.measurementTargetLocked(*p.Target)
		if err != nil {
			return MeasurementInspection{}, err
		}
	}
	c, _, _ := s.compileGraphLocked()
	return s.measurementInspectionLocked(c, target), nil
}
func (s *Service) measurementInspectionLocked(c Compiled, target *MeasurementTarget) MeasurementInspection {
	out := MeasurementInspection{ResultKind: "none", Objects: []string{}, Terminals: []string{}, Wires: []string{}, Nets: []Net{}, Readings: []MeasurementReading{}}
	if target == nil {
		return out
	}
	t := *target
	out.Target = &t
	context := s.readingContextLocked()
	out.ResultKind = "none"
	if context.sample.frame != nil {
		out.ResultKind = "dc"
		if s.transient != nil {
			out.ResultKind = "transient"
		}
	}
	appendNet := func(id string) {
		for _, old := range out.Nets {
			if old.ID == id {
				return
			}
		}
		for _, net := range c.Nets {
			if net.ID == id {
				out.Nets = append(out.Nets, net)
				return
			}
		}
	}
	appendTerminal := func(ref string) {
		if owner, _, ok := s.graph.Terminal(ref); ok {
			out.Objects = appendUnique(out.Objects, owner.ID)
			out.Terminals = appendUnique(out.Terminals, ref)
			appendNet(c.TerminalNet[ref])
		}
	}
	anchor := t.Anchor
	switch t.Kind {
	case "object":
		if part, ok := s.graph.Component(t.ID); ok {
			out.Available = true
			for _, probe := range probeInventory(s.graph) {
				owner, _, _ := s.graph.Terminal(probe.Positive)
				if owner.ID == part.ID {
					out.Readings = append(out.Readings, context.reading(probe.ID))
				}
			}
			for _, pin := range part.Terminals {
				appendTerminal(part.ID + "." + pin)
			}
			return out
		}
		for _, wire := range s.graph.Wires {
			if wire.ID == t.ID {
				if !wire.Connected {
					out.Reason = "Wire is not electrically connected; inspect its endpoints"
					return out
				}
				anchor = wire.From
				break
			}
		}
	case "probe":
		if probe, ok := context.available[t.ID]; ok {
			out.Available = true
			out.Readings = append(out.Readings, context.reading(t.ID))
			appendTerminal(probe.Positive)
			appendTerminal(probe.Reference)
			return out
		}
	case "terminal", "net":
		// The captured anchor, rather than an old net ID, owns this selection.
	default:
		out.Reason = "Unknown inspection target"
		return out
	}
	if _, _, ok := s.graph.Terminal(anchor); !ok {
		out.Reason = "Source removed; undo restores it or select another measurement"
		return out
	}
	appendTerminal(anchor)
	if t.Kind == "terminal" {
		out.Available = true
		out.Readings = append(out.Readings, context.reading("v:"+anchor))
	}
	id := c.TerminalNet[anchor]
	if id == "" {
		out.Reason = "Connectivity is incomplete; no current net association is available"
		return out
	}
	out.Available = true
	if t.Kind == "net" {
		t.ID = id
	}
	if t.Kind != "terminal" {
		out.Readings = append(out.Readings, context.reading("v:"+anchor))
	}
	for _, net := range out.Nets {
		for _, ref := range net.Terminals {
			out.Terminals = appendUnique(out.Terminals, ref)
			owner, _, _ := s.graph.Terminal(ref)
			out.Objects = appendUnique(out.Objects, owner.ID)
		}
	}
	for _, wire := range s.graph.Wires {
		if wire.Connected && c.TerminalNet[wire.From] == id && c.TerminalNet[wire.To] == id {
			out.Wires = append(out.Wires, wire.ID)
		}
	}
	return out
}
