package workbench

import (
	"context"
	"math"
)

const NPNTransientAnalysis = "NPN transient"
const NPNTransientEngineID = "elab.npn-switch-quasistatic.v1"

func IsTransientAnalysis(analysis string) bool {
	return analysis == "RC transient" || analysis == NPNTransientAnalysis || analysis == MixedAnalysis
}
func NPNPulseExampleGraph() Graph {
	g := NPNExampleGraph()
	for i, c := range g.Components {
		if c.ID == "VIN" {
			p, _ := MakeComponent(c.ID, "pulse-source", c.Position, "5 V")
			g.Components[i] = p
		}
	}
	return g
}
func pulseHigh(cfg TransientSettings, at float64) bool {
	period := cfg.HighSeconds + cfg.LowSeconds
	phase := math.Mod(at, period)
	epsilon := period * 1e-10
	if phase >= period-epsilon {
		phase = 0
	}
	return phase < cfg.HighSeconds-epsilon
}
func nextPulseEdge(cfg TransientSettings, at float64) float64 {
	period := cfg.HighSeconds + cfg.LowSeconds
	cycle := math.Floor((at + period*1e-10) / period)
	edge := cycle*period + cfg.HighSeconds
	if edge <= at+period*1e-10 {
		edge = (cycle + 1) * period
	}
	return edge
}
func npnPulseInput(c Compiled, t *Transient, at float64) float64 {
	if t.Mode == "low" {
		return 0
	}
	if t.Mode == "high" || pulseHigh(t.Settings, at) {
		return c.Input.Value
	}
	return 0
}
func npnTraceSample(c Compiled, high, low *NPNResult, at, input float64) TraceSample {
	r := low
	if input > 0 {
		r = high
	}
	p := TraceSample{Time: at, Input: input, Voltage: r.VCE, Current: r.CollectorCurrent, BaseCurrent: r.BaseCurrent, VBE: r.VBE, State: r.State}
	currents := map[string]float64{c.Transistor.ID + ".base": r.BaseCurrent, c.Transistor.ID + ".collector": r.CollectorCurrent, c.Transistor.ID + ".emitter": r.EmitterCurrent, c.LED.ID: r.CollectorCurrent, c.Source.ID: r.CollectorCurrent, c.Input.ID: r.BaseCurrent}
	if c.Source.ID == c.Input.ID {
		currents[c.Source.ID] = r.CollectorCurrent + r.BaseCurrent
	}
	for _, part := range []Component{c.Resistor, c.BaseResistor} {
		currents[part.ID] = (r.Nodes[c.TerminalNet[part.ID+".1"]] - r.Nodes[c.TerminalNet[part.ID+".2"]]) / part.Value
	}
	return captureFrame(c, p, r.Nodes, currents)
}
func appendInputEvent(t *Transient, e InputEvent) {
	t.Events = append(t.Events, e)
	if len(t.Events) > 64 {
		t.Events = append([]InputEvent{}, t.Events[len(t.Events)-64:]...)
	}
}
func advanceNPNSample(c Compiled, t *Transient, high, low *NPNResult, end float64) {
	at := t.Sample.Time
	if t.Mode == "auto" {
		for {
			edge := nextPulseEdge(t.Settings, at)
			if edge > end || edge <= at {
				break
			}
			// Explicit left/right samples at the same time retain the discontinuity,
			// even when a pulse edge falls between requested sampling times.
			appendTrace(t, npnTraceSample(c, high, low, edge, npnPulseInput(c, t, at)))
			appendTrace(t, npnTraceSample(c, high, low, edge, npnPulseInput(c, t, edge)))
			mode := "pulse-low"
			if pulseHigh(t.Settings, edge) {
				mode = "pulse-high"
			}
			appendInputEvent(t, InputEvent{edge, mode})
			at = edge
		}
	}
	t.Sample = npnTraceSample(c, high, low, end, npnPulseInput(c, t, end))
	appendTrace(t, t.Sample)
}
func (s *Service) publishNPNTransientLocked() {
	r := s.npnLow
	if s.transient.Sample.Input > 0 {
		r = s.npnHigh
	}
	s.npn = cloneNPN(r)
}
func (s *Service) startNPNTransientLocked(c Compiled) {
	// A memoryless device model only needs two solves per electrical run. Time
	// stepping selects these immutable phase results without holding the service
	// lock for a nonlinear solve on every live sample.
	high, low, err := solveNPNPhases(context.Background(), c)
	if err == nil {
		s.npnHigh = high
		s.npnLow = low
	}
	if err != nil {
		s.status = "error"
		s.diagnostics = []Diagnostic{{"solve-failed", err.Error(), c.Transistor.ID}}
		return
	}
	for _, r := range []*NPNResult{s.npnHigh, s.npnLow} {
		r.RunID = s.run
		r.ElectricalRevision = s.electrical
	}
	if s.pendingInputMode != "" {
		s.inputMode = s.pendingInputMode
		s.pendingInputMode = ""
	}
	if s.inputMode != "auto" && s.inputMode != "high" && s.inputMode != "low" {
		s.inputMode = "auto"
	}
	t := &Transient{Engine: NPNTransientEngineID, RunID: s.run, ElectricalRevision: s.electrical, SourceID: c.Input.ID, SupplyID: c.Source.ID, ResistorID: c.Resistor.ID, BaseResistorID: c.BaseResistor.ID, TransistorID: c.Transistor.ID, LEDID: c.LED.ID, OutputNet: c.Collector, Settings: s.transientSettings, Mode: s.inputMode, Samples: []TraceSample{}, Events: []InputEvent{{0, "restart"}}}
	t.Sample = npnTraceSample(c, s.npnHigh, s.npnLow, 0, npnPulseInput(c, t, 0))
	appendTrace(t, t.Sample)
	s.result = nil
	s.transient = t
	s.publishNPNTransientLocked()
	if s.paused {
		s.status = "paused"
	} else {
		s.status = "running"
		s.launchTransientLocked(c)
	}
}
func (s *Service) advanceTransientLocked(c Compiled, end float64) error {
	if c.Analysis == MixedAnalysis {
		st, t, err := advanceMixed(s.mixedState, s.transient, end)
		if err != nil {
			return err
		}
		s.mixedState = st
		s.mixed = st.result
		s.transient = t
		return nil
	}
	if c.Analysis == NPNTransientAnalysis {
		advanceNPNSample(c, s.transient, s.npnHigh, s.npnLow, end)
		s.publishNPNTransientLocked()
	} else {
		advanceRCSample(c, s.transient, end)
	}
	return nil
}
func (s *Service) applyTransientInputLocked(c Compiled, mode string) error {
	if c.Analysis == MixedAnalysis {
		st, t, err := overrideMixed(s.mixedState, s.transient, mode)
		if err != nil {
			return &actionError{"solve-failed", err.Error()}
		}
		s.mixedState = st
		s.mixed = st.result
		s.transient = t
		s.inputMode = mode
		s.pendingInputMode = ""
		return nil
	}
	if c.Analysis != NPNTransientAnalysis {
		s.applyRCInputLocked(c, mode)
		return nil
	}
	t := s.transient
	s.inputMode = mode
	s.pendingInputMode = ""
	t.Mode = mode
	t.PendingMode = ""
	t.Sample = npnTraceSample(c, s.npnHigh, s.npnLow, t.Sample.Time, npnPulseInput(c, t, t.Sample.Time))
	appendTrace(t, t.Sample)
	appendInputEvent(t, InputEvent{t.Sample.Time, mode})
	s.publishNPNTransientLocked()
	return nil
}
