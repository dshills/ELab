package workbench

import (
	"context"
	"fmt"
	"math"
)

const MixedAnalysis = "RC/NPN transient"

const (
	maxMixedNewtonIterations = 100
	maxMixedNewtonDelta      = .2
	maxMixedNodeVoltage      = 30.
)
const MixedEngineID = "elab.mixed-mna-sdirk2.v1"

type CapacitorResult struct {
	ID      string  `json:"id"`
	Voltage float64 `json:"voltage_v"`
	Current float64 `json:"current_a"`
	Energy  float64 `json:"energy_j"`
}
type BranchResult struct {
	ID      string  `json:"id"`
	Current float64 `json:"current_a"`
	Voltage float64 `json:"voltage_v"`
	Power   float64 `json:"power_w"`
}
type MixedTransistor struct {
	TransistorID     string   `json:"transistor_id"`
	Model            NPNModel `json:"model"`
	BaseCurrent      float64  `json:"base_current_a"`
	CollectorCurrent float64  `json:"collector_current_a"`
	EmitterCurrent   float64  `json:"emitter_current_a"`
	VBE              float64  `json:"vbe_v"`
	VCE              float64  `json:"vce_v"`
	TransistorPower  float64  `json:"power_w"`
	State            string   `json:"state"`
}

type MixedResult struct {
	Optocouplers       []OptocouplerResult   `json:"optocouplers,omitempty"`
	Motors             []MotorResult         `json:"motors,omitempty"`
	Batteries          []BatteryResult       `json:"batteries,omitempty"`
	Sensors            []SensorResult        `json:"sensors,omitempty"`
	Regulators         []RegulatorResult     `json:"regulators,omitempty"`
	Controls           []ControlResult       `json:"controls,omitempty"`
	Displays           []DisplayResult       `json:"displays,omitempty"`
	Sequential         []SequentialResult    `json:"sequential,omitempty"`
	LogicGates         []LogicResult         `json:"logic_gates,omitempty"`
	OpAmps             []OpAmpResult         `json:"opamps,omitempty"`
	Inductors          []InductorResult      `json:"inductors,omitempty"`
	MOSFETs            []MOSFETResult        `json:"mosfets,omitempty"`
	Potentiometers     []PotentiometerResult `json:"potentiometers,omitempty"`
	Timers555          []Timer555Result      `json:"timers555,omitempty"`
	Diodes             []BranchResult        `json:"diodes,omitempty"`
	Switches           []BranchResult        `json:"switches,omitempty"`
	Engine             string                `json:"engine"`
	RunID              uint64                `json:"run_id"`
	ElectricalRevision uint64                `json:"electrical_revision"`
	Time               float64               `json:"time_s"`
	Nodes              map[string]float64    `json:"nodes_v"`
	Transistors        []MixedTransistor     `json:"transistors"`
	Capacitors         []CapacitorResult     `json:"capacitors"`
	LEDs               []BranchResult        `json:"leds"`
	Resistors          []BranchResult        `json:"resistors"`
	Sources            []BranchResult        `json:"sources"`
	KCLResidual        float64               `json:"kcl_residual_a"`
	PowerResidual      float64               `json:"power_residual_w"`
	NumericalDamping   float64               `json:"numerical_damping_w"`
	Stale              bool                  `json:"stale"`
}

func cloneMixed(r *MixedResult) *MixedResult {
	if r == nil {
		return nil
	}
	o := *r
	o.Optocouplers = append([]OptocouplerResult(nil), r.Optocouplers...)
	o.Motors = append([]MotorResult(nil), r.Motors...)
	o.Batteries = append([]BatteryResult(nil), r.Batteries...)
	o.Sensors = append([]SensorResult(nil), r.Sensors...)
	o.Regulators = append([]RegulatorResult(nil), r.Regulators...)
	o.Controls = append([]ControlResult(nil), r.Controls...)
	o.Displays = append([]DisplayResult(nil), r.Displays...)
	o.Sequential = append([]SequentialResult(nil), r.Sequential...)
	o.LogicGates = append([]LogicResult(nil), r.LogicGates...)
	o.Timers555 = append([]Timer555Result(nil), r.Timers555...)
	o.OpAmps = append([]OpAmpResult(nil), r.OpAmps...)
	o.Inductors = append([]InductorResult(nil), r.Inductors...)
	o.Nodes = map[string]float64{}
	for k, v := range r.Nodes {
		o.Nodes[k] = v
	}
	o.MOSFETs = append([]MOSFETResult(nil), r.MOSFETs...)
	o.Transistors = append([]MixedTransistor{}, r.Transistors...)

	o.Capacitors = append([]CapacitorResult{}, r.Capacitors...)
	o.LEDs = append([]BranchResult{}, r.LEDs...)
	o.Resistors = append([]BranchResult{}, r.Resistors...)
	o.Sources = append([]BranchResult{}, r.Sources...)
	o.Potentiometers = append([]PotentiometerResult{}, r.Potentiometers...)
	o.Diodes = append([]BranchResult{}, r.Diodes...)
	o.Switches = append([]BranchResult{}, r.Switches...)
	return &o
}

