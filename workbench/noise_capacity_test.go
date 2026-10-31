package workbench

import (
	"compress/gzip"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestNoiseCapacitySavedCaptureReopensAndRejectsMaterialChanges(t *testing.T) {
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
	document, err := decodeDocument(data)
	if err != nil {
		t.Fatal("ELab-generated capture must reopen", err)
	}
	path := filepath.Join(t.TempDir(), "noise.elab")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	s := New()
	defer s.Close()
	call(t, s, "project.load", LoadParams{Path: path}, "load")
	state := s.Snapshot()
	if state.DCSweep.Result == nil || len(state.DCSweep.Result.Points) != 201 || len(state.DCSweep.Result.Noise.Contributors) != 62 {
		t.Fatal("missing capacity capture")
	}
	for _, pair := range [][2]float64{{1e-20, 1.001e-20}, {0, 1e-30}, {1e-9, 2e-9}, {math.MaxFloat64, 1}, {1, math.Inf(1)}} {
		if nonnegativeRoundoffEqual(pair[0], pair[1]) {
			t.Fatal("material noise corruption accepted", pair)
		}
	}
	cfg := document.Sweeps.Entries[0].Metadata.Settings
	plan, err := newSweepPlan(Graph{state.Components, state.Wires}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateNoiseRecord(plan, cloneSweepResult(state.DCSweep.Result)); err != nil {
		t.Fatal("unmodified capture rejected", err)
	}
	if state.DCSweep.Result.Points[0].Values[0] <= 0 || state.DCSweep.Result.Noise.Contributors[0].RMS[0] <= 0 {
		t.Fatal("fixture must have positive noise")
	}
	result := cloneSweepResult(state.DCSweep.Result)
	result.Points[0].Values[0] *= 1.001
	if validateNoiseRecord(plan, result) == nil {
		t.Fatal("modified total accepted")
	}
	result = cloneSweepResult(state.DCSweep.Result)
	result.Noise.Contributors[0].RMS[0] *= 1.001
	if validateNoiseRecord(plan, result) == nil {
		t.Fatal("modified contributor RMS accepted")
	}
}
