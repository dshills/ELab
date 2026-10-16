package workbench

import (
	"context"
	"fmt"
	"math"
)

const MixedAnalysis = "RC/NPN transient"
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
	c             Compiled
	nodes         []string
	index         map[string]int
	sources, caps []Component
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
		if IsVoltageSource(p.Kind) {
			n.sources = append(n.sources, p)
		}
		if p.Kind == "capacitor" {
			n.caps = append(n.caps, p)
		}
	}
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
	size := len(n.nodes) + len(n.sources)
	if dt == 0 {
		size += len(n.caps)
	}
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
			i, g := diodeCurrent(voltage(x, u) - voltage(x, v))
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
	for k, p := range n.caps {
		u, v := n.pin(p, "1"), n.pin(p, "2")
		if dt == 0 {
			branch(u, v, len(n.nodes)+len(n.sources)+k, old[k])
		} else {
			g := p.Value / dt
			flow(u, v, g*(voltage(x, u)-voltage(x, v)-old[k]), g)
		}
	}
	return a, f
}
func linearSolve(a [][]float64, b []float64) ([]float64, error) {
	n := len(b)
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
		for row := col + 1; row < n; row++ {
			factor := a[row][col] / a[col][col]
			for j := col + 1; j < n; j++ {
				a[row][j] -= factor * a[col][j]
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
	result, err := n.newton(ctx, guess, old, dt, t, at)
	if err == nil {
		return result, nil
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
		seed := make([]float64, len(n.nodes)+len(n.sources)+len(n.caps))
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
	size := len(n.nodes) + len(n.sources)
	if dt == 0 {
		size += len(n.caps)
	}
	x := make([]float64, size)
	copy(x, guess)
	for iter := 0; iter < 100; iter++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		a, f := n.equations(x, old, dt, t, at)
		good := true
		for i, v := range f {
			tol := 1e-10
			if i >= len(n.nodes) {
				tol = 1e-8
			}
			if !finite(v) || math.Abs(v) > tol {
				good = false
			}
		}
		if good {
			// A supplied guess can already satisfy KCL/constraints. Apply the
			// same range guard before accepting it, not only after an update.
			for _, v := range x[:len(n.nodes)] {
				if !finite(v) || math.Abs(v) > 30 {
					return nil, fmt.Errorf("mixed node voltage exceeds the ±30 V educational model range")
				}
			}
			return n.result(x, old, dt, t, at), nil
		}
		for i := range f {
			f[i] = -f[i]
		}
		delta, err := linearSolve(a, f)
		if err != nil {
			return nil, err
		}
		scale := 1.
		for i := range n.nodes {
			if math.Abs(delta[i]) > .2 {
				scale = math.Min(scale, .2/math.Abs(delta[i]))
			}
		}
		for i := range x {
			x[i] += scale * delta[i]
		}
		for i := range n.nodes {
			if math.Abs(x[i]) > 30 {
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
				i, _ = diodeCurrent(v)
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
		if dt == 0 {
			i = x[len(n.nodes)+len(n.sources)+k]
			energyRate += v * i
		} else {
			i = p.Value * (v - old[k]) / dt
			energyRate += .5 * p.Value * (v*v - old[k]*old[k]) / dt
			r.NumericalDamping += .5 * p.Value * (v - old[k]) * (v - old[k]) / dt
		}
		r.Capacitors = append(r.Capacitors, CapacitorResult{p.ID, v, i, .5 * p.Value * v * v})
	}
	_, f := n.equations(x, old, dt, t, at)
	for i := range n.nodes {
		r.KCLResidual = math.Max(r.KCLResidual, math.Abs(f[i]))
	}
	r.PowerResidual = sourcePower - powers - energyRate - r.NumericalDamping
	return &mixedState{n, x, capV, r}
}

// Adaptive backward Euler: compare one full step with two half steps. Accept
// the latter, retaining stable integration through rapid transistor transitions.
func (state *mixedState) advance(t *Transient, end float64) (*mixedState, error) {
	return state.advanceBudget(t, end, newIntegrationBudget())
}
func (state *mixedState) advanceBudget(t *Transient, end float64, budget *integrationBudget) (*mixedState, error) {
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
		if h < 1e-10 {
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