type mixedNetwork struct {
	authored                     Graph
	dc                           bool
	nonlinear                    bool
	hasOptocoupler, hasRegulator bool
	hasMOSFET                    bool
	hasGeneralBipolar            bool
	c                            Compiled
	nodes                        []string
	index                        map[string]int
	sources, caps                []Component
	inductors                    []Component
	controls                     []Component
	controlMemory                []ControlMemory
	displays                     []Component
	logic                        []Component
	opamps                       []Component
	timers                       []Component
	timerMemory                  []timer555Memory
	sequential                   []Component
	sequenceMemory               []SequentialMemory
	inductorCuts                 []inductorCut
	inductorPins                 map[string]int
}
type mixedState struct {
	net     *mixedNetwork
	x, capV []float64
	result  *MixedResult
}

func newMixedNetwork(c Compiled) *mixedNetwork {
	c, authored := lowerPowerDevices(c)
	n := &mixedNetwork{authored: authored, c: c, index: map[string]int{c.Ground: -1}}
	for _, node := range c.Nets {
		if node.ID != c.Ground {
			n.index[node.ID] = len(n.nodes)
			n.nodes = append(n.nodes, node.ID)
		}
	}
	for _, p := range c.MixedGraph.Components {
		n.hasOptocoupler = n.hasOptocoupler || p.Kind == "optocoupler"
		n.hasRegulator = n.hasRegulator || IsRegulatorKind(p.Kind)
		n.hasMOSFET = n.hasMOSFET || IsMOSFETKind(p.Kind)
		n.hasGeneralBipolar = n.hasGeneralBipolar || p.Kind == "pnp" || p.Kind == "npn" && c.TerminalNet[p.ID+".emitter"] != c.Ground
		switch p.Kind {
		case "thermistor", "ldr", "resistor", "capacitor", "inductor", "switch", "potentiometer":
		case "dc-source", "pulse-source", "sine-source", "finite-pulse-source":
		case "ground", "junction", "net-label":
		default:
			// Unknown future devices must not silently receive linear stepping.
			n.nonlinear = true
		}
		if IsVoltageSource(p.Kind) {
			n.sources = append(n.sources, p)
		}
		if p.Kind == "capacitor" {
			n.caps = append(n.caps, p)
		}
		if p.Kind == "inductor" || p.Kind == "relay" {
			n.inductors = append(n.inductors, p)
		}
		if p.Kind == "timer555" {
			n.timers = append(n.timers, p)
			n.timerMemory = append(n.timerMemory, timer555Memory{})
		}
		if IsSequentialKind(p.Kind) {
			n.sequential = append(n.sequential, p)
			n.sequenceMemory = append(n.sequenceMemory, SequentialMemory{})
		}
		if IsControlKind(p.Kind) {
			n.controls = append(n.controls, p)
			n.controlMemory = append(n.controlMemory, ControlMemory{})
		}
		if IsDisplayKind(p.Kind) {
			n.displays = append(n.displays, p)
		}
		if IsLogicKind(p.Kind) {
			n.logic = append(n.logic, p)
		}
		if p.Kind == "opamp" {
			n.opamps = append(n.opamps, p)
		}
	}
	if len(n.inductors) > 0 {
		// A compiled plan is immutable. Resolve the second map lookup once
		// instead of repeating it in every coupled stage/Newton iteration.
		n.inductorPins = make(map[string]int, len(c.TerminalNet))
		for terminal, net := range c.TerminalNet {
			n.inductorPins[terminal] = n.index[net]
		}
	}
	n.inductorCuts = n.inductorStartupCuts()
	return n
}
func (n *mixedNetwork) pin(p Component, t string) int {
	if p.Kind == "relay" {
		if t == "1" {
			t = "coil1"
		}
		if t == "2" {
			t = "coil2"
		}
	}
	if n.inductorPins != nil {
		if node, ok := n.inductorPins[p.ID+"."+t]; ok {
			return node
		}
	}
	return n.index[n.c.TerminalNet[p.ID+"."+t]]
}
func voltage(x []float64, i int) float64 {
	if i < 0 {
		return 0
	}
	return x[i]
}
func (n *mixedNetwork) sourceVoltage(p Component, t *Transient, at float64) float64 {
	return sourceVoltageAt(p, t, at, p.ID == n.c.Source.ID)
}
func (n *mixedNetwork) sourceStimulus(p Component, t *Transient, at, dt float64) float64 {
	// Legacy squares retain left-hand phase integration at discontinuities.
	if dt > 0 && p.Kind == "pulse-source" {
		at -= dt / 2
	}
	return n.sourceVoltage(p, t, at)
}
func diodeCurrent(v float64) (i, g float64) {
	if v <= 0 {
		return 0, 0
	}
	lo, hi := 0., v/Model().SeriesResistance
	for k := 0; k < 65; k++ {
		mid := (lo + hi) / 2
		if ledDrop(mid) > v {
			hi = mid
		} else {
			lo = mid
		}
	}
	i = (lo + hi) / 2
	m := Model()
	vt := 1.380649e-23 * (273.15 + m.Temperature) / 1.602176634e-19
	return i, 1 / (m.SeriesResistance + m.EmissionCoefficient*vt/(i+m.SaturationCurrent))
}

