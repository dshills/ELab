package workbench

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"
	"time"
)

func capacitorDecayFixture(t *testing.T) (*mixedState, *Transient) {
	t.Helper()
	graph := highPassRCGraph()
	graph.Components[0].Value = 0
	c, _, diagnostic := graph.Compile()
	if diagnostic != nil {
		t.Fatal(diagnostic)
	}
	trace := &Transient{Settings: DefaultTransientSettings(), Mode: "low"}
	state, err := newMixedNetwork(c).solve(nil, []float64{1}, 0, trace, 0)
	if err != nil {
		t.Fatal(err)
	}
	return state, trace
}

func TestCapacitorStageStabilityFunctionAndSecondOrder(t *testing.T) {
	for _, h := range []float64{.0001, .001, .1, 10} {
		state, trace := capacitorDecayFixture(t)
		budget := &integrationBudget{elapsed: func() time.Duration { return 0 }}
		next, err := state.capacitorStep(trace, h, h, budget)
		if err != nil {
			t.Fatal(err)
		}
		z, gamma := -h/.01, 1-1/math.Sqrt2
		want := (1 + (1-2*gamma)*z) / math.Pow(1-gamma*z, 2)
		if math.Abs(next.capV[0]-want) > 1e-10 {
			t.Fatal("capacitor stages differ from independent stability function", h, next.capV[0], want)
		}
	}
	previous := 0.
	for _, steps := range []int{4, 8, 16} {
		state, trace := capacitorDecayFixture(t)
		h := .01 / float64(steps)
		for i := 1; i <= steps; i++ {
			var err error
			state, err = state.capacitorStep(trace, h, float64(i)*h, &integrationBudget{elapsed: func() time.Duration { return 0 }})
			if err != nil {
				t.Fatal(err)
			}
		}
		error := math.Abs(state.capV[0] - math.Exp(-1))
		if previous > 0 && (previous/error < 3.8 || previous/error > 4.3) {
			t.Fatal("halving the step does not produce second-order decay convergence", steps, previous, error)
		}
		previous = error
	}
}

func TestCapacitorStageReportsInstantaneousPower(t *testing.T) {
	state, trace := capacitorDecayFixture(t)
	next, err := state.capacitorStep(trace, .001, .001, &integrationBudget{elapsed: func() time.Duration { return 0 }})
	if err != nil {
		t.Fatal(err)
	}
	r := next.result
	if r.NumericalDamping != 0 {
		t.Fatal("SDIRK stage history must not be reported as backward-Euler damping", r.NumericalDamping)
	}
	power := 0.
	for _, source := range r.Sources {
		power += source.Power
	}
	for _, resistor := range r.Resistors {
		power -= resistor.Power
	}
	for _, capacitor := range r.Capacitors {
		power -= capacitor.Voltage * capacitor.Current
	}
	if math.Abs(power) > 1e-12 || math.Abs(r.PowerResidual-power) > 1e-15 {
		t.Fatal("reported power balance differs from instantaneous terminal power", r.PowerResidual, power)
	}
}

func TestCapacitorStageBudgetFailurePreservesAcceptedState(t *testing.T) {
	state, trace := capacitorDecayFixture(t)
	x, caps := append([]float64(nil), state.x...), append([]float64(nil), state.capV...)
	result, err := json.Marshal(state.result)
	if err != nil {
		t.Fatal(err)
	}
	checks := 0
	budget := &integrationBudget{elapsed: func() time.Duration {
		checks++
		if checks >= 2 {
			return MaxIntegrationWallTime + time.Nanosecond
		}
		return 0
	}}
	next, err := state.capacitorStep(trace, .001, .001, budget)
	after, encodeErr := json.Marshal(state.result)
	if next != nil || simulationFailureCode(err, "other") != "integration-time-limit" || !reflect.DeepEqual(state.x, x) || !reflect.DeepEqual(state.capV, caps) || encodeErr != nil || !bytes.Equal(after, result) {
		t.Fatal("failed intermediate stage publishes or changes accepted memory", err)
	}
}

func TestCapacityFourCapacitorMOSFETKernelWithinAttemptBudget(t *testing.T) {
	data, err := os.ReadFile("../internal/conformance/testdata/capacity-rc-mosfet.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Graph    Graph             `json:"graph"`
		Settings TransientSettings `json:"settings"`
	}
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	c, _, diagnostic := fixture.Graph.Compile()
	if diagnostic != nil || SimulatedComponentCount(fixture.Graph) != 64 || len(c.Nets) != 60 {
		t.Fatal("capacity fixture identity differs", diagnostic)
	}
	kernel, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: fixture.Settings, mode: "auto", run: 1, revision: 1})
	if err != nil {
		t.Fatal(err)
	}
	peakAttempts := 0
	for sample := 1; sample <= 200; sample++ {
		// Numerical/race regression: keep the real attempt bound and exclude
		// machine-speed assertions. Ordinary public runs qualify wall budgets.
		budget := &integrationBudget{elapsed: func() time.Duration { return 0 }}
		next, trace, e := advanceMixedBudget(kernel.mixed, kernel.trace, float64(sample)*.001, budget)
		if e != nil {
			t.Fatalf("sample %d, attempts %d: %v", sample, budget.attempts, e)
		}
		kernel.mixed, kernel.trace = next, trace
		r := next.result
		peakAttempts = max(peakAttempts, budget.attempts)
		if r.Engine != NetworkEngineID || len(r.Capacitors) != 4 || len(r.MOSFETs) != 1 || len(r.LEDs) != 1 || r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
			t.Fatal("capacity state loses its model, storage or conservation", sample)
		}
		if sample == 50 && r.LEDs[0].Current < .002 || sample == 150 && r.LEDs[0].Current > 1e-6 {
			t.Fatal("four-stage timing does not turn the LED on and off", sample, r.LEDs)
		}
	}
	t.Logf("64 parts /60 nodes /four capacitors: maximum %d/%d attempts per 1 ms output", peakAttempts, MaxIntegrationAttempts)
}
