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
const MixedEngineID = "elab.mixed-mna-be.v1"

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
	Inductors          []InductorResult      `json:"inductors,omitempty"`
	MOSFETs            []MOSFETResult        `json:"mosfets,omitempty"`
	Potentiometers     []PotentiometerResult `json:"potentiometers,omitempty"`
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
	dc            bool
	nonlinear     bool
	c             Compiled
	nodes         []string
	index         map[string]int
	sources, caps []Component
	inductors     []Component
	inductorCuts  []inductorCut
}
type mixedState struct {
	net     *mixedNetwork
	x, capV []float64
	result  *MixedResult
}

func newMixedNetwork(c Compiled) *mixedNetwork {
	n := &mixedNetwork{c: c, index: map[string]int{c.Ground: -1}}
	for _, node := range c.Nets {
		if node.ID != c.Ground {
			n.index[node.ID] = len(n.nodes)
			n.nodes = append(n.nodes, node.ID)
		}
	}
	for _, p := range c.MixedGraph.Components {
		switch p.Kind {
		case "resistor", "capacitor", "inductor", "switch", "potentiometer":
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
		if p.Kind == "inductor" {
			n.inductors = append(n.inductors, p)
		}
	}
	n.inductorCuts = n.inductorStartupCuts()
	return n
}
func (n *mixedNetwork) pin(p Component, t string) int { return n.index[n.c.TerminalNet[p.ID+"."+t]] }
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
	a := make([][]float64, size)
	f := make([]float64, size)
	for i := range a {
		a[i] = make([]float64, size)
	}
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
		case "resistor", "switch":
			u, v := n.pin(p, "1"), n.pin(p, "2")
			resistance := p.Value
			if p.Kind == "switch" {
				resistance = switchResistance(p)
			}
			g := 1 / resistance
			flow(u, v, (voltage(x, u)-voltage(x, v))*g, g)
		case "potentiometer":
			upper, lower := potentiometerSegments(p)
			one, two, w := n.pin(p, "1"), n.pin(p, "2"), n.pin(p, "wiper")
			flow(one, w, (voltage(x, one)-voltage(x, w))/upper, 1/upper)
			flow(two, w, (voltage(x, two)-voltage(x, w))/lower, 1/lower)

		case "diode":
			u, v := n.pin(p, "anode"), n.pin(p, "cathode")
			i, g := siliconDiodeCurrent(voltage(x, u) - voltage(x, v))
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
		case "npn":
			b, c := n.pin(p, "base"), n.pin(p, "collector")
			ib, ic, bb, bc, cb, cc := npnLinear(voltage(x, b), voltage(x, c))
			f[b] += ib
			f[c] += ic
			add(b, b, bb)
			add(b, c, bc)
			add(c, b, cb)
			add(c, c, cc)
		}
	}
	for k, p := range n.sources {
		branch(n.pin(p, "positive"), n.pin(p, "negative"), len(n.nodes)+k, n.sourceStimulus(p, t, at, dt))
	}
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
	return linearSolveMode(a, b, false)
}