// NPN currents and analytic derivatives of the reciprocal transport equations.
func npnLinear(vb, vc float64) (ib, ic float64, db, dc, cb, cc float64) {
	m := TransistorModel()
	vt := 1.380649e-23 * (273.15 + m.Temperature) / 1.602176634e-19
	eb, ec := math.Exp(math.Max(-80, math.Min(80, vb/vt))), math.Exp(math.Max(-80, math.Min(80, (vb-vc)/vt)))
	f, r := m.SaturationCurrent*(eb-1), m.SaturationCurrent*(ec-1)
	gf, gr := m.SaturationCurrent*eb/vt, m.SaturationCurrent*ec/vt
	return f/m.ForwardBeta + r/m.ReverseBeta, f - (1+1/m.ReverseBeta)*r, gf/m.ForwardBeta + gr/m.ReverseBeta, -gr / m.ReverseBeta, gf - (1+1/m.ReverseBeta)*gr, (1 + 1/m.ReverseBeta) * gr
}
func (n *mixedNetwork) equations(x, old []float64, dt float64, t *Transient, at float64) ([][]float64, []float64) {
	a, f := n.deviceEquations(x, old, dt, t, at)
	if dt == 0 && !n.dc {
		n.stampInductorStartup(x, a, f)
	}
	return a, f
}

// deviceEquations retains physical KCL even when startup substitutes a
// differentiated cutset equation to determine otherwise undefined L voltages.
func (n *mixedNetwork) deviceEquations(x, old []float64, dt float64, t *Transient, at float64) ([][]float64, []float64) {
	return n.mnaEquations(x, old, dt, t, at, true)
}

