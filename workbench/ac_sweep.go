package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/cmplx"
)

const ACSweepEngineID = "elab.small-signal-mna.v1"
const acDBFloor = -600.

type ACResponse struct {
	Real         float64 `json:"real_v_per_v"`
	Imaginary    float64 `json:"imaginary_v_per_v"`
	PhaseDefined bool    `json:"phase_defined"`
}
type ACBand struct {
	ProbeID       string    `json:"probe_id"`
	PeakFrequency float64   `json:"sampled_peak_hz"`
	PeakDB        float64   `json:"sampled_peak_db"`
	HalfPowerHz   []float64 `json:"half_power_estimates_hz"`
}

func acSupported(g Graph) error {
	for _, p := range g.Components {
		switch p.Kind {
		case "current-source", "electronic-load", "speaker", "npn", "pnp", "nmos", "pmos", "diode", "schottky", "zener", "led", "transformer", "ground", "net-label", "junction", "resistor", "potentiometer", "capacitor", "inductor", "dc-source", "sine-source", "pulse-source", "finite-pulse-source", "opamp", "thermistor", "ldr", "switch":
		default:
			return fmt.Errorf("AC small-signal analysis does not yet support %s; use transient simulation", p.Kind)
		}
	}
	return nil
}
func acProbes(probes []Probe) []Probe {
	out := []Probe{}
	for _, p := range probes {
		out = append(out, Probe{ID: "gain:" + p.ID, Label: p.Label + " gain", Unit: "dB"}, Probe{ID: "phase:" + p.ID, Label: p.Label + " phase", Unit: "°"})
	}
	return out
}
func acFormat(parameter string) (string, string) {
	if parameter == "frequency" {
		return "elab.ac-sweep", ACSweepEngineID
	}
	return "elab.dc-sweep", DCSweepEngineID
}
func acAssumptions() []string {
	return []string{
		"Semiconductor differential conductances come from the solved DC operating point; generic v1 devices have no intrinsic charge/bandwidth and a Zener knee uses the blocking-side derivative",
		"Normalized 1 V small-signal excitation of the selected source around a solved DC bias; all other ideal voltage sources are AC grounded",
		"Capacitor admittance jωC; inductor impedance jωL; op-amp analytic Jacobian, including its compensation capacitor when optional dynamics are enabled; small-signal AC does not predict large-signal slew distortion",
		"Configured sine offset/pulse low/DC source levels establish bias; waveform amplitude, startup and live trajectory are ignored",
		"Gain is output differential voltage / selected source voltage; phase unwrap follows the requested point order. Below 1e-30 V/V gain is plotted at -600 dB. Zero transfer has undefined phase, displayed as zero with phase_defined=false",
		"Half-power estimates use logarithmic interpolation between sampled dB crossings relative to sampled peak, not a certified pole/resonance; expand the sweep for passband reference",
	}
}
func acValues(z complex128, previous float64, hasPrevious bool) (float64, float64, bool) {
	magnitude := cmplx.Abs(z)
	db := acDBFloor
	if magnitude > 1e-30 {
		db = 20 * math.Log10(magnitude)
	}
	phase := 0.
	defined := magnitude > 0
	if defined {
		phase = cmplx.Phase(z) * 180 / math.Pi
		if hasPrevious {
			phase += 360 * math.Round((previous-phase)/360)
		}
	}
	return db, phase, defined
}
func acBands(r *SweepResult) []ACBand {
	out := []ACBand{}
	for j := 0; j < len(r.Probes); j += 2 {
		peak := 0
		for i, p := range r.Points {
			if p.Values[j] > r.Points[peak].Values[j] {
				peak = i
			}
		}
		band := ACBand{ProbeID: r.Probes[j].ID, PeakFrequency: r.Points[peak].Input, PeakDB: r.Points[peak].Values[j], HalfPowerHz: []float64{}}
		level := band.PeakDB - 10*math.Log10(2)
		for i := 1; i < len(r.Points); i++ {
			a, b := r.Points[i-1], r.Points[i]
			y0, y1 := a.Values[j]-level, b.Values[j]-level
			if y0 == 0 {
				band.HalfPowerHz = append(band.HalfPowerHz, a.Input)
			}
			if y0*y1 < 0 {
				band.HalfPowerHz = append(band.HalfPowerHz, a.Input*math.Exp(math.Log(b.Input/a.Input)*(-y0)/(y1-y0)))
			}
			if i == len(r.Points)-1 && y1 == 0 {
				band.HalfPowerHz = append(band.HalfPowerHz, b.Input)
			}
		}
		out = append(out, band)
	}
	return out
}
func computeACSweep(ctx context.Context, plan sweepPlan, progress func(int)) (*SweepResult, error) {
	compiled, _, diagnostic := plan.graph.Compile()
	if diagnostic != nil {
		return nil, fmt.Errorf("%s", diagnostic.Message)
	}
	compiled.Analysis = NetworkDCAnalysis
	compiled.MixedGraph = plan.graph.Clone()
	n := newMixedNetwork(compiled)
	caps := n.caps
	n.caps = nil
	n.dc = true
	for i, p := range n.sources {
		for _, s := range plan.sources {
			if p.ID == s.ID {
				p.Kind = "dc-source"
				p.Value = s.DCLevel
				p.Sine = nil
				p.Pulse = nil
				n.sources[i] = p
			}
		}
	}
	t := &Transient{Mode: "auto", Settings: DefaultTransientSettings()}
	state, err := n.solveContext(ctx, nil, nil, 0, t, 0)
	if err != nil {
		return nil, fmt.Errorf("AC bias: %w", err)
	}
	jac, _ := n.equations(state.x, nil, 0, t, 0)
	if _, err := linearSolve(jac, make([]float64, len(jac))); err != nil {
		return nil, fmt.Errorf("AC bias is undetermined: %w", err)
	}
	// The real rank check eliminates in place; restore the physical Jacobian.
	jac, _ = n.equations(state.x, nil, 0, t, 0)
	sourceIndex := -1
	for k, p := range n.sources {
		if p.ID == plan.target.ComponentID {
			sourceIndex = len(n.nodes) + k
		}
	}
	if sourceIndex < 0 {
		return nil, fmt.Errorf("AC source not found")
	}
	out := &SweepResult{Target: plan.target, Spacing: plan.settings.Spacing, Start: plan.start, Stop: plan.stop, Probes: acProbes(plan.probes), Points: []SweepPoint{}, Statistics: []SweepStatistics{}}
	previous := make([]float64, len(plan.probes))
	hasPrevious := make([]bool, len(previous))
	for i := 0; i < plan.settings.Points; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		frequency := sweepAxisPoint(plan.start, plan.stop, plan.settings.Points, i, plan.settings.Spacing)
		w := 2 * math.Pi * frequency
		a := make([][]complex128, len(jac))
		rhs := make([]complex128, len(jac))
		rhs[sourceIndex] = 1
		for row := range a {
			a[row] = make([]complex128, len(jac))
			for col, v := range jac[row] {
				a[row][col] = complex(v, 0)
			}
		}
		add := func(row, col int, value complex128) {
			if row >= 0 && col >= 0 {
				a[row][col] += value
			}
		}
		for _, p := range caps {
			u, v := n.pin(p, "1"), n.pin(p, "2")
			y := passiveACAdmittance(p, w)
			add(u, u, y)
			add(v, v, y)
			add(u, v, -y)
			add(v, u, -y)
		}
		for k, p := range n.inductors {
			add(n.inductorIndex(k), n.inductorIndex(k), complex(0, -w*p.Value))
		}
		for _, c := range n.couplings {
			one, two := n.inductorIndex(c.first), n.inductorIndex(c.second)
			add(one, two, complex(0, -w*c.mutual))
			add(two, one, complex(0, -w*c.mutual))
		}
		x, err := complexSolve(ctx, a, rhs)
		if err != nil {
			return nil, fmt.Errorf("AC %.9g Hz: %w", frequency, err)
		}
		point := SweepPoint{Input: frequency, Values: []float64{}, AC: []ACResponse{}}
		// Report complex linear-system residual, in amperes for KCL rows. AC average
		// power is not a DC/transient conservation certificate; its legacy field is zero.
		for row := range a {
			res := complex(0, 0)
			scale := cmplx.Abs(rhs[row])
			for col, c := range a[row] {
				res += c * x[col]
				scale += cmplx.Abs(c * x[col])
			}
			residual := cmplx.Abs(res - rhs[row])
			if residual > 1e-10+1e-9*scale {
				return nil, fmt.Errorf("AC %.9g Hz linear residual too large", frequency)
			}
			if row < len(n.nodes) {
				point.KCLResidual = math.Max(point.KCLResidual, residual)
			}
		}

		for j, p := range plan.probes {
			node := func(ref string) int {
				if ref == "ground" {
					return -1
				}
				return acNode(n, compiled.TerminalNet[ref])
			}
			z := complexVoltage(x, node(p.Positive)) - complexVoltage(x, node(p.Reference))
			db, phase, defined := acValues(z, previous[j], hasPrevious[j])
			if defined {
				previous[j], hasPrevious[j] = phase, true
			}
			point.Values = append(point.Values, db, phase)
			point.AC = append(point.AC, ACResponse{real(z), imag(z), defined})
		}
		out.Points = append(out.Points, point)
		progress(i + 1)
	}
	for j, p := range out.Probes {
		s := SweepStatistics{p.ID, math.Inf(1), math.Inf(-1)}
		for _, point := range out.Points {
			s.Minimum = math.Min(s.Minimum, point.Values[j])
			s.Maximum = math.Max(s.Maximum, point.Values[j])
		}
		out.Statistics = append(out.Statistics, s)
	}
	out.AC = acBands(out)
	return out, ctx.Err()
}
func complexVoltage(x []complex128, index int) complex128 {
	if index < 0 {
		return 0
	}
	return x[index]
}

