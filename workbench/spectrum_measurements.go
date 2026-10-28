package workbench

import (
	"encoding/json"
	"fmt"
	"sort"
)

const MaxSpectrumPeaks = 5

const spectrumPeakDescription = "At most five non-DC local maxima above the display floor and within 60 dB of the strongest; plateau/ties choose lower frequency; labels are FFT bins, not interpolated estimates"

type SpectrumCursorParams struct {
	Source    string   `json:"source"`
	Cursor    string   `json:"cursor"`
	Frequency *float64 `json:"frequency_hz,omitempty"`
	Peak      int      `json:"peak,omitempty"`
	Clear     bool     `json:"clear,omitempty"`
	Select    bool     `json:"select,omitempty"`
	ResultID  string   `json:"result_id,omitempty"`
}
type SpectrumReading struct {
	Index     int     `json:"index"`
	Frequency float64 `json:"frequency_hz"`
	Peak      float64 `json:"peak"`
	DB        float64 `json:"db_reference_1_unit"`
	DBFloored bool    `json:"db_floored"`
}
type SpectrumPeak struct {
	Rank    int             `json:"rank"`
	Reading SpectrumReading `json:"reading"`
}
type SpectrumCursor struct {
	Requested float64          `json:"requested_hz"`
	Valid     bool             `json:"valid"`
	Reason    string           `json:"reason,omitempty"`
	Reading   *SpectrumReading `json:"reading,omitempty"`
}
type SpectrumMeasurements struct {
	Source    string          `json:"source"`
	ResultID  string          `json:"result_id,omitempty"`
	Unit      string          `json:"unit"`
	Visible   bool            `json:"visible"`
	Active    string          `json:"active_cursor"`
	Peaks     []SpectrumPeak  `json:"peaks"`
	A         *SpectrumCursor `json:"a,omitempty"`
	B         *SpectrumCursor `json:"b,omitempty"`
	DeltaHz   *float64        `json:"delta_hz,omitempty"`
	DeltaPeak *float64        `json:"delta_peak,omitempty"`
	DeltaDB   *float64        `json:"delta_db,omitempty"`
}
type spectrumCursorStore struct {
	a, b       *float64
	active, id string
	liveKey    liveSpectrumKey
}
type spectrumMeasurementStore struct {
	hidden         bool
	live, recorded spectrumCursorStore
}

func spectrumReading(r *SpectrumResult, index int) SpectrumReading {
	p := r.Bins[index]
	return SpectrumReading{index, p.Frequency, p.Peak, p.DB, p.Peak <= 1e-15}
}

