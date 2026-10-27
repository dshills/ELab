package workbench

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"math/cmplx"
)

const MaxSpectrumSamples = 4096

type SpectrumParams struct {
	ID          string  `json:"id,omitempty"`
	ProbeID     string  `json:"probe_id,omitempty"`
	Start       float64 `json:"start_s,omitempty"`
	End         float64 `json:"end_s,omitempty"`
	Fundamental float64 `json:"fundamental_hz,omitempty"`
	Samples     int     `json:"samples,omitempty"`
	Harmonics   int     `json:"harmonics,omitempty"`
	Window      string  `json:"window,omitempty"`
}

// SpectrumInput is input-only so unit text never widens saved/output numeric fields.
type SpectrumInput SpectrumParams

type SpectrumBin struct {
	Frequency float64 `json:"frequency_hz"`
	Peak      float64 `json:"peak"`
	DB        float64 `json:"db_reference_1_unit"`
}
type SpectrumHarmonic struct {
	Order     int     `json:"order"`
	Frequency float64 `json:"frequency_hz"`
	Peak      float64 `json:"peak"`
}
type SpectrumResult struct {
	ID                 string             `json:"id"`
	ElectricalRevision uint64             `json:"electrical_revision"`
	DocumentGeneration uint64             `json:"document_generation"`
	Format             string             `json:"format"`
	Version            int                `json:"version"`
	SourceSHA256       string             `json:"recording_sha256"`
	Settings           SpectrumParams     `json:"settings"`
	Probe              Probe              `json:"probe"`
	SampleStep         float64            `json:"sample_step_s"`
	Nyquist            float64            `json:"nyquist_hz"`
	Resolution         float64            `json:"resolution_hz"`
	Mean               float64            `json:"mean"`
	Bins               []SpectrumBin      `json:"bins"`
	Harmonics          []SpectrumHarmonic `json:"harmonics"`
	THD                *float64           `json:"thd_percent,omitempty"`
	THDReason          string             `json:"thd_unavailable_reason,omitempty"`
	Assumptions        []string           `json:"assumptions"`
}
type SpectrumState struct {
	Available []Probe         `json:"available"`
	Settings  SpectrumParams  `json:"settings"`
	Result    *SpectrumResult `json:"result,omitempty"`
}