// Row-scaled partial pivoting keeps conductance and voltage constraints from
// competing on units. Work is bounded by the existing compiled network size.
func complexSolve(ctx context.Context, a [][]complex128, b []complex128) ([]complex128, error) {
	n := len(b)
	m := make([][]complex128, n)
	for i := range m {
		m[i] = append(append([]complex128{}, a[i]...), b[i])
		scale := 0.
		for _, v := range a[i] {
			scale = math.Max(scale, cmplx.Abs(v))
		}
		if scale == 0 {
			return nil, fmt.Errorf("singular complex system")
		}
		for j := range m[i] {
			m[i][j] /= complex(scale, 0)
		}
	}
	for k := 0; k < n; k++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pivot := k
		for i := k + 1; i < n; i++ {
			if cmplx.Abs(m[i][k]) > cmplx.Abs(m[pivot][k]) {
				pivot = i
			}
		}
		if cmplx.Abs(m[pivot][k]) < 1e-15 {
			return nil, fmt.Errorf("singular complex system")
		}
		m[k], m[pivot] = m[pivot], m[k]
		for i := k + 1; i < n; i++ {
			q := m[i][k] / m[k][k]
			for j := k + 1; j <= n; j++ {
				m[i][j] -= q * m[k][j]
			}
			m[i][k] = 0
		}
	}
	x := make([]complex128, n)
	for i := n - 1; i >= 0; i-- {
		v := m[i][n]
		for j := i + 1; j < n; j++ {
			v -= m[i][j] * x[j]
		}
		x[i] = v / m[i][i]
		if !finite(real(x[i])) || !finite(imag(x[i])) {
			return nil, fmt.Errorf("nonfinite AC solution")
		}
	}
	return x, nil
}

