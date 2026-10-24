package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

const MaxScopeProbes = 6

type probeFrame struct{ values map[string]float64 }

// Probe identities use component/pin names, so moving parts cannot change them.
type Probe struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Unit      string `json:"unit"`
	Positive  string `json:"positive"`
	Reference string `json:"reference"`
}
type ScopePoint struct {
	Time  float64 `json:"time_s"`
	Value float64 `json:"value"`
}
type ScopeTrace struct {
	Probe  Probe        `json:"probe"`
	Points []ScopePoint `json:"points"`
}
type ScopeCursor struct {
	Requested float64 `json:"requested_s"`
	Time      float64 `json:"time_s"`
	Valid     bool    `json:"valid"`
	Reason    string  `json:"reason,omitempty"`
}
type CursorReading struct {
	ProbeID string  `json:"probe_id"`
	A       float64 `json:"a"`
	B       float64 `json:"b"`
	Delta   float64 `json:"delta"`
}
type UnavailableProbe struct {
	ID     string `json:"id"`
	Unit   string `json:"unit"`
	Reason string `json:"reason"`
}

type Scope struct {
	Unavailable  []UnavailableProbe `json:"unavailable,omitempty"`
	Window       ScopeWindow        `json:"window"`
	WindowValid  bool               `json:"window_valid"`
	WindowReason string             `json:"window_reason,omitempty"`
	Statistics   []ScopeStatistics  `json:"statistics"`
	Available    []Probe            `json:"available"`
	Selected     []string           `json:"selected"`
	Traces       []ScopeTrace       `json:"traces"`
	RunID        uint64             `json:"run_id"`
	Stale        bool               `json:"stale"`
	ActiveCursor string             `json:"active_cursor"`
	A            *ScopeCursor       `json:"a,omitempty"`
	B            *ScopeCursor       `json:"b,omitempty"`
	DeltaSeconds *float64           `json:"delta_s,omitempty"`
	Frequency    *float64           `json:"interval_hz,omitempty"`
	Readings     []CursorReading    `json:"readings"`
}
type ProbeParams struct {
	IDs []string `json:"ids"`
}
type CursorParams struct {
	Cursor string  `json:"cursor"`
	Time   float64 `json:"time_s"`
}
type CursorSelectParams struct {
	Cursor string `json:"cursor"`
}