// Collocation needs the same physical device/source equations and branch
// injections, then supplies its own coupled storage rows for all stages.
func (n *mixedNetwork) mnaEquations(x, old []float64, dt float64, t *Transient, at float64, storageRows bool) ([][]float64, []float64) {
	size := n.matrixSize(dt)
	a := denseMatrix(size)
	f := make([]float64, size)
	add := func(row, col int, v float64) {
		if row >= 0 && col >= 0 {
			a[row][col] += v
		}
	}
	flow := func(p, q int, i, g float64) {
		if p >= 0 {
			f[p] += i
		}
		if q >= 0 {
			f[q] -= i
		}
		add(p, p, g)
		add(q, q, g)
		add(p, q, -g)
		add(q, p, -g)
	}
	branch := func(p, q, k int, value float64) {
		i := x[k]
		if p >= 0 {
			f[p] += i
		}
		if q >= 0 {
			f[q] -= i
		}
		f[k] = voltage(x, p) - voltage(x, q) - value
		add(p, k, 1)
		add(q, k, -1)
		add(k, p, 1)
		add(k, q, -1)
	}
	for _, p := range n.c.MixedGraph.Components {
		switch p.Kind {
		case "optocoupler":
			n.stampOptocoupler(p, x, add, f)
		case "thermistor", "ldr":
			u, v := n.pin(p, "1"), n.pin(p, "2")
			r := sensorResistance(p)
			flow(u, v, (voltage(x, u)-voltage(x, v))/r, 1/r)
		case "resistor", "switch":
			u, v := n.pin(p, "1"), n.pin(p, "2")
			resistance := p.Value
			if p.Kind == "switch" {
				resistance = switchResistance(p)
			}
			g := 1 / resistance
			flow(u, v, (voltage(x, u)-voltage(x, v))*g, g)
		case "linear-regulator", "ldo-regulator":
			n.stampRegulator(p, x, add, f)
		case "relay", "comparator", "pushbutton", "bouncing-pushbutton":
			n.stampControl(p, x, add, f)
		case "bcd-decoder", "seven-segment":
			n.stampDisplay(p, x, add, f)
		case "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor":
			n.stampLogic(p, x, add, f)
		case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter":
			n.stampSequential(p, x, add, f)
		case "timer555":
			n.stampTimer555(p, x, add, f)
		case "potentiometer":
			upper, lower := potentiometerSegments(p)
			one, two, w := n.pin(p, "1"), n.pin(p, "2"), n.pin(p, "wiper")
			flow(one, w, (voltage(x, one)-voltage(x, w))/upper, 1/upper)
			flow(two, w, (voltage(x, two)-voltage(x, w))/lower, 1/lower)

		case "diode", "zener", "schottky":
			u, v := n.pin(p, "anode"), n.pin(p, "cathode")
			i, g := deviceDiodeCurrent(p, voltage(x, u)-voltage(x, v))
			flow(u, v, i, g)
		case "led":
			u, v := n.pin(p, "anode"), n.pin(p, "cathode")
			i, g := n.ledCurrent(voltage(x, u) - voltage(x, v))
			flow(u, v, i, g)
		case "nmos", "pmos":
			g, d, s := n.pin(p, "gate"), n.pin(p, "drain"), n.pin(p, "source")
			id, gm, gd := mosfetLinear(p.Kind, voltage(x, g)-voltage(x, s), voltage(x, d)-voltage(x, s))
			if d >= 0 {
				f[d] += id
			}
			if s >= 0 {
				f[s] -= id
			}
			add(d, g, gm)
			add(d, d, gd)
			add(d, s, -gm-gd)
			add(s, g, -gm)
			add(s, d, -gd)
			add(s, s, gm+gd)
		case "opamp":
			n.stampOpAmp(p, x, add, f)
		case "npn", "pnp":
			n.stampBipolar(p, x, add, f)
		}
	}
	for k, p := range n.sources {
		branch(n.pin(p, "positive"), n.pin(p, "negative"), len(n.nodes)+k, n.sourceStimulus(p, t, at, dt))
	}
	n.stampMotorCoupling(x, add, f)
	if !storageRows {
		for _, parts := range [][]Component{n.inductors, n.caps} {
			for k, p := range parts {
				row := n.inductorIndex(k)
				if p.Kind == "capacitor" {
					row = n.capacitorIndex(k)
				}
				u, v := n.pin(p, "1"), n.pin(p, "2")
				if u >= 0 {
					f[u] += x[row]
				}
				if v >= 0 {
					f[v] -= x[row]
				}
				add(u, row, 1)
				add(v, row, -1)
			}
		}
		return a, f
	}
	n.stampInductors(x, old, add, f)
	for k, p := range n.caps {
		u, v := n.pin(p, "1"), n.pin(p, "2")
		if dt == 0 {
			branch(u, v, n.capacitorIndex(k), old[k])
		} else {
			g := p.Value / dt
			flow(u, v, g*(voltage(x, u)-voltage(x, v)-old[k]), g)
		}
	}
	return a, f
}
func linearSolve(a [][]float64, b []float64) ([]float64, error) {
	return linearSolveMode(a, b, len(b) >= minimumZeroSkipSize)
}

