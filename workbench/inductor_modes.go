package workbench

import (
	"context"
	"math"
	"math/cmplx"

	"gonum.org/v1/gonum/mat"
)

// Initially qualify modal stepping only for the newly supported larger analog
// BJT/speaker networks. All other models keep their original LC ceiling.
func (n *mixedNetwork) usesModalLCCeiling() bool {
	bipolar, speaker := 0, false
	for _, p := range n.authored.Components {
		switch p.Kind {
		case "npn", "pnp":
			bipolar++
		case "speaker":
			speaker = true
		case "resistor", "potentiometer", "capacitor", "inductor", "thermistor", "ldr", "switch", "led", "diode", "schottky", "zener", "dc-source", "sine-source", "pulse-source", "finite-pulse-source", "ground", "junction", "net-label":
		default:
			return false
		}
	}
	return bipolar > 2 && speaker
}

// The differential state is [capacitor voltages, inductor currents]. Eliminate
// the algebraic unknowns from the actual local MNA Jacobian, holding each storage
// state independently fixed. Energy scaling balances disparate capacitances and
// inductances without changing the eigenvalues. No finite-difference stimulus,
// state mutation, pole fitting or assumed transistor bandwidth is involved.
func (n *mixedNetwork) storageJacobian(state *inductorStage, trace *Transient, at float64, budget *integrationBudget) ([][]float64, bool) {
	count := len(n.caps) + len(n.inductors)
	if count == 0 || count > MaxNetworkCapacitors+MaxNetworkInductors {
		return nil, false
	}
	if len(state.x) != n.matrixSize(1) || len(state.capV) != len(n.caps) {
		return nil, false
	}
	// mnaEquations returns (fresh owned Jacobian, residual vector), not an
	// error or a cached matrix. Its empty storage rows are filled below.
	base, residuals := n.mnaEquations(state.x, nil, 1, trace, at, false)
	size := len(base)
	if size != len(state.x) || len(residuals) != size {
		return nil, false
	}
	rows, scales := make([]int, count), make([]float64, count)
	for k, p := range n.caps {
		row := n.capacitorIndex(k)
		rows[k], scales[k] = row, math.Sqrt(p.Value)
		for j := range base[row] {
			base[row][j] = 0
		}
		if u := n.pin(p, "1"); u >= 0 {
			base[row][u] = 1
		}
		if v := n.pin(p, "2"); v >= 0 {
			base[row][v] = -1
		}
		base[row][row] -= capacitorESR(p)
	}
	for k, p := range n.inductors {
		row, j := n.inductorIndex(k), len(n.caps)+k
		rows[j], scales[j] = row, math.Sqrt(p.Value)
		for col := range base[row] {
			base[row][col] = 0
		}
		base[row][row] = 1
	}
	// Coupled windings are not admitted by the modal whitelist above.
	a := denseMatrix(count)
	for col, row := range rows {
		if budget.checkTime() != nil {
			return nil, false
		}
		jac := denseMatrix(size)
		for k := range base {
			copy(jac[k], base[k])
		}
		rhs := make([]float64, size)
		rhs[row] = 1
		x, err := linearSolveMode(jac, rhs, true)
		if err != nil {
			return nil, false
		}
		for j := range base {
			residual := 0.
			magnitude := 0.
			for k, v := range base[j] {
				residual += v * x[k]
				magnitude += math.Abs(v * x[k])
			}
			want := 0.
			if j == row {
				want = 1
			}
			if !finite(residual) || math.Abs(residual-want) > 1e-12*(1+magnitude) {
				return nil, false
			}
		}
		for k, p := range n.caps {
			a[k][col] = x[n.capacitorIndex(k)] / p.Value * scales[k] / scales[col]
		}
		for k, p := range n.inductors {
			j := len(n.caps) + k
			a[j][col] = (voltage(x, n.pin(p, "1")) - voltage(x, n.pin(p, "2")) - inductorWindingResistance(p)*x[n.inductorIndex(k)]) / p.Value * scales[j] / scales[col]
		}
	}
	for _, row := range a {
		for _, v := range row {
			if !finite(v) {
				return nil, false
			}
		}
	}
	return a, true
}

