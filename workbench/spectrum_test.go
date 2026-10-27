package workbench

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func spectrumReferenceRecord() *verificationRecord {
	r := &verificationRecord{metadata: VerificationMetadata{OperationID: "reference", Duration: .4, Probes: []Probe{{ID: "v:OUT", Label: "Output", Unit: "V"}}}}
	for i := 0; i <= 1024; i++ {
		at := float64(i) * .4 / 1024
		phase := 2 * math.Pi * 10 * at
		r.rows = append(r.rows, VerificationRow{Time: at, Values: []float64{.25 + math.Sin(phase+.3) + .1*math.Sin(2*phase-.2) + .05*math.Sin(3*phase+.7)}})
	}
	return r
}
func TestSpectrumFFTAmplitudeWindowsAndTHDAgainstIndependentSignal(t *testing.T) {
	for _, window := range []string{"rectangular", "hann"} {
		t.Run(window, func(t *testing.T) {
			r, err := analyzeSpectrum(spectrumReferenceRecord(), SpectrumParams{ProbeID: "v:OUT", End: .4, Samples: 1024, Harmonics: 5, Fundamental: 10, Window: window})
			if err != nil {
				t.Fatal(err)
			}
			if math.Abs(r.Mean-.25) > 1e-12 || math.Abs(r.Bins[4].Peak-1) > 1e-12 || math.Abs(r.Bins[8].Peak-.1) > 1e-12 || math.Abs(r.Bins[12].Peak-.05) > 1e-12 || r.THD == nil || math.Abs(*r.THD-100*math.Sqrt(.1*.1+.05*.05)) > 1e-9 {
				t.Fatal("known Fourier series", r.Mean, r.Harmonics, r.THD)
			}
			if r.Nyquist != 1280 || r.Resolution != 2.5 || r.SampleStep != .4/1024 || len(r.SourceSHA256) != 64 || len(r.Bins) != 513 {
				t.Fatal("FFT grid/provenance", r)
			}
		})
	}
}
func TestSpectrumRejectsAliasingAndExplainsUnavailableTHD(t *testing.T) {
	record := spectrumReferenceRecord()
	for _, p := range []SpectrumParams{{ProbeID: "v:missing"}, {ProbeID: "v:OUT", Samples: 4096}, {ProbeID: "v:OUT", Samples: 100}, {ProbeID: "v:OUT", Start: .5, End: .6}, {ProbeID: "v:OUT", Window: "blackman"}, {ProbeID: "v:OUT", Harmonics: 11}, {ProbeID: "v:OUT", Fundamental: math.NaN()}} {
		if _, err := analyzeSpectrum(record, p); err == nil {
			t.Fatal("invalid analysis accepted", p)
		}
	}
	for _, p := range []SpectrumParams{{ProbeID: "v:OUT"}, {ProbeID: "v:OUT", End: .37, Fundamental: 10}, {ProbeID: "v:OUT", Fundamental: 1000}} {
		r, err := analyzeSpectrum(record, p)
		if err != nil {
			t.Fatal(err)
		}
		if r.THD != nil || r.THDReason == "" || len(r.Bins) < 17 {
			t.Fatal("unreliable THD must be explicit", r)
		}
	}
	// Right-side edge choice is deterministic, and the source rows stay exact.
	edge := VerificationRow{Time: record.rows[400].Time, Values: []float64{7}}
	rows := append([]VerificationRow{}, record.rows[:401]...)
	rows = append(rows, edge)
	rows = append(rows, record.rows[401:]...)
	record.rows = rows
	before, _ := json.Marshal(record.rows)
	r, err := analyzeSpectrum(record, SpectrumParams{ProbeID: "v:OUT", Samples: 256})
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(record.rows)
	if string(before) != string(after) || r.Settings.Samples != 256 {
		t.Fatal("analysis mutated source")
	}
	record.rows[300].Time = -1
	if _, err := analyzeSpectrum(record, SpectrumParams{ProbeID: "v:OUT"}); err == nil {
		t.Fatal("unordered source accepted")
	}
}
func TestSpectrumSharedCommandsExportsAndRecordedProjectReopen(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "discrete-class-ab"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	before := settled(t, s)
	duration, step := .4, .0005
	_, captured := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:VIN.positive", "v:RL"}}, "capture")
	cfg := SpectrumParams{ID: captured.Operation.ID, ProbeID: "v:RL", Start: .2, End: .4, Fundamental: 10, Window: "hann", Harmonics: 5}
	call(t, s, "spectrum.settings.set", cfg, "settings")
	call(t, s, "spectrum.analyze", struct{}{}, "analyze")
	state := s.Snapshot()
	r := state.Spectrum.Result
	if r == nil || r.THD == nil || len(r.Harmonics) != 5 || len(state.Spectrum.Available) != 2 || !reflect.DeepEqual(before.Mixed, state.Mixed) {
		t.Fatal("shared spectrum state", state.Spectrum)
	}
	r.Bins[2].Peak = 99
	r.Harmonics[0].Peak = 99
	*r.THD = 99
	if s.Snapshot().Spectrum.Result.Bins[2].Peak == 99 || s.Snapshot().Spectrum.Result.Harmonics[0].Peak == 99 || *s.Snapshot().Spectrum.Result.THD == 99 {
		t.Fatal("snapshot aliases spectrum")
	}
	for _, format := range []string{"json", "csv"} {
		path := filepath.Join(t.TempDir(), "spectrum."+format)
		call(t, s, "spectrum.export", VerificationExportParams{ID: s.Snapshot().Spectrum.Result.ID, Path: path, Format: format}, "export"+format)
		data, err := os.ReadFile(path)
		if err != nil || len(data) < 100 {
			t.Fatal("spectrum export", err)
		}
		if format == "json" {
			var exported SpectrumResult
			if err := json.Unmarshal(data, &exported); err != nil || exported.SourceSHA256 != s.Snapshot().Spectrum.Result.SourceSHA256 {
				t.Fatal("export provenance", err)
			}
		}
	}
	path := filepath.Join(t.TempDir(), "recorded.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	loaded := settled(t, fresh)
	if len(loaded.Operations) != 1 {
		t.Fatal("recorded source missing after reopen")
	}
	cfg.ID = loaded.Operations[0].ID
	call(t, fresh, "spectrum.analyze", cfg, "recompute")
	again := fresh.Snapshot().Spectrum.Result
	if !reflect.DeepEqual(again.Bins, s.Snapshot().Spectrum.Result.Bins) || *again.THD != *s.Snapshot().Spectrum.Result.THD {
		t.Fatal("saved recording changed FFT")
	}
	// Draft changes retire the prior derived result; stale export cannot borrow it.
	oldID := s.Snapshot().Spectrum.Result.ID
	cfg.ID = captured.Operation.ID
	cfg.Window = "rectangular"
	call(t, s, "spectrum.settings.set", cfg, "change")
	if s.Snapshot().Spectrum.Result != nil {
		t.Fatal("settings retained stale spectrum")
	}
	refuse(t, s, "spectrum.export", VerificationExportParams{ID: oldID, Path: filepath.Join(t.TempDir(), "stale.json"), Format: "json"}, "not-ready")
	call(t, s, "spectrum.analyze", struct{}{}, "reanalyze")
	if s.Snapshot().Spectrum.Result.ID == oldID {
		t.Fatal("analysis settings did not retire exact result identity")
	}
	refuse(t, s, "spectrum.export", VerificationExportParams{ID: oldID, Path: filepath.Join(t.TempDir(), "old.json"), Format: "json"}, "not-ready")
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "clear")
	if s.Snapshot().Spectrum.Result != nil || s.Snapshot().Spectrum.Settings.ID != "" {
		t.Fatal("document replacement retained spectrum")
	}
}