// Omit zero updates in larger ordinary systems and collocation's mostly zero
// stage blocks. Both paths use the same pivot search and nonzero update order.
func linearSolveMode(a [][]float64, b []float64, skipZeros bool) ([]float64, error) {
	n := len(b)
	var columns []int
	if skipZeros {
		columns = make([]int, 0, n)
	}
	for col := 0; col < n; col++ {
		pivot := col
		pivotMagnitude := math.Abs(a[col][col])
		for row := col + 1; row < n; row++ {
			if magnitude := math.Abs(a[row][col]); magnitude > pivotMagnitude {
				pivot = row
				pivotMagnitude = magnitude
			}
		}
		if math.Abs(a[pivot][col]) < 1e-30 || !finite(a[pivot][col]) {
			return nil, fmt.Errorf("singular mixed circuit; check floating nodes or inconsistent ideal constraints")
		}
		a[col], a[pivot] = a[pivot], a[col]
		b[col], b[pivot] = b[pivot], b[col]
		pivotValues := a[col]
		if skipZeros {
			columns = columns[:0]
			for j := col + 1; j < n; j++ {
				if pivotValues[j] != 0 {
					columns = append(columns, j)
				}
			}
		}
		for row := col + 1; row < n; row++ {
			rowValues := a[row]
			factor := rowValues[col] / pivotValues[col]
			if skipZeros {
				if factor == 0 {
					continue
				}
				for _, j := range columns {
					rowValues[j] -= factor * pivotValues[j]
				}
			} else {
				for j := col + 1; j < n; j++ {
					a[row][j] -= factor * a[col][j]
				}
			}
			b[row] -= factor * b[col]
		}
	}
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		sum := b[i]
		rowValues := a[i]
		for j := i + 1; j < n; j++ {
			sum -= rowValues[j] * x[j]
		}
		x[i] = sum / rowValues[i]
		if !finite(x[i]) {
			return nil, fmt.Errorf("nonfinite mixed solution")
		}
	}
	return x, nil
}
func (n *mixedNetwork) solve(guess, old []float64, dt float64, t *Transient, at float64) (*mixedState, error) {
	return n.solveContext(context.Background(), guess, old, dt, t, at)
}
func (n *mixedNetwork) solveContext(ctx context.Context, guess, old []float64, dt float64, t *Transient, at float64) (*mixedState, error) {
	result, err := n.solveAnalogContext(ctx, guess, old, dt, t, at)
	if err != nil || len(n.timers)+len(n.sequential)+len(n.controls) == 0 || dt > 0 {
		return result, err
	}
	if len(n.sequential)+len(n.controls) > 0 {
		return result.resolveSequential(ctx, t)
	}
	return result.resolveTimers555(ctx, t)
}
func (n *mixedNetwork) solveAnalogContext(ctx context.Context, guess, old []float64, dt float64, t *Transient, at float64) (*mixedState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(n.inductors) > 0 {
		if dt > 0 {
			return nil, fmt.Errorf("dynamic inductor states require coupled collocation stepping")
		}
		old = n.reactiveHistory(guess, old)
		if dt == 0 && !n.dc {
			if err := n.checkInductorStartup(old); err != nil {
				return nil, err
			}
		}
	}
	result, err := n.newton(ctx, guess, old, dt, t, at)
	if err == nil {
		return result, nil
	}
	// Preserve the normal starting point and its solution basin. If a cold
	// nonlinear solve fails, make one bounded retry with only grounded ideal
	// voltages initialized, avoiding an artificial slow ramp of a fixed rail.
	if n.nonlinear && len(guess) == 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		seed := n.initialConstraintGuess(old, dt, t, at)
		changed := false
		for _, v := range seed[:len(n.nodes)] {
			changed = changed || v != 0
		}
		if changed {
			var seededErr error
			if result, seededErr = n.newton(ctx, seed, old, dt, t, at); seededErr == nil {
				return result, nil
			}
			err = seededErr
		}
	}
	qs := []Component{}
	for _, p := range n.c.MixedGraph.Components {
		if IsBipolarKind(p.Kind) {
			qs = append(qs, p)
		}
	}
	for choice := 0; choice < len(qs); choice++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		seed := make([]float64, n.matrixSize(dt))
		copy(seed, guess)
		for i, q := range qs {
			b, c := n.pin(q, "base"), n.pin(q, "collector")
			emitter, sign := voltage(seed, n.pin(q, "emitter")), bipolarPolarity(q.Kind)
			if i == choice {
				if b >= 0 {
					seed[b] = emitter + sign*.7
				}
				if c >= 0 {
					seed[c] = emitter + sign*.1
				}
			} else {
				if b >= 0 {
					seed[b] = emitter - sign
				}
				if c >= 0 {
					seed[c] = emitter + sign*3
				}
			}
		}
		if result, e := n.newton(ctx, seed, old, dt, t, at); e == nil {
			return result, nil
		}
	}
	return nil, err
}
func (n *mixedNetwork) newton(ctx context.Context, guess, old []float64, dt float64, t *Transient, at float64) (*mixedState, error) {
	size := n.matrixSize(dt)
	x := make([]float64, size)
	copy(x, guess)
	if dt == 0 && !n.dc {
		for k := range n.inductors {
			// Start on the immutable current constraint, including pA-scale
			// intent; a residual tolerance must not erase a small startup.
			x[n.inductorIndex(k)] = old[len(n.caps)+k]
		}
	}
	for iter := 0; iter < maxMixedNewtonIterations; iter++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		a, f := n.equations(x, old, dt, t, at)
		good := true
		for i, v := range f {
			tol := n.residualTolerance(i, x, a, old, dt)
			if !finite(v) || math.Abs(v) > tol {
				good = false
			}
		}
		var delta []float64
		if good && (n.hasRegulator || len(n.inductors) > 0 || len(n.opamps) > 0 || n.hasMOSFET || n.hasGeneralBipolar || n.hasOptocoupler || len(n.timers) > 0 || len(n.logic) > 0 || len(n.sequential) > 0 || len(n.displays) > 0 || len(n.controls) > 0) {
			rhs := make([]float64, size)
			for i := range f {
				rhs[i] = -f[i]
			}
			var err error
			delta, err = linearSolve(a, rhs)
			if err != nil {
				return nil, err
			}
			good = n.voltageCorrectionConverged(x, delta, 1)
		}
		if good {
			if iter == 0 && len(n.inductors) == 0 {
				// A small residual is not evidence of a unique operating point.
				// Check rank with zero RHS without moving an already acceptable
				// guess or rebuilding all nonlinear device equations a second time.
				if _, err := linearSolve(a, make([]float64, size)); err != nil {
					return nil, err
				}
			}
			// A supplied guess can already satisfy KCL/constraints. Apply the
			// same range guard before accepting it, not only after an update.
			for _, v := range x[:len(n.nodes)] {
				if !finite(v) || math.Abs(v) > maxMixedNodeVoltage {
					return nil, fmt.Errorf("mixed node voltage exceeds the ±30 V educational model range")
				}
			}
			result := n.result(x, old, dt, t, at)
			if len(n.inductors) > 0 {
				if !finite(result.result.PowerResidual) || !finite(result.result.KCLResidual) {
					return nil, fmt.Errorf("RLC result exceeds the finite energy/power range")
				}
				if result.result.KCLResidual > 1e-9 {
					return nil, fmt.Errorf("RLC currents do not satisfy physical node current balance")
				}
				for _, p := range result.result.Inductors {
					if !finite(p.Voltage) || !finite(p.Current) || !finite(p.Energy) {
						return nil, fmt.Errorf("inductor %s exceeds the finite state range", p.ID)
					}
				}
			}
			return result, nil
		}
		if delta == nil {
			for i := range f {
				f[i] = -f[i]
			}
			var err error
			delta, err = linearSolve(a, f)
			if err != nil {
				return nil, err
			}
		}
		scale := 1.
		if n.nonlinear {
			if len(n.inductors) > 0 {
				scale = n.inductorNewtonScale(x, delta, 1)
			} else {
				for i := range n.nodes {
					if math.Abs(delta[i]) > maxMixedNewtonDelta {
						scale = math.Min(scale, maxMixedNewtonDelta/math.Abs(delta[i]))
					}
				}
				scale = n.opAmpNewtonScale(x, delta, 1, scale)
				scale = n.regulatorNewtonScale(x, delta, 1, scale)
			}
		}
		for i := range x {
			x[i] += scale * delta[i]
		}
		for i := range n.nodes {
			if math.Abs(x[i]) > maxMixedNodeVoltage {
				return nil, fmt.Errorf("mixed node voltage exceeds the ±30 V educational model range")
			}
		}
	}
	return nil, fmt.Errorf("mixed Newton solver did not converge; reduce sample step or check initial conditions")
}
func (n *mixedNetwork) result(x, old []float64, dt float64, t *Transient, at float64) *mixedState {
	r := &MixedResult{Engine: networkEngine(n.c.Analysis), RunID: t.RunID, ElectricalRevision: t.ElectricalRevision, Time: at, Nodes: map[string]float64{n.c.Ground: 0}, Transistors: []MixedTransistor{}, Capacitors: []CapacitorResult{}, LEDs: []BranchResult{}, Resistors: []BranchResult{}, Sources: []BranchResult{}}
	for i, id := range n.nodes {
		if !privatePart(id) {
			r.Nodes[id] = x[i]
		}
	}
	capV := make([]float64, len(n.caps))
	powers := 0.
	sourcePower := 0.
	energyRate := 0.
	for _, p := range n.c.MixedGraph.Components {
		switch p.Kind {
		case "resistor", "led", "switch", "diode", "zener", "schottky":
			a, b := "1", "2"
			if p.Kind == "led" || IsDiodeKind(p.Kind) {
				a, b = "anode", "cathode"
			}
			v := voltage(x, n.pin(p, a)) - voltage(x, n.pin(p, b))
			resistance := p.Value
			if p.Kind == "switch" {
				resistance = switchResistance(p)
			}
			var i float64
			switch p.Kind {
			case "led":
				i, _ = n.ledCurrent(v)
			case "diode", "zener", "schottky":
				i, _ = deviceDiodeCurrent(p, v)
			default:
				i = v / resistance
			}
			br := BranchResult{p.ID, i, v, i * v}
			powers += br.Power
			if p.Kind == "led" {
				r.LEDs = append(r.LEDs, br)
			} else if IsDiodeKind(p.Kind) {
				r.Diodes = append(r.Diodes, br)
			} else if p.Kind == "switch" {
				r.Switches = append(r.Switches, br)
			} else {
				if !privatePart(p.ID) {
					r.Resistors = append(r.Resistors, br)
				}
			}
		case "optocoupler":
			device := n.optoResult(p, x)
			r.Optocouplers = append(r.Optocouplers, device)
			powers += device.Power
		case "thermistor", "ldr":
			device := n.sensorResult(p, x)
			r.Sensors = append(r.Sensors, device)
			powers += device.Power
		case "linear-regulator", "ldo-regulator":
			device := n.regulatorResult(p, x)
			r.Regulators = append(r.Regulators, device)
			powers += device.Power
		case "relay", "comparator", "pushbutton", "bouncing-pushbutton":
			device := n.controlResult(p, x)
			r.Controls = append(r.Controls, device)
			powers += device.Power
		case "bcd-decoder", "seven-segment":
			device := n.displayResult(p, x)
			r.Displays = append(r.Displays, device)
			powers += device.Power
		case "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor":
			device := n.logicResult(p, x)
			r.LogicGates = append(r.LogicGates, device)
			powers += device.Power
		case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter":
			device := n.sequentialResult(p, x)
			r.Sequential = append(r.Sequential, device)
			powers += device.Power
		case "timer555":
			device := n.timer555Result(p, x)
			r.Timers555 = append(r.Timers555, device)
			powers += device.Power
		case "potentiometer":
			upper, lower := potentiometerSegments(p)
			v1, v2, vw := voltage(x, n.pin(p, "1")), voltage(x, n.pin(p, "2")), voltage(x, n.pin(p, "wiper"))
			i1, i2 := (v1-vw)/upper, (v2-vw)/lower
			power := (v1-vw)*i1 + (v2-vw)*i2
			powers += power
			r.Potentiometers = append(r.Potentiometers, PotentiometerResult{ID: p.ID, Resistance: p.Value, WiperPosition: *p.WiperPosition, UpperResistance: upper, LowerResistance: lower, Terminal1Voltage: v1, Terminal2Voltage: v2, WiperVoltage: vw, Terminal1Current: i1, Terminal2Current: i2, WiperCurrent: -i1 - i2, Power: power})

		case "nmos", "pmos":
			device := n.mosfetResult(p, x)
			powers += device.Power
			r.MOSFETs = append(r.MOSFETs, device)
		case "opamp":
			device := n.opAmpResult(p, x)
			powers += device.Power
			r.OpAmps = append(r.OpAmps, device)
		case "npn", "pnp":
			device := n.bipolarResult(p, x)
			powers += device.TransistorPower
			r.Transistors = append(r.Transistors, device)
		}
	}
	for k, p := range n.sources {
		v := n.sourceStimulus(p, t, at, dt)
		i := -x[len(n.nodes)+k]
		br := BranchResult{p.ID, i, v, v * i}
		if !privatePart(p.ID) {
			r.Sources = append(r.Sources, br)
		}
		sourcePower += br.Power
	}
	for k, p := range n.caps {
		v := voltage(x, n.pin(p, "1")) - voltage(x, n.pin(p, "2"))
		capV[k] = v
		i := 0.
		if dt == 0 || n.usesCollocation() {
			i = x[n.capacitorIndex(k)]
		} else {
			i = p.Value * (v - old[k]) / dt
		}
		// SDIRK history is a stage construction, not the previous accepted
		// voltage. Balance instantaneous terminal power for every method.
		energyRate += v * i
		if !privatePart(p.ID) {
			r.Capacitors = append(r.Capacitors, CapacitorResult{p.ID, v, i, .5 * p.Value * v * v})
		}
	}
	for k, p := range n.inductors {
		v := voltage(x, n.pin(p, "1")) - voltage(x, n.pin(p, "2"))
		i := x[n.inductorIndex(k)]
		if p.Kind == "relay" {
			energyRate += (v - inductorWindingResistance(p)*i) * i
			continue
		}
		energyRate += v * i
		if !privatePart(p.ID) {
			r.Inductors = append(r.Inductors, InductorResult{ID: p.ID, Voltage: v, Current: i, Energy: .5 * p.Value * i * i})
		}
	}
	_, f := n.mnaEquations(x, old, dt, t, at, !n.usesCollocation())
	for i := range n.nodes {
		r.KCLResidual = math.Max(r.KCLResidual, math.Abs(f[i]))
	}
	n.publishPower(r, x)
	r.PowerResidual = sourcePower - powers - energyRate - r.NumericalDamping
	return &mixedState{n, x, capV, r}
}

