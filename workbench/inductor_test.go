package workbench

import (
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/dshills/elab/internal/testutil"
)

func inductorTestGraph(t *testing.T, supply, resistance, inductance float64, capacitance *float64) Graph {
	t.Helper()
	g := EmptyGraph()
	for k, p := range []struct{ id, kind, value string }{
		{"V1", "dc-source", FormatValue(supply, "V")}, {"R1", "resistor", FormatValue(resistance, "Ω")},
		{"L1", "inductor", FormatValue(inductance, "H")}, {"GND1", "ground", ""},
	} {
		part, err := MakeComponent(p.id, p.kind, Point{X: float64(k) * 200}, p.value)
		if err != nil {
			t.Fatal(err)
		}
		g.Components = append(g.Components, part)
	}
	end := "GND1.reference"
	if capacitance != nil {
		part, err := MakeComponent("C1", "capacitor", Point{X: 800}, FormatValue(*capacitance, "F"))
		if err != nil {
			t.Fatal(err)
		}
		g.Components = append(g.Components, part)
		g.Wires = append(g.Wires, Wire{ID: "WC", From: "C1.2", To: end, Connected: true})
		end = "C1.1"
	}
	for k, pins := range [][2]string{{"V1.positive", "R1.1"}, {"R1.2", "L1.1"}, {"L1.2", end}, {"V1.negative", "GND1.reference"}} {
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", k), From: pins[0], To: pins[1], Connected: true})
	}
	return g
}

func initializeInductorTest(t *testing.T, g Graph) (*mixedState, *Transient) {
	t.Helper()
	c, _, diagnostic := g.Compile()
	if diagnostic != nil || c.Analysis != InductorAnalysis {
		t.Fatal("inductor network compilation", c.Analysis, diagnostic)
	}
	tr := &Transient{Settings: DefaultTransientSettings(), Mode: "auto", Engine: InductorEngineID}
	net := newMixedNetwork(c)
	old := make([]float64, len(net.caps))
	for k, p := range net.caps {
		if p.InitialVoltage != nil {
			old[k] = *p.InitialVoltage
		}
	}
	state, err := net.solve(nil, old, 0, tr, 0)
	if err != nil {
		t.Fatal(err)
	}
	return state, tr
}

func TestInductorRLAnalyticStartupDecayAndSameTimeContinuity(t *testing.T) {
	for _, input := range []struct{ voltage, current float64 }{{5, 0}, {0, .05}, {5, -.02}, {0, 1e-12}, {0, -1e-12}} {
		g := inductorTestGraph(t, input.voltage, 100, 1, nil)
		g.Components[2].InitialCurrent = &input.current
		state, tr := initializeInductorTest(t, g)
		if state.result.Inductors[0].Current != input.current {
			t.Fatal("fresh run lost configured current")
		}
		for sample := 1; sample <= 60; sample++ {
			var err error
			at := float64(sample) * .001
			state, err = state.advance(tr, at)
			if err != nil {
				t.Fatal(input, sample, err)
			}
			want := input.voltage/100 + (input.current-input.voltage/100)*math.Exp(-at/.01)
			got := state.result.Inductors[0]
			if math.Abs(got.Current-want) > 1e-5 || math.Abs(got.Voltage-(input.voltage-100*want)) > .001 || math.Abs(got.Energy-.5*want*want) > 1e-6 || state.result.KCLResidual > 1e-9 || math.Abs(state.result.PowerResidual) > 1e-8 {
				t.Fatal("RL exponential/current/energy/conservation", input, at, got, want, state.result.PowerResidual)
			}
		}
		before := cloneMixed(state.result)
		x, caps := append([]float64{}, state.x...), append([]float64{}, state.capV...)
		right, err := state.net.solve(state.x, state.capV, 0, tr, state.result.Time)
		if err != nil || math.Abs(right.result.Inductors[0].Current-before.Inductors[0].Current) > 1e-12 || !reflect.DeepEqual(cloneMixed(state.result), before) || !reflect.DeepEqual(x, state.x) || !reflect.DeepEqual(caps, state.capV) {
			t.Fatal("same-time solve changed inductor memory or mutated accepted state", err)
		}
	}
}