func probeInventory(g Graph) []Probe {
	out := []Probe{}
	for _, p := range g.Components {
		for _, pin := range p.Terminals {
			ref := p.ID + "." + pin
			name := ref + " / ground"
			if p.Kind == "net-label" {
				name = p.Label + " / ground"
			}
			out = append(out, Probe{"v:" + ref, name, "V", ref, "ground"})
		}
		if len(p.Terminals) == 2 {
			out = append(out, Probe{"v:" + p.ID, p.ID + " voltage", "V", p.ID + "." + p.Terminals[0], p.ID + "." + p.Terminals[1]})
		}
		switch p.Kind {
		case "resistor", "capacitor", "inductor", "led", "switch", "diode", "zener", "schottky":
			out = append(out, Probe{"i:" + p.ID, p.ID + " current", "A", p.ID + "." + p.Terminals[0], p.ID + "." + p.Terminals[1]})
		case "dc-source", "pulse-source", "sine-source", "finite-pulse-source":
			out = append(out, Probe{"i:" + p.ID, p.ID + " delivered current", "A", p.ID + ".positive", "out of positive"})
		case "potentiometer":
			out = append(out, Probe{"v:" + p.ID, p.ID + " track voltage", "V", p.ID + ".1", p.ID + ".2"})
			for _, pin := range p.Terminals {
				ref := p.ID + "." + pin
				out = append(out, Probe{"i:" + ref, ref + " current into device", "A", ref, "into device"})
			}
		case "nmos", "pmos":
			out = append(out, Probe{"v:" + p.ID, p.ID + " drain-source voltage", "V", p.ID + ".drain", p.ID + ".source"}, Probe{"v:" + p.ID + ".gate-source", p.ID + " gate-source voltage", "V", p.ID + ".gate", p.ID + ".source"})
			for _, pin := range p.Terminals {
				ref := p.ID + "." + pin
				direction := "into device"
				if pin == "source" {
					direction = "out of device"
				}
				out = append(out, Probe{"i:" + ref, ref + " current", "A", ref, direction})
			}
		case "bcd-decoder", "seven-segment":
			for _, pin := range p.Terminals {
				direction := "into device"
				if p.Kind == "bcd-decoder" && len(pin) == 1 {
					direction = "out of device"
				}
				out = append(out, Probe{"i:" + p.ID + "." + pin, p.ID + "." + pin + " current", "A", p.ID + "." + pin, direction})
			}
		case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter":
			primary := "q"
			if IsCounterKind(p.Kind) {
				primary = "q0"
			}
			out = append(out, Probe{"v:" + p.ID, p.ID + " output / local ground", "V", p.ID + "." + primary, p.ID + ".gnd"}, Probe{"i:" + p.ID, p.ID + " delivered primary output current", "A", p.ID + "." + primary, "out of device"})
			for _, pin := range p.Terminals {
				direction := "into device"
				if sequentialOutput(pin) {
					direction = "out of device"
				}
				out = append(out, Probe{"i:" + p.ID + "." + pin, p.ID + "." + pin + " current", "A", p.ID + "." + pin, direction})
			}
		case "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor", "timer555":
			out = append(out, Probe{"v:" + p.ID, p.ID + " output / local ground", "V", p.ID + ".out", p.ID + ".gnd"}, Probe{"i:" + p.ID, p.ID + " delivered output current", "A", p.ID + ".out", "out of device"})
			for _, pin := range p.Terminals {
				ref := p.ID + "." + pin
				direction := "into device"
				if pin == "out" {
					direction = "out of device"
				}
				out = append(out, Probe{"i:" + ref, ref + " current", "A", ref, direction})
			}
		case "opamp":
			out = append(out,
				Probe{"v:" + p.ID, p.ID + " output / ground", "V", p.ID + ".out", "ground"},
				Probe{"v:" + p.ID + ".input-difference", p.ID + " noninverting minus inverting", "V", p.ID + ".noninverting", p.ID + ".inverting"},
				Probe{"v:" + p.ID + ".supply", p.ID + " positive minus negative supply", "V", p.ID + ".vplus", p.ID + ".vminus"},
				Probe{"i:" + p.ID, p.ID + " delivered output current", "A", p.ID + ".out", "out of device"})
			for _, pin := range p.Terminals {
				ref, direction := p.ID+"."+pin, "into device"
				if pin == "out" {
					direction = "out of device"
				}
				out = append(out, Probe{"i:" + ref, ref + " current", "A", ref, direction})
			}
		case "npn", "pnp":
			out = append(out, Probe{"v:" + p.ID, p.ID + " collector-emitter voltage", "V", p.ID + ".collector", p.ID + ".emitter"}, Probe{"v:" + p.ID + ".base-emitter", p.ID + " base-emitter voltage", "V", p.ID + ".base", p.ID + ".emitter"})
			for _, pin := range p.Terminals {
				ref := p.ID + "." + pin
				direction := "into device"
				if pin == "emitter" {
					direction = "out of device"
				}
				out = append(out, Probe{"i:" + ref, ref + " current", "A", ref, direction})
			}
		}
	}
	return out
}

