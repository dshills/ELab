package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/elab/internal/testutil"
)

var sequentialKinds = []string{"d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter"}

func seqVolts() [15]float64 { v := [15]float64{}; v[13] = 5; return v }
func seqDecide(t *testing.T, kind string, v [15]float64, m SequentialMemory) SequentialMemory {
	t.Helper()
	n, err := sequentialDecision(kind, v, m, true)
	if err != nil {
		t.Fatal(err)
	}
	return n
}
func TestSequentialTruthTablesResetPriorityAndEdgeArming(t *testing.T) {
	for _, kind := range sequentialKinds {
		for oldValue := 0; oldValue < sequentialModulus(kind); oldValue++ {
			for inputs := 0; inputs < 4; inputs++ {
				v := seqVolts()
				if inputs&1 != 0 {
					for _, pin := range []int{0, 1, 3, 6} {
						v[pin] = 5
					}
				}
				if inputs&2 != 0 {
					v[2] = 5
				}
				m := SequentialMemory{Value: oldValue, Powered: true, ClockKnown: true}
				v[4] = 5
				n := seqDecide(t, kind, v, m)
				want := oldValue
				switch kind {
				case "d-flipflop":
					want = inputs & 1
				case "t-flipflop":
					if inputs&1 != 0 {
						want = 1 - oldValue
					}
				case "jk-flipflop":
					switch inputs {
					case 1:
						want = 1
					case 2:
						want = 0
					case 3:
						want = 1 - oldValue
					}
				default:
					if inputs&1 != 0 {
						want = (oldValue + 1) % sequentialModulus(kind)
					}
				}
				if n.Value != want {
					t.Fatal("rising edge truth", kind, oldValue, inputs, n, want)
				}
				if hold := seqDecide(t, kind, v, n); hold != n {
					t.Fatal("counted held clock", kind, hold, n)
				}
				v[4] = 0
				fall := seqDecide(t, kind, v, n)
				if fall.Value != n.Value {
					t.Fatal("falling edge counted")
				}
				v[5] = 5
				v[4] = 5
				if reset := seqDecide(t, kind, v, fall); reset.Value != 0 || !reset.ResetHigh {
					t.Fatal("reset priority", reset)
				}
			}
		}
	}
	for _, kind := range sequentialKinds {
		v := seqVolts()
		v[4] = 5
		v[0] = 5
		v[1] = 5
		v[2] = 5
		v[3] = 5
		v[6] = 5
		initial := seqDecide(t, kind, v, SequentialMemory{})
		if initial.Value != 0 || !initial.ClockHigh {
			t.Fatal("phantom startup edge", initial)
		}
		v[4] = 2.5
		if hold := seqDecide(t, kind, v, initial); hold != initial {
			t.Fatal("clock band did not hold")
		}
		v[4] = 0
		m := seqDecide(t, kind, v, initial)
		v[4] = 5
		m = seqDecide(t, kind, v, m)
		if m.Value != 1 {
			t.Fatal("first real edge lost", kind, m)
		}
		v[13] = 1.9
		off := seqDecide(t, kind, v, m)
		if off != (SequentialMemory{}) {
			t.Fatal("power loss retained memory")
		}
		v[13] = 5
		if on := seqDecide(t, kind, v, off); on.Value != 0 {
			t.Fatal("power restore counted high clock")
		}
		v[13] = 13
		if _, err := sequentialDecision(kind, v, m, true); err == nil {
			t.Fatal("overvoltage accepted")
		}
	}
}
func TestSequentialSampledIntermediateInputRefusedOnlyAtEdge(t *testing.T) {
	for _, kind := range sequentialKinds {
		v := seqVolts()
		for _, pin := range []int{0, 1, 2, 3, 6} {
			v[pin] = 2.5
		}
		m := SequentialMemory{Powered: true, ClockKnown: true}
		if _, err := sequentialDecision(kind, v, m, true); err != nil {
			t.Fatal("unsampled data refused", err)
		}
		v[4] = 5
		if _, err := sequentialDecision(kind, v, m, false); err != nil {
			t.Fatal("guard detection sampled data", err)
		}
		if _, err := sequentialDecision(kind, v, m, true); err == nil {
			t.Fatal("intermediate sampled data invented", kind)
		}
		v[5] = 5
		if reset := seqDecide(t, kind, v, m); reset.Value != 0 {
			t.Fatal("reset failed to override invalid data")
		}
	}
}
func TestSequentialPinCurrentsJacobianAndPassivePower(t *testing.T) {
	for _, kind := range sequentialKinds {
		for _, powered := range []bool{false, true} {
			v := [15]float64{}
			for k := range v {
				v[k] = float64(k%7) - 2
			}
			for value := 0; value < sequentialModulus(kind); value++ {
				m := SequentialMemory{Value: value, Powered: powered}
				e := sequentialCurrents(kind, v, m)
				sum, power := 0., 0.
				for k, i := range e.Current {
					sum += i
					power += (v[k] - v[14]) * i
				}
				if math.Abs(sum) > 1e-12 || power < -1e-12 {
					t.Fatal("current/power balance", kind, power, sum)
				}
				for col := range v {
					p, q := v, v
					p[col] += 1e-6
					q[col] -= 1e-6
					ep, eq := sequentialCurrents(kind, p, m), sequentialCurrents(kind, q, m)
					for row := range v {
						want := (ep.Current[row] - eq.Current[row]) / 2e-6
						if math.Abs(want-e.Jacobian[row][col]) > 1e-8 {
							t.Fatal("sequential Jacobian", kind, row, col)
						}
					}
				}
			}
		}
	}
}
func TestSequentialExampleSequencesAndElectricalBalance(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	for _, kind := range sequentialKinds {
		t.Run(kind, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: kind}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			initial := settled(t, s)
			if initial.SequentialModel == nil || initial.Analysis != SequentialAnalysis || initial.Mixed == nil || initial.Mixed.Sequential[0].Memory.Value != 0 || initial.Document.Dirty {
				t.Fatal("sequential startup", initial.Diagnostics)
			}
			if warnings := waveformSamplingWarnings(Graph{initial.Components, initial.Wires}, initial.TransientSettings); len(warnings) != 0 {
				t.Fatal("example sampling", warnings)
			}
			duration, step := 3.3, .005
			probes := []string{"v:U1.clk", "v:U1"}
			if IsCounterKind(kind) {
				probes = []string{"v:U1.clk", "v:U1.q0", "v:U1.q1", "v:U1.q2", "v:U1.q3"}
			}
			_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: probes}, "capture")
			record := s.operations[view.Operation.ID].record
			if record.metadata.SequentialModel == nil || record.metadata.Engine != SequentialEngineID {
				t.Fatal("sequence provenance")
			}
			edges := 0
			for k := 1; k < len(record.rows); k++ {
				a, b := record.rows[k-1], record.rows[k]
				if a.Time != b.Time {
					continue
				}
				if IsCounterKind(kind) {
					decode := func(v []float64) int {
						n := 0
						for i, x := range v[1:] {
							if x > 3.5 {
								n |= 1 << i
							}
						}
						return n
					}
					old, next := decode(a.Values), decode(b.Values)
					if old != next {
						edges++
						if next != edges%sequentialModulus(kind) {
							t.Fatal("counter sequence", kind, edges, b)
						}
						wantTime := .014 + float64(edges-1)*.2
						if math.Abs(b.Time-wantTime) > 2e-8 {
							t.Fatal("clock-edge localization", b.Time, wantTime)
						}
					}
				}
			}
			if IsCounterKind(kind) && edges != 17 {
				t.Fatal("missed counter edges/wrap", kind, edges)
			}
			call(t, s, "simulation.step", StepParams{Duration: .3}, "step")
			live := s.Snapshot()
			if live.Mixed.KCLResidual > 1e-8 || math.Abs(live.Mixed.PowerResidual) > 1e-8 {
				t.Fatal("sequence electrical balance", live.Mixed)
			}
			if kind == "t-flipflop" || kind == "jk-flipflop" {
				if live.Mixed.Sequential[0].Memory.Value != 0 {
					t.Fatal("divider did not toggle twice")
				}
			}
		})
	}
}
func TestSequentialProjectsCapturesModelsAndStrictArrays(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "binary-counter"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	duration, step := .4, .002
	_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:U1", "i:U1.q0"}}, "capture")
	record := s.operations[view.Operation.ID].record
	raw, _ := json.Marshal(VerificationDocument{Metadata: record.metadata, Rows: record.rows})
	if _, err := decodeRecording(raw); err != nil {
		t.Fatal("valid sequence record", err)
	}
	path := filepath.Join(t.TempDir(), "counter.elab")
	call(t, s, "simulation.step", StepParams{Duration: .25}, "step")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	restored := settled(t, fresh)
	if restored.SequentialModel == nil || restored.Document.RecordingCount != 1 || restored.Document.Dirty || restored.Mixed.Sequential[0].Memory.Value != 0 {
		t.Fatal("restart/project startup should clear while retaining recording")
	}
	var capture map[string]any
	json.Unmarshal(raw, &capture)
	meta := capture["metadata"].(map[string]any)
	u := meta["mixed"].(map[string]any)["sequential"].([]any)[0].(map[string]any)
	for _, field := range []string{"memory", "kind", "model", "pin_currents_a", "bit_voltages_v"} {
		saved := u[field]
		delete(u, field)
		bad, _ := json.Marshal(capture)
		if _, err := decodeRecording(bad); err == nil {
			t.Fatal("missing sequential field accepted", field)
		}
		u[field] = saved
	}
	pins := u["pin_currents_a"]
	u["pin_currents_a"] = pins.([]any)[:14]
	bad, _ := json.Marshal(capture)
	if _, err := decodeRecording(bad); err == nil {
		t.Fatal("short pin array accepted")
	}
	u["pin_currents_a"] = pins
	memory := u["memory"].(map[string]any)
	memory["value"] = 16
	bad, _ = json.Marshal(capture)
	if _, err := decodeRecording(bad); err == nil {
		t.Fatal("invalid count accepted")
	}
	before := s.Snapshot()
	before.SequentialModel.OutputResistance = 1
	before.Mixed.Sequential[0].Memory.Value = 9
	if s.Snapshot().SequentialModel.OutputResistance != 50 || s.Snapshot().Mixed.Sequential[0].Memory.Value == 9 {
		t.Fatal("sequence snapshot alias")
	}
	d := s.documentLocked()
	if inspectDocument(d).SequentialModel == nil {
		t.Fatal("recovery lost sequential model")
	}
	d.SequentialModel = nil
	raw, _ = json.Marshal(d)
	if _, err := decodeDocument(raw); err == nil {
		t.Fatal("missing sequential model accepted")
	}
	cloned := cloneVerificationMetadata(record.metadata)
	cloned.SequentialModel.OutputResistance = 2
	if reflect.DeepEqual(cloned.SequentialModel, record.metadata.SequentialModel) {
		t.Fatal("sequence metadata clone aliases")
	}
}
func TestSequentialFailureDoesNotLeakMemory(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "binary-counter"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	state := cloneMixedState(s.mixedState)
	trace := cloneTransient(s.transient)
	old := append([]SequentialMemory{}, state.net.sequenceMemory...)
	before := cloneMixed(state.result)
	budget := newIntegrationBudget()
	budget.attempts = 512
	if _, err := state.advanceSequential(trace, .05, budget); err == nil {
		t.Fatal("budget failure expected")
	}
	if !reflect.DeepEqual(old, state.net.sequenceMemory) || !reflect.DeepEqual(before, state.result) {
		t.Fatal("failed trial leaked sequential state")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := state.resolveSequential(ctx, trace); err != context.Canceled {
		t.Fatal(err)
	}
}

func seqKernelFixture(t *testing.T, feedback bool, reverse bool) *mixedNetwork {
	f := newOpAmpNetworkFixture()
	f.add(t, "VP", "dc-source", "5", "vcc", "0")
	f.add(t, "CLK", "finite-pulse-source", "5", "clock", "0")
	f.graph.Components[1].Pulse = &PulseWaveform{HighVoltage: 5, LowVoltage: 0, HighSeconds: .1, LowSeconds: .1, RiseSeconds: .02, FallSeconds: .02}
	data := "vcc"
	if feedback {
		data = "nq1"
	}
	f.add(t, "U1", "d-flipflop", "", data, "clock", "0", "output", "nq1", "vcc", "0")
	if !feedback {
		f.add(t, "U2", "d-flipflop", "", "output", "clock", "0", "q2", "nq2", "vcc", "0")
		if reverse {
			f.graph.Components[2], f.graph.Components[3] = f.graph.Components[3], f.graph.Components[2]
		}
	}
	n := f.network()
	n.c.Analysis = SequentialAnalysis
	return n
}
func TestSequentialCommonClockPipelineAndFeedbackToggle(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	for _, reverse := range []bool{false, true} {
		n := seqKernelFixture(t, false, reverse)
		trace := &Transient{Engine: SequentialEngineID, Mode: "auto", Settings: DefaultTransientSettings()}
		state, err := n.solve(nil, nil, 0, trace, 0)
		if err != nil {
			t.Fatal(err)
		}
		for edge := 1; edge <= 3; edge++ {
			state, err = state.advanceSequential(trace, .02+float64(edge-1)*.2, newIntegrationBudget())
			if err != nil {
				t.Fatal(err)
			}
			for _, u := range state.result.Sequential {
				want := 1
				if u.ID == "U2" && edge == 1 {
					want = 0
				}
				if u.Memory.Value != want {
					t.Fatal("common-clock pipeline sampled updated output", reverse, edge, u)
				}
			}
		}
	}
	n := seqKernelFixture(t, true, false)
	trace := &Transient{Engine: SequentialEngineID, Mode: "auto", Settings: DefaultTransientSettings()}
	state, err := n.solve(nil, nil, 0, trace, 0)
	if err != nil {
		t.Fatal(err)
	}
	for edge := 1; edge <= 4; edge++ {
		state, err = state.advanceSequential(trace, .02+float64(edge-1)*.2, newIntegrationBudget())
		if err != nil {
			t.Fatal(err)
		}
		if state.result.Sequential[0].Memory.Value != edge%2 {
			t.Fatal("D feedback toggle", edge, state.result.Sequential)
		}
	}
}
func TestSequentialLiveEnableResetAndNoReleaseEdge(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "binary-counter"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	settled(t, s)
	call(t, s, "simulation.step", StepParams{Duration: .05}, "first")
	if s.Snapshot().Mixed.Sequential[0].Memory.Value != 1 {
		t.Fatal("first count")
	}
	call(t, s, "switch.set", SwitchParams{ID: "SENABLE", Closed: false}, "disable")
	call(t, s, "simulation.step", StepParams{Duration: .2}, "hold")
	if s.Snapshot().Mixed.Sequential[0].Memory.Value != 1 {
		t.Fatal("enable did not hold")
	}
	call(t, s, "switch.set", SwitchParams{ID: "SRESET", Closed: true}, "reset")
	call(t, s, "simulation.step", StepParams{Duration: .002}, "reset-apply")
	if s.Snapshot().Mixed.Sequential[0].Memory.Value != 0 {
		t.Fatal("asynchronous reset")
	}
	call(t, s, "switch.set", SwitchParams{ID: "SRESET", Closed: false}, "release")
	call(t, s, "switch.set", SwitchParams{ID: "SENABLE", Closed: true}, "enable")
	call(t, s, "simulation.step", StepParams{Duration: .002}, "release-apply")
	if s.Snapshot().Mixed.Sequential[0].Memory.Value != 0 {
		t.Fatal("release invented rising edge")
	}
	call(t, s, "simulation.step", StepParams{Duration: .2}, "next-edge")
	if s.Snapshot().Mixed.Sequential[0].Memory.Value != 1 {
		t.Fatal("next enabled edge lost")
	}
}
func TestSequentialMixedStorageAndTimerDrivenClock(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	// Capacitor voltage and coil current must survive a memory/output transition.
	f := newOpAmpNetworkFixture()
	f.add(t, "VP", "dc-source", "5", "vcc", "0")
	f.add(t, "CLK", "finite-pulse-source", "5", "clock", "0")
	f.graph.Components[1].Pulse = &PulseWaveform{HighVoltage: 5, LowVoltage: 0, HighSeconds: .1, LowSeconds: .1, RiseSeconds: .02, FallSeconds: .02}
	f.add(t, "U1", "t-flipflop", "", "vcc", "clock", "0", "output", "nq", "vcc", "0")
	f.add(t, "C1", "capacitor", "100u", "output", "0")
	f.add(t, "RL", "resistor", "1k", "output", "coil")
	f.add(t, "L1", "inductor", "100m", "coil", "0")
	n := f.network()
	n.c.Analysis = SequentialAnalysis
	trace := &Transient{Engine: SequentialEngineID, Mode: "auto", Settings: DefaultTransientSettings()}
	state, err := n.solve(nil, []float64{0}, 0, trace, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Keep each requested step bounded under the unchanged production budget.
	for k := 1; k <= 20; k++ {
		state, err = state.advanceSequential(trace, float64(k)*.001, newIntegrationBudget())
		if err != nil {
			t.Fatal(err)
		}
	}
	pairs := 0
	for k := 1; k < len(trace.Samples); k++ {
		a, b := trace.Samples[k-1], trace.Samples[k]
		if a.Time == b.Time && a.State != b.State {
			pairs++
			for _, id := range []string{"v:C1.1", "i:L1"} {
				if math.Abs(a.frame.values[id]-b.frame.values[id]) > 1e-10 {
					t.Fatal("storage jumped at a sequential output edge", id, a.Time)
				}
			}
		}
	}
	if pairs == 0 {
		t.Fatal("sequential RLC transition did not retain both edge sides")
	}
	if state.result.Sequential[0].Memory.Value != 1 || state.result.Capacitors[0].Voltage <= 0 || state.result.Inductors[0].Current <= 0 || math.Abs(state.result.PowerResidual) > 1e-7 {
		t.Fatal("sequential RLC integration", state.result)
	}
	g := Timer555FlasherExampleGraph()
	p, _ := MakeComponent("U2", "t-flipflop", Point{500, -200}, "")
	g.Components = append(g.Components, p)
	for _, pair := range [][2]string{{"U1.out", "U2.clk"}, {"U1.vcc", "U2.vcc"}, {"U1.vcc", "U2.t"}, {"U1.gnd", "U2.reset"}, {"U1.gnd", "U2.gnd"}} {
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", len(g.Wires)+1), From: pair[0], To: pair[1], Connected: true})
	}
	c, _, d := g.Compile()
	if d != nil {
		t.Fatal(d)
	}
	kernel, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	if kernel.mixed.result.Sequential[0].Memory.Value != 0 {
		t.Fatal("derived timer clock invented a startup edge")
	}
	seen := map[int]bool{}
	for k := 1; k <= 1600; k++ {
		if err = kernel.advance(t.Context(), float64(k)*.001); err != nil {
			t.Fatal(err)
		}
		seen[kernel.mixed.result.Sequential[0].Memory.Value] = true
	}
	if len(seen) != 2 {
		t.Fatal("555 clock failed to drive toggle", seen)
	}
}

func TestSequentialDiscoveryRequiredPinsAndSweepRefusal(t *testing.T) {
	if parts := filteredParts("counter"); len(parts) != 2 {
		t.Fatal("counter search is not specific", parts)
	}
	for _, kind := range sequentialKinds {
		g := sequentialExample(kind)
		if _, _, d := g.Compile(); d != nil {
			t.Fatal(d)
		}
		for _, pin := range sequentialPins(kind) {
			if sequentialOutput(pin) {
				continue
			}
			broken := g.Clone()
			broken.Wires = nil
			for _, w := range g.Wires {
				if w.From != "U1."+pin && w.To != "U1."+pin {
					broken.Wires = append(broken.Wires, w)
				}
			}
			if _, _, d := broken.Compile(); d == nil {
				t.Fatal("required sequential pin open", kind, pin)
			}
		}
		if _, err := newSweepPlan(g, SweepSettings{ComponentID: "R1", Parameter: "resistance", Start: "1k", Stop: "2k", Points: 3}); err == nil {
			t.Fatal("DC sweep invented sequential history")
		}
	}
	registry := SchemaRegistry()
	var schema map[string]any
	json.Unmarshal(registry["urn:elab:sequential-result:1"], &schema)
	pinSchema := schema["properties"].(map[string]any)["pin_currents_a"].(map[string]any)
	if pinSchema["minItems"] != float64(15) || pinSchema["maxItems"] != float64(15) {
		t.Fatal("sequential pin-current schema")
	}
}