func TestInductorLosslessLCAndLargeStepRefusal(t *testing.T) {
	capacitance, initial := 10e-6, 1.
	g := inductorTestGraph(t, 0, 100, .1, &capacitance)
	// Isolate the oscillator from the zero-source/resistor branch, sharing
	// only ground. The LC loop starts with electric energy and zero current.
	g.Wires[2].To = "GND1.reference"
	g.Wires[3].To = "GND1.reference"
	g.Wires = append(g.Wires, Wire{ID: "WL", From: "L1.1", To: "C1.1", Connected: true})
	g.Components[4].InitialVoltage = &initial
	state, tr := initializeInductorTest(t, g)
	before := cloneMixed(state.result)
	if next, err := state.advance(tr, 1); err == nil || next != nil || !reflect.DeepEqual(before, cloneMixed(state.result)) {
		t.Fatal("oversized oscillatory request must refuse without artificial damping or publication", err)
	}
	for sample := 1; sample <= 60; sample++ {
		var err error
		at := float64(sample) * .001
		state, err = state.advance(tr, at)
		if err != nil {
			t.Fatal(sample, err)
		}
		cap, ind := state.result.Capacitors[0], state.result.Inductors[0]
		wantV, wantI := math.Cos(1000*at), .01*math.Sin(1000*at)
		if math.Abs(cap.Voltage-wantV) > .001 || math.Abs(ind.Current-wantI) > 1e-5 || math.Abs(cap.Energy+ind.Energy-5e-6) > 1e-12 || math.Abs(state.result.PowerResidual) > 1e-8 {
			t.Fatal("LC phase/energy reference", at, cap, ind, wantV, wantI, state.result.PowerResidual)
		}
	}
}

func rlcStepReference(at, resistance float64) (voltage, current float64) {
	alpha, omega := resistance/.2, 1000.
	if alpha == omega {
		return 5 * (1 - math.Exp(-alpha*at)*(1+alpha*at)), 50 * at * math.Exp(-alpha*at)
	}
	if alpha < omega {
		damped := math.Sqrt(omega*omega - alpha*alpha)
		e := math.Exp(-alpha * at)
		return 5 * (1 - e*(math.Cos(damped*at)+alpha/damped*math.Sin(damped*at))), 50 / damped * e * math.Sin(damped*at)
	}
	d := math.Sqrt(alpha*alpha - omega*omega)
	one, two := -alpha+d, -alpha-d
	e1, e2 := math.Exp(one*at), math.Exp(two*at)
	return 5 * (1 + (two*e1-one*e2)/(one-two)), 50 * (e1 - e2) / (one - two)
}

func TestInductorRLCIndependentThreeDampingRegimes(t *testing.T) {
	for _, resistance := range []float64{50, 200, 500} {
		capacitance := 10e-6
		state, tr := initializeInductorTest(t, inductorTestGraph(t, 5, resistance, .1, &capacitance))
		for sample := 1; sample <= 60; sample++ {
			at := float64(sample) * .001
			var err error
			state, err = state.advance(tr, at)
			if err != nil {
				t.Fatal(resistance, sample, err)
			}
			wantV, wantI := rlcStepReference(at, resistance)
			cap, ind := state.result.Capacitors[0], state.result.Inductors[0]
			if math.Abs(cap.Voltage-wantV) > .001 || math.Abs(ind.Current-wantI) > 1e-5 || math.Abs(ind.Voltage-(5-resistance*wantI-wantV)) > .001 || state.result.KCLResidual > 1e-9 || math.Abs(state.result.PowerResidual) > 1e-8 {
				t.Fatal("RLC independent reference", resistance, at, cap, ind, wantV, wantI, state.result.PowerResidual)
			}
		}
	}
}