// Frames are immutable private solver history. Only chosen channels are exported.
func captureFrame(c Compiled, p TraceSample, nodes, currents map[string]float64) TraceSample {
	p.frame = &probeFrame{values: map[string]float64{}}
	for ref, net := range c.TerminalNet {
		if value, ok := nodes[net]; ok {
			p.frame.values["v:"+ref] = value
		}
	}
	for id, value := range currents {
		p.frame.values["i:"+id] = value
	}
	return p
}
func frameValue(p TraceSample, probe Probe) (float64, bool) {
	if p.frame == nil {
		return 0, false
	}
	if probe.Unit == "A" {
		v, ok := p.frame.values[probe.ID]
		return v, ok
	}
	a, ok := p.frame.values["v:"+probe.Positive]
	if !ok {
		return 0, false
	}
	if probe.Reference == "ground" {
		return a, true
	}
	b, ok := p.frame.values["v:"+probe.Reference]
	return a - b, ok
}
func defaultProbes(c Compiled) []string {
	if IsNetworkAnalysis(c.Analysis) {
		out := []string{}
		for _, p := range c.MixedGraph.Components {
			if IsSequentialKind(p.Kind) {
				out = append(out, "v:"+p.ID+".clk", "v:"+p.ID)
				if IsCounterKind(p.Kind) {
					out = append(out, "v:"+p.ID+".q1", "v:"+p.ID+".q2", "v:"+p.ID+".q3")
				} else {
					out = append(out, "v:"+p.ID+".nq")
				}
			}
			if IsLogicKind(p.Kind) {
				out = append(out, "v:"+p.ID+".a", "v:"+p.ID, "i:"+p.ID)
			}
			if p.Kind == "timer555" {
				out = append(out, "v:"+p.ID, "v:"+p.ID+".threshold", "i:"+p.ID)
			}
			if p.Kind == "opamp" {
				out = append(out, "v:"+p.ID, "v:"+p.ID+".noninverting", "i:"+p.ID)
			}
			if IsMOSFETKind(p.Kind) {
				out = append(out, "v:"+p.ID, "v:"+p.ID+".gate-source", "i:"+p.ID+".drain")
			}
			if IsBipolarKind(p.Kind) {
				out = append(out, "v:"+p.ID+".collector")
			}
		}
		for _, p := range c.MixedGraph.Components {
			if p.Kind == "led" || IsDiodeKind(p.Kind) {
				out = append(out, "i:"+p.ID)
			}
		}
		for _, p := range c.MixedGraph.Components {
			if p.Kind == "potentiometer" {
				out = append(out, "v:"+p.ID+".wiper", "i:"+p.ID+".wiper")
			}
		}
		for _, part := range c.MixedGraph.Components {
			if part.Kind == "inductor" {
				out = append(out, "i:"+part.ID, "v:"+part.ID)
			}
		}
		if c.Capacitor.ID != "" {
			out = append(out, "v:"+c.Capacitor.ID)
		}
		out = append(out, "v:"+c.Source.ID+".positive")
		for _, p := range c.MixedGraph.Components {
			if p.Kind == "switch" {
				out = append(out, "i:"+p.ID)
			}
		}
		if len(out) > MaxScopeProbes {
			out = out[:MaxScopeProbes]
		}
		return out
	}
	if c.Analysis == NPNTransientAnalysis {
		return []string{"v:" + c.Transistor.ID + ".collector", "v:" + c.Input.ID + ".positive", "i:" + c.Transistor.ID + ".collector"}
	}
	if c.Analysis == "RC transient" {
		pin := "1"
		if c.TerminalNet[c.Capacitor.ID+".1"] != c.Output {
			pin = "2"
		}
		return []string{"v:" + c.Capacitor.ID + "." + pin, "v:" + c.Source.ID + ".positive"}
	}
	return []string{}
}
func (s *Service) scopeLocked(c Compiled) Scope {
	out := Scope{Available: probeInventory(s.graph), Selected: []string{}, Traces: []ScopeTrace{}, Readings: []CursorReading{}, Statistics: []ScopeStatistics{}, ActiveCursor: s.activeCursor, RunID: s.run}
	if out.ActiveCursor == "" {
		out.ActiveCursor = "a"
	}
	ids := s.scopeProbes
	if !s.scopeConfigured {
		ids = defaultProbes(c)
	}
	byID := map[string]Probe{}
	for _, p := range out.Available {
		byID[p.ID] = p
	}
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			out.Selected = append(out.Selected, id)
			out.Traces = append(out.Traces, ScopeTrace{Probe: p, Points: []ScopePoint{}})
		} else if s.scopeConfigured {
			unit := "V"
			if strings.HasPrefix(id, "i:") {
				unit = "A"
			}
			out.Unavailable = append(out.Unavailable, UnavailableProbe{ID: id, Unit: unit, Reason: "Source removed; undo can restore it, or remove/reassign this probe."})
		}
	}
	t := s.transient
	if t == nil {
		return out
	}
	out.RunID = t.RunID
	out.Stale = t.Stale
	if len(t.Samples) > 0 {
		out.Window = ScopeWindow{Mode: "live", Start: t.Samples[0].Time, End: t.Sample.Time}
		out.WindowValid = true
	}
	if s.scopeWindow != nil {
		out.Window = *s.scopeWindow
		out.WindowValid = len(t.Samples) > 0 && scopeTimeInside(out.Window.Start, t.Samples[0].Time, t.Sample.Time) && scopeTimeInside(out.Window.End, t.Samples[0].Time, t.Sample.Time)
		if !out.WindowValid {
			out.WindowReason = "fixed interval is outside fresh retained history"
		}
	}

	if out.WindowValid && s.scopeWindow != nil {
		found := false
		for _, sample := range t.Samples {
			if scopeWindowTimeInside(sample.Time, out.Window.Start, out.Window.End) {
				found = true
				break
			}
		}
		if !found {
			out.WindowValid = false
			out.WindowReason = "No exact saved samples in this interval"
		}
	}

	for i := range out.Traces {
		for _, sample := range t.Samples {
			if value, ok := frameValue(sample, out.Traces[i].Probe); ok && out.WindowValid && scopeWindowTimeInside(sample.Time, out.Window.Start, out.Window.End) {
				out.Traces[i].Points = append(out.Traces[i].Points, ScopePoint{sample.Time, value})
			}
		}
	}
	for _, trace := range out.Traces {
		if len(trace.Points) > 0 && !t.Stale {
			out.Statistics = append(out.Statistics, scopeStatistics(trace.Points, trace.Probe.ID))
		}
	}
	cursor := func(request *float64) *ScopeCursor {
		if request == nil {
			return nil
		}
		r := &ScopeCursor{Requested: *request, Reason: "outside retained history"}
		if len(t.Samples) == 0 || !scopeTimeInside(*request, t.Samples[0].Time, t.Sample.Time) {
			return r
		}
		index := nearestSample(t.Samples, *request)
		r.Time = t.Samples[index].Time
		r.Valid = !t.Stale
		if s.scopeWindow != nil && (!out.WindowValid || !scopeWindowTimeInside(r.Time, out.Window.Start, out.Window.End)) {
			r.Valid = false
			r.Reason = "outside visible window"
			return r
		}
		r.Reason = ""
		if t.Stale {
			r.Reason = "stale run"
		}
		return r
	}
	out.A = cursor(s.cursorA)
	out.B = cursor(s.cursorB)
	if out.A != nil && out.B != nil && out.A.Valid && out.B.Valid {
		delta := out.B.Time - out.A.Time
		out.DeltaSeconds = &delta
		if delta != 0 {
			f := 1 / math.Abs(delta)
			if finite(f) {
				out.Frequency = &f
			}
		}
		ai, bi := nearestSample(t.Samples, out.A.Time), nearestSample(t.Samples, out.B.Time)
		for _, trace := range out.Traces {
			a, aok := frameValue(t.Samples[ai], trace.Probe)
			b, bok := frameValue(t.Samples[bi], trace.Probe)
			if aok && bok {
				out.Readings = append(out.Readings, CursorReading{trace.Probe.ID, a, b, b - a})
			}
		}
	}
	return out
}

