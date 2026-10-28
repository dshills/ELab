package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/cmplx"
	"reflect"
)

const NoiseEngineID = "elab.small-signal-white-noise.v1"
const noiseBoltzmann = 1.380649e-23
const noiseCharge = 1.602176634e-19

type NoiseContributor struct {
	ID          string    `json:"id"`
	ComponentID string    `json:"component_id"`
	Mechanism   string    `json:"mechanism"`
	Density     float64   `json:"source_density"`
	Unit        string    `json:"source_unit"`
	RMS         []float64 `json:"output_rms_v"`
}
type NoisePoint struct {
	LinearResidual float64      `json:"adjoint_residual"`
	PSD            [][]float64  `json:"contribution_psd_v2_per_hz"`
	Gain           []ACResponse `json:"input_transfer"`
	InputDensity   []float64    `json:"input_density_v_per_sqrt_hz"`
	InputDefined   []bool       `json:"input_defined"`
}
type NoiseSummary struct {
	Model        string             `json:"model"`
	Bias         *MixedResult       `json:"bias"`
	Contributors []NoiseContributor `json:"contributors"`
	OutputRMS    []float64          `json:"output_rms_v"`
	InputRMS     []float64          `json:"input_rms_v"`
	InputDefined []bool             `json:"input_defined"`
}
type noiseDrive struct {
	info   NoiseContributor
	vector []float64
}

