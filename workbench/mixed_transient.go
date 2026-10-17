package workbench

import (
	"context"
	"fmt"
	"math"
)

func IsNPNTransientAnalysis(a string) bool {
	return a == InductorAnalysis || a == WaveformAnalysis || a == NPNTransientAnalysis || a == MixedAnalysis || a == NetworkTransientAnalysis
}
func IsNPNTransientEngine(e string) bool {
	return e == InductorEngineID || e == WaveformEngineID || e == NPNTransientEngineID || e == MixedEngineID || e == NetworkEngineID || e == "elab.network-waveforms-mna-be.v1" || e == "elab.mixed-mna-be.v1" || e == "elab.network-mna-be.v1"
}
func mixedSample(compiled Compiled, r *MixedResult, input float64) TraceSample {
	_, region := MixedRegion(r)
	p := TraceSample{Time: r.Time, Input: input, Voltage: r.Nodes[compiled.Output], State: region}
	if len(r.MOSFETs) > 0 {
		m := r.MOSFETs[0]
		p.Voltage, p.Current = m.VDS, m.DrainCurrent
	}
	if len(r.Transistors) > 0 {
		q := r.Transistors[0]
		p.Voltage = q.VCE
		p.Current = q.CollectorCurrent
		p.BaseCurrent = q.BaseCurrent
		p.VBE = q.VBE
	}
	if len(r.LEDs) > 0 {
		p.Current = r.LEDs[0].Current
	} else if len(r.Diodes) > 0 {
		p.Current = r.Diodes[0].Current
	} else if len(r.Resistors) > 0 {
		p.Current = r.Resistors[0].Current
	} else if len(r.Switches) > 0 {
		p.Current = r.Switches[0].Current
	} else if len(r.Potentiometers) > 0 {
		p.Current = r.Potentiometers[0].Terminal1Current
	}
	if len(r.OpAmps) > 0 && len(r.Transistors) == 0 && len(r.MOSFETs) == 0 {
		p.Voltage, p.Current = r.OpAmps[0].OutputVoltage, r.OpAmps[0].OutputCurrent
	}
	if len(r.Capacitors) > 0 {
		cap := r.Capacitors[0]
		p.Energy = cap.Energy
		p.CapacitorVoltage = cap.Voltage
		if len(r.Transistors) == 0 && len(r.MOSFETs) == 0 {
			p.Voltage = r.Nodes[compiled.Output]
		}
	}
	if len(r.Inductors) > 0 {
		p.Energy = 0
		for _, part := range r.Inductors {
			p.Energy += part.Energy
		}
		for _, part := range r.Capacitors {
			p.Energy += part.Energy
		}
	}
	currents := map[string]float64{}
	for _, u := range r.OpAmps {
		currents[u.ID] = u.OutputCurrent
		currents[u.ID+".noninverting"] = u.NoninvertingCurrent
		currents[u.ID+".inverting"] = u.InvertingCurrent
		currents[u.ID+".out"] = u.OutputCurrent
		currents[u.ID+".vplus"] = u.PositiveSupplyCurrent
		currents[u.ID+".vminus"] = u.NegativeSupplyCurrent
	}
	for _, p := range r.Potentiometers {
		currents[p.ID+".1"] = p.Terminal1Current
		currents[p.ID+".2"] = p.Terminal2Current
		currents[p.ID+".wiper"] = p.WiperCurrent
	}
	for _, m := range r.MOSFETs {
		currents[m.ID+".gate"] = m.GateCurrent
		currents[m.ID+".drain"] = m.DrainCurrent
		currents[m.ID+".source"] = m.SourceCurrent
	}
	for _, q := range r.Transistors {
		currents[q.TransistorID+".base"] = q.BaseCurrent
		currents[q.TransistorID+".collector"] = q.CollectorCurrent
		currents[q.TransistorID+".emitter"] = q.EmitterCurrent
	}
	for _, inductor := range r.Inductors {
		currents[inductor.ID] = inductor.Current
	}
	for _, cap := range r.Capacitors {
		currents[cap.ID] = cap.Current
	}
	for _, parts := range [][]BranchResult{r.LEDs, r.Resistors, r.Sources, r.Switches, r.Diodes} {
		for _, part := range parts {
			currents[part.ID] = part.Current
		}
	}
	return captureFrame(compiled, p, r.Nodes, currents)
}
func (s *Service) startMixedLocked(c Compiled) { s.startInitialLocked(c) }

