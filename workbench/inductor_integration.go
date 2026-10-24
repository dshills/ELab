package workbench

import (
	"fmt"
	"math"
)

// Three-stage Radau IIA collocation. Integrating each quadratic Lagrange
// polynomial at these nodes gives A; the final row is the quadrature weight.
// The method is fifth order and stiffly accurate (the final stage is output).
var radauNodes = [3]float64{(4 - math.Sqrt(6)) / 10, (4 + math.Sqrt(6)) / 10, 1}
var radauWeights = func() (weights [3][3]float64) {
	for i, c := range radauNodes {
		for j, node := range radauNodes {
			one, two := radauNodes[(j+1)%3], radauNodes[(j+2)%3]
			weights[i][j] = (c*c*c/3 - (one+two)*c*c/2 + one*two*c) / ((node - one) * (node - two))
		}
	}
	return weights
}()

// Trials contain only the private unknowns and storage values needed by the
// error estimator. A complete, publishable result is constructed once the
// whole output interval has succeeded.
type inductorStage struct {
	x, capV []float64
}

type inductorRoundoffError struct{}

func (*inductorRoundoffError) Error() string {
	return "RLC collocation corrections stalled at floating-point resolution"
}

func (n *mixedNetwork) radauEquations(x, old []float64, h float64, trace *Transient, end float64) ([][]float64, []float64, []float64) {
	size := n.matrixSize(h)
	dimension := 3 * size
	matrix := denseMatrix(dimension)
	residual, tolerances := make([]float64, dimension), make([]float64, dimension)
	for stage, c := range radauNodes {
		offset := stage * size
		local := x[offset : offset+size]
		// A positive interval retains the legacy square's left stimulus at a
		// corner. All stage nodes lie inside this already corner-split interval.
		a, f := n.mnaEquations(local, nil, c*h, trace, end-(1-c)*h, false)
		for row := 0; row < n.inductorIndex(0); row++ {
			copy(matrix[offset+row][offset:offset+size], a[row])
			residual[offset+row] = f[row]
			tolerances[offset+row] = n.residualTolerance(row, local, a, old, c*h)
		}
		for k, part := range n.inductors {
			index := n.inductorIndex(k)
			row := offset + index
			current, previous := local[index], old[len(n.caps)+k]
			residual[row] = current - previous
			matrix[row][row] = 1
			magnitude := math.Abs(current) + math.Abs(previous)
			one, two := n.pin(part, "1"), n.pin(part, "2")
			for other := range radauNodes {
				base := other * size
				coefficient := h * radauWeights[stage][other] / part.Value
				positive, negative := voltage(x[base:base+size], one), voltage(x[base:base+size], two)
				delta := coefficient * (positive - negative - inductorWindingResistance(part)*x[base+index])
				matrix[row][base+index] += coefficient * inductorWindingResistance(part)
				residual[row] -= delta
				magnitude += math.Abs(coefficient) * (math.Abs(positive) + math.Abs(negative))
				if one >= 0 {
					matrix[row][base+one] -= coefficient
				}
				if two >= 0 {
					matrix[row][base+two] += coefficient
				}
			}
			tolerances[row] = inductorKCLAbsolute + inductorKCLRelative*magnitude
		}
		for k, part := range n.caps {
			row := offset + n.capacitorIndex(k)
			one, two := n.pin(part, "1"), n.pin(part, "2")
			value := voltage(local, one) - voltage(local, two)
			residual[row] = value - old[k]
			magnitude := math.Abs(voltage(local, one)) + math.Abs(voltage(local, two)) + math.Abs(old[k])
			if one >= 0 {
				matrix[row][offset+one] = 1
			}
			if two >= 0 {
				matrix[row][offset+two] = -1
			}
			for other := range radauNodes {
				column := other*size + n.capacitorIndex(k)
				coefficient := h * radauWeights[stage][other] / part.Value
				delta := coefficient * x[column]
				residual[row] -= delta
				magnitude += math.Abs(delta)
				matrix[row][column] -= coefficient
			}
			tolerances[row] = 1e-22 + 1e-12*magnitude
		}
	}
	return matrix, residual, tolerances
}

func (n *mixedNetwork) radauStep(state *inductorStage, trace *Transient, h, end float64, budget *integrationBudget) (*inductorStage, error) {
	size := n.matrixSize(h)
	x := make([]float64, 3*size)
	for stage := range radauNodes {
		copy(x[stage*size:(stage+1)*size], state.x)
	}
	old := n.reactiveHistory(state.x, state.capV)
	bestCorrection, stagnant := math.Inf(1), 0
	for iteration := 0; iteration < maxMixedNewtonIterations; iteration++ {
		if err := budget.checkTime(); err != nil {
			return nil, err
		}
		budget.newtonIterations++
		if err := n.radauNodeRange(x, size); err != nil {
			return nil, err
		}
		a, f, tolerances := n.radauEquations(x, old, h, trace, end)
		good := true
		kcl := 0.
		for row, value := range f {
			if !finite(value) || math.Abs(value) > tolerances[row] {
				good = false
			}
			if row >= 2*size && row < 2*size+len(n.nodes) {
				kcl = math.Max(kcl, math.Abs(value))
			}
		}
		for row := range f {
			f[row] = -f[row]
		}
		delta, err := linearSolveMode(a, f, true)
		if err != nil {
			return nil, err
		}
		residualGood := good
		good = good && n.voltageCorrectionConverged(x, delta, 3)
		if good {
			if !finite(kcl) || kcl > 1e-9 {
				return nil, fmt.Errorf("RLC collocation result exceeds current residual range")
			}
			endState := x[2*size:]
			for k, part := range n.inductors {
				current := endState[n.inductorIndex(k)]
				if !finite(.5 * part.Value * current * current) {
					return nil, fmt.Errorf("inductor %s exceeds finite state range", part.ID)
				}
			}
			caps := make([]float64, len(n.caps))
			for k, part := range n.caps {
				caps[k] = voltage(endState, n.pin(part, "1")) - voltage(endState, n.pin(part, "2"))
			}
			return &inductorStage{x: endState, capV: caps}, nil
		}
		// A small, stagnant correction with already satisfied physical rows
		// signals cancellation, not success. Reject this trial so the caller
		// can propose a separately checked interval instead of shrinking
		// indefinitely toward the floating-point time floor.
		correction := n.inductorCorrectionRatio(x, delta, 3)
		if residualGood && correction <= 1 {
			if correction < bestCorrection*.5 {
				bestCorrection, stagnant = correction, 0
			} else {
				stagnant++
			}
		} else {
			bestCorrection, stagnant = math.Inf(1), 0
		}
		if stagnant >= 6 {
			return nil, &inductorRoundoffError{}
		}
		scale := 1.
		if n.nonlinear {
			scale = n.inductorNewtonScale(x, delta, 3)
		}
		for i := range x {
			x[i] += scale * delta[i]
		}
		if err := n.radauNodeRange(x, size); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("RLC collocation Newton solver did not converge")
}

func (n *mixedNetwork) radauNodeRange(x []float64, size int) error {
	for stage := range radauNodes {
		for node := range n.nodes {
			value := x[stage*size+node]
			if !finite(value) || math.Abs(value) > maxMixedNodeVoltage {
				return fmt.Errorf("RLC collocation node exceeds the ±30 V educational model range")
			}
		}
	}
	return nil
}
