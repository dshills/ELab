package workbench

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const MaxToleranceTrials = 64
const ToleranceEngineID = "elab.tolerance.v1"

type ToleranceSettings struct {
	Mode        string  `json:"mode,omitempty"`
	SourceID    string  `json:"source_id,omitempty"`
	ProbeID     string  `json:"probe_id,omitempty"`
	Frequency   float64 `json:"frequency_hz,omitempty"`
	Trials      int     `json:"trials,omitempty"`
	Seed        uint64  `json:"seed,omitempty"`
	Resistance  float64 `json:"resistance_fraction,omitempty"`
	Capacitance float64 `json:"capacitance_fraction,omitempty"`
	Beta        float64 `json:"beta_fraction,omitempty"`
}
type ToleranceVariation struct {
	ID        string  `json:"id"`
	Parameter string  `json:"parameter"`
	Nominal   float64 `json:"nominal"`
	Actual    float64 `json:"actual"`
}
type ToleranceTrial struct {
	Index      int                  `json:"index"`
	Parameters []ToleranceVariation `json:"parameters"`
	Value      *float64             `json:"value,omitempty"`
	Failure    string               `json:"failure,omitempty"`
}
type ToleranceStatistics struct {
	Successful        int     `json:"successful"`
	Failed            int     `json:"failed"`
	Minimum           float64 `json:"minimum"`
	Maximum           float64 `json:"maximum"`
	Mean              float64 `json:"mean"`
	StandardDeviation float64 `json:"population_standard_deviation"`
}
type ToleranceResult struct {
	ID                 string              `json:"id"`
	Format             string              `json:"format"`
	Version            int                 `json:"version"`
	Engine             string              `json:"engine"`
	DocumentGeneration uint64              `json:"document_generation"`
	ElectricalRevision uint64              `json:"electrical_revision"`
	SourceSHA256       string              `json:"source_sha256"`
	Source             DCCircuit           `json:"source"`
	Settings           ToleranceSettings   `json:"settings"`
	Probe              Probe               `json:"probe"`
	Trials             []ToleranceTrial    `json:"trials"`
	Statistics         ToleranceStatistics `json:"statistics"`
	Assumptions        []string            `json:"assumptions"`
}
type ToleranceState struct {
	Settings      ToleranceSettings `json:"settings"`
	ID            string            `json:"id,omitempty"`
	Status        string            `json:"status,omitempty"`
	Completed     int               `json:"completed"`
	SourceCurrent bool              `json:"source_current"`
	Failure       string            `json:"failure,omitempty"`
	Result        *ToleranceResult  `json:"result,omitempty"`
}
type toleranceJob struct {
	view                   ToleranceState
	cancel                 context.CancelFunc
	generation, electrical uint64
}

