package workbench

import (
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/internal/testutil"
)

func TestInductorRadauCollocationMomentsAndFifthOrder(t *testing.T) {
	for stage, node := range radauNodes {
		for degree := 0; degree <= 2; degree++ {
			integral := 0.
			for k, weight := range radauWeights[stage] {
				integral += weight * math.Pow(radauNodes[k], float64(degree))
			}
			want := math.Pow(node, float64(degree+1)) / float64(degree+1)
			if math.Abs(integral-want) > 2e-15 {
				t.Fatal("stage does not integrate its collocation polynomial", stage, degree, integral, want)
			}
		}
	}
	for degree := 0; degree <= 4; degree++ {
		integral := 0.
		for k, weight := range radauWeights[2] {
			integral += weight * math.Pow(radauNodes[k], float64(degree))
		}
		if math.Abs(integral-1/float64(degree+1)) > 2e-15 {
			t.Fatal("final quadrature lacks fifth order", degree, integral)
		}
	}
	previous := 0.
	for _, steps := range []int{1, 2, 4} {
		state, trace := initializeInductorTest(t, inductorTestGraph(t, 5, 100, 1, nil))
		h := .01 / float64(steps)
		trial := &inductorStage{x: state.x, capV: state.capV}
		for step := 1; step <= steps; step++ {
			var err error
			trial, err = state.net.radauStep(trial, trace, h, float64(step)*h, newIntegrationBudget())
			if err != nil {
				t.Fatal(err)
			}
		}
		difference := math.Abs(trial.x[state.net.inductorIndex(0)] - .05*-math.Expm1(-1))
		if difference > 5e-5 || previous > 0 && previous/difference < 20 {
			t.Fatal("fixed-step RL error lacks fifth-order refinement", steps, difference, previous)
		}
		previous = difference
	}
}

func TestInductorRadauCoupledJacobian(t *testing.T) {
	state, trace := initializeInductorTest(t, inductorNPNCapGraph(t))
	var err error
	state, err = state.advance(trace, .001)
	if err != nil {
		t.Fatal(err)
	}
	n := state.net
	size := len(state.x)
	x := make([]float64, 3*size)
	for stage := range radauNodes {
		copy(x[stage*size:], state.x)
	}
	old := n.reactiveHistory(state.x, state.capV)
	matrix, _, _ := n.radauEquations(x, old, .0001, trace, .0011)
	const h = 1e-6
	for col := range x {
		one, two := append([]float64{}, x...), append([]float64{}, x...)
		one[col] += h
		two[col] -= h
		_, positive, _ := n.radauEquations(one, old, .0001, trace, .0011)
		_, negative, _ := n.radauEquations(two, old, .0001, trace, .0011)
		for row := range x {
			derivative := (positive[row] - negative[row]) / (2 * h)
			if math.Abs(matrix[row][col]-derivative) > 1e-7+1e-5*math.Abs(matrix[row][col]) {
				t.Fatal("coupled stage Jacobian differs from independent finite difference", row, col, matrix[row][col], derivative)
			}
		}
	}
	// The optimized elimination must solve the same actual coupled device
	// matrix, including its cross-stage storage rows and row pivoting.
	_, rhs, _ := n.radauEquations(x, old, .0001, trace, .0011)
	copyMatrix := func() [][]float64 {
		out := make([][]float64, len(matrix))
		for row := range matrix {
			out[row] = append([]float64{}, matrix[row]...)
		}
		return out
	}
	dense, err := linearSolve(copyMatrix(), append([]float64{}, rhs...))
	if err != nil {
		t.Fatal(err)
	}
	optimized, err := linearSolveMode(copyMatrix(), append([]float64{}, rhs...), true)
	if err != nil {
		t.Fatal(err)
	}
	for row := range dense {
		if math.Abs(dense[row]-optimized[row]) > 1e-12*(1+math.Abs(dense[row])) {
			t.Fatal("zero-update optimization changed the coupled solution", row, dense[row], optimized[row])
		}
	}
}

func TestInductorRadauStiffDecay(t *testing.T) {
	// Raw kernel stability, deliberately without the adaptive accuracy
	// controller: increasingly unresolved passive decay must not retain
	// alternating near-unit gain as trapezoidal integration can.
	for _, sample := range []struct{ step, maximumGain float64 }{{.01, .4}, {1, .03}, {100, .00031}} {
		g := inductorTestGraph(t, 0, 100, 1, nil)
		initial := .05
		g.Components[2].InitialCurrent = &initial
		state, trace := initializeInductorTest(t, g)
		trial := &inductorStage{x: state.x, capV: state.capV}
		trial, err := state.net.radauStep(trial, trace, sample.step, sample.step, newIntegrationBudget())
		if err != nil {
			t.Fatal(err)
		}
		gain := trial.x[state.net.inductorIndex(0)] / initial
		if gain < 0 || gain > sample.maximumGain {
			t.Fatal("stiff passive mode was not damped", sample.step, gain)
		}
	}
}

func TestInductorMixedOutputWorkEnvelope(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	state, trace := initializeInductorTest(t, inductorNPNCapGraph(t))
	maximumTime, maximumAttempts := time.Duration(0), 0
	for sample := 1; sample <= 120; sample++ {
		budget := newIntegrationBudget()
		var err error
		state, trace, err = advanceMixedBudget(state, trace, float64(sample)*.001, budget)
		if err != nil {
			t.Fatal(sample, err)
		}
		maximumTime = max(maximumTime, time.Since(budget.started))
		maximumAttempts = max(maximumAttempts, budget.attempts)
	}
	t.Logf("120 mixed RC/NPN/L outputs: peak %s, peak %d/%d adaptive attempts; wall budget %s", maximumTime, maximumAttempts, MaxIntegrationAttempts, MaxIntegrationWallTime)
}

func TestInductorStageDeadlineRetainsAcceptedState(t *testing.T) {
	state, trace := initializeInductorTest(t, RLCExampleGraph())
	before, vector := cloneMixed(state.result), append([]float64{}, state.x...)
	budget := newIntegrationBudget()
	calls := 0
	budget.elapsed = func() time.Duration {
		calls++
		if calls == 1 {
			return 0 // Admission succeeds; the first stage sees the deadline.
		}
		return MaxIntegrationWallTime + time.Nanosecond
	}
	result, err := state.advanceBudget(trace, .001, budget)
	var limit *integrationBudgetError
	if result != nil || !errors.As(err, &limit) || limit.code != "integration-time-limit" || !reflect.DeepEqual(before, cloneMixed(state.result)) || !reflect.DeepEqual(vector, state.x) {
		t.Fatal("stage deadline did not refuse atomically", result, err)
	}
}