// Equal-distance and duplicate-time ties choose the latest/right-hand sample.
func nearestSample(samples []TraceSample, at float64) int {
	best := 0
	distance := math.Inf(1)
	for i, p := range samples {
		d := math.Abs(p.Time - at)
		if d <= distance {
			best = i
			distance = d
		}
	}
	return best
}
func (s *Service) scopeCommandLocked(command string, raw json.RawMessage) error {
	switch command {
	case "scope.trace.select":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		return s.focusTraceLocked(p)
	case "scope.probe.remove":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		ids := s.scopeProbes
		if !s.scopeConfigured {
			c, _, _ := s.graph.Compile()
			ids = defaultProbes(c)
		}
		for i, id := range ids {
			if id == p.ID {
				s.scopeProbes = append(append([]string{}, ids[:i]...), ids[i+1:]...)
				s.scopeConfigured = true
				if s.focusProbe == p.ID {
					s.focusProbe = ""
				}
				return nil
			}
		}
		return &actionError{"not-found", "Probe is not selected"}
	case "scope.probes.set":
		var p ProbeParams
		if err := params(raw, &p, "ids"); err != nil {
			return err
		}
		if p.IDs == nil || len(p.IDs) > MaxScopeProbes {
			return fmt.Errorf("ids must be an array of at most six probes")
		}
		available := map[string]bool{}
		for _, probe := range probeInventory(s.graph) {
			available[probe.ID] = true
		}
		seen := map[string]bool{}
		for _, id := range p.IDs {
			if !available[id] || seen[id] {
				return fmt.Errorf("unknown or duplicate probe %q", id)
			}
			seen[id] = true
		}
		s.scopeProbes = append([]string{}, p.IDs...)
		s.scopeConfigured = true
		if !seen[s.focusProbe] {
			s.focusProbe = ""
		}
	case "scope.cursor.select", "scope.cursor.clear":
		var p CursorSelectParams
		if err := params(raw, &p, "cursor"); err != nil {
			return err
		}
		if p.Cursor != "a" && p.Cursor != "b" {
			return fmt.Errorf("cursor must be a or b")
		}
		if command == "scope.cursor.select" {
			s.activeCursor = p.Cursor
		} else if p.Cursor == "a" {
			s.cursorA = nil
		} else {
			s.cursorB = nil
		}
	case "scope.cursor.set":
		var p CursorParams
		if err := params(raw, &p, "cursor", "time_s"); err != nil {
			return err
		}
		if p.Cursor != "a" && p.Cursor != "b" {
			return fmt.Errorf("cursor must be a or b")
		}
		if !finite(p.Time) || s.transient == nil || s.transient.Stale || len(s.transient.Samples) == 0 || !scopeTimeInside(p.Time, s.transient.Samples[0].Time, s.transient.Sample.Time) {
			return fmt.Errorf("cursor time must be inside a fresh retained transient history")
		}
		if p.Cursor == "a" {
			s.cursorA = &p.Time
		} else {
			s.cursorB = &p.Time
		}
	}
	return nil
}
func cloneScope(in Scope) Scope {
	out := in
	if in.Unavailable != nil {
		out.Unavailable = append([]UnavailableProbe{}, in.Unavailable...)
	}
	out.Statistics = append([]ScopeStatistics{}, in.Statistics...)
	out.Available = append([]Probe{}, in.Available...)
	out.Selected = append([]string{}, in.Selected...)
	out.Readings = append([]CursorReading{}, in.Readings...)
	out.Traces = append([]ScopeTrace{}, in.Traces...)
	for i := range out.Traces {
		out.Traces[i].Points = append([]ScopePoint{}, in.Traces[i].Points...)
	}
	if in.A != nil {
		v := *in.A
		out.A = &v
	}
	if in.B != nil {
		v := *in.B
		out.B = &v
	}
	if in.DeltaSeconds != nil {
		v := *in.DeltaSeconds
		out.DeltaSeconds = &v
	}
	if in.Frequency != nil {
		v := *in.Frequency
		out.Frequency = &v
	}
	return out
}

// Keep the traces that were already visible when a drawing becomes incomplete
// or loses a plotted source. Undo can restore their stable associations.
func (s *Service) retainInterruptedDefaultProbesLocked(before Graph) {
	if s.scopeConfigured || s.transient == nil {
		return
	}
	previous, _, diagnostic := before.Compile()
	if diagnostic != nil {
		return
	}
	ids := defaultProbes(previous)
	available := map[string]bool{}
	for _, p := range probeInventory(s.graph) {
		available[p.ID] = true
	}
	_, _, diagnostic = s.graph.Compile()
	interrupted := diagnostic != nil
	for _, id := range ids {
		if !available[id] {
			interrupted = true
		}
	}
	if interrupted {
		s.scopeProbes = append([]string{}, ids...)
		s.scopeConfigured = true
	}
}