func defaultTolerance(g Graph) ToleranceSettings {
	p := ToleranceSettings{Mode: "ac-gain", Trials: 16, Seed: 1, Frequency: 1000, Resistance: .05, Capacitance: .1, Beta: .2}
	// Derive defaults without rebuilding the full probe inventory on every
	// live snapshot. These are the same first pin / C1 differential probes.
	for _, c := range g.Components {
		if p.SourceID == "" && IsVoltageSource(c.Kind) && c.Kind != "battery" {
			p.SourceID = c.ID
		}
		if p.ProbeID == "" && len(c.Terminals) > 0 {
			p.ProbeID = "v:" + c.ID + "." + c.Terminals[0]
		}
		if c.ID == "C1" && len(c.Terminals) == 2 {
			p.ProbeID = "v:C1"
		}
	}

	return p
}
func cloneToleranceResult(r *ToleranceResult) *ToleranceResult {
	if r == nil {
		return nil
	}
	c := *r
	c.Source = cloneSweepMetadata(SweepMetadata{Source: r.Source}).Source
	c.Assumptions = append([]string{}, r.Assumptions...)
	c.Trials = append([]ToleranceTrial{}, r.Trials...)
	for i, t := range c.Trials {
		c.Trials[i].Parameters = append([]ToleranceVariation{}, t.Parameters...)
		if t.Value != nil {
			v := *t.Value
			c.Trials[i].Value = &v
		}
	}
	return &c
}
func cloneTolerance(v *ToleranceState) *ToleranceState {
	if v == nil {
		return nil
	}
	c := *v
	c.Result = cloneToleranceResult(v.Result)
	return &c
}
func (s *Service) toleranceViewLocked() *ToleranceState {
	v := &ToleranceState{Settings: defaultTolerance(s.graph)}
	if s.toleranceSettings != nil {
		v.Settings = *s.toleranceSettings
	}
	if j := s.tolerance; j != nil {
		v = cloneTolerance(&j.view)
		v.SourceCurrent = j.generation == s.documentGeneration && j.electrical == s.electrical
	}
	return v
}
func validateTolerance(g Graph, p ToleranceSettings) (Probe, error) {
	if p.Mode != "dc" && p.Mode != "ac-gain" {
		return Probe{}, fmt.Errorf("mode must be dc or ac-gain")
	}
	if p.Trials < 2 || p.Trials > MaxToleranceTrials || p.Seed > 9007199254740991 {
		return Probe{}, fmt.Errorf("trials must be 2–64 and seed an exact integer 0–9007199254740991")
	}
	for _, f := range []float64{p.Resistance, p.Capacitance, p.Beta} {
		if !finite(f) || f < 0 || f > .5 {
			return Probe{}, fmt.Errorf("tolerance fractions must be 0–0.5")
		}
	}
	if !finite(p.Frequency) || p.Frequency < 0 || p.Frequency > 100000 || p.Mode == "ac-gain" && p.Frequency < .01 {
		return Probe{}, fmt.Errorf("AC frequency must be 0.01–100000 Hz; DC permits omitted/zero frequency")
	}
	var probe Probe
	for _, v := range probeInventory(g) {
		if v.ID == p.ProbeID {
			probe = v
			break
		}
	}
	if probe.ID == "" {
		return Probe{}, fmt.Errorf("choose an existing probe")
	}
	// Reuse analysis admission: history-bearing devices are never silently frozen.
	if p.Mode == "ac-gain" {
		_, err := newSweepPlan(g, SweepSettings{ComponentID: p.SourceID, Parameter: "frequency", Start: strconv.FormatFloat(p.Frequency, 'g', -1, 64), Stop: strconv.FormatFloat(p.Frequency*1.001, 'g', -1, 64), Points: 2, ProbeIDs: []string{p.ProbeID}})
		if err != nil {
			return Probe{}, err
		}
		probe.ID = "gain:" + probe.ID
		probe.Label += " gain"
		probe.Unit = "dB"
	} else {
		if controlModelForGraph(g) != nil || sequentialModelForGraph(g) != nil || timer555ModelForGraph(g) != nil {
			return Probe{}, fmt.Errorf("history-bearing circuits require transient analysis")
		}
		if _, _, d := g.Compile(); d != nil {
			return Probe{}, fmt.Errorf("%s", d.Message)
		}
	}
	return probe, nil
}