func TestSpectrumUnitsAndDetailedDiscovery(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "spectrum.settings.set", map[string]any{"start_s": "100 ms", "end_s": ".2 s", "fundamental_hz": ".01 kHz", "window": "hann"}, "units")
	p := s.Snapshot().Spectrum.Settings
	if p.Start != .1 || p.End != .2 || p.Fundamental != 10 {
		t.Fatal("spectrum SI normalization", p)
	}
	for _, id := range []string{"spectrum.analyze", "spectrum.settings.set", "spectrum.export"} {
		caps, err := detailedCapabilities(id)
		if err != nil || len(caps) != 1 {
			t.Fatal("missing discovery", id, err)
		}
	}
	refuse(t, s, "spectrum.settings.set", map[string]any{"fundamental_hz": "1 V"}, "invalid-params")
	if !reflect.DeepEqual(p, s.Snapshot().Spectrum.Settings) {
		t.Fatal("refused quantity changed settings")
	}
}

func TestSpectrumProtocolDefaultsUseActualRowsAndRefuseNull(t *testing.T) {
	s := New()
	defer s.Close()
	record := spectrumReferenceRecord()
	record.metadata.Statistics = []ScopeStatistics{{}}
	record.metadata.Duration = .5 // A nominal horizon is not the retained end time.
	s.mu.Lock()
	s.operations = map[string]*verificationOperation{"reference": {record: record, summary: OperationSummary{ID: "reference", Status: "completed", Kind: "verify"}}}
	s.operationOrder = []string{"reference"}
	s.selectedOperation = "reference"
	s.mu.Unlock()
	if s.Snapshot().Spectrum.Settings.End != .4 {
		t.Fatal("default interval used nominal rather than retained end")
	}
	before := s.Snapshot()
	rev, gen := before.CommandRevision, before.Document.Generation
	response := s.Dispatch(t.Context(), Request{Version: 1, ID: "omitted-params", Command: "spectrum.analyze", ExpectedCommandRevision: &rev, ExpectedDocumentGeneration: &gen, Key: "omitted"}, "agent")
	if response.Error != nil {
		t.Fatal("omitted params did not use current settings", response.Error)
	}
	actual := s.Snapshot()
	rev = actual.CommandRevision
	response = s.Dispatch(t.Context(), Request{Version: 1, ID: "null-params", Command: "spectrum.analyze", Params: json.RawMessage(`null`), ExpectedCommandRevision: &rev, ExpectedDocumentGeneration: &gen, Key: "null"}, "agent")
	if response.Error == nil || response.Error.Code != "invalid-params" {
		t.Fatal("null must retain the protocol's object-only refusal", response.Error)
	}
	if !reflect.DeepEqual(actual, s.Snapshot()) {
		t.Fatal("null input changed analysis")
	}
}
