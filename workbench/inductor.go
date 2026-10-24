package workbench

import "math"

const InductorModelID = "ideal-inductor.v1"
const InductorAnalysis = "RLC network transient"
const InductorEngineID = "elab.rlc-mna-radau5.v1"
const MaxNetworkInductors = 4
const MaxInductorInitialCurrent = 1.

// The clipped LED's forward knee begins at Is=1e-20 A. Resolving that
// startup with the smallest supported L needs sub-attosecond internal steps;
// public sample bounds and the per-output time/attempt budgets are unchanged.
const minimumInductorStep = 1e-28

const (
	inductorKCLAbsolute = 1e-30
	inductorKCLRelative = 1e-12
)

type InductorResult struct {
	ID      string  `json:"id"`
	Voltage float64 `json:"voltage_v"`
	Current float64 `json:"current_a"`
	Energy  float64 `json:"energy_j"`
}

// Small current residuals do not bound voltage error near a controlled
// current source. Check the actual Newton correction before accepting an L,
// powered op-amp or MOSFET network; this is tighter than integration accuracy.
func (n *mixedNetwork) voltageCorrectionConverged(x, delta []float64, stages int) bool {
	size := len(x) / stages
	for stage := range stages {
		for node := range n.nodes {
			index := stage*size + node
			if !finite(delta[index]) || math.Abs(delta[index]) > 1e-10+1e-11*math.Abs(x[index]) {
				return false
			}
		}
	}
	return true
}

func (n *mixedNetwork) residualTolerance(row int, x []float64, jacobian [][]float64, old []float64, dt float64) float64 {
	if row < len(n.nodes) {
		if len(n.inductors) == 0 {
			return 1e-10 // Preserve legacy absolute KCL acceptance.
		}
		if dt == 0 && !n.dc {
			for _, cut := range n.inductorCuts {
				if row == cut.row {
					return 1e-8 // Differentiated current balance is a voltage row.
				}
			}
		}
		// A fixed absolute KCL floor would accept arbitrary LED voltages at
		// Is-scale startup currents. Scale roundoff to this node's local
		// Jacobian/current magnitude, without weakening legacy paths.
		scale := 0.
		for col, coefficient := range jacobian[row] {
			scale += math.Abs(coefficient * x[col])
		}
		return inductorKCLAbsolute + inductorKCLRelative*scale
	}
	if dt == 0 && !n.dc && row >= n.inductorIndex(0) && row < n.capacitorIndex(0) {
		return inductorKCLAbsolute + inductorKCLRelative*math.Abs(old[len(n.caps)+row-n.inductorIndex(0)])
	}
	return 1e-8 // Source/C voltage and positive-time storage equations.
}

func (n *mixedNetwork) inductorIndex(k int) int { return len(n.nodes) + len(n.sources) + k }
func (n *mixedNetwork) capacitorIndex(k int) int {
	return len(n.nodes) + len(n.sources) + len(n.inductors) + k
}
func (n *mixedNetwork) matrixSize(dt float64) int {
	size := n.capacitorIndex(0)
	if dt == 0 || n.usesCollocation() {
		size += len(n.caps)
	}
	return size
}

// Snapshot only storage states, before Newton seeds can change the guess.
// Layout: capacitor voltages, inductor currents. Algebraic voltages and cap
// currents remain in the accepted vector for readings, not integration memory.
func (n *mixedNetwork) reactiveHistory(guess, caps []float64) []float64 {
	old := make([]float64, len(n.caps)+len(n.inductors))
	copy(old, caps)
	for k, p := range n.inductors {
		i := 0.
		if p.InitialCurrent != nil {
			i = *p.InitialCurrent
		}
		if index := n.inductorIndex(k); index < len(guess) {
			i = guess[index]
		}
		old[len(n.caps)+k] = i
	}
	return old
}

func (n *mixedNetwork) stampInductors(x, old []float64, add func(int, int, float64), f []float64) {
	for k, p := range n.inductors {
		u, v, row := n.pin(p, "1"), n.pin(p, "2"), n.inductorIndex(k)
		current := x[row]
		if u >= 0 {
			f[u] += current
		}
		if v >= 0 {
			f[v] -= current
		}
		add(u, row, 1)
		add(v, row, -1)
		if !n.dc {
			f[row] = current - old[len(n.caps)+k]
			add(row, row, 1)
			continue
		}
		f[row] = voltage(x, u) - voltage(x, v) - inductorWindingResistance(p)*current
		add(row, row, -inductorWindingResistance(p))
		add(row, u, 1)
		add(row, v, -1)
	}
}

// Guard against stepping over a whole LC oscillation before the full/half-step
// comparison can observe it. Adaptive current/voltage checks remain mandatory.
func (n *mixedNetwork) maximumInductorStep() float64 {
	if len(n.inductors) == 0 || len(n.caps) == 0 {
		return math.Inf(1)
	}
	inductance, capacitance := math.Inf(1), math.Inf(1)
	for _, p := range n.inductors {
		inductance = math.Min(inductance, p.Value)
	}
	for _, p := range n.caps {
		capacitance = math.Min(capacitance, p.Value)
	}
	return .1 * math.Sqrt(inductance*capacitance) / float64(len(n.inductors)*len(n.caps))
}

func (n *mixedNetwork) inductorError(previous, full, refined *inductorStage) float64 {
	absoluteV, absoluteI, relative := n.inductorAccuracy()
	ratio := 0.
	for k := range n.inductors {
		index := n.inductorIndex(k)
		tolerance := absoluteI + relative*math.Max(math.Abs(previous.x[index]), math.Abs(refined.x[index]))
		ratio = math.Max(ratio, math.Abs(full.x[index]-refined.x[index])/tolerance)
	}
	for index := range n.nodes {
		tolerance := absoluteV + relative*math.Max(math.Abs(previous.x[index]), math.Abs(refined.x[index]))
		ratio = math.Max(ratio, math.Abs(full.x[index]-refined.x[index])/tolerance)
	}
	for k, value := range refined.capV {
		tolerance := 5e-8 + 5e-7*math.Max(math.Abs(value), math.Abs(previous.capV[k]))
		ratio = math.Max(ratio, math.Abs(value-full.capV[k])/tolerance)
	}
	return ratio
}

func (n *mixedNetwork) inductorAccuracy() (absoluteV, absoluteI, relative float64) {
	if len(n.caps) == 0 {
		return 5e-7, 5e-9, 5e-6
	}
	return 5e-8, 1e-10, 5e-7
}

func (n *mixedNetwork) inductorCorrectionRatio(x, delta []float64, stages int) float64 {
	absolute, _, relative := n.inductorAccuracy()
	ratio, size := 0., len(x)/stages
	for stage := range stages {
		for node := range n.nodes {
			i := stage*size + node
			if !finite(delta[i]) {
				return math.Inf(1)
			}
			ratio = math.Max(ratio, math.Abs(delta[i])/(absolute+relative*math.Abs(x[i])))
		}
	}
	return ratio
}
