package workbench

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func pageQuery(t *testing.T, s *Service, p AnalysisPageParams) AnalysisPage {
	t.Helper()
	raw, _ := json.Marshal(p)
	r := s.Dispatch(t.Context(), Request{Version: 1, ID: "page", Command: "analysis.page", Params: raw}, "reader")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	return r.Result.(AnalysisPage)
}
func TestAnalysisCapacityPagesReassembleAndOwnData(t *testing.T) {
	f, err := os.Open("testdata/noise-capacity.elab.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	data, err := io.ReadAll(z)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "capacity.elab")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	s := New()
	defer s.Close()
	call(t, s, "project.load", LoadParams{Path: path}, "load")
	before := settled(t, s)
	r := s.Dispatch(t.Context(), Request{Version: 1, ID: "list", Command: "analysis.list", Params: json.RawMessage(`{}`)}, "reader")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	catalog := r.Result.(AnalysisCatalog)
	if len(catalog.Analyses) != 1 || catalog.Analyses[0].Rows != 201 || catalog.CommandRevision != before.CommandRevision {
		t.Fatal("compact catalog", catalog)
	}
	for _, limit := range []int{1, 25, 50} {
		points := []SweepPoint{}
		offset := 0
		for {
			p := pageQuery(t, s, AnalysisPageParams{Kind: "sweep", ID: catalog.Analyses[0].ID, DocumentGeneration: catalog.DocumentGeneration, Offset: offset, Limit: limit})
			encoded, _ := json.Marshal(p)
			if len(encoded) > 512*1024 || len(p.Points) > limit {
				t.Fatal("page not bounded", len(encoded))
			}
			points = append(points, p.Points...)
			if p.Done {
				break
			}
			if p.NextOffset <= offset {
				t.Fatal("page did not advance")
			}
			offset = p.NextOffset
		}
		if !reflect.DeepEqual(points, before.DCSweep.Result.Points) {
			t.Fatal("page reconstruction lost values")
		}
	}
	p := pageQuery(t, s, AnalysisPageParams{Kind: "sweep", ID: catalog.Analyses[0].ID, DocumentGeneration: catalog.DocumentGeneration})
	p.Points[0].Values[0] = 999
	p.Points[0].Noise.PSD[0][0] = 999
	p.Analysis.NoiseContributors[0].RMS[0] = 999
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("paged read changed physics/state or borrowed data")
	}
	for _, change := range []AnalysisPageParams{{Kind: "sweep", ID: catalog.Analyses[0].ID, DocumentGeneration: catalog.DocumentGeneration, Limit: 51}, {Kind: "sweep", ID: catalog.Analyses[0].ID, DocumentGeneration: catalog.DocumentGeneration, Offset: -1}, {Kind: "sweep", ID: catalog.Analyses[0].ID, DocumentGeneration: catalog.DocumentGeneration, Offset: 202}, {Kind: "wrong", ID: "x", DocumentGeneration: catalog.DocumentGeneration}} {
		raw, _ := json.Marshal(change)
		r = s.Dispatch(t.Context(), Request{Version: 1, ID: "bad", Command: "analysis.page", Params: raw}, "reader")
		if r.Error == nil || r.Error.Code != "invalid-params" {
			t.Fatal("invalid page admitted", r.Error)
		}
	}
	terminal := pageQuery(t, s, AnalysisPageParams{Kind: "sweep", ID: catalog.Analyses[0].ID, DocumentGeneration: catalog.DocumentGeneration, Offset: 201})
	if !terminal.Done || len(terminal.Points) != 0 {
		t.Fatal("terminal page")
	}
	call(t, s, "project.clear", map[string]bool{"discard_changes": true}, "clear")
	raw, _ := json.Marshal(AnalysisPageParams{Kind: "sweep", ID: catalog.Analyses[0].ID, DocumentGeneration: catalog.DocumentGeneration})
	r = s.Dispatch(t.Context(), Request{Version: 1, ID: "stale", Command: "analysis.page", Params: raw}, "reader")
	if r.Error == nil || r.Error.Code != "conflict" {
		t.Fatal("old document page admitted")
	}
}