// One marker per local-maximum plateau. DC and the FFT display floor are not
// peaks. Markers below -60 dB relative to the strongest candidate are omitted.
// Ties choose lower frequency; these are sampled bins, never interpolated peaks.
func spectrumPeaks(r *SpectrumResult) []SpectrumPeak {
	peaks := []SpectrumPeak{}
	if r == nil {
		return peaks
	}
	for i := 1; i < len(r.Bins); {
		end := i
		for end+1 < len(r.Bins) && r.Bins[end+1].Peak == r.Bins[i].Peak {
			end++
		}
		p := r.Bins[i]
		if p.Peak > 1e-15 && p.Peak > r.Bins[i-1].Peak && (end+1 == len(r.Bins) || p.Peak > r.Bins[end+1].Peak) {
			peaks = append(peaks, SpectrumPeak{Reading: spectrumReading(r, i)})
		}
		i = end + 1
	}
	sort.Slice(peaks, func(i, j int) bool {
		a, b := peaks[i].Reading, peaks[j].Reading
		if a.Peak == b.Peak {
			return a.Frequency < b.Frequency
		}
		return a.Peak > b.Peak
	})
	if len(peaks) > 0 {
		floor := peaks[0].Reading.Peak * 1e-3
		n := 0
		for n < len(peaks) && n < MaxSpectrumPeaks && peaks[n].Reading.Peak >= floor {
			peaks[n].Rank = n + 1
			n++
		}
		peaks = peaks[:n]
	}
	return peaks
}
func spectrumCursorReading(r *SpectrumResult, frequency *float64) *SpectrumCursor {
	if frequency == nil {
		return nil
	}
	out := &SpectrumCursor{Requested: *frequency}
	if r == nil || len(r.Bins) == 0 {
		out.Reason = "No current spectrum"
		return out
	}
	if *frequency < 0 || *frequency > r.Nyquist {
		out.Reason = "Frequency is outside this frame's range"
		return out
	}
	index := sort.Search(len(r.Bins), func(i int) bool { return r.Bins[i].Frequency >= *frequency })
	if index == len(r.Bins) {
		index--
	} else if index > 0 && *frequency-r.Bins[index-1].Frequency <= r.Bins[index].Frequency-*frequency {
		index--
	}
	reading := spectrumReading(r, index)
	out.Valid = true
	out.Reading = &reading
	return out
}
func (s *Service) spectrumCursorStoreLocked(source string) *spectrumCursorStore {
	if source == "live" {
		return &s.spectrumMeasurements.live
	}
	return &s.spectrumMeasurements.recorded
}
func (s *Service) spectrumCursorBindingLocked(source string, r *SpectrumResult, store *spectrumCursorStore) bool {
	if r == nil {
		return false
	}
	if source == "live" {
		return store.liveKey == s.liveSpectrumKeyLocked()
	}
	return store.id == r.ID
}
func (s *Service) spectrumMeasurementsLocked(source string, r *SpectrumResult) *SpectrumMeasurements {
	store := s.spectrumCursorStoreLocked(source)
	out := &SpectrumMeasurements{Source: source, Visible: !s.spectrumMeasurements.hidden, Active: store.active, Peaks: spectrumPeaks(r)}
	if out.Active == "" {
		out.Active = "a"
	}
	if r != nil {
		out.ResultID = r.ID
		out.Unit = r.Probe.Unit
	}
	if s.spectrumCursorBindingLocked(source, r, store) {
		out.A = spectrumCursorReading(r, store.a)
		out.B = spectrumCursorReading(r, store.b)
	}
	if out.A != nil && out.B != nil && out.A.Valid && out.B.Valid {
		hz := out.B.Reading.Frequency - out.A.Reading.Frequency
		peak := out.B.Reading.Peak - out.A.Reading.Peak
		out.DeltaHz, out.DeltaPeak = &hz, &peak
		if !out.A.Reading.DBFloored && !out.B.Reading.DBFloored {
			db := out.B.Reading.DB - out.A.Reading.DB
			out.DeltaDB = &db
		}
	}
	return out
}
func (s *Service) currentSpectrumResultLocked(source string) *SpectrumResult {
	if source == "live" {
		if v := s.liveSpectrumViewLocked(); v != nil {
			return v.Result
		}
		return nil
	}
	if s.spectrum != nil {
		return s.spectrum.Result
	}
	return nil
}
func (s *Service) spectrumCursorCommandLocked(raw json.RawMessage) error {
	var p SpectrumCursorParams
	if err := params(raw, &p, "source", "cursor"); err != nil {
		return err
	}
	if p.Source != "live" && p.Source != "recorded" {
		return fmt.Errorf("source must be live or recorded")
	}
	if p.Cursor != "a" && p.Cursor != "b" {
		return fmt.Errorf("cursor must be a or b")
	}
	actions := 0
	for _, chosen := range []bool{p.Frequency != nil, p.Peak != 0, p.Clear, p.Select} {
		if chosen {
			actions++
		}
	}
	if actions != 1 {
		return fmt.Errorf("choose exactly one of frequency_hz, peak, clear or select")
	}
	if p.Frequency != nil && (!finite(*p.Frequency) || *p.Frequency < 0) {
		return fmt.Errorf("frequency_hz must be finite and nonnegative")
	}
	r := s.currentSpectrumResultLocked(p.Source)
	if p.ResultID != "" && (r == nil || r.ID != p.ResultID) {
		return &actionError{"conflict", "Spectrum frame changed; inspect the current result"}
	}
	var frequency float64
	if p.Frequency != nil || p.Peak != 0 {
		if r == nil {
			return &actionError{"not-ready", "A current spectrum is required"}
		}
		if p.Frequency != nil {
			frequency = *p.Frequency
			if frequency > r.Nyquist {
				return fmt.Errorf("frequency exceeds this frame's Nyquist frequency")
			}
		} else {
			peaks := spectrumPeaks(r)
			if p.Peak < 1 || p.Peak > len(peaks) {
				return &actionError{"not-found", "Choose a current peak rank"}
			}
			frequency = peaks[p.Peak-1].Reading.Frequency
		}
	}
	store := s.spectrumCursorStoreLocked(p.Source)
	if !s.spectrumCursorBindingLocked(p.Source, r, store) {
		active := store.active
		*store = spectrumCursorStore{active: active, liveKey: s.liveSpectrumKeyLocked()}
		if r != nil {
			store.id = r.ID
		}
	}
	target := &store.a
	if p.Cursor == "b" {
		target = &store.b
	}
	if p.Clear {
		*target = nil
	} else if p.Select {
		store.active = p.Cursor
	} else {
		*target = &frequency
		store.active = p.Cursor
	}
	return nil
}
func addSpectrumMeasurementDetails(id string, schema map[string]any) bool {
	if id != "spectrum.markers.set" && id != "spectrum.cursor.set" {
		return false
	}

	if id == "spectrum.markers.set" {
		schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "spectrum-measurements", Effects: []string{"Session-only peak visibility shared by live and recorded plots; no FFT recomputation, physics, history or dirty-state change", "Ranked peak data remains available in snapshots when markers are hidden"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "A boolean visible field is required"}}}
		schema["x-peak-markers"] = spectrumPeakDescription
		return true
	}
	if id == "spectrum.cursor.set" {
		inputProperty(schema, "source")["enum"] = []string{"live", "recorded"}
		inputProperty(schema, "cursor")["enum"] = []string{"a", "b"}
		frequency := inputProperty(schema, "frequency_hz")
		delete(frequency, "type")
		delete(frequency, "anyOf")
		frequency["anyOf"] = []any{map[string]any{"type": "number", "minimum": 0}, map[string]any{"type": "string", "minLength": 1, "maxLength": 128}}
		inputProperty(schema, "peak")["minimum"] = 1
		inputProperty(schema, "peak")["maximum"] = MaxSpectrumPeaks
	}
	schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "spectrum-measurements", Effects: []string{"Session-only peak visibility and A/B frequency anchors; no FFT recomputation, physics, history or dirty-state change", "Cursors snap to exact bins, nearest ties choose lower frequency; amplitude is peak in the probe unit; live anchors follow new frames of the same source", "Source changes retire anchors; result_id optionally pins one exact frame; one of frequency_hz, peak, clear or select is required"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Invalid source/cursor/frequency or ambiguous action"}, {"not-ready", "admission", "Frequency/peak placement requires a current spectrum"}, {"conflict", "admission", "Optional result_id is no longer current"}, {"not-found", "admission", "Peak rank is unavailable"}}}
	schema["x-peak-markers"] = spectrumPeakDescription
	return true
}

func cloneSpectrumMeasurements(in *SpectrumMeasurements) *SpectrumMeasurements {
	if in == nil {
		return nil
	}
	out := *in
	out.Peaks = append([]SpectrumPeak{}, in.Peaks...)
	cloneCursor := func(c *SpectrumCursor) *SpectrumCursor {
		if c == nil {
			return nil
		}
		copy := *c
		if c.Reading != nil {
			r := *c.Reading
			copy.Reading = &r
		}
		return &copy
	}
	out.A, out.B = cloneCursor(in.A), cloneCursor(in.B)
	for _, pair := range []struct {
		from *float64
		to   **float64
	}{{in.DeltaHz, &out.DeltaHz}, {in.DeltaPeak, &out.DeltaPeak}, {in.DeltaDB, &out.DeltaDB}} {
		if pair.from != nil {
			v := *pair.from
			*pair.to = &v
		}
	}
	return &out
}