// Check a complete eigenbasis and its backward residual. The uncertainty radius
// includes eigenvector conditioning and floating-point accumulation. Unresolved,
// singular or ill-conditioned bases keep the conservative original ceiling.
func checkedOscillationRate(a [][]float64) (float64, bool) {
	count := len(a)
	if count == 0 {
		return 0, false
	}
	data := make([]float64, 0, count*count)
	normA := 0.
	for _, row := range a {
		if len(row) != count {
			return 0, false
		}
		sum := 0.
		for _, v := range row {
			if !finite(v) {
				return 0, false
			}
			sum += math.Abs(v)
		}
		normA = math.Max(normA, sum)
		data = append(data, row...)
	}
	var eigen mat.Eigen
	if !eigen.Factorize(mat.NewDense(count, count, data), mat.EigenRight) {
		return 0, false
	}
	values := eigen.Values(nil)
	var vectors mat.CDense
	eigen.VectorsTo(&vectors)
	v, inv := make([][]complex128, count), make([][]complex128, count)
	normV := 0.
	for i := range v {
		v[i] = make([]complex128, count)
		inv[i] = make([]complex128, count)
		sum := 0.
		for j := range v[i] {
			v[i][j] = vectors.At(i, j)
			sum += cmplx.Abs(v[i][j])
		}
		normV = math.Max(normV, sum)
	}
	for col := range count {
		copyV := make([][]complex128, count)
		for j := range v {
			copyV[j] = append([]complex128{}, v[j]...)
		}
		rhs := make([]complex128, count)
		rhs[col] = 1
		x, err := complexSolve(context.Background(), copyV, rhs)
		if err != nil {
			return 0, false
		}
		for row, z := range x {
			inv[row][col] = z
		}
	}
	normInv, identityError, residualNorm := 0., 0., 0.
	for i := range count {
		sum, identity, residual := 0., 0., 0.
		for j := range count {
			sum += cmplx.Abs(inv[i][j])
			one, r := complex(0, 0), -v[i][j]*values[j]
			for k := range count {
				one += v[i][k] * inv[k][j]
				r += complex(a[i][k], 0) * v[k][j]
			}
			if i == j {
				one -= 1
			}
			identity += cmplx.Abs(one)
			residual += cmplx.Abs(r)
		}
		normInv = math.Max(normInv, sum)
		identityError = math.Max(identityError, identity)
		residualNorm = math.Max(residualNorm, residual)
	}
	condition := normV * normInv
	if !finite(condition) || condition > 1e8 || identityError > 1e-10 {
		return 0, false
	}
	const epsilon = 2.220446049250313e-16
	radius := condition * (residualNorm*normInv + normA*(identityError+256*float64(count)*epsilon)) / (1 - identityError)
	rate := 0.
	for _, value := range values {
		rate = math.Max(rate, math.Max(math.Abs(imag(value)), math.Max(0, real(value))))
	}
	rate += radius
	return rate, finite(rate)
}

func (n *mixedNetwork) transientLCCeiling(state *inductorStage, trace *Transient, at float64, budget *integrationBudget) float64 {
	ceiling, _ := n.transientLCProposal(state, trace, at, budget)
	return ceiling
}

func (n *mixedNetwork) transientLCProposal(state *inductorStage, trace *Transient, at float64, budget *integrationBudget) (ceiling, initial float64) {
	fallback := n.maximumInductorStep()
	if !n.usesModalLCCeiling() {
		return fallback, fallback
	}
	budget.modalAnalyses++
	a, ok := n.storageJacobian(state, trace, at, budget)
	if !ok || budget.checkTime() != nil {
		return fallback, fallback
	}
	rate, ok := checkedOscillationRate(a)
	if !ok || budget.checkTime() != nil {
		return fallback, fallback
	}
	// Resolve a newly initialized fast decay before trying a large output
	// interval. This is a starting guess, not a permanent ceiling: after an
	// accepted step the ordinary estimator proposes the next interval.
	norm := 0.
	for _, row := range a {
		sum := 0.
		for _, value := range row {
			sum += math.Abs(value)
		}
		norm = math.Max(norm, sum)
	}
	initial = math.Inf(1)
	if norm > 0 {
		initial = .1 / norm
	}
	if rate == 0 {
		return math.Inf(1), initial
	}
	// Keep the original conservative per-cycle/count factor, now applied to
	// actual local oscillation/growth rates instead of every possible L/C pair.
	return .1 / rate / float64(max(1, len(n.inductors)*len(n.caps))), initial
}
