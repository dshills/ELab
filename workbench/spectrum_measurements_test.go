package workbench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestSpectrumMeasurementsKnownSignalAndNearestBin(t *testing.T) {
	r, err := analyzeSpectrum(spectrumReferenceRecord(), SpectrumParams{ProbeID: "v:OUT", End: .4, Samples: 1024, Fundamental: 10, Window: "rectangular"})
	if err != nil {
		t.Fatal(err)
	}
	before := cloneSpectrumResult(r)
	peaks := spectrumPeaks(r)
	if len(peaks) != 3 {
		t.Fatalf("peaks: %+v", peaks)
	}
	for i, want := range []struct{ hz, peak float64 }{{10, 1}, {20, .1}, {30, .05}} {
		p := peaks[i]
		if p.Rank != i+1 || p.Reading.Frequency != want.hz || math.Abs(p.Reading.Peak-want.peak) > 1e-12 {
			t.Fatal("known peak", p)
		}
	}
	for _, tt := range []struct {
		want  float64
		index int
	}{{0, 0}, {11.25, 4}, {11.250001, 5}, {1280, 512}} {
		got := spectrumCursorReading(r, &tt.want)
		if !got.Valid || got.Reading.Index != tt.index {
			t.Fatalf("nearest bin %g: %+v", tt.want, got)
		}
	}
	invalid := 1281.
	if spectrumCursorReading(r, &invalid).Valid || spectrumCursorReading(nil, &invalid).Valid || spectrumCursorReading(r, nil) != nil {
		t.Fatal("invalid cursor reading")
	}
	if !reflect.DeepEqual(before, r) {
		t.Fatal("measurements modified FFT")
	}
}
func TestSpectrumPeaksPlateausTiesFloorThresholdAndBound(t *testing.T) {
	r := &SpectrumResult{}
	// DC is larger than all AC candidates. Two equal plateaus use lower-frequency
	// bins; candidates at exactly -60 dB are retained. Nyquist is a real bin.
	for i, p := range []float64{20, 0, 1, 1, 0, 1, 0, .001, 0, .000999, 0, 1e-15, 0, .5} {
		r.Bins = append(r.Bins, SpectrumBin{Frequency: float64(i), Peak: p})
	}
	p := spectrumPeaks(r)
	if len(p) != 4 {
		t.Fatal(p)
	}
	for i, hz := range []float64{2, 5, 13, 7} {
		if p[i].Rank != i+1 || p[i].Reading.Frequency != hz {
			t.Fatal("peak ordering/threshold", p)
		}
	}
	r.Bins = nil
	for i := range 30 {
		v := 0.
		if i%2 == 1 {
			v = 1
		}
		r.Bins = append(r.Bins, SpectrumBin{Frequency: float64(i), Peak: v})
	}
	p = spectrumPeaks(r)
	if len(p) != MaxSpectrumPeaks || p[4].Reading.Frequency != 9 {
		t.Fatal("peak bound", p)
	}
	if len(spectrumPeaks(nil)) != 0 {
		t.Fatal("nil peaks")
	}
}