// Compare one full step with two half steps and accept the latter. Capacitor
// networks use SDIRK2; L networks use Radau IIA collocation.
func (state *mixedState) advance(t *Transient, end float64) (*mixedState, error) {
	return state.advanceBudget(t, end, newIntegrationBudget())
}
func (state *mixedState) advanceBudget(t *Transient, end float64, budget *integrationBudget) (*mixedState, error) {
	if len(state.net.sequential)+len(state.net.controls) > 0 {
		return state.advanceSequential(t, end, budget)
	}
	if len(state.net.timers) > 0 {
		return state.advanceTimers555(t, end, budget)
	}
	return state.advanceAnalogBudget(t, end, budget)
}
func (state *mixedState) advanceAnalogBudget(t *Transient, end float64, budget *integrationBudget) (*mixedState, error) {
	if state.net.usesCollocation() {
		return state.advanceInductorBudget(t, end, budget)
	}
	cur := state
	at := state.result.Time
	h := end - at
	if len(state.net.caps) == 0 {
		if err := budget.take(); err != nil {
			return nil, err
		}
		return state.net.solve(state.x, state.capV, end-at, t, end)
	}
	var failure string
	for at < end {
		if err := budget.take(); err != nil {
			return nil, fmt.Errorf("%w at %.9g s, step %.3g: %s", err, at, h, failure)
		}
		h = math.Min(h, end-at)
		if state.net.c.Analysis == WaveformAnalysis {
			h = math.Min(h, cur.net.maximumWaveformStep(t, at))
		}
		if h < 1e-10 || at+h <= at {
			return nil, fmt.Errorf("mixed timestep too small at %.9g, h %.3g: %s", at, h, failure)
		}
		full, e1 := cur.capacitorStep(t, h, at+h, budget)
		half, e2 := cur.capacitorStep(t, h/2, at+h/2, budget)
		var refined *mixedState
		var e3 error
		if e2 == nil {
			refined, e3 = half.capacitorStep(t, h/2, at+h, budget)
		}
		errRatio := math.Inf(1)
		if e1 == nil && e2 == nil && e3 == nil {
			errRatio = 0
			for i, v := range refined.capV {
				absolute, relative := 2e-6, 2e-4
				if cur.net.c.Analysis != MixedAnalysis {
					absolute, relative = 5e-8, 5e-6
					if cur.net.c.Analysis == WaveformAnalysis {
						absolute, relative = 5e-7, 5e-6
					}
				}
				if len(cur.net.opamps) > 0 {
					// Controlled buffers expose accumulated capacitor error at
					// their outputs. Use the established coupling-node tolerance
					// across their analysis families, including waveform circuits.
					absolute, relative = 5e-8, 1.25e-6
				}
				tol := absolute + relative*math.Max(math.Abs(v), math.Abs(cur.capV[i]))
				errRatio = math.Max(errRatio, math.Abs(v-full.capV[i])/tol)
				if (cur.net.c.Analysis != MixedAnalysis || len(cur.net.opamps) > 0) && cur.net.c.Analysis != WaveformAnalysis {
					part := cur.net.caps[i]
					one, two := cur.net.pin(part, "1"), cur.net.pin(part, "2")
					if one >= 0 && two >= 0 {
						// Coupling-capacitor voltage can be large while its output
						// node is small. Check both absolute node voltages too;
						// grounded capacitors retain their established error path.
						for _, node := range [2]int{one, two} {
							previous, next := voltage(cur.x, node), voltage(refined.x, node)
							nodeTolerance := 5e-8 + 1.25e-6*math.Max(math.Abs(previous), math.Abs(next))
							errRatio = math.Max(errRatio, math.Abs(next-voltage(full.x, node))/nodeTolerance)
						}
					}
				}
			}
		}
		failure = fmt.Sprintf("ratio %.3g, errors %v / %v / %v", errRatio, e1, e2, e3)
		if errRatio <= 1 {
			if len(cur.net.timers)+len(cur.net.sequential)+len(cur.net.controls) > 0 {
				for _, candidate := range []*mixedState{half, refined} {
					changed, err := candidate.hybridGuardChanged()
					if err != nil {
						return nil, err
					}
					if changed {
						return candidate, nil
					}
				}
			}
			cur = refined
			at += h
			if end-at < 1e-13 {
				at = end
			}
			cur.result.Time = at
			// Full-vs-two-half SDIRK2 error scales with h³. Keep the
			// existing acceptance bounds; independently check every proposal.
			if errRatio == 0 {
				h *= 2
			} else {
				h *= math.Min(2, math.Cbrt(.75/errRatio))
			}
		} else {
			h /= 2
		}
	}
	return cur, nil
}