func TestAnalysisRecordingSpectrumTolerancePages(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "rc"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	settled(t, s)
	duration, step := .05, .001
	_, capture := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:C1"}}, "capture")
	call(t, s, "spectrum.analyze", SpectrumParams{ID: capture.Operation.ID, ProbeID: "v:C1", Samples: 32}, "spectrum")
	call(t, s, "tolerance.settings.set", toleranceRCSettings(), "settings")
	call(t, s, "tolerance.run", struct{}{}, "tolerance")
	deadline := time.Now().Add(5 * time.Second)
	for {
		state := s.Snapshot()
		if state.Tolerance.Status == "completed" {
			break
		}
		if state.Tolerance.Status == "failed" || time.Now().After(deadline) {
			t.Fatal("tolerance not completed", state.Tolerance)
		}
		time.Sleep(time.Millisecond)
	}
	before := s.Snapshot()
	for _, test := range []struct {
		kind, id string
		rows     int
	}{{"recording", capture.Operation.ID, 51}, {"spectrum", before.Spectrum.Result.ID, len(before.Spectrum.Result.Bins)}, {"tolerance", before.Tolerance.ID, len(before.Tolerance.Result.Trials)}} {
		t.Run(test.kind, func(t *testing.T) {
			samples := []VerificationRow{}
			bins := []SpectrumBin{}
			trials := []ToleranceTrial{}
			offset := 0
			for {
				p := pageQuery(t, s, AnalysisPageParams{Kind: test.kind, ID: test.id, DocumentGeneration: before.Document.Generation, Offset: offset, Limit: 7})
				samples = append(samples, p.Samples...)
				bins = append(bins, p.Bins...)
				trials = append(trials, p.Trials...)
				if p.Done {
					break
				}
				offset = p.NextOffset
			}
			switch test.kind {
			case "recording":
				s.mu.Lock()
				want := append([]VerificationRow{}, s.operations[test.id].record.rows...)
				s.mu.Unlock()
				if !reflect.DeepEqual(samples, want) {
					t.Fatal("recording page differs")
				}
				samples[0].Values[0] = 999
			case "spectrum":
				if !reflect.DeepEqual(bins, before.Spectrum.Result.Bins) {
					t.Fatal("spectrum page differs")
				}
				bins[0].Peak = 999
			case "tolerance":
				if !reflect.DeepEqual(trials, before.Tolerance.Result.Trials) {
					t.Fatal("trials differ")
				}
				trials[0].Parameters[0].Actual = 999
				if trials[0].Value != nil {
					*trials[0].Value = 999
				}
			}
			if !reflect.DeepEqual(before, s.Snapshot()) {
				t.Fatal("page alias/state change")
			}
		})
	}
}
func TestAnalysisUnknownAndNotReady(t *testing.T) {
	s := Service{documentGeneration: 1, operations: map[string]*verificationOperation{"pending": {summary: OperationSummary{Status: "running"}}}}
	for _, tc := range []struct{ id, code string }{{"missing", "not-found"}, {"pending", "not-ready"}} {
		_, err := s.analysisPageLocked(AnalysisPageParams{Kind: "recording", ID: tc.id, DocumentGeneration: 1})
		if err == nil || err.(*actionError).code != tc.code {
			t.Fatal(fmt.Sprint(err))
		}
	}
}

func TestAnalysisPageEvictionAndDiscovery(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "noise-rc"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	first := ""
	for i := 0; i < MaxDCSweepOperations+1; i++ {
		if i == MaxDCSweepOperations {
			call(t, s, "sweep.keep.set", RecordingKeepParams{ID: first, Keep: false}, "exclude")
		}
		cfg := noiseSettings("V1", "v:C1")
		cfg.Points = 2
		call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, fmt.Sprint(i))
		state := awaitSweep(t, s, "completed")
		if first == "" {
			first = state.DCSweep.Selected
		}
	}
	state := s.Snapshot()
	raw, _ := json.Marshal(AnalysisPageParams{Kind: "sweep", ID: first, DocumentGeneration: state.Document.Generation})
	r := s.Dispatch(t.Context(), Request{Version: 1, ID: "old", Command: "analysis.page", Params: raw}, "reader")
	if r.Error == nil || r.Error.Code != "not-found" {
		t.Fatal("evicted data returned", r.Error)
	}
	caps, err := detailedCapabilities("analysis.page")
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err = json.Unmarshal(caps[0].Input, &schema); err != nil {
		t.Fatal(err)
	}
	if inputProperty(schema, "limit")["maximum"] != float64(MaxAnalysisPageRows) || schema["x-command-contract"] == nil {
		t.Fatal("missing detailed bounds/contract")
	}
}

func TestReadyToExitWithoutSnapshot(t *testing.T) {
	s := New()
	defer s.Close()
	settled(t, s)
	if s.ReadyToExit() {
		t.Fatal("live service reports exit ready")
	}
	if n := testing.AllocsPerRun(100, func() { s.ReadyToExit() }); n != 0 {
		t.Fatal("lifecycle poll allocated", n)
	}
	s.Close()
	if !s.ReadyToExit() || !s.Snapshot().ReadyToExit() {
		t.Fatal("lifecycle readiness differs")
	}
}