func TestInductorRLCDefaultHalfQuarterOutputGrid(t *testing.T) {
	s := New()
	call(t, s, "example.load", ExampleLoadParams{ID: "rlc"}, "rlc")
	step := s.Snapshot().TransientSettings.StepSeconds
	s.Close()
	if step != .0001 {
		t.Fatal("review refinement grid when the guided RLC sample step changes", step)
	}
	for _, resistance := range []float64{50, 200, 500} {
		t.Run(fmt.Sprint(resistance), func(t *testing.T) {
			states := make([]*mixedState, 3)
			traces := make([]*Transient, 3)
			capacitance := 10e-6
			for k := range states {
				states[k], traces[k] = initializeInductorTest(t, inductorTestGraph(t, 5, resistance, .1, &capacitance))
				substeps := 1 << k
				traces[k].Settings.StepSeconds = step / float64(substeps)
			}
			maxVoltage, maxCurrent := 0., 0.
			for sample := 1; sample <= 100; sample++ {
				at := float64(sample) * step
				wantV, wantI := rlcStepReference(at, resistance)
				for k := range states {
					substeps := 1 << k
					for sub := 1; sub <= substeps; sub++ {
						end := (float64(sample-1) + float64(sub)/float64(substeps)) * step
						var err error
						states[k], err = states[k].advance(traces[k], end)
						if err != nil {
							t.Fatal("RLC grid refinement refused", k, at, err)
						}
					}
					r := states[k].result
					cap, coil := r.Capacitors[0], r.Inductors[0]
					if math.Abs(cap.Voltage-wantV) > .001 || math.Abs(coil.Current-wantI) > 1e-5 || math.Abs(coil.Voltage-(5-resistance*wantI-wantV)) > .001 || r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
						t.Fatal("RLC grid disagrees with independent reference", k, at, cap, coil, wantV, wantI)
					}
				}
				for k := 0; k < 2; k++ {
					coarse, fine := states[k].result, states[2].result
					maxVoltage = math.Max(maxVoltage, math.Abs(coarse.Capacitors[0].Voltage-fine.Capacitors[0].Voltage))
					maxCurrent = math.Max(maxCurrent, math.Abs(coarse.Inductors[0].Current-fine.Inductors[0].Current))
				}
			}
			if maxVoltage > .001 || maxCurrent > 1e-5 {
				t.Fatal("default/half outputs differ from quarter-grid outputs", maxVoltage, maxCurrent)
			}
			t.Logf("100/50/25 µs grids, 10 ms: maximum difference %.9g V / %.9g A", maxVoltage, maxCurrent)
		})
	}
}

func TestInductorSeriesStartupVoltagesAndCurrentConsistency(t *testing.T) {
	g := inductorTestGraph(t, 5, 100, .4, nil)
	second, err := MakeComponent("L2", "inductor", Point{X: 800}, "600 mH")
	if err != nil {
		t.Fatal(err)
	}
	g.Components = append(g.Components, second)
	g.Wires[2].To = "L2.1"
	g.Wires = append(g.Wires, Wire{ID: "WL2", From: "L2.2", To: "GND1.reference", Connected: true})
	state, tr := initializeInductorTest(t, g)
	if math.Abs(state.result.Inductors[0].Voltage-2) > 1e-8 || math.Abs(state.result.Inductors[1].Voltage-3) > 1e-8 {
		t.Fatal("series coils need consistent startup voltage derivatives", state.result.Inductors)
	}
	for sample := 1; sample <= 20; sample++ {
		at := float64(sample) * .001
		state, err = state.advance(tr, at)
		if err != nil {
			t.Fatal(err)
		}
		want := .05 * -math.Expm1(-at/.01)
		for _, p := range state.result.Inductors {
			if math.Abs(p.Current-want) > 1e-5 || state.result.KCLResidual > 1e-9 {
				t.Fatal("series inductance/current reference", at, p, want)
			}
		}
	}
	for _, initial := range []float64{.01, 1e-12} {
		g.Components[4].InitialCurrent = &initial // L1 still has implicit zero.
		c, _, diagnostic := g.Compile()
		if diagnostic != nil {
			t.Fatal(diagnostic)
		}
		if _, err := newMixedNetwork(c).solve(nil, nil, 0, tr, 0); err == nil {
			t.Fatal("incompatible series initial currents accepted", initial)
		}
	}
}

