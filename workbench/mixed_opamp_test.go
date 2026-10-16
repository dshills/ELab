package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// This fixture bypasses the public compiler deliberately: opamp is not a
// registered palette/file kind until its persistence, probes and UI are ready.
// The actual MNA/Newton, result and integration implementations run unchanged.
type opAmpNetworkFixture struct {
	graph Graph
	pins  map[string]string
}

func newOpAmpNetworkFixture() *opAmpNetworkFixture {
	return &opAmpNetworkFixture{pins: map[string]string{}}
}

func (f *opAmpNetworkFixture) add(t *testing.T, id, kind, value string, nets ...string) {
	t.Helper()
	var part Component
	if kind == "opamp" {
		part = Component{ID: id, Kind: kind, Terminals: []string{"noninverting", "inverting", "out", "vplus", "vminus"}}
	} else {
		var err error
		// Public sources store nonnegative magnitudes. Reverse their terminals
		// to create a negative rail without expanding accepted source values.
		if kind == "dc-source" && strings.HasPrefix(value, "-") {
			value = strings.TrimPrefix(value, "-")
			nets[0], nets[1] = nets[1], nets[0]
		}
		part, err = MakeComponent(id, kind, Point{}, value)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(nets) != len(part.Terminals) {
		t.Fatal("fixture has incomplete terminal assignment", id, part.Terminals, nets)
	}
	for k, terminal := range part.Terminals {
		f.pins[id+"."+terminal] = nets[k]
	}
	f.graph.Components = append(f.graph.Components, part)
}

func (f *opAmpNetworkFixture) network() *mixedNetwork {
	unique := map[string]bool{"0": true}
	for _, net := range f.pins {
		unique[net] = true
	}
	ids := make([]string, 0, len(unique))
	for net := range unique {
		ids = append(ids, net)
	}
	sort.Strings(ids)
	compiled := Compiled{MixedGraph: f.graph, Ground: "0", TerminalNet: f.pins, Analysis: NetworkTransientAnalysis}
	for _, id := range ids {
		compiled.Nets = append(compiled.Nets, Net{ID: id})
	}
	for _, part := range f.graph.Components {
		if IsVoltageSource(part.Kind) && compiled.Source.ID == "" {
			compiled.Source = part
			compiled.Supply = f.pins[part.ID+".positive"]
			compiled.Circuit.Voltage = part.Value
		}
	}
	compiled.Output = "output"
	return newMixedNetwork(compiled)
}

func opAmpAmplifierFixture(t *testing.T, kind string, input, high, low, load float64) *opAmpNetworkFixture {
	t.Helper()
	f := newOpAmpNetworkFixture()
	f.add(t, "V1", "dc-source", fmt.Sprint(input), "input", "0")
	f.add(t, "VP", "dc-source", fmt.Sprint(high), "positive", "0")
	f.add(t, "VN", "dc-source", fmt.Sprint(low), "negative", "0")
	f.add(t, "RL", "resistor", fmt.Sprint(load), "output", "0")
	plus, minus := "input", "output"
	switch kind {
	case "noninverting":
		minus = "feedback"
		f.add(t, "RF", "resistor", "90k", "output", "feedback")
		f.add(t, "RG", "resistor", "10k", "feedback", "0")
	case "inverting":
		plus, minus = "0", "feedback"
		f.add(t, "RF", "resistor", "20k", "output", "feedback")
		f.add(t, "RI", "resistor", "10k", "feedback", "input")
	}
	f.add(t, "U1", "opamp", "", plus, minus, "output", "positive", "negative")
	return f
}

func checkOpAmpNetworkBalance(t *testing.T, s *mixedState) {
	t.Helper()
	r := s.result
	if !finite(r.KCLResidual) || !finite(r.PowerResidual) || r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
		t.Fatal("network violates KCL/power conservation", r.KCLResidual, r.PowerResidual)
	}
	for _, device := range r.OpAmps {
		if device.Power < -1e-12 || math.Abs(device.NoninvertingCurrent+device.InvertingCurrent-device.OutputCurrent+device.PositiveSupplyCurrent+device.NegativeSupplyCurrent) > 1e-14 {
			t.Fatal("result loses signed terminal current/power", device)
		}
	}
}