// Retain the original arithmetic path for existing circuits. Collocation's
// mostly zero stage blocks can omit zero updates without changing pivoting.
func linearSolveMode(a [][]float64, b []float64, skipZeros bool) ([]float64, error) {
	n := len(b)
	var columns []int
	if skipZeros {
		columns = make([]int, 0, n)
	}
	for col := 0; col < n; col++ {
		pivot := col
		for row := col + 1; row < n; row++ {
			if math.Abs(a[row][col]) > math.Abs(a[pivot][col]) {
				pivot = row
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
			factor := a[row][col] / a[col][col]
			if skipZeros {
				if factor == 0 {
					continue
				}
				rowValues := a[row]
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
		for j := i + 1; j < n; j++ {
			sum -= a[i][j] * x[j]
		}
		x[i] = sum / a[i][i]
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
		if p.Kind == "npn" {
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
			if i == choice {
				seed[b] = .7
				seed[c] = .1
			} else {
				seed[b] = -1
				seed[c] = 3
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
		if good && len(n.inductors) > 0 {
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
			for i := range n.nodes {
				if math.Abs(delta[i]) > maxMixedNewtonDelta {
					scale = math.Min(scale, maxMixedNewtonDelta/math.Abs(delta[i]))
				}
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
		r.Nodes[id] = x[i]
	}
	capV := make([]float64, len(n.caps))
	powers := 0.
	sourcePower := 0.
	energyRate := 0.
	for _, p := range n.c.MixedGraph.Components {
		switch p.Kind {
		case "resistor", "led", "switch", "diode":
			a, b := "1", "2"
			if p.Kind == "led" || p.Kind == "diode" {
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
			case "diode":
				i, _ = siliconDiodeCurrent(v)
			default:
				i = v / resistance
			}
			br := BranchResult{p.ID, i, v, i * v}
			powers += br.Power
			if p.Kind == "led" {
				r.LEDs = append(r.LEDs, br)
			} else if p.Kind == "diode" {
				r.Diodes = append(r.Diodes, br)
			} else if p.Kind == "switch" {
				r.Switches = append(r.Switches, br)
			} else {
				r.Resistors = append(r.Resistors, br)
			}
		case "potentiometer":
			upper, lower := potentiometerSegments(p)
			v1, v2, vw := voltage(x, n.pin(p, "1")), voltage(x, n.pin(p, "2")), voltage(x, n.pin(p, "wiper"))
			i1, i2 := (v1-vw)/upper, (v2-vw)/lower
			power := (v1-vw)*i1 + (v2-vw)*i2
			powers += power
			r.Potentiometers = append(r.Potentiometers, PotentiometerResult{ID: p.ID, Resistance: p.Value, WiperPosition: *p.WiperPosition, UpperResistance: upper, LowerResistance: lower, Terminal1Voltage: v1, Terminal2Voltage: v2, WiperVoltage: vw, Terminal1Current: i1, Terminal2Current: i2, WiperCurrent: -i1 - i2, Power: power})

		case "nmos", "pmos":
			vgs := voltage(x, n.pin(p, "gate")) - voltage(x, n.pin(p, "source"))
			vds := voltage(x, n.pin(p, "drain")) - voltage(x, n.pin(p, "source"))
			id, _, _ := mosfetLinear(p.Kind, vgs, vds)
			power := vds * id
			powers += power
			r.MOSFETs = append(r.MOSFETs, MOSFETResult{ID: p.ID, Model: mosfetModel(p.Kind), DrainCurrent: id, SourceCurrent: id, VGS: vgs, VDS: vds, Power: power, State: mosfetState(p.Kind, vgs, vds)})
		case "npn":
			vb, vc := voltage(x, n.pin(p, "base")), voltage(x, n.pin(p, "collector"))
			ib, ic, _, _, _, _ := npnLinear(vb, vc)
			state := "active"
			if math.Abs(ib) < 1e-8 && math.Abs(ic) < 1e-8 {
				state = "cutoff"
			} else if vb > vc {
				state = "saturation"
			}
			power := vb*ib + vc*ic
			powers += power
			r.Transistors = append(r.Transistors, MixedTransistor{TransistorID: p.ID, Model: TransistorModel(), BaseCurrent: ib, CollectorCurrent: ic, EmitterCurrent: ib + ic, VBE: vb, VCE: vc, State: state, TransistorPower: power})
		}
	}
	for k, p := range n.sources {
		v := n.sourceStimulus(p, t, at, dt)
		i := -x[len(n.nodes)+k]
		br := BranchResult{p.ID, i, v, v * i}
		r.Sources = append(r.Sources, br)
		sourcePower += br.Power
	}
	for k, p := range n.caps {
		v := voltage(x, n.pin(p, "1")) - voltage(x, n.pin(p, "2"))
		capV[k] = v
		i := 0.
		if dt == 0 || len(n.inductors) > 0 {
			i = x[n.capacitorIndex(k)]
			energyRate += v * i
		} else {
			i = p.Value * (v - old[k]) / dt
			energyRate += .5 * p.Value * (v*v - old[k]*old[k]) / dt
			r.NumericalDamping += .5 * p.Value * (v - old[k]) * (v - old[k]) / dt
		}
		r.Capacitors = append(r.Capacitors, CapacitorResult{p.ID, v, i, .5 * p.Value * v * v})
	}
	for k, p := range n.inductors {
		v := voltage(x, n.pin(p, "1")) - voltage(x, n.pin(p, "2"))
		i := x[n.inductorIndex(k)]
		energyRate += v * i
		r.Inductors = append(r.Inductors, InductorResult{ID: p.ID, Voltage: v, Current: i, Energy: .5 * p.Value * i * i})
	}
	_, f := n.mnaEquations(x, old, dt, t, at, len(n.inductors) == 0)
	for i := range n.nodes {
		r.KCLResidual = math.Max(r.KCLResidual, math.Abs(f[i]))
	}
	r.PowerResidual = sourcePower - powers - energyRate - r.NumericalDamping
	return &mixedState{n, x, capV, r}
}

// Compare one full step with two half steps and accept the latter. Legacy
// networks use backward Euler; L networks use Radau IIA collocation.
func (state *mixedState) advance(t *Transient, end float64) (*mixedState, error) {
	return state.advanceBudget(t, end, newIntegrationBudget())
}
func (state *mixedState) advanceBudget(t *Transient, end float64, budget *integrationBudget) (*mixedState, error) {
	if len(state.net.inductors) > 0 {
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
		full, e1 := cur.net.solve(cur.x, cur.capV, h, t, at+h)
		half, e2 := cur.net.solve(cur.x, cur.capV, h/2, t, at+h/2)
		var refined *mixedState
		var e3 error
		if e2 == nil {
			refined, e3 = cur.net.solve(half.x, half.capV, h/2, t, at+h)
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
				tol := absolute + relative*math.Max(math.Abs(v), math.Abs(cur.capV[i]))
				errRatio = math.Max(errRatio, math.Abs(v-full.capV[i])/tol)
				if cur.net.c.Analysis != MixedAnalysis && cur.net.c.Analysis != WaveformAnalysis {
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
			cur = refined
			at += h
			if end-at < 1e-13 {
				at = end
			}
			cur.result.Time = at
			if cur.net.c.Analysis == WaveformAnalysis {
				// Full-vs-two-half backward-Euler error scales with h².
				// Target three quarters of the existing acceptance bound instead of holding
				// a halved step until a binary doubling becomes admissible.
				// Every proposed next step is still independently checked.
				if errRatio == 0 {
					h *= 2
				} else {
					h *= math.Min(2, math.Sqrt(.75/errRatio))
				}
			} else if errRatio < .2 {
				h *= 2
			}
		} else {
			h /= 2
		}
	}
	return cur, nil
}