func cloneSpectrumResult(p *SpectrumResult) *SpectrumResult {
	if p == nil {
		return nil
	}
	v := *p
	v.Bins = append([]SpectrumBin{}, p.Bins...)
	v.Harmonics = append([]SpectrumHarmonic{}, p.Harmonics...)
	v.Assumptions = append([]string{}, p.Assumptions...)
	if p.THD != nil {
		n := *p.THD
		v.THD = &n
	}
	return &v
}
func cloneSpectrum(p *SpectrumState) *SpectrumState {
	if p == nil {
		return nil
	}
	v := *p
	v.Result = cloneSpectrumResult(p.Result)
	v.Available = append([]Probe{}, p.Available...)
	return &v
}
func (p SpectrumParams) validate() error {
	if !finite(p.Start) || !finite(p.End) || p.Start < 0 || p.End < 0 || (p.End != 0 && p.End <= p.Start) || !finite(p.Fundamental) || p.Fundamental < 0 || p.Fundamental > 1e6 {
		return fmt.Errorf("choose a finite nonnegative interval and fundamental frequency; end 0 uses the capture end")
	}
	if p.Samples != 0 && (p.Samples < 32 || p.Samples > MaxSpectrumSamples || p.Samples&(p.Samples-1) != 0) {
		return fmt.Errorf("spectrum samples must be 0 (automatic) or a power of two from 32 to 4096")
	}
	if p.Window != "" && p.Window != "rectangular" && p.Window != "hann" {
		return fmt.Errorf("window must be rectangular or hann")
	}
	if p.Harmonics != 0 && (p.Harmonics < 2 || p.Harmonics > 10) {
		return fmt.Errorf("choose 2–10 harmonics (0 uses five)")
	}
	return nil
}
func spectrumFFT(x []complex128) {
	n := len(x)
	j := 0
	for i := 1; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}
	for size := 2; size <= n; size *= 2 {
		root := cmplx.Rect(1, -2*math.Pi/float64(size))
		for lo := 0; lo < n; lo += size {
			w := complex(1., 0)
			for k := 0; k < size/2; k++ {
				a, b := x[lo+k], w*x[lo+k+size/2]
				x[lo+k], x[lo+k+size/2] = a+b, a-b
				w *= root
			}
		}
	}
}
func analyzeSpectrum(record *verificationRecord, p SpectrumParams) (*SpectrumResult, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	if record == nil || len(record.rows) < 33 {
		return nil, fmt.Errorf("spectrum needs a completed recording with at least 32 sample intervals")
	}
	index := -1
	var probe Probe
	for i, v := range record.metadata.Probes {
		if v.ID == p.ProbeID {
			index, probe = i, v
			break
		}
	}
	if index < 0 {
		return nil, fmt.Errorf("choose a probe stored in this recording")
	}
	if p.ID == "" {
		p.ID = record.metadata.OperationID
	}
	if p.End == 0 {
		p.End = record.rows[len(record.rows)-1].Time
	}
	if p.Window == "" {
		p.Window = "rectangular"
	}
	if p.Harmonics == 0 {
		p.Harmonics = 5
	}
	if p.Start < record.rows[0].Time || p.End > record.rows[len(record.rows)-1].Time || p.End <= p.Start {
		return nil, fmt.Errorf("spectrum interval must lie inside recorded time")
	}
	// Keep the accepted right side of duplicate edge timestamps. Do not invent
	// intermediate samples across a gap wider than the analysis grid.
	rows := make([]VerificationRow, 0, len(record.rows))
	for _, row := range record.rows {
		if len(row.Values) <= index || !finite(row.Time) || !finite(row.Values[index]) {
			return nil, fmt.Errorf("recording has invalid spectrum samples")
		}
		if len(rows) > 0 && row.Time < rows[len(rows)-1].Time {
			return nil, fmt.Errorf("recording times are not ordered")
		}
		if len(rows) > 0 && row.Time == rows[len(rows)-1].Time {
			rows[len(rows)-1] = row
		} else {
			rows = append(rows, row)
		}
	}
	if len(rows) < 33 {
		return nil, fmt.Errorf("spectrum needs at least 32 distinct captured intervals")
	}
	duration := p.End - p.Start
	maxGap := 0.
	for i := 1; i < len(rows); i++ {
		if rows[i].Time > p.Start && rows[i-1].Time < p.End {
			maxGap = math.Max(maxGap, rows[i].Time-rows[i-1].Time)
		}
	}
	if maxGap <= 0 {
		return nil, fmt.Errorf("recording has no distinct sample times")
	}
	n := p.Samples
	if n == 0 {
		n = 32
		for n < MaxSpectrumSamples && float64(n*2)*maxGap <= duration*(1+1e-10) {
			n *= 2
		}
	}
	dt := duration / float64(n)
	if dt <= 0 || !finite(dt) || !finite(1/(2*dt)) || !finite(1/duration) {
		return nil, fmt.Errorf("spectrum interval is outside representable frequency bounds")
	}
	if dt < maxGap*(1-1e-10) {
		return nil, fmt.Errorf("requested spectrum grid exceeds recorded sampling; use fewer samples or capture at a finer step")
	}
	p.Samples = n
	values := make([]float64, n)
	at := 1
	mean := 0.
	for i := range values {
		t := p.Start + float64(i)*dt
		for at < len(rows)-1 && rows[at].Time < t {
			at++
		}
		a, b := rows[at-1], rows[at]
		f := (t - a.Time) / (b.Time - a.Time)
		values[i] = a.Values[index] + f*(b.Values[index]-a.Values[index])
		mean += values[i] / float64(n)
	}
	x := make([]complex128, n)
	weight := 0.
	for i, v := range values {
		w := 1.
		if p.Window == "hann" {
			w = .5 - .5*math.Cos(2*math.Pi*float64(i)/float64(n))
		}
		weight += w
		x[i] = complex((v-mean)*w, 0)
	}
	spectrumFFT(x)
	raw, err := json.Marshal(VerificationDocument{Metadata: record.metadata, Rows: record.rows})
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(raw)
	r := &SpectrumResult{ElectricalRevision: record.metadata.ElectricalRevision, DocumentGeneration: record.metadata.DocumentGeneration, Format: "elab.spectrum", Version: 1, SourceSHA256: fmt.Sprintf("%x", hash), Settings: p, Probe: probe, SampleStep: dt, Nyquist: 1 / (2 * dt), Resolution: 1 / duration, Mean: mean, Bins: []SpectrumBin{}, Harmonics: []SpectrumHarmonic{}, Assumptions: []string{"Recorded data only; mean removed before FFT; no live circuit or time change", "Linear interpolation to a uniform end-exclusive grid; never finer than the widest captured interval; duplicate timestamps use the accepted right side", "Peak amplitudes corrected for window coherent gain; dB relative to one probe unit; no calibrated noise floor or THD+N", "THD needs a declared coherent fundamental, at least two complete cycles and all requested harmonics strictly below Nyquist; v1 physical model omissions remain"}}
	identity, _ := json.Marshal(struct {
		Source   string
		Settings SpectrumParams
	}{r.SourceSHA256, r.Settings})
	digest := sha256.Sum256(identity)
	r.ID = fmt.Sprintf("spectrum-%x", digest)
	for i := 0; i <= n/2; i++ {
		peak := 2 * cmplx.Abs(x[i]) / weight
		if i == 0 || i == n/2 {
			peak /= 2
		}
		db := -300.
		if peak > 1e-15 {
			db = 20 * math.Log10(peak)
		}
		r.Bins = append(r.Bins, SpectrumBin{float64(i) * r.Resolution, peak, db})
	}
	if p.Fundamental == 0 {
		r.THDReason = "Set the known fundamental frequency to measure THD"
		return r, nil
	}
	bin := p.Fundamental / r.Resolution
	k := int(math.Round(bin))
	if k < 2 || math.Abs(bin-float64(k)) > 1e-7 || k*p.Harmonics >= n/2 {
		r.THDReason = "Use an interval with at least two complete fundamental cycles, integer FFT bins and every harmonic below Nyquist"
		return r, nil
	}
	fundamental := r.Bins[k].Peak
	if fundamental < 1e-12 {
		r.THDReason = "Fundamental amplitude is too small for a reliable ratio"
		return r, nil
	}
	sum := 0.
	for h := 1; h <= p.Harmonics; h++ {
		v := r.Bins[h*k].Peak
		r.Harmonics = append(r.Harmonics, SpectrumHarmonic{h, float64(h) * p.Fundamental, v})
		if h > 1 {
			sum += v * v
		}
	}
	thd := 100 * math.Sqrt(sum) / fundamental
	r.THD = &thd
	return r, nil
}
func (s *Service) spectrumViewLocked() *SpectrumState {
	fresh := s.spectrum == nil
	v := cloneSpectrum(s.spectrum)
	if fresh {
		v = &SpectrumState{Settings: SpectrumParams{ID: s.selectedOperation, Window: "rectangular", Harmonics: 5}}
	}
	p := &v.Settings
	if p.ID == "" {
		p.ID = s.selectedOperation
	}
	if p.Window == "" {
		p.Window = "rectangular"
	}
	if p.Harmonics == 0 {
		p.Harmonics = 5
	}
	v.Available = []Probe{}
	if op := s.operations[p.ID]; op != nil && op.record != nil {
		r := op.record
		v.Available = append(v.Available, r.metadata.Probes...)
		if p.ProbeID == "" && len(v.Available) > 0 {
			p.ProbeID = v.Available[0].ID
		}
		if fresh && len(r.rows) > 0 {
			p.Start = r.rows[0].Time
			p.End = r.rows[len(r.rows)-1].Time
			for _, part := range r.metadata.Graph.Components {
				if part.Sine != nil {
					p.Fundamental = part.Sine.Frequency
					break
				}
			}
		}
	}
	return v
}
func (s *Service) spectrumCommandLocked(command string, raw json.RawMessage) error {
	p := s.spectrumViewLocked().Settings
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	if len(fields) > 0 {
		p = SpectrumParams{}
		if err := params(raw, &p); err != nil {
			return err
		}
	}
	if err := p.validate(); err != nil {
		return err
	}
	if p.ID == "" {
		p.ID = s.selectedOperation
	}
	if command == "spectrum.settings.set" {
		if s.spectrum != nil && s.spectrum.Settings == p {
			return nil
		}
		s.spectrum = &SpectrumState{Settings: p}
		return nil
	}
	op := s.operations[p.ID]
	if op == nil {
		return &actionError{"not-found", "Choose a retained completed recording"}
	}
	if op.record == nil {
		return &actionError{"not-ready", "Spectrum needs a completed recording"}
	}
	if p.ProbeID == "" && len(op.record.metadata.Probes) > 0 {
		p.ProbeID = op.record.metadata.Probes[0].ID
	}
	result, err := analyzeSpectrum(op.record, p)
	if err != nil {
		return err
	}
	s.spectrum = &SpectrumState{Settings: p, Result: result}
	return nil
}
func encodeSpectrumExport(r *SpectrumResult, format string) ([]byte, error) {
	if format != "json" && format != "csv" {
		return nil, fmt.Errorf("spectrum format must be json or csv")
	}
	if format == "json" {
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(data, '\n'), nil
	}
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	if err := w.Write([]string{"recording_sha256", "probe_id", "frequency_hz", "peak_" + r.Probe.Unit, "db_reference_1_unit"}); err != nil {
		return nil, err
	}
	for _, p := range r.Bins {
		if err := w.Write([]string{r.SourceSHA256, r.Probe.ID, number(p.Frequency), number(p.Peak), number(p.DB)}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return b.Bytes(), w.Error()
}