func TestOpAmpActualNetworkAmplifiersAgainstDecimalReferences(t *testing.T) {
	for _, reference := range opAmpReferences(t).Amplifiers {
		for _, dc := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%g/%g/dc=%v", reference.Kind, reference.Vin, reference.Load, dc), func(t *testing.T) {
				n := opAmpAmplifierFixture(t, reference.Kind, reference.Vin, reference.High, reference.Low, reference.Load).network()
				n.dc = dc
				trace := &Transient{Settings: DefaultTransientSettings()}
				s, err := n.solve(nil, nil, 0, trace, 0)
				if err != nil {
					t.Fatal("cold network initialization", err)
				}
				for _, guess := range [][]float64{nil, append([]float64(nil), s.x...)} {
					if guess != nil {
						guess[n.index["output"]] += .01
					}
					s, err = n.solve(guess, nil, 0, trace, 0)
					if err != nil {
						t.Fatal("cold/warm network solve", err)
					}
					checkOpAmpNetworkBalance(t, s)
					if len(s.result.OpAmps) != 1 || math.Abs(s.result.Nodes["output"]-reference.Output) > 1e-7 {
						t.Fatal("actual network differs from independent scalar reference", s.result.Nodes["output"], reference.Output)
					}
				}
				t.Logf("output %.12g V; KCL %.3g A; power residual %.3g W", s.result.Nodes["output"], s.result.KCLResidual, s.result.PowerResidual)
			})
		}
	}
}

func TestOpAmpNetworkMergedStampJacobian(t *testing.T) {
	f := opAmpAmplifierFixture(t, "follower", 1, 5, -5, 1000)
	n := f.network()
	x := make([]float64, n.matrixSize(0))
	for id, value := range map[string]float64{"input": 1.000003, "output": 1, "positive": 5, "negative": -5} {
		x[n.index[id]] = value
	}
	trace := &Transient{Settings: DefaultTransientSettings()}
	a, _ := n.equations(x, nil, 0, trace, 0)
	for column := range x {
		h := 1e-7
		if column == n.index["input"] || column == n.index["output"] {
			h = 1e-10
		}
		plus, minus := append([]float64(nil), x...), append([]float64(nil), x...)
		plus[column] += h
		minus[column] -= h
		_, fp := n.equations(plus, nil, 0, trace, 0)
		_, fm := n.equations(minus, nil, 0, trace, 0)
		for row := range x {
			measured := (fp[row] - fm[row]) / (2 * h)
			if math.Abs(a[row][column]-measured) > 1e-8+3e-6*math.Abs(measured) {
				t.Fatal("merged follower stamp differs from finite perturbation", row, column, a[row][column], measured)
			}
		}
	}
}

func TestOpAmpNetworkOffAndReversedSupplies(t *testing.T) {
	for _, high := range []float64{0, -5} {
		f := newOpAmpNetworkFixture()
		f.add(t, "VP", "dc-source", fmt.Sprint(high), "positive", "0")
		f.add(t, "VO", "dc-source", "5", "output", "0")
		f.add(t, "U1", "opamp", "", "0", "0", "output", "positive", "0")
		n := f.network()
		s, err := n.solve(nil, nil, 0, &Transient{}, 0)
		if err != nil {
			t.Fatal(err)
		}
		checkOpAmpNetworkBalance(t, s)
		u := s.result.OpAmps[0]
		want := "unpowered"
		if high < 0 {
			want = "reversed-supply"
		}
		if u.State != want || math.Abs(u.OutputCurrent+5e-9) > 1e-15 || u.PositiveSupplyCurrent != 0 || math.Abs(u.NegativeSupplyCurrent+5e-9) > 1e-15 {
			t.Fatal("off-state result does not retain only declared passive paths", u)
		}
	}
}