// Stage a whole requested output sample with one shared work budget across all
// per-source corners. A failure never publishes partial capacitor memory/time.
func advanceMixed(state *mixedState, t *Transient, end float64) (*mixedState, *Transient, error) {
	return advanceMixedBudget(state, t, end, newIntegrationBudget())
}

func advanceMixedBudget(state *mixedState, t *Transient, end float64, budget *integrationBudget) (*mixedState, *Transient, error) {
	trace := cloneTransient(t)
	cur := state
	for cur.result.Time < end {
		at := cur.result.Time
		corner := cur.net.nextSourceCorner(trace, at)
		stop := math.Min(end, corner)
		mergeOutput := end-stop < 1e-13
		if len(cur.net.inductors) > 0 {
			// A fast coil can change measurably in tens of femtoseconds.
			// Reconcile representations of the same time, not distinct
			// source/output boundaries separated by an absolute duration.
			mergeOutput = sameSampleTime(stop, end)
		}
		if mergeOutput {
			stop = end
		}
		next, err := cur.advanceBudget(trace, stop, budget)
		if err != nil {
			return nil, nil, err
		}
		cur = next
		if sourceCornerReached(corner, end) {
			legacy := false
			for _, p := range cur.net.sources {
				if nextSourceCorner(p, trace, at, p.ID == cur.net.c.Source.ID) == corner && p.Kind == "pulse-source" {
					legacy = true
				}
			}
			if legacy {
				appendTrace(trace, mixedSample(cur.net.c, cur.result, cur.result.Nodes[cur.net.c.Supply]))
			}
			if err := budget.take(); err != nil {
				return nil, nil, err
			}
			right, err := cur.net.solve(cur.x, cur.capV, 0, trace, stop)
			if err != nil {
				return nil, nil, err
			}
			cur = right
			appendTrace(trace, mixedSample(cur.net.c, cur.result, cur.result.Nodes[cur.net.c.Supply]))
			for _, p := range cur.net.sources {
				if nextSourceCorner(p, trace, at, p.ID == cur.net.c.Source.ID) == corner {
					mode := "pulse:" + p.ID + ":corner"
					if p.Kind == "pulse-source" {
						mode = "pulse-low"
						if pulseHigh(trace.Settings, stop) {
							mode = "pulse-high"
						}
					}
					appendInputEvent(trace, InputEvent{Time: stop, Mode: mode})
				}
			}
		}
	}
	trace.Sample = mixedSample(cur.net.c, cur.result, cur.result.Nodes[cur.net.c.Supply])
	appendTrace(trace, trace.Sample)
	return cur, trace, nil
}
func overrideMixed(state *mixedState, t *Transient, mode string) (*mixedState, *Transient, error) {
	return overrideMixedContext(context.Background(), state, t, mode)
}
func overrideMixedContext(ctx context.Context, state *mixedState, t *Transient, mode string) (*mixedState, *Transient, error) {
	trace := cloneTransient(t)
	trace.Mode = mode
	trace.PendingMode = ""
	next, err := state.net.solveContext(ctx, state.x, state.capV, 0, trace, state.result.Time)
	if err != nil {
		return nil, nil, err
	}
	trace.Sample = mixedSample(next.net.c, next.result, next.net.sourceVoltage(next.net.c.Source, trace, next.result.Time))
	appendTrace(trace, trace.Sample)
	appendInputEvent(trace, InputEvent{next.result.Time, mode})
	return next, trace, nil
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
