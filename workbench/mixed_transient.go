package workbench

import (
	"fmt"
	"math"
)

func IsNPNTransientAnalysis(a string) bool { return a == NPNTransientAnalysis || a == MixedAnalysis }
func IsNPNTransientEngine(e string) bool   { return e == NPNTransientEngineID || e == MixedEngineID }
func mixedSample(compiled Compiled, r *MixedResult, input float64) TraceSample {
	q := r.Transistors[0]
	current := q.CollectorCurrent
	if len(r.LEDs) > 0 {
		current = r.LEDs[0].Current
	}
	c := r.Capacitors[0]
	p := TraceSample{Time: r.Time, Input: input, Voltage: q.VCE, Current: current, Energy: c.Energy, BaseCurrent: q.BaseCurrent, VBE: q.VBE, State: q.State, CapacitorVoltage: c.Voltage}
	currents := map[string]float64{}
	for _, q := range r.Transistors {
		currents[q.TransistorID+".base"] = q.BaseCurrent
		currents[q.TransistorID+".collector"] = q.CollectorCurrent
		currents[q.TransistorID+".emitter"] = q.EmitterCurrent
	}
	for _, cap := range r.Capacitors {
		currents[cap.ID] = cap.Current
	}
	for _, parts := range [][]BranchResult{r.LEDs, r.Resistors, r.Sources} {
		for _, part := range parts {
			currents[part.ID] = part.Current
		}
	}
	return captureFrame(compiled, p, r.Nodes, currents)
}
func (s *Service) startMixedLocked(c Compiled) {
	n := newMixedNetwork(c)
	old := make([]float64, len(n.caps))
	for i, p := range n.caps {
		old[i] = s.transientSettings.InitialVoltage
		if p.InitialVoltage != nil {
			old[i] = *p.InitialVoltage
		}
	}
	mode := s.inputMode
	if s.pendingInputMode != "" {
		mode = s.pendingInputMode
	}
	if mode != "auto" && mode != "high" && mode != "low" {
		mode = "auto"
	}
	t := &Transient{Engine: MixedEngineID, RunID: s.run, ElectricalRevision: s.electrical, SourceID: c.Source.ID, TransistorID: c.Transistor.ID, LEDID: c.LED.ID, CapacitorID: c.Capacitor.ID, OutputNet: c.Output, Settings: s.transientSettings, Mode: mode, Samples: []TraceSample{}, Events: []InputEvent{{0, "restart"}}}
	st, err := n.solve(nil, old, 0, t, 0)
	if err != nil {
		s.status = "error"
		s.diagnostics = []Diagnostic{{"mixed-initialize-failed", err.Error(), ""}}
		return
	}
	s.inputMode = mode
	s.pendingInputMode = ""
	s.result = nil
	s.npn = nil
	s.npnHigh = nil
	s.npnLow = nil
	s.transient = t
	s.mixedState = st
	s.mixed = st.result
	t.Sample = mixedSample(c, st.result, n.sourceVoltage(c.Source, t, 0))
	appendTrace(t, t.Sample)
	if s.paused {
		s.status = "paused"
	} else {
		s.status = "running"
		s.launchTransientLocked(c)
	}
}
func autoMixedPulse(n *mixedNetwork, t *Transient) bool {
	for _, p := range n.sources {
		if p.Kind == "pulse-source" && (p.ID != n.c.Source.ID || t.Mode == "auto") {
			return true
		}
	}
	return false
}

// Stage a whole requested output sample. A convergence failure cannot publish
// partially advanced capacitor memory, device results, events or time.
func advanceMixed(state *mixedState, t *Transient, end float64) (*mixedState, *Transient, error) {
	trace := cloneTransient(t)
	cur := state
	at := cur.result.Time
	for at < end {
		stop := end
		edgeHit := false
		if autoMixedPulse(cur.net, trace) {
			edge := nextPulseEdge(trace.Settings, at)
			if edge > at && edge <= stop {
				stop = edge
				edgeHit = true
			}
		}
		next, err := cur.advance(trace, stop)
		if err != nil {
			return nil, nil, err
		}
		cur = next
		at = stop
		if edgeHit {
			// The dynamic step used the source's left-hand value. Preserve capacitor
			// voltages while solving the algebraic right-hand state at the exact edge.
			left := mixedSample(cur.net.c, cur.result, cur.net.sourceVoltage(cur.net.c.Source, trace, math.Max(0, at-(trace.Settings.HighSeconds+trace.Settings.LowSeconds)*1e-9)))
			appendTrace(trace, left)
			right, err := cur.net.solve(cur.x, cur.capV, 0, trace, at)
			if err != nil {
				return nil, nil, err
			}
			cur = right
			appendTrace(trace, mixedSample(cur.net.c, cur.result, cur.net.sourceVoltage(cur.net.c.Source, trace, at)))
			mode := "pulse-low"
			if pulseHigh(trace.Settings, at) {
				mode = "pulse-high"
			}
			appendInputEvent(trace, InputEvent{at, mode})
		}
	}
	trace.Sample = mixedSample(cur.net.c, cur.result, cur.net.sourceVoltage(cur.net.c.Source, trace, end))
	appendTrace(trace, trace.Sample)
	return cur, trace, nil
}
func overrideMixed(state *mixedState, t *Transient, mode string) (*mixedState, *Transient, error) {
	trace := cloneTransient(t)
	trace.Mode = mode
	trace.PendingMode = ""
	next, err := state.net.solve(state.x, state.capV, 0, trace, state.result.Time)
	if err != nil {
		return nil, nil, err
	}
	trace.Sample = mixedSample(next.net.c, next.result, next.net.sourceVoltage(next.net.c.Source, trace, next.result.Time))
	appendTrace(trace, trace.Sample)
	appendInputEvent(trace, InputEvent{next.result.Time, mode})
	return next, trace, nil
}
func (s *Service) stepMixedLocked(duration float64) error {
	cur, t := s.mixedState, s.transient
	mode := t.Mode
	if t.PendingMode != "" {
		var err error
		mode = t.PendingMode
		cur, t, err = overrideMixed(cur, t, mode)
		if err != nil {
			return &actionError{"solve-failed", err.Error()}
		}
	}
	end := t.Sample.Time + duration
	for t.Sample.Time < end {
		next := math.Min(end, t.Sample.Time+s.transientSettings.StepSeconds)
		if end-next < 1e-12 {
			next = end
		}
		var err error
		cur, t, err = advanceMixed(cur, t, next)
		if err != nil {
			return &actionError{"solve-failed", err.Error()}
		}
	}
	s.mixedState = cur
	s.mixed = cur.result
	s.transient = t
	s.inputMode = mode
	s.pendingInputMode = ""
	return nil
}
func (s *Service) capacitorInitialLocked(p ValueParams) error {
	c, ok := s.graph.Component(p.ID)
	if !ok || c.Kind != "capacitor" {
		return fmt.Errorf("choose a capacitor to set its initial voltage")
	}
	var v *float64
	if p.Value != "default" {
		value, err := ParseValue(p.Value, "V")
		if err != nil {
			return err
		}
		if !finite(value) || math.Abs(value) > 12 {
			return fmt.Errorf("capacitor initial voltage must be within ±12 V")
		}
		if value == 0 {
			value = 0
		}
		v = &value
	}
	for i := range s.graph.Components {
		if s.graph.Components[i].ID == p.ID {
			s.graph.Components[i].InitialVoltage = v
			break
		}
	}
	return nil
}