func TestInductorFloatingConductiveIslands(t *testing.T) {
	for _, floatingSource := range []bool{false, true} {
		g := inductorTestGraph(t, 5, 100, .4, nil)
		second, _ := MakeComponent("L2", "inductor", Point{X: 800}, "600 mH")
		initial := .02
		g.Components[2].InitialCurrent = &initial
		second.InitialCurrent = &initial
		g.Components = append(g.Components, second)
		if floatingSource {
			// Source positive → L1 → R → ground → L2 → source negative.
			g.Wires[0].To = "L1.1"
			g.Wires[1].From, g.Wires[1].To = "L1.2", "R1.1"
			g.Wires[2].From = "R1.2"
			g.Wires[3].From, g.Wires[3].To = "L2.1", "GND1.reference"
			g.Wires = append(g.Wires, Wire{ID: "WL2", From: "L2.2", To: "V1.negative", Connected: true})
		} else {
			// Source → L1 → floating R island → L2 → ground.
			g.Wires[0].To = "L1.1"
			g.Wires[1].From, g.Wires[1].To = "L1.2", "R1.1"
			g.Wires[2].From = "L2.2"
			g.Wires = append(g.Wires, Wire{ID: "WL2", From: "R1.2", To: "L2.1", Connected: true})
		}
		state, tr := initializeInductorTest(t, g)
		if math.Abs(state.result.Inductors[0].Voltage-1.2) > 1e-9 || math.Abs(state.result.Inductors[1].Voltage-1.8) > 1e-9 || state.result.KCLResidual > 1e-10 {
			t.Fatal("floating island startup derivative/physical KCL", floatingSource, state.result)
		}
		for sample := 1; sample <= 20; sample++ {
			at := float64(sample) * .001
			var err error
			state, err = state.advance(tr, at)
			if err != nil {
				t.Fatal(floatingSource, err)
			}
			want := .05 - .03*math.Exp(-at/.01)
			for _, p := range state.result.Inductors {
				if math.Abs(p.Current-want) > 1e-5 || state.result.KCLResidual > 1e-9 || math.Abs(state.result.PowerResidual) > 1e-8 {
					t.Fatal("floating island reference", floatingSource, at, p, want)
				}
			}
		}
	}
}

func TestInductorParallelAndPersistentCirculatingCurrent(t *testing.T) {
	for _, circulating := range []bool{false, true} {
		g := inductorTestGraph(t, 5, 100, .4, nil)
		second, _ := MakeComponent("L2", "inductor", Point{X: 800}, "600 mH")
		if circulating {
			g.Components[0].Value = 0
			one, two := .03, -.03
			g.Components[2].InitialCurrent, second.InitialCurrent = &one, &two
		}
		g.Components = append(g.Components, second)
		g.Wires = append(g.Wires, Wire{ID: "WL21", From: "L2.1", To: "L1.1", Connected: true}, Wire{ID: "WL22", From: "L2.2", To: "GND1.reference", Connected: true})
		state, tr := initializeInductorTest(t, g)
		for sample := 1; sample <= 20; sample++ {
			at := float64(sample) * .001
			var err error
			state, err = state.advance(tr, at)
			if err != nil {
				t.Fatal(err)
			}
			one, two := .03*-math.Expm1(-at/.0024), .02*-math.Expm1(-at/.0024)
			if circulating {
				one, two = .03, -.03
			}
			if math.Abs(state.result.Inductors[0].Current-one) > 1e-5 || math.Abs(state.result.Inductors[1].Current-two) > 1e-5 || state.result.KCLResidual > 1e-9 || math.Abs(state.result.PowerResidual) > 1e-8 {
				t.Fatal("parallel coil current reference", circulating, at, state.result.Inductors, one, two)
			}
		}
	}
}

