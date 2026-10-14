package workbench

import (
	"context"
	"fmt"
)

// Verification reuses the same numerical kernels with private circuit state.
// It never creates a live clock or installs its readings into the editor.
type verificationKernel struct {
	compiled  Compiled
	trace     *Transient
	mixed     *mixedState
	high, low *NPNResult
	led       *Result
	npn       *NPNResult
}

func newVerificationKernel(ctx context.Context, input verificationInput) (*verificationKernel, error) {
	c := input.compiled
	k := &verificationKernel{compiled: c, trace: &Transient{RunID: input.serial, ElectricalRevision: input.metadata.ElectricalRevision, Settings: input.settings, Mode: input.mode, Samples: []TraceSample{}, Events: []InputEvent{}}}
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
			r.RunID = input.serial
			r.ElectricalRevision = input.metadata.ElectricalRevision
		}
		t.Engine = NPNTransientEngineID
		t.Sample = npnTraceSample(c, high, low, 0, npnPulseInput(c, t, 0))
		k.updateNPN()
	case MixedAnalysis, NetworkDCAnalysis, NetworkTransientAnalysis, WaveformAnalysis:
		n := newMixedNetwork(c)
		old := make([]float64, len(n.caps))
		for i, p := range n.caps {
			old[i] = input.settings.InitialVoltage
			if p.InitialVoltage != nil {
				old[i] = *p.InitialVoltage
			}
		}
		t.Engine = networkEngine(c.Analysis)
		state, err := n.solve(nil, old, 0, t, 0)
		if err != nil {
			return nil, err
		}
		k.mixed = state
		t.Sample = mixedSample(c, state.result, n.sourceVoltage(c.Source, t, 0))
	case NPNAnalysis:
		r, err := SolveNPN(ctx, c)
		if err != nil {
			return nil, err
		}
		r.RunID = input.serial
		r.ElectricalRevision = input.metadata.ElectricalRevision
		k.npn = &r
		t.Engine = NPNEngineID
		t.Sample = npnTraceSample(c, &r, &r, 0, c.Input.Value)
	default:
		r, err := SolveLED(ctx, c.Circuit)
		if err != nil {
			return nil, err
		}
		r = c.Annotate(r)
		r.RunID = input.serial
		r.ElectricalRevision = input.metadata.ElectricalRevision
		k.led = &r
		t.Engine = EngineID
		currents := map[string]float64{c.Source.ID: r.Current, c.LED.ID: r.Current, c.Resistor.ID: (r.Nodes[c.TerminalNet[c.Resistor.ID+".1"]] - r.Nodes[c.TerminalNet[c.Resistor.ID+".2"]]) / c.Resistor.Value}
		t.Sample = captureFrame(c, TraceSample{Time: 0, Input: c.Source.Value, Voltage: r.LEDVoltage, Current: r.Current}, r.Nodes, currents)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	appendTrace(t, t.Sample)
	return k, nil
}
func (k *verificationKernel) updateNPN() {
	r := k.low
	if k.trace.Sample.Input > 0 {
		r = k.high
	}
	k.npn = cloneNPN(r)
}
func (k *verificationKernel) advance(ctx context.Context, end float64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch k.compiled.Analysis {
	case MixedAnalysis, NetworkDCAnalysis, NetworkTransientAnalysis, WaveformAnalysis:
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
