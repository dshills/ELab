package workbench

import (
	"context"
	"fmt"
)

// Live initialization and verification reuse numerical kernels with private circuit state.
// It never creates a live clock or installs its readings into the editor.
type kernelPlan struct {
	compiled      Compiled
	settings      TransientSettings
	mode          string
	run, revision uint64
}

type simulationKernel struct {
	compiled  Compiled
	trace     *Transient
	mixed     *mixedState
	high, low *NPNResult
	led       *Result
	npn       *NPNResult
}

func newSimulationKernel(ctx context.Context, input kernelPlan) (*simulationKernel, error) {
	c := input.compiled
	k := &simulationKernel{compiled: c, trace: &Transient{RunID: input.run, ElectricalRevision: input.revision, Settings: input.settings, Mode: input.mode, Samples: []TraceSample{}, Events: []InputEvent{}}}
	t := k.trace
	switch c.Analysis {
	case "RC transient":
		t.Engine = RCEngineID
		t.TimeConstant = c.Resistor.Value * c.Capacitor.Value
		v := input.settings.InitialVoltage
		if c.Capacitor.InitialVoltage != nil {
			v = *c.Capacitor.InitialVoltage
			if c.TerminalNet[c.Capacitor.ID+".1"] != c.Output {
				v = -v
			}
		}
		t.Sample = rcSample(c, t, 0, rcInput(c, t, 0), v)
	case NPNTransientAnalysis:
		high, low, err := solveNPNPhases(ctx, c)
		if err != nil {
			return nil, err
		}
		k.high, k.low = high, low
		for _, r := range []*NPNResult{high, low} {
			r.RunID = input.run
			r.ElectricalRevision = input.revision
		}
		t.Engine = NPNTransientEngineID
		t.Sample = npnTraceSample(c, high, low, 0, npnPulseInput(c, t, 0))
		k.updateNPN()
	case MixedAnalysis, NetworkDCAnalysis, NetworkTransientAnalysis, WaveformAnalysis, InductorAnalysis, Timer555Analysis, SequentialAnalysis, ControlAnalysis:
		n := newMixedNetwork(c)
		old := make([]float64, len(n.caps))
		for i, p := range n.caps {
			old[i] = input.settings.InitialVoltage
			if p.InitialVoltage != nil {
				old[i] = *p.InitialVoltage
			}
		}
		t.Engine = networkEngine(c.Analysis)
		state, err := n.solveContext(ctx, nil, old, 0, t, 0)
		if err != nil {
			return nil, err
		}
		k.mixed = state
		input := n.sourceVoltage(c.Source, t, 0)
		if c.Source.Kind == "battery" || c.Source.Kind == "current-source" {
			input = mixedSourceVoltage(c, state.result)
		}
		t.Sample = mixedSample(c, state.result, input)
	case NPNAnalysis:
		r, err := SolveNPN(ctx, c)
		if err != nil {
			return nil, err
		}
		r.RunID = input.run
		r.ElectricalRevision = input.revision
		k.npn = &r
		t.Engine = NPNEngineID
		t.Sample = npnTraceSample(c, &r, &r, 0, c.Input.Value)
	default:
		r, err := SolveLED(ctx, c.Circuit)
		if err != nil {
			return nil, err
		}
		r = c.Annotate(r)
		r.RunID = input.run
		r.ElectricalRevision = input.revision
		k.led = &r
		t.Engine = EngineID
		t.Sample = ledTraceSample(c, r)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	appendTrace(t, t.Sample)
	return k, nil
}
func (k *simulationKernel) updateNPN() {
	r := k.low
	if k.trace.Sample.Input > 0 {
		r = k.high
	}
	k.npn = cloneNPN(r)
}
func (k *simulationKernel) advance(ctx context.Context, end float64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch k.compiled.Analysis {
	case MixedAnalysis, NetworkDCAnalysis, NetworkTransientAnalysis, WaveformAnalysis, InductorAnalysis, Timer555Analysis, SequentialAnalysis, ControlAnalysis:
		state, trace, err := advanceMixed(k.mixed, k.trace, end)
		if err != nil {
			return err
		}
		k.mixed, k.trace = state, trace
	case NPNTransientAnalysis:
		advanceNPNSample(k.compiled, k.trace, k.high, k.low, end)
		k.updateNPN()
	case "RC transient":
		advanceRCSample(k.compiled, k.trace, end)
	default:
		return fmt.Errorf("DC verification has no transient time")
	}
	return ctx.Err()
}
func verificationRow(sample TraceSample, probes []Probe) (VerificationRow, error) {
	row := VerificationRow{Time: sample.Time, Values: []float64{}}
	for _, probe := range probes {
		v, ok := frameValue(sample, probe)
		if !ok || !finite(v) {
			return row, fmt.Errorf("solver did not produce a finite reading for probe %s", probe.ID)
		}
		row.Values = append(row.Values, v)
	}
	return row, nil
}
func solveNPNPhases(ctx context.Context, c Compiled) (*NPNResult, *NPNResult, error) {
	dc := c
	dc.Analysis = NPNAnalysis
	dc.Input.Kind = "dc-source"
	high, err := SolveNPN(ctx, dc)
	if err != nil {
		return nil, nil, err
	}
	dc.Input.Value = 0
	low, err := SolveNPN(ctx, dc)
	if err != nil {
		return nil, nil, err
	}
	return &high, &low, nil
}