func TestOpAmpNetworkRejectsFloatingCommonModesAndCancellation(t *testing.T) {
	f := newOpAmpNetworkFixture()
	f.add(t, "V1", "dc-source", "5", "positive", "0")
	f.add(t, "U1", "opamp", "", "input", "input", "output", "positive", "0")
	n := f.network()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := n.solveContext(ctx, nil, nil, 0, &Transient{}, 0); err != context.Canceled {
		t.Fatal("cancelled solve did not stop", err)
	}
	if _, err := n.solve(nil, nil, 0, &Transient{}, 0); err == nil || !strings.Contains(err.Error(), "singular") {
		t.Fatal("unreferenced circuit acquired a hidden ground", err)
	}
}

func TestOpAmpResultCloneAndLegacyShape(t *testing.T) {
	n := opAmpAmplifierFixture(t, "follower", 1, 5, -5, 1000).network()
	s, err := n.solve(nil, nil, 0, &Transient{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	clone := cloneMixed(s.result)
	if !reflect.DeepEqual(s.result.OpAmps, clone.OpAmps) {
		t.Fatal("cloned result differs")
	}
	clone.OpAmps[0].OutputCurrent = 99
	clone.OpAmps[0].Model.ID = "mutated"
	if s.result.OpAmps[0].OutputCurrent == 99 || s.result.OpAmps[0].Model.ID != OpAmpModelID {
		t.Fatal("snapshot aliases solver-owned op-amp results")
	}
	legacy, err := json.Marshal(MixedResult{})
	if err != nil || strings.Contains(string(legacy), "opamps") {
		t.Fatal("unused experimental model changes legacy JSON shape", string(legacy), err)
	}
}

func TestOpAmpActualNetworkBufferedRC(t *testing.T) {
	f := opAmpAmplifierFixture(t, "follower", 1, 5, -5, 1000)
	f.pins["U1.noninverting"] = "rc"
	f.add(t, "RI", "resistor", "10k", "input", "rc")
	f.add(t, "C1", "capacitor", "1uF", "rc", "0")
	n := f.network()
	trace := &Transient{Settings: DefaultTransientSettings()}
	s, err := n.solve(nil, []float64{0}, 0, trace, 0)
	if err != nil {
		t.Fatal("RC startup", err)
	}
	for sample := 1; sample <= 50; sample++ {
		at := float64(sample) * .001
		s, err = s.advance(trace, at)
		if err != nil {
			t.Fatal("buffered RC advance", sample, err)
		}
		checkOpAmpNetworkBalance(t, s)
		// A 10 kOhm / 1 uF input filter tends to 1-exp(-t/.01).
		// Finite-gain loading changes it by <1 uV; allow the existing BE
		// integrator's accumulated error rather than claiming ideal op-amp dynamics.
		ideal := -math.Expm1(-at / .01)
		if math.Abs(s.result.Nodes["rc"]-ideal) > .0007 || math.Abs(s.result.Nodes["output"]-s.result.Nodes["rc"]) > .0001 {
			t.Fatal("buffered RC differs from its analytic envelope", at, s.result.Nodes, ideal)
		}
	}
}

func TestOpAmpNetworkUnloadedEqualRails(t *testing.T) {
	f := newOpAmpNetworkFixture()
	f.add(t, "V1", "dc-source", "5", "rail", "0")
	f.add(t, "U1", "opamp", "", "0", "0", "output", "rail", "rail")
	s, err := f.network().solve(nil, nil, 0, &Transient{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	checkOpAmpNetworkBalance(t, s)
	if math.Abs(s.result.Nodes["output"]-5) > 1e-9 || s.result.OpAmps[0].State != "unpowered" {
		t.Fatal("small bleed current residual hid a large voltage error", s.result)
	}
}

func TestOpAmpTwoStageNetwork(t *testing.T) {
	f := opAmpAmplifierFixture(t, "follower", 1, 5, -5, 1000)
	f.add(t, "U2", "opamp", "", "output", "second", "second", "positive", "negative")
	f.add(t, "RL2", "resistor", "1k", "second", "0")
	n := f.network()
	s, err := n.solve(nil, nil, 0, &Transient{}, 0)
	if err != nil {
		t.Fatal("two-stage startup", err)
	}
	checkOpAmpNetworkBalance(t, s)
	if len(s.result.OpAmps) != 2 || math.Abs(s.result.Nodes["second"]-1) > .0001 {
		t.Fatal("cascaded buffers do not track their input", s.result)
	}
}

func TestOpAmpNewtonInputLimitAcrossAllCollocationStages(t *testing.T) {
	n := opAmpAmplifierFixture(t, "follower", 1, 5, -5, 1000).network()
	for _, stages := range []int{1, 3} {
		for _, sample := range []struct{ difference, change, initial, want float64 }{
			{1, -10, 1, (1 + 5e-6) / 10},
			{-1, 10, 1, (1 + 5e-6) / 10},
			{0, .001, 1, .005},
			{0, -.001, 1, .005},
			{1, -.5, 1, 1},
			{-1, .5, .1, .1},
			{1, .5, 1, 1e-5},
			{-1, -.5, 1, 1e-5},
		} {
			size := n.matrixSize(0)
			x, delta := make([]float64, stages*size), make([]float64, stages*size)
			// Only the final stage is powered. Earlier stages must not hide it.
			offset := (stages - 1) * size
			x[offset+n.index["positive"]] = 5
			x[offset+n.index["negative"]] = -5
			x[offset+n.index["input"]] = sample.difference
			delta[offset+n.index["input"]] = sample.change
			got := n.opAmpNewtonScale(x, delta, stages, sample.initial)
			if math.Abs(got-sample.want) > 1e-14 || got > sample.initial || got <= 0 {
				t.Fatal("incorrect bounded differential-input proposal", stages, sample, got)
			}
		}
	}
}

func TestOpAmpActualNetworkBufferedRLC(t *testing.T) {
	f := opAmpAmplifierFixture(t, "follower", .2, 5, -5, 1000)
	f.add(t, "RS", "resistor", "1k", "output", "coil")
	f.add(t, "L1", "inductor", "100mH", "coil", "filtered")
	f.add(t, "C1", "capacitor", "1uF", "filtered", "0")
	f.add(t, "RFILT", "resistor", "1k", "filtered", "0")
	c := f.network().c
	c.Analysis = InductorAnalysis
	kernels := make([]*simulationKernel, 2)
	for k := range kernels {
		var err error
		kernels[k], err = newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
		if err != nil {
			t.Fatal("buffered RLC startup", err)
		}
	}
	for sample := 1; sample <= 20; sample++ {
		for k, kernel := range kernels {
			for sub := 1; sub <= k+1; sub++ {
				end := (float64(sample-1) + float64(sub)/float64(k+1)) * .0005
				if err := kernel.advance(t.Context(), end); err != nil {
					t.Fatal("buffered RLC integration", sample, k, err)
				}
			}
			checkOpAmpNetworkBalance(t, kernel.mixed)
		}
		a, b := kernels[0].mixed.result, kernels[1].mixed.result
		if math.Abs(a.Nodes["filtered"]-b.Nodes["filtered"]) > 1e-7 || math.Abs(a.Inductors[0].Current-b.Inductors[0].Current) > 1e-9 {
			t.Fatal("RLC output disagrees with half-step run", sample, a, b)
		}
	}
	// The DC passive divider tends to one half of the buffer output.
	r := kernels[0].mixed.result
	if math.Abs(r.Nodes["filtered"]-.1) > .0001 || math.Abs(r.Nodes["output"]-.2) > .0001 {
		t.Fatal("buffered RLC missed its DC limit", r)
	}
}
