package workbench

const NPNTransientAnalysis = "NPN transient"
const NPNTransientEngineID = "elab.npn-switch-quasistatic.v1"

func IsTransientAnalysis(analysis string) bool {
	return analysis == InductorAnalysis || analysis == WaveformAnalysis || analysis == "RC transient" || analysis == NPNTransientAnalysis || analysis == MixedAnalysis || analysis == NetworkTransientAnalysis
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
			if !sourceCornerReached(edge, end) || edge <= at {
				break
			}
			if edge > end {
				edge = end
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
func (s *Service) startNPNTransientLocked(c Compiled) { s.startInitialLocked(c) }
func (s *Service) advanceTransientLocked(c Compiled, end float64) error {
	if IsNetworkAnalysis(c.Analysis) {
		st, t, err := s.networkAdvance(s.mixedState, s.transient, end, s.graph)
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
	if IsNetworkAnalysis(c.Analysis) {
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