func TestInductorLEDZeroCurrentStartup(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	g := inductorTestGraph(t, 5, 100, 1, nil)
	led, _ := MakeComponent("D1", "led", Point{X: 800}, "")
	g.Components = append(g.Components, led)
	g.Wires[2].To = "D1.anode"
	g.Wires = append(g.Wires, Wire{ID: "WD", From: "D1.cathode", To: "GND1.reference", Connected: true})
	state, trace := initializeInductorTest(t, g)
	if state.result.LEDs[0].Voltage != 0 || state.result.Inductors[0].Current != 0 {
		t.Fatal("zero-current forward startup", state.result)
	}
	for _, at := range []float64{1e-6, 1e-5, .001, .005, .02} {
		var err error
		state, err = state.advance(trace, at)
		if err != nil {
			t.Fatal("series LED/coil did not start", at, err)
		}
		want := inductorLEDReference(at)
		wantV := 2*1.380649e-23*300.15/1.602176634e-19*math.Log1p(want/1e-20) + want
		if math.Abs(state.result.Inductors[0].Current-want) > 1e-5 || math.Abs(state.result.LEDs[0].Voltage-wantV) > .001 || state.result.KCLResidual > 1e-9 || math.Abs(state.result.PowerResidual) > 1e-8 {
			t.Fatal("independent LED/coil integral reference", at, state.result, want, wantV)
		}
	}
}

// Separate time-of-flight quadrature: t=∫ L/(5-RI-Vled(I)) dI. The
// substitution I=Is*expm1(s) resolves the forward knee without MNA, Newton
// histories or the production timestep method. Constants pin the v1 model.
func inductorLEDReference(at float64) float64 {
	const nv = 2 * 1.380649e-23 * 300.15 / 1.602176634e-19
	lo, hi := 0., .05
	for k := 0; k < 80; k++ {
		i := (lo + hi) / 2
		if 101*i+nv*math.Log1p(i/1e-20) > 5 {
			hi = i
		} else {
			lo = i
		}
	}
	lo, hi = 0, .999999*lo
	for k := 0; k < 55; k++ {
		i := (lo + hi) / 2
		end := math.Log1p(i / 1e-20)
		const panels = 4096
		h := end / panels
		sum := 0.
		for j := 0; j <= panels; j++ {
			s := h * float64(j)
			current := 1e-20 * math.Expm1(s)
			value := 1e-20 * math.Exp(s) / (5 - 101*current - nv*s)
			weight := 2.
			if j == 0 || j == panels {
				weight = 1
			} else if j%2 != 0 {
				weight = 4
			}
			sum += weight * value
		}
		if sum*h/3 > at {
			hi = i
		} else {
			lo = i
		}
	}
	return (lo + hi) / 2
}

func TestInductorLEDSmallAndLargeInductanceStartup(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	for _, inductance := range []float64{1e-6, .001, 100} {
		g := inductorTestGraph(t, 5, 100, inductance, nil)
		led, _ := MakeComponent("D1", "led", Point{X: 800}, "")
		g.Components = append(g.Components, led)
		g.Wires[2].To = "D1.anode"
		g.Wires = append(g.Wires, Wire{ID: "WD", From: "D1.cathode", To: "GND1.reference", Connected: true})
		state, trace := initializeInductorTest(t, g)
		state, err := state.advance(trace, inductance*.001)
		if err != nil {
			t.Fatal("scaled LED startup", inductance, err)
		}
		if math.Abs(state.result.Inductors[0].Current-inductorLEDReference(.001)) > 1e-5 {
			t.Fatal("inductance time scaling", inductance, state.result)
		}
	}
}