func acNode(n *mixedNetwork, id string) int {
	if id == n.c.Ground {
		return -1
	}
	if index, ok := n.index[id]; ok {
		return index
	}
	return -1
}

func validateACRecord(plan sweepPlan, r *SweepResult) error {
	if plan.settings.Parameter != "frequency" {
		if len(r.AC) > 0 {
			return fmt.Errorf("DC record contains AC measurements")
		}
		for _, p := range r.Points {
			if len(p.AC) > 0 {
				return fmt.Errorf("DC point contains AC response")
			}
		}
		return nil
	}
	previous := make([]float64, len(plan.probes))
	hasPrevious := make([]bool, len(previous))
	for _, p := range r.Points {
		if len(p.AC) != len(plan.probes) {
			return fmt.Errorf("AC response count differs from outputs")
		}
		for j, z := range p.AC {
			if !finite(z.Real) || !finite(z.Imaginary) {
				return fmt.Errorf("nonfinite AC response")
			}
			db, phase, defined := acValues(complex(z.Real, z.Imaginary), previous[j], hasPrevious[j])
			if defined {
				previous[j], hasPrevious[j] = phase, true
			}
			if p.Values[2*j] != db || p.Values[2*j+1] != phase || z.PhaseDefined != defined {
				return fmt.Errorf("AC response disagrees with gain/phase")
			}
		}
	}
	if !reflectACBands(r.AC, acBands(r)) {
		return fmt.Errorf("AC band estimates differ from captured points")
	}
	return nil
}
func reflectACBands(a, b []ACBand) bool {
	if len(a) != len(b) {
		return false
	}
	for i, x := range a {
		y := b[i]
		if x.ProbeID != y.ProbeID || x.PeakFrequency != y.PeakFrequency || x.PeakDB != y.PeakDB || len(x.HalfPowerHz) != len(y.HalfPowerHz) {
			return false
		}
		for j, f := range x.HalfPowerHz {
			if f != y.HalfPowerHz[j] {
				return false
			}
		}
	}
	return true
}

func validateACFields(raw json.RawMessage, points []json.RawMessage) error {
	var r struct {
		Bands []json.RawMessage `json:"ac_bands"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return err
	}
	if len(r.Bands) == 0 {
		return fmt.Errorf("AC bands required")
	}
	for _, b := range r.Bands {
		if err := params(b, &ACBand{}, "probe_id", "sampled_peak_hz", "sampled_peak_db", "half_power_estimates_hz"); err != nil {
			return err
		}
	}
	for _, p := range points {
		var point struct {
			AC []json.RawMessage `json:"ac_response"`
		}
		if err := json.Unmarshal(p, &point); err != nil {
			return err
		}
		if len(point.AC) == 0 {
			return fmt.Errorf("AC complex responses required")
		}
		for _, v := range point.AC {
			if err := params(v, &ACResponse{}, "real_v_per_v", "imaginary_v_per_v", "phase_defined"); err != nil {
				return err
			}
		}
	}
	return nil
}