// SplitMix64: stable seed, component order and independent uniform [-t,+t].
func toleranceUnit(state *uint64) float64 {
	*state += 0x9e3779b97f4a7c15
	z := *state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	z ^= z >> 31
	return float64(z>>11) / 9007199254740992
}
func toleranceGraph(g Graph, p ToleranceSettings, state *uint64) (Graph, []ToleranceVariation) {
	out := g.Clone()
	values := []ToleranceVariation{}
	for i, c := range out.Components {
		amount, parameter, nominal := 0., "", 0.
		if c.Kind == "resistor" {
			amount, parameter, nominal = p.Resistance, "resistance_ohm", c.Value
		}
		if c.Kind == "capacitor" {
			amount, parameter, nominal = p.Capacitance, "capacitance_f", c.Value
		}
		if hasTransistorProfile(c) {
			amount, parameter, nominal = p.Beta, "forward_beta", c.Device.Bipolar.ForwardBeta
		}
		if amount == 0 || parameter == "" {
			continue
		}
		actual := nominal * (1 + amount*(2*toleranceUnit(state)-1))
		values = append(values, ToleranceVariation{c.ID, parameter, nominal, actual})
		if parameter == "forward_beta" {
			out.Components[i].Device.Bipolar.ForwardBeta = actual
		} else {
			out.Components[i].Value = actual
		}
	}
	return out, values
}
func computeTolerance(ctx context.Context, source DCCircuit, p ToleranceSettings, progress func(int)) (*ToleranceResult, error) {
	probe, err := validateTolerance(source.Graph, p)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(raw)
	r := &ToleranceResult{Format: "elab.tolerance", Version: 1, Engine: ToleranceEngineID, SourceSHA256: fmt.Sprintf("%x", hash), Source: source, Settings: p, Probe: probe, Trials: []ToleranceTrial{}, Assumptions: []string{"Independent uniform fractional variation of authored R/C and opted-in profile forward beta only; generic-v1 transistor parameters do not vary", "SplitMix64 top-53-bit uniform draws in captured component order; deterministic seed. Values outside device/admission bounds become explicit failed trials, never clamped or silently omitted", "DC opens capacitors; use AC gain at a declared frequency to observe capacitance tolerance. Each trial solves independent configured bias with existing model, accuracy and work guards", "Statistics include successful trials only, report failed count, and use population standard deviation. Sampled ranges are not worst-case guarantees or physical manufacturing distributions"}}
	state := p.Seed
	mean, m2 := 0., 0.
	for i := 0; i < p.Trials; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		g, variations := toleranceGraph(source.Graph, p, &state)
		trial := ToleranceTrial{Index: i + 1, Parameters: variations}
		value := 0.
		if p.Mode == "dc" {
			_, frame, e := solveDCOperatingPoint(ctx, g, dcSourceLevels(g))
			err = e
			if e == nil {
				var ok bool
				value, ok = frameValue(frame, probe)
				if !ok {
					err = fmt.Errorf("probe unavailable")
				}
			}
		} else {
			plan, e := newSweepPlan(g, SweepSettings{ComponentID: p.SourceID, Parameter: "frequency", Start: strconv.FormatFloat(p.Frequency, 'g', -1, 64), Stop: strconv.FormatFloat(p.Frequency*1.001, 'g', -1, 64), Points: 2, ProbeIDs: []string{p.ProbeID}})
			err = e
			if e == nil {
				var response *SweepResult
				response, err = computeACSweep(ctx, plan, func(int) {})
				if err == nil {
					if response == nil || len(response.Points) != 2 || len(response.Points[0].Values) != 2 {
						err = fmt.Errorf("AC trial did not produce the requested gain/phase pair")
					} else {
						value = response.Points[0].Values[0]
					}
				}
			}
		}
		if err != nil || !finite(value) {
			if err == nil {
				err = fmt.Errorf("nonfinite trial")
			}
			trial.Failure = err.Error()
			r.Statistics.Failed++
		} else {
			v := value
			trial.Value = &v
			r.Statistics.Successful++
			if r.Statistics.Successful == 1 {
				r.Statistics.Minimum, r.Statistics.Maximum = v, v
			}
			r.Statistics.Minimum = math.Min(r.Statistics.Minimum, v)
			r.Statistics.Maximum = math.Max(r.Statistics.Maximum, v)
			delta := v - mean
			mean += delta / float64(r.Statistics.Successful)
			m2 += delta * (v - mean)
		}
		r.Trials = append(r.Trials, trial)
		progress(i + 1)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.Statistics.Mean = mean
	if r.Statistics.Successful > 0 {
		r.Statistics.StandardDeviation = math.Sqrt(math.Max(0, m2/float64(r.Statistics.Successful)))
	}
	return r, nil
}
func (s *Service) toleranceCommandLocked(command string, raw json.RawMessage) error {
	if command == "tolerance.cancel" {
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		if s.tolerance == nil || s.tolerance.view.ID != p.ID {
			return &actionError{"not-found", "Choose the current tolerance run"}
		}
		if operationActive(s.tolerance.view.Status) {
			s.tolerance.cancel()
			s.tolerance.view.Status = "canceling"
		}
		return nil
	}
	p := s.toleranceViewLocked().Settings
	var fields map[string]json.RawMessage
	if len(raw) > 0 {
		if err := params(raw, &fields); err != nil {
			return err
		}
		if len(fields) > 0 {
			if value, ok := fields["trials"]; ok {
				var n float64
				if err := json.Unmarshal(value, &n); err != nil || !finite(n) || n != math.Trunc(n) || n < 2 || n > 64 {
					return fmt.Errorf("trials must be a whole number from 2 to 64")
				}
			}
			p = ToleranceSettings{}
			if err := params(raw, &p); err != nil {
				return err
			}
		}
	}
	if _, err := validateTolerance(s.graph, p); err != nil {
		return err
	}
	if s.toleranceWorkers > 0 {
		return &actionError{"busy", "Wait for the active tolerance worker to retire, or cancel it"}
	}
	if command == "tolerance.settings.set" && s.toleranceSettings != nil && *s.toleranceSettings == p {
		return nil
	}
	s.toleranceSettings = &p
	if command == "tolerance.settings.set" {
		s.tolerance = nil
		return nil
	}
	s.toleranceSerial++
	id := fmt.Sprintf("tolerance-%d", s.toleranceSerial)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	j := &toleranceJob{view: ToleranceState{Settings: p, ID: id, Status: "running"}, cancel: cancel, generation: s.documentGeneration, electrical: s.electrical}
	s.tolerance = j
	s.toleranceWorkers++
	source := dcCircuitSnapshot(s.graph, s.transientSettings)
	s.liveWorkers.Add(1)
	go func() {
		defer s.liveWorkers.Done()
		defer cancel()
		result, err := computeTolerance(ctx, source, p, func(done int) {
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.tolerance == j && !s.closed {
				j.view.Completed = done
				s.revision++
				s.signalLocked("jobs")
			}
		})
		s.mu.Lock()
		defer s.mu.Unlock()
		s.toleranceWorkers--
		if s.tolerance != j || s.closed {
			return
		}
		j.view.Status = "completed"
		if err != nil {
			j.view.Status = "failed"
			j.view.Failure = err.Error()
			if ctx.Err() == context.Canceled {
				j.view.Status = "canceled"
			}
		} else {
			result.ID = id
			result.DocumentGeneration = j.generation
			result.ElectricalRevision = j.electrical
			j.view.Result = result
		}
		s.revision++
		s.signalLocked("jobs")
	}()
	return nil
}
func encodeToleranceExport(r *ToleranceResult, format string) ([]byte, error) {
	if format == "json" {
		b, e := json.MarshalIndent(r, "", "  ")
		if e != nil {
			return nil, e
		}
		return append(b, '\n'), nil
	}
	if format != "csv" {
		return nil, fmt.Errorf("tolerance format must be json or csv")
	}
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	if e := w.Write([]string{"source_sha256", "trial", "probe", "unit", "value", "failure", "parameters_json"}); e != nil {
		return nil, e
	}
	for _, t := range r.Trials {
		v := ""
		if t.Value != nil {
			v = number(*t.Value)
		}
		params, e := json.Marshal(t.Parameters)
		if e != nil {
			return nil, e
		}
		if e = w.Write([]string{r.SourceSHA256, strconv.Itoa(t.Index), toleranceCSVText(r.Probe.ID), toleranceCSVText(r.Probe.Unit), v, toleranceCSVText(t.Failure), string(params)}); e != nil {
			return nil, e
		}
	}
	w.Flush()
	return b.Bytes(), w.Error()
}

func toleranceCSVText(s string) string {
	trim := strings.TrimLeft(s, " ")
	if len(trim) > 0 && strings.ContainsRune("=+-@\t\r\n", rune(trim[0])) {
		return "'" + s
	}
	return s
}