func IsNoiseSweep(parameter string) bool { return parameter == "noise" }
func IsSpectralSweep(parameter string) bool {
	return IsFrequencySweep(parameter) || IsNoiseSweep(parameter)
}
func noiseAssumptions() []string {
	return []string{
		NoiseEngineID + ": stationary, uncorrelated one-sided white sources about a solved DC operating point; no live random waveform, flicker, avalanche, excess noise, correlations or THD+N",
		"Resistor Johnson current PSD 4*k*T/R (T=300.15 K unless its temperature model is enabled); diode terminal shot PSD 2*q*abs(I); BJT independent base/emitter and collector/emitter shot PSD 2*q*abs(IB/IC), approximate in saturation/reverse operation",
		"JFET/MOS channel PSD (8/3)*k*300.15*abs(gm); gate noise and parasitic bleeds omitted. Op-amp illustrative white input voltage 10 nV/sqrt(Hz) and independent input currents 1 pA/sqrt(Hz) referred to V-; no device-fit claim",
		"Only declared noise mechanisms count; ideal capacitors/inductors/sources/connectivity are noiseless. Lossy storage, speakers, transformers, switches/sensors, current devices and Zener avalanche are refused rather than silently omitted",
		"Selected ideal voltage source defines input transfer; other ideal voltage sources AC grounded. Input-referred density is unavailable where gain is below 1e-30 V/V",
		"Output PSD is the sum of independent transferred source powers. RMS is sqrt(trapezoidal integral of linear PSD over explicit ascending frequency samples); refine the grid/band to check quadrature. Input RMS is unavailable if any sampled gain is undefined",
	}
}
func noiseSupported(g Graph) error {
	physical, err := g.PhysicalGraph()
	if err != nil {
		return err
	}
	for _, p := range physical.Components {
		if hasPassiveLoss(p) {
			return fmt.Errorf("noise analysis does not yet model lossy storage; disable losses or use explicit resistors")
		}
		switch p.Kind {
		case "ground", "net-label", "junction", "dc-source", "pulse-source", "sine-source", "finite-pulse-source", "resistor", "capacitor", "inductor", "diode", "schottky", "led", "npn", "pnp", "njfet", "pjfet", "nmos", "pmos", "opamp":
		default:
			return fmt.Errorf("noise analysis does not yet declare noise for %s", p.Kind)
		}
	}
	return nil
}
func noiseProbes(in []Probe) []Probe {
	out := make([]Probe, len(in))
	for k, p := range in {
		out[k] = Probe{ID: "noise:" + p.ID, Label: p.Label + " noise density", Unit: "V/√Hz"}
	}
	return out
}
func (n *mixedNetwork) noiseDrives(x []float64) []noiseDrive {
	out := []noiseDrive{}
	add := func(p Component, mechanism string, psd float64, unit string, vector []float64) {
		out = append(out, noiseDrive{NoiseContributor{ID: p.ID + ":" + mechanism, ComponentID: p.ID, Mechanism: mechanism, Density: math.Sqrt(psd), Unit: unit, RMS: []float64{}}, vector})
	}
	pair := func(p Component, mechanism, a, b string, psd float64) {
		v := make([]float64, len(x))
		u, w := n.pin(p, a), n.pin(p, b)
		if u >= 0 {
			v[u] -= 1
		}
		if w >= 0 {
			v[w] += 1
		}
		add(p, mechanism, psd, "A/√Hz", v)
	}
	graph := n.authored
	if len(graph.Components) == 0 {
		graph = n.c.MixedGraph
	}
	for _, p := range graph.Components {
		temp := 300.15
		if hasThermal(p) {
			temp = n.thermalTemperature(p, x) + 273.15
		}
		switch p.Kind {
		case "resistor":
			r := p.Value
			if hasThermal(p) {
				r = n.evaluateThermal(p, x).resistance
			}
			pair(p, "johnson", "1", "2", 4*noiseBoltzmann*temp/r)
		case "diode", "schottky", "led":
			v := voltage(x, n.pin(p, "anode")) - voltage(x, n.pin(p, "cathode"))
			i := 0.
			if p.Kind == "led" {
				i, _ = n.ledCurrent(v)
			} else {
				i, _ = deviceDiodeCurrent(p, v)
			}
			if hasThermal(p) {
				i = n.evaluateThermal(p, x).currents[0]
			}
			pair(p, "terminal-shot", "anode", "cathode", 2*noiseCharge*math.Abs(i))
		case "npn", "pnp":
			q := n.bipolarResult(p, x)
			pair(p, "base-shot", "base", "emitter", 2*noiseCharge*math.Abs(q.BaseCurrent-q.JunctionBECurrent-q.JunctionBCCurrent))
			pair(p, "collector-shot", "collector", "emitter", 2*noiseCharge*math.Abs(q.CollectorCurrent+q.JunctionBCCurrent))
		case "njfet", "pjfet":
			_, gm, _ := jfetChannel(p, voltage(x, n.pin(p, "gate"))-voltage(x, n.pin(p, "source")), voltage(x, n.pin(p, "drain"))-voltage(x, n.pin(p, "source")))
			pair(p, "channel-white", "drain", "source", 8./3*noiseBoltzmann*temp*math.Abs(gm))
		case "nmos", "pmos":
			_, gm, _ := mosfetLinear(p.Kind, voltage(x, n.pin(p, "gate"))-voltage(x, n.pin(p, "source")), voltage(x, n.pin(p, "drain"))-voltage(x, n.pin(p, "source")))
			pair(p, "channel-white", "drain", "source", 8./3*noiseBoltzmann*temp*math.Abs(gm))
		case "opamp":
			pins := n.opAmpPins(p)
			v := n.opAmpVoltages(p, x)
			drive := make([]float64, len(x))
			inject := func(row int, d float64) {
				if row >= 0 {
					drive[row] -= d
				}
			}
			if hasDynamics(p) {
				pole := n.index[poleNode(p.ID)]
				pv := voltage(x, pole)
				e := opAmpCurrentsControlled(v, pv, true)
				for k, row := range pins {
					inject(row, e.jacobian[k][0])
				}
				_, d, rail := poleCurrent(p, v, pv)
				inject(pins[rail], d[0])
				inject(pole, -d[0])
			} else {
				e := opAmpCurrents(v)
				for k, row := range pins {
					inject(row, e.jacobian[k][0])
				}
			}
			add(p, "input-voltage-white", 1e-16, "V/√Hz", drive)
			pair(p, "noninverting-current-white", "noninverting", "vminus", 1e-24)
			pair(p, "inverting-current-white", "inverting", "vminus", 1e-24)
		}
	}
	return out
}
func smallSignalMatrix(n *mixedNetwork, jac [][]float64, caps []Component, frequency float64) [][]complex128 {
	a := make([][]complex128, len(jac))
	for k, row := range jac {
		a[k] = make([]complex128, len(row))
		for j, v := range row {
			a[k][j] = complex(v, 0)
		}
	}
	add := func(r, c int, v complex128) {
		if r >= 0 && c >= 0 {
			a[r][c] += v
		}
	}
	w := 2 * math.Pi * frequency
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
		a, b := n.inductorIndex(c.first), n.inductorIndex(c.second)
		add(a, b, complex(0, -w*c.mutual))
		add(b, a, complex(0, -w*c.mutual))
	}
	return a
}
func computeNoiseSweep(ctx context.Context, plan sweepPlan, progress func(int)) (*SweepResult, error) {
	physical, err := plan.graph.PhysicalGraph()
	if err != nil {
		return nil, err
	}
	compiled, _, d := physical.Compile()
	compiled.Analysis = NetworkDCAnalysis
	compiled.MixedGraph = physical
	for _, p := range physical.Components {
		if p.Kind == "ground" {
			compiled.Ground = compiled.TerminalNet[p.ID+".reference"]
			break
		}
	}
	if d != nil {
		return nil, fmt.Errorf("%s", d.Message)
	}
	bias, _, err := solveDCOperatingPoint(ctx, physical, plan.sources)
	if err != nil {
		return nil, fmt.Errorf("noise bias: %w", err)
	}
	n, x, err := recordedDeviceNodes(compiled, bias)
	if err != nil {
		return nil, err
	}
	caps := n.caps
	n.caps = nil
	n.dc = true
	x = x[:n.matrixSize(0)]
	for k, p := range n.sources {
		for _, level := range plan.sources {
			if level.ID == p.ID {
				p.Kind = "dc-source"
				p.Value = level.DCLevel
				p.Sine = nil
				p.Pulse = nil
				n.sources[k] = p
			}
		}
	}
	tr := &Transient{Mode: "auto", Settings: DefaultTransientSettings()}
	jac, _ := n.equations(x, nil, 0, tr, 0)
	if _, err := linearSolve(jac, make([]float64, len(jac))); err != nil {
		return nil, fmt.Errorf("noise bias undetermined: %w", err)
	}
	jac, _ = n.equations(x, nil, 0, tr, 0)
	drives := n.noiseDrives(x)
	if len(drives) == 0 {
		return nil, fmt.Errorf("add a supported noisy resistor or semiconductor")
	}
	summary := &NoiseSummary{Model: NoiseEngineID, Bias: cloneMixed(bias), Contributors: []NoiseContributor{}}
	for _, s := range drives {
		summary.Contributors = append(summary.Contributors, s.info)
	}
	out := &SweepResult{Noise: summary, Target: plan.target, Spacing: plan.settings.Spacing, Start: plan.start, Stop: plan.stop, Probes: noiseProbes(plan.probes), Points: []SweepPoint{}, Statistics: []SweepStatistics{}}
	sourceIndex := -1
	for k, p := range n.sources {
		if p.ID == plan.target.ComponentID {
			sourceIndex = len(n.nodes) + k
		}
	}
	if sourceIndex < 0 {
		return nil, fmt.Errorf("noise input source missing")
	}
	for k := 0; k < plan.settings.Points; k++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		f := sweepAxisPoint(plan.start, plan.stop, plan.settings.Points, k, plan.settings.Spacing)
		a := smallSignalMatrix(n, jac, caps, f)
		transpose := make([][]complex128, len(a))
		for i := range a {
			transpose[i] = make([]complex128, len(a))
			for j := range a {
				transpose[i][j] = a[j][i]
			}
		}
		np := &NoisePoint{PSD: make([][]float64, len(drives)), Gain: []ACResponse{}, InputDensity: []float64{}, InputDefined: []bool{}}
		for i := range np.PSD {
			np.PSD[i] = make([]float64, len(plan.probes))
		}
		point := SweepPoint{Input: f, Values: []float64{}, Noise: np}
		for j, p := range plan.probes {
			rhs := make([]complex128, len(a))
			node := func(ref string) int {
				if ref == "ground" {
					return -1
				}
				return acNode(n, compiled.TerminalNet[ref])
			}
			if u := node(p.Positive); u >= 0 {
				rhs[u] += 1
			}
			if v := node(p.Reference); v >= 0 {
				rhs[v] -= 1
			}
			adjoint, err := complexSolve(ctx, transpose, rhs)
			if err != nil {
				return nil, err
			}
			// A^T z = output selector, without complex conjugation: z^T b is transfer.
			for row := range transpose {
				res := complex(0, 0)
				scale := cmplx.Abs(rhs[row])
				for col, v := range transpose[row] {
					res += v * adjoint[col]
					scale += cmplx.Abs(v * adjoint[col])
				}
				res -= rhs[row]
				if cmplx.Abs(res) > 1e-10+1e-9*scale {
					return nil, fmt.Errorf("noise adjoint residual too large")
				}
				np.LinearResidual = math.Max(np.LinearResidual, cmplx.Abs(res))
			}
			total := 0.
			for i, s := range drives {
				z := complex(0, 0)
				for col, v := range s.vector {
					z += adjoint[col] * complex(v, 0)
				}
				psd := cmplx.Abs(z) * cmplx.Abs(z) * s.info.Density * s.info.Density
				if !finite(psd) {
					return nil, fmt.Errorf("nonfinite noise contribution")
				}
				np.PSD[i][j] = psd
				total += psd
			}
			density := math.Sqrt(total)
			if !finite(density) {
				return nil, fmt.Errorf("nonfinite total noise density")
			}
			point.Values = append(point.Values, density)
			gain := adjoint[sourceIndex]
			defined := cmplx.Abs(gain) > 1e-30
			np.Gain = append(np.Gain, ACResponse{Real: real(gain), Imaginary: imag(gain), PhaseDefined: defined})
			input := 0.
			if defined {
				input = density / cmplx.Abs(gain)
			}
			if !finite(input) {
				return nil, fmt.Errorf("nonfinite input-referred density")
			}
			np.InputDensity = append(np.InputDensity, input)
			np.InputDefined = append(np.InputDefined, defined)
		}
		out.Points = append(out.Points, point)
		progress(k + 1)
	}
	summarizeNoise(out)
	return out, ctx.Err()
}
func summarizeNoise(r *SweepResult) {
	n := r.Noise
	count := len(r.Probes)
	n.OutputRMS = make([]float64, count)
	n.InputRMS = make([]float64, count)
	n.InputDefined = make([]bool, count)
	r.Statistics = []SweepStatistics{}
	for j, p := range r.Probes {
		min, max := math.Inf(1), math.Inf(-1)
		n.InputDefined[j] = true
		for _, point := range r.Points {
			min = math.Min(min, point.Values[j])
			max = math.Max(max, point.Values[j])
			n.InputDefined[j] = n.InputDefined[j] && point.Noise.InputDefined[j]
		}
		r.Statistics = append(r.Statistics, SweepStatistics{p.ID, min, max})
		for i := 1; i < len(r.Points); i++ {
			a, b := r.Points[i-1], r.Points[i]
			df := b.Input - a.Input
			n.OutputRMS[j] += df * (a.Values[j]*a.Values[j] + b.Values[j]*b.Values[j]) / 2
			n.InputRMS[j] += df * (a.Noise.InputDensity[j]*a.Noise.InputDensity[j] + b.Noise.InputDensity[j]*b.Noise.InputDensity[j]) / 2
		}
		n.OutputRMS[j] = math.Sqrt(n.OutputRMS[j])
		if n.InputDefined[j] {
			n.InputRMS[j] = math.Sqrt(n.InputRMS[j])
		} else {
			n.InputRMS[j] = 0
		}
	}
	for k := range n.Contributors {
		n.Contributors[k].RMS = make([]float64, count)
		for j := 0; j < count; j++ {
			area := 0.
			for i := 1; i < len(r.Points); i++ {
				a, b := r.Points[i-1], r.Points[i]
				area += (b.Input - a.Input) * (a.Noise.PSD[k][j] + b.Noise.PSD[k][j]) / 2
			}
			n.Contributors[k].RMS[j] = math.Sqrt(area)
		}
	}
}
func cloneNoisePoint(p *NoisePoint) *NoisePoint {
	if p == nil {
		return nil
	}
	q := *p
	q.Gain = append([]ACResponse{}, p.Gain...)
	q.InputDensity = append([]float64{}, p.InputDensity...)
	q.InputDefined = append([]bool{}, p.InputDefined...)
	q.PSD = make([][]float64, len(p.PSD))
	for k, v := range p.PSD {
		q.PSD[k] = append([]float64{}, v...)
	}
	return &q
}
func cloneNoiseSummary(n *NoiseSummary) *NoiseSummary {
	if n == nil {
		return nil
	}
	out := *n
	out.Bias = cloneMixed(n.Bias)
	out.OutputRMS = append([]float64{}, n.OutputRMS...)
	out.InputRMS = append([]float64{}, n.InputRMS...)
	out.InputDefined = append([]bool{}, n.InputDefined...)
	out.Contributors = append([]NoiseContributor{}, n.Contributors...)
	for k, v := range n.Contributors {
		out.Contributors[k].RMS = append([]float64{}, v.RMS...)
	}
	return &out
}
func validateNoiseRecord(plan sweepPlan, r *SweepResult) error {
	if !IsNoiseSweep(plan.settings.Parameter) {
		if r.Noise != nil {
			return fmt.Errorf("unrelated noise summary")
		}
		for _, p := range r.Points {
			if p.Noise != nil {
				return fmt.Errorf("unrelated noise point")
			}
		}
		return nil
	}
	if err := noiseResultShape(r); err != nil {
		return err
	}
	if r.Noise == nil || r.Noise.Model != NoiseEngineID || r.Noise.Bias == nil {
		return fmt.Errorf("missing noise model/bias")
	}
	physical, err := plan.graph.PhysicalGraph()
	if err != nil {
		return err
	}
	c, _, d := physical.Compile()
	c.Analysis = NetworkDCAnalysis
	c.MixedGraph = physical
	for _, p := range physical.Components {
		if p.Kind == "ground" {
			c.Ground = c.TerminalNet[p.ID+".reference"]
			break
		}
	}
	if d != nil {
		return fmt.Errorf("noise graph: %s", d.Message)
	}
	if err := validateRecordedResults(VerificationMetadata{Analysis: NetworkDCAnalysis, Engine: DCSweepEngineID, Graph: physical, Mixed: r.Noise.Bias}, c); err != nil {
		return err
	}
	n, x, err := recordedDeviceNodes(c, r.Noise.Bias)
	if err != nil {
		return err
	}
	n.caps = nil
	n.dc = true
	x = x[:n.matrixSize(0)]
	drives := n.noiseDrives(x)
	if len(drives) != len(r.Noise.Contributors) || len(drives) == 0 {
		return fmt.Errorf("noise contributor inventory differs")
	}
	for k, s := range drives {
		a := r.Noise.Contributors[k]
		if a.ID != s.info.ID || a.ComponentID != s.info.ComponentID || a.Mechanism != s.info.Mechanism || a.Density != s.info.Density || a.Unit != s.info.Unit {
			return fmt.Errorf("noise source law differs from captured bias")
		}
	}
	for _, p := range r.Points {
		q := p.Noise
		if q == nil || !finite(q.LinearResidual) || q.LinearResidual < 0 || len(q.PSD) != len(drives) || len(q.Gain) != len(plan.probes) || len(q.InputDensity) != len(plan.probes) || len(q.InputDefined) != len(plan.probes) {
			return fmt.Errorf("noise point shape differs")
		}
		for _, v := range q.PSD {
			if len(v) != len(plan.probes) {
				return fmt.Errorf("noise contribution width")
			}
		}
		for j, density := range p.Values {
			sum := 0.
			for _, v := range q.PSD {
				if !finite(v[j]) || v[j] < 0 {
					return fmt.Errorf("invalid noise PSD")
				}
				sum += v[j]
			}
			if density != math.Sqrt(sum) {
				return fmt.Errorf("noise total differs from contributions")
			}
			g := q.Gain[j]
			gain := cmplx.Abs(complex(g.Real, g.Imaginary))
			defined := gain > 1e-30
			if !finite(gain) || g.PhaseDefined != defined || q.InputDefined[j] != defined {
				return fmt.Errorf("invalid noise input transfer")
			}
			want := 0.
			if defined {
				want = density / gain
			}
			if !finite(want) || q.InputDensity[j] != want {
				return fmt.Errorf("input-referred density differs")
			}
		}
	}
	expected := cloneSweepResult(r)
	summarizeNoise(expected)
	if !reflect.DeepEqual(expected.Noise.OutputRMS, r.Noise.OutputRMS) || !reflect.DeepEqual(expected.Noise.InputRMS, r.Noise.InputRMS) || !reflect.DeepEqual(expected.Noise.InputDefined, r.Noise.InputDefined) || !reflect.DeepEqual(expected.Noise.Contributors, r.Noise.Contributors) {
		return fmt.Errorf("noise integrals differ from samples")
	}
	return nil
}
func validateNoiseFields(raw json.RawMessage, points []json.RawMessage) error {
	var obj struct {
		Noise json.RawMessage `json:"noise"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return err
	}
	if len(obj.Noise) == 0 {
		return fmt.Errorf("missing noise summary fields")
	}
	if err := recordingFields(obj.Noise, reflect.TypeOf(NoiseSummary{})); err != nil {
		return err
	}
	for _, p := range points {
		var obj struct {
			Noise json.RawMessage `json:"noise"`
		}
		if err := json.Unmarshal(p, &obj); err != nil {
			return err
		}
		if len(obj.Noise) == 0 {
			return fmt.Errorf("missing noise point fields")
		}
		if err := recordingFields(obj.Noise, reflect.TypeOf(NoisePoint{})); err != nil {
			return err
		}
	}
	return nil
}

// Guard every consumer, including standalone export/render helpers. Imports also
// validate model/source law and numerical relationships after this shape check.
func noiseResultShape(r *SweepResult) error {
	if r == nil || r.Noise == nil {
		return fmt.Errorf("noise summary unavailable")
	}
	count := len(r.Probes)
	n := r.Noise
	if count < 1 || count > 3 || len(n.OutputRMS) != count || len(n.InputRMS) != count || len(n.InputDefined) != count {
		return fmt.Errorf("noise summary shape differs from outputs")
	}
	for _, c := range n.Contributors {
		if len(c.RMS) != count {
			return fmt.Errorf("noise contributor RMS shape differs")
		}
	}
	for _, p := range r.Points {
		q := p.Noise
		if len(p.Values) != count || q == nil || len(q.PSD) != len(n.Contributors) || len(q.Gain) != count || len(q.InputDensity) != count || len(q.InputDefined) != count {
			return fmt.Errorf("noise point shape differs from outputs")
		}
		for _, v := range q.PSD {
			if len(v) != count {
				return fmt.Errorf("noise contribution PSD shape differs")
			}
		}
	}
	return nil
}

// NoiseSummaryValid is a read-only defensive check for presentation consumers.
func NoiseSummaryValid(r *SweepResult) bool { return noiseResultShape(r) == nil }