func recordedSpectrumMeasurementFixture(t *testing.T) *Service {
	t.Helper()
	s := liveSpectrumFixture(t)
	duration, step := .05, .001
	_, capture := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:C1", "i:R1"}}, "measurement-capture")
	call(t, s, "spectrum.analyze", SpectrumParams{ID: capture.Operation.ID, ProbeID: "v:C1"}, "measurement-analyze")
	return s
}
func TestSpectrumCursorCommandsRecordedGuardsOwnershipAndRetirement(t *testing.T) {
	s := recordedSpectrumMeasurementFixture(t)
	before := s.Snapshot()
	r := before.Spectrum.Result
	call(t, s, "spectrum.cursor.set", map[string]any{"source": "recorded", "cursor": "a", "frequency_hz": "0 Hz", "result_id": r.ID}, "cursor-a")
	call(t, s, "spectrum.cursor.set", map[string]any{"source": "recorded", "cursor": "b", "frequency_hz": r.Resolution * 2}, "cursor-b")
	got := s.Snapshot()
	m := got.Spectrum.Measurements
	if m.A == nil || !m.A.Valid || m.B == nil || !m.B.Valid || m.Active != "b" || m.Unit != "V" || *m.DeltaHz != 2*r.Resolution || *m.DeltaPeak != m.B.Reading.Peak-m.A.Reading.Peak {
		t.Fatal("readings", m)
	}
	if !reflect.DeepEqual(before.Transient, got.Transient) || !reflect.DeepEqual(before.Spectrum.Result, got.Spectrum.Result) || before.Document.Dirty != got.Document.Dirty || before.RunID != got.RunID || before.ElectricalRevision != got.ElectricalRevision || !reflect.DeepEqual(before.UndoEntry, got.UndoEntry) || !reflect.DeepEqual(before.RedoEntry, got.RedoEntry) {
		t.Fatal("measurement changed simulation/document/history")
	}
	original := cloneSpectrumMeasurements(m)
	// Both the public snapshot and explicit clone own every pointer and slice.
	clone := cloneSpectrum(got.Spectrum)
	*m.DeltaHz = 999
	*m.DeltaPeak = 999
	m.A.Reading.Peak = 999
	if m.DeltaDB != nil {
		*m.DeltaDB = 999
	}
	if len(m.Peaks) > 0 {
		m.Peaks[0].Reading.Peak = 999
	}
	if !reflect.DeepEqual(original, s.Snapshot().Spectrum.Measurements) || !reflect.DeepEqual(original, clone.Measurements) {
		t.Fatal("borrowed measurement data")
	}
	bad := []string{
		`{"source":"bad","cursor":"a","frequency_hz":1}`,
		`{"source":"recorded","cursor":"x","clear":true}`,
		`{"source":"recorded","cursor":"a"}`,
		`{"source":"recorded","cursor":"a","clear":true,"select":true}`,
		`{"source":"recorded","cursor":"a","frequency_hz":-1}`,
		`{"source":"recorded","cursor":"a","frequency_hz":1e99}`,
		`{"source":"recorded","cursor":"a","frequency_hz":"1 V"}`,
		`{"source":"recorded","cursor":"a","peak":6}`,
		`{"source":"recorded","cursor":"a","peak":-1}`,
		`{"source":"recorded","cursor":"a","clear":true,"result_id":"retired"}`,
		`{"source":"recorded","cursor":"a","clear":true,"extra":1}`,
	}
	for i, raw := range bad {
		rev := s.Snapshot().CommandRevision
		resp := s.Dispatch(t.Context(), Request{Version: 1, ID: fmt.Sprint(i), Key: fmt.Sprint(i), Command: "spectrum.cursor.set", Params: json.RawMessage(raw), ExpectedCommandRevision: &rev}, "agent")
		if resp.Error == nil || !reflect.DeepEqual(original, s.Snapshot().Spectrum.Measurements) {
			t.Fatal("invalid input changed state", raw, resp.Error)
		}
	}
	call(t, s, "spectrum.markers.set", ScopeVisibilityParams{Visible: false}, "hide-peaks")
	if s.Snapshot().Spectrum.Measurements.Visible {
		t.Fatal("visibility")
	}
	call(t, s, "spectrum.cursor.set", SpectrumCursorParams{Source: "recorded", Cursor: "a", Select: true}, "select-a")
	if s.Snapshot().Spectrum.Measurements.A.Reading.Frequency != 0 || s.Snapshot().Spectrum.Measurements.Active != "a" {
		t.Fatal("select moved cursor")
	}
	call(t, s, "spectrum.cursor.set", SpectrumCursorParams{Source: "recorded", Cursor: "a", Clear: true}, "clear-a")
	if s.Snapshot().Spectrum.Measurements.A != nil || s.Snapshot().Spectrum.Measurements.DeltaHz != nil {
		t.Fatal("clear retained delta")
	}
	p := r.Settings
	p.ProbeID = "i:R1"
	call(t, s, "spectrum.settings.set", p, "retire-recorded")
	if m := s.Snapshot().Spectrum.Measurements; m.A != nil || m.B != nil {
		t.Fatal("retired result borrowed cursor")
	}
	call(t, s, "spectrum.analyze", struct{}{}, "current-recorded")
	if s.Snapshot().Spectrum.Measurements.Unit != "A" || s.Snapshot().Spectrum.Measurements.B != nil {
		t.Fatal("new source borrowed anchor")
	}
	call(t, s, "project.clear", map[string]bool{"discard_changes": true}, "clear-document")
	if m := s.Snapshot().Spectrum.Measurements; m.A != nil || m.B != nil || !m.Visible {
		t.Fatal("document retained measurements", m)
	}
}
func TestSpectrumCursorLiveTracksBinsAndRetiresSource(t *testing.T) {
	s := liveSpectrumFixture(t)
	call(t, s, "scope.view.set", ScopeViewParams{View: "spectrum"}, "measurement-tab")
	first := awaitLiveSpectrum(t, s)
	f := first.Result.Resolution * 2
	call(t, s, "spectrum.cursor.set", SpectrumCursorParams{Source: "live", Cursor: "a", Frequency: &f}, "live-a")
	call(t, s, "spectrum.cursor.set", SpectrumCursorParams{Source: "live", Cursor: "b", Peak: 1}, "live-b-peak")
	anchors := s.Snapshot().Spectrum.Live.Measurements
	if anchors.B == nil || anchors.B.Reading.Frequency != anchors.Peaks[0].Reading.Frequency {
		t.Fatal("peak selection", anchors)
	}
	p := first.Settings
	p.Hold = true
	call(t, s, "spectrum.live.set", p, "measurement-hold")
	if !reflect.DeepEqual(anchors, s.Snapshot().Spectrum.Live.Measurements) {
		t.Fatal("hold retired anchors")
	}
	call(t, s, "simulation.step", StepParams{Duration: .01}, "measurement-step")
	if !reflect.DeepEqual(anchors, s.Snapshot().Spectrum.Live.Measurements) {
		t.Fatal("held measurements moved")
	}
	p.Hold = false
	call(t, s, "spectrum.live.set", p, "measurement-follow")
	deadline := time.Now().Add(3 * time.Second)
	var next *LiveSpectrumState
	for time.Now().Before(deadline) {
		next = s.Snapshot().Spectrum.Live
		if next.End > first.End {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if next.End <= first.End || next.Measurements.A == nil || next.Measurements.A.Requested != f || !reflect.DeepEqual(next.Measurements.A, spectrumCursorReading(next.Result, &f)) {
		t.Fatal("live frequency did not track accepted bins", next)
	}
	// A delayed exact-frame request cannot borrow the latest FFT.
	rev := s.Snapshot().CommandRevision
	raw, _ := json.Marshal(SpectrumCursorParams{Source: "live", Cursor: "a", Clear: true, ResultID: first.Result.ID})
	response := s.Dispatch(t.Context(), Request{Version: 1, ID: "stale-frame", Key: "stale-frame", Command: "spectrum.cursor.set", Params: raw, ExpectedCommandRevision: &rev}, "agent")
	if response.Error == nil || response.Error.Code != "conflict" {
		t.Fatal("retired frame admitted", response.Error)
	}
	p.ProbeID = "i:R1"
	call(t, s, "spectrum.live.set", p, "measurement-current")
	next = awaitLiveSpectrum(t, s)
	if next.Measurements.A != nil || next.Measurements.B != nil || next.Measurements.Unit != "A" {
		t.Fatal("source change borrowed voltage anchors")
	}
	call(t, s, "simulation.restart", struct{}{}, "measurement-restart")
	if m := s.Snapshot().Spectrum.Live.Measurements; m.A != nil || m.B != nil {
		t.Fatal("restart borrowed anchors")
	}
}
func TestSpectrumCursorFloorAndDiscovery(t *testing.T) {
	s := New()
	defer s.Close()
	s.mu.Lock()
	r := &SpectrumResult{ID: "floor", Nyquist: 2, Probe: Probe{Unit: "A"}, Bins: []SpectrumBin{{Frequency: 0, Peak: 0, DB: -300}, {Frequency: 1, Peak: 1e-9, DB: -180}, {Frequency: 2, Peak: 0, DB: -300}}}
	a, b := 0., 1.
	s.spectrum = &SpectrumState{Result: r}
	s.spectrumMeasurements.recorded = spectrumCursorStore{a: &a, b: &b, id: r.ID}
	m := s.spectrumMeasurementsLocked("recorded", r)
	s.mu.Unlock()
	if !m.A.Reading.DBFloored || m.B.Reading.DBFloored || m.DeltaDB != nil || m.DeltaPeak == nil || *m.DeltaPeak != 1e-9 {
		t.Fatal("floor claimed exact dB ratio", m)
	}
	data, err := json.Marshal(Catalog())
	if err != nil || len(data) > 64*1024 {
		t.Fatal("catalog budget", len(data), err)
	}
	t.Logf("catalog %d bytes; %d commands", len(data), len(Catalog()))
	var input map[string]any
	if err = json.Unmarshal(SchemaRegistry()["urn:elab:spectrum-cursor-input:1"], &input); err != nil {
		t.Fatal(err)
	}
	if inputProperty(input, "frequency_hz")["anyOf"] == nil || inputProperty(input, "peak")["maximum"] != float64(MaxSpectrumPeaks) {
		t.Fatal("missing detailed input schema")
	}
}

func TestSpectrumMarkerAndCursorDetailedContractsAreDistinct(t *testing.T) {
	marker := sweepDetailSchema(t, "spectrum.markers.set")
	cursor := sweepDetailSchema(t, "spectrum.cursor.set")
	m, _ := json.Marshal(marker["x-command-contract"])
	c, _ := json.Marshal(cursor["x-command-contract"])
	if bytes.Contains(m, []byte("frequency_hz")) || !bytes.Contains(m, []byte("visible")) || !bytes.Contains(c, []byte("frequency_hz")) {
		t.Fatal("marker discovery incorrectly requires cursor parameters", string(m), string(c))
	}
}
