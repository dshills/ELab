package workbench

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func toleranceRCSettings() ToleranceSettings {
	return ToleranceSettings{Mode: "ac-gain", SourceID: "V1", ProbeID: "v:C1", Frequency: 1000, Trials: 16, Seed: 123, Resistance: .05, Capacitance: .1, Beta: .2}
}
func TestToleranceSeededRCGainAgainstIndependentReferences(t *testing.T) {
	g := ACRCExampleGraph()
	p := toleranceRCSettings()
	r, err := computeTolerance(t.Context(), dcCircuitSnapshot(g, DefaultTransientSettings()), p, func(int) {})
	if err != nil {
		t.Fatal(err)
	}
	if r.Statistics.Successful != p.Trials || r.Statistics.Failed != 0 || len(r.SourceSHA256) != 64 {
		t.Fatal("trial completeness", r)
	}
	var nominalR, nominalC float64
	for _, c := range g.Components {
		if c.Kind == "resistor" {
			nominalR = c.Value
		}
		if c.Kind == "capacitor" {
			nominalC = c.Value
		}
	}
	mean := 0.
	for _, trial := range r.Trials {
		rv, cv := nominalR, nominalC
		for _, v := range trial.Parameters {
			switch v.Parameter {
			case "resistance_ohm":
				rv = v.Actual
				if rv < nominalR*.95 || rv > nominalR*1.05 {
					t.Fatal("R range")
				}
			case "capacitance_f":
				cv = v.Actual
				if cv < nominalC*.9 || cv > nominalC*1.1 {
					t.Fatal("C range")
				}
			}
		}
		want := -10 * math.Log10(1+math.Pow(2*math.Pi*p.Frequency*rv*cv, 2))
		if trial.Value == nil || math.Abs(*trial.Value-want) > 1e-10 {
			t.Fatal("independent RC transfer", trial, want)
		}
		mean += *trial.Value / float64(p.Trials)
	}
	variance := 0.
	for _, v := range r.Trials {
		variance += math.Pow(*v.Value-mean, 2) / float64(p.Trials)
	}
	if math.Abs(r.Statistics.Mean-mean) > 1e-12 || math.Abs(r.Statistics.StandardDeviation-math.Sqrt(variance)) > 1e-12 {
		t.Fatal("independent statistics", r.Statistics)
	}
	again, err := computeTolerance(t.Context(), r.Source, p, func(int) {})
	if err != nil || !reflect.DeepEqual(r, again) {
		t.Fatal("seeded reproducibility", err)
	}
	p.Seed++
	other, err := computeTolerance(t.Context(), r.Source, p, func(int) {})
	if err != nil || reflect.DeepEqual(r.Trials, other.Trials) {
		t.Fatal("different seed ignored", err)
	}
	if !reflect.DeepEqual(g, r.Source.Graph) {
		t.Fatal("captured circuit mutated")
	}
}
func waitTolerance(t *testing.T, s *Service) *ToleranceState {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		v := s.Snapshot().Tolerance
		if v != nil && v.Status != "" && !operationActive(v.Status) {
			return v
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("tolerance worker did not retire")
	return nil
}
func TestTolerancePublicWorkflowCaptureExportsAndNoLiveMutation(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "ac-rc"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	before := settled(t, s)
	p := toleranceRCSettings()
	call(t, s, "tolerance.settings.set", p, "settings")
	call(t, s, "tolerance.run", struct{}{}, "run")
	v := waitTolerance(t, s)
	if v.Status != "completed" || v.Result == nil || !v.SourceCurrent {
		t.Fatal(v)
	}
	after := s.Snapshot()
	if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Mixed, after.Mixed) || !reflect.DeepEqual(before.Transient, after.Transient) || before.Document.Dirty != after.Document.Dirty || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatalf("analysis changed graph=%t mixed=%t dirty=%t history=%t; before=%+v after=%+v", !reflect.DeepEqual(before.Components, after.Components), !reflect.DeepEqual(before.Mixed, after.Mixed), before.Document.Dirty != after.Document.Dirty, !reflect.DeepEqual(before.UndoEntry, after.UndoEntry), before.Mixed, after.Mixed)
	}
	id := v.Result.ID
	v.Result.Trials[0].Parameters[0].Actual = 999
	*v.Result.Trials[0].Value = 999
	v.Result.Source.Graph.Components[0].Name = "alias"
	owned := s.Snapshot().Tolerance
	if owned.Result.Trials[0].Parameters[0].Actual == 999 || *owned.Result.Trials[0].Value == 999 || owned.Result.Source.Graph.Components[0].Name == "alias" {
		t.Fatal("snapshot aliases parameter slice, value pointer or source graph")
	}
	for _, format := range []string{"json", "csv"} {
		path := filepath.Join(t.TempDir(), "trials."+format)
		call(t, s, "tolerance.export", VerificationExportParams{ID: id, Path: path, Format: format}, "export-"+format)
		data, err := os.ReadFile(path)
		if err != nil || len(data) < 100 {
			t.Fatal("trial export", err)
		}
		if format == "json" {
			var r ToleranceResult
			if err := json.Unmarshal(data, &r); err != nil || r.SourceSHA256 != s.Snapshot().Tolerance.Result.SourceSHA256 || len(r.Trials) != 16 {
				t.Fatal("export provenance", err)
			}
		}
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "edit")
	if s.Snapshot().Tolerance.SourceCurrent {
		t.Fatal("captured trial source not marked old")
	}
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "clear")
	if s.Snapshot().Tolerance.Result != nil {
		t.Fatal("replacement retained result")
	}
	refuse(t, s, "tolerance.export", VerificationExportParams{ID: id, Path: filepath.Join(t.TempDir(), "stale.json"), Format: "json"}, "not-ready")
}
func TestToleranceFailuresBoundsZeroVariationAndCancellation(t *testing.T) {
	g := EmitterFollowerExampleGraph()
	v := DefaultTransistorProfile()
	v.ForwardBeta = 1
	v.CBE, v.CBC = 0, 0
	for i, c := range g.Components {
		if c.ID == "Q1" {
			g.Components[i].Device = &DeviceParameters{Bipolar: &v}
		}
	}
	p := ToleranceSettings{Mode: "dc", ProbeID: "v:Q1.emitter", Frequency: 1000, Trials: 16, Seed: 1, Beta: .5}
	r, err := computeTolerance(t.Context(), dcCircuitSnapshot(g, DefaultTransientSettings()), p, func(int) {})
	if err != nil {
		t.Fatal(err)
	}
	if r.Statistics.Failed == 0 || r.Statistics.Successful == 0 || r.Statistics.Failed+r.Statistics.Successful != 16 {
		t.Fatal("failed trials were lost", r.Statistics)
	}
	for _, q := range r.Trials {
		if (q.Value == nil) == (q.Failure == "") {
			t.Fatal("trial outcome ambiguous", q)
		}
	}
	p.Beta = 0
	r, err = computeTolerance(t.Context(), r.Source, p, func(int) {})
	if err != nil || r.Statistics.StandardDeviation != 0 || r.Statistics.Failed != 0 {
		t.Fatal("zero tolerance", err)
	}
	for _, change := range []func(*ToleranceSettings){func(p *ToleranceSettings) { p.Trials = 65 }, func(p *ToleranceSettings) { p.Seed = 9007199254740992 }, func(p *ToleranceSettings) { p.Resistance = .51 }, func(p *ToleranceSettings) { p.Frequency = math.NaN() }, func(p *ToleranceSettings) { p.Mode = "transient" }} {
		bad := p
		change(&bad)
		if _, err := validateTolerance(g, bad); err == nil {
			t.Fatal("invalid settings accepted", bad)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := computeTolerance(ctx, r.Source, p, func(int) {}); !errors.Is(err, context.Canceled) {
		t.Fatal("cancel ignored", err)
	}
}

func TestToleranceOmittedParamsZeroSeedAndCooperativeProgressCancellation(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "rc-tolerance"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	p := toleranceRCSettings()
	p.Seed = 0
	p.Resistance = 0
	p.Capacitance = 0
	p.Beta = 0
	call(t, s, "tolerance.settings.set", p, "configure-zero")
	state := s.Snapshot()
	rev, gen := state.CommandRevision, state.Document.Generation
	response := s.Dispatch(t.Context(), Request{Version: 1, ID: "omitted", Command: "tolerance.run", Key: "omitted", ExpectedCommandRevision: &rev, ExpectedDocumentGeneration: &gen}, "agent")
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	v := waitTolerance(t, s)
	if v.Result == nil || v.Result.Settings.Seed != 0 || v.Result.Statistics.StandardDeviation != 0 {
		t.Fatal("zero settings lost", v)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	calls := 0
	_, err := computeTolerance(ctx, v.Result.Source, p, func(int) { calls++; cancel() })
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatal("progress cancellation", calls, err)
	}
}

func TestToleranceDCMayOmitFrequencyAndCSVEscapesText(t *testing.T) {
	p := ToleranceSettings{Mode: "dc", ProbeID: "v:C1", Trials: 2}
	if _, err := validateTolerance(ACRCExampleGraph(), p); err != nil {
		t.Fatal("irrelevant DC frequency required", err)
	}
	r := &ToleranceResult{Probe: Probe{ID: "v:C1", Unit: "V"}, Trials: []ToleranceTrial{{Index: 1, Failure: "=1+1", Parameters: []ToleranceVariation{}}}}
	data, err := encodeToleranceExport(r, "csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil || len(rows) != 2 || len(rows[1]) != 7 || rows[1][5] != "'=1+1" || r.Trials[0].Failure != "=1+1" {
		t.Fatal("CSV text convention", err, string(data))
	}
}
