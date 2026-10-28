package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestCapacityNoiseDrawingActionsAfterCapture(t *testing.T) {
	f, err := os.Open("../workbench/testdata/noise-capacity.elab.gz")
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
	var d workbench.Document
	if err = json.Unmarshal(data, &d); err != nil {
		t.Fatal(err)
	}
	d.Sweeps = nil
	d.Version = workbench.DocumentVersion
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "capacity.elab")
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s, done := testScreen(t)
	defer done()
	agentAction(t, s.service, "project.load", workbench.LoadParams{Path: path})
	agentAction(t, s.service, "sweep.settings.set", workbench.SweepSettings{ComponentID: "V1", Parameter: "noise", Start: "1 Hz", Stop: "100 kHz", Spacing: "logarithmic", Points: 201, ProbeIDs: []string{"v:C1", "v:R1", "v:R2"}})
	uiAction(t, s, "inspector-section-simulation", nil)
	uiAction(t, s, "sweep-start", nil)
	state := awaitNativeSweep(t, s)
	if len(state.DCSweep.Result.Points) != 201 || len(state.DCSweep.Result.Noise.Contributors) != 62 {
		t.Fatal("capacity sweep incomplete")
	}
	uiAction(t, s, "sweep-back", nil)
	agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
	uiAction(t, s, "inspector-section-component", nil)
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "2k"})
	uiAction(t, s, "apply-value", nil)
	state = s.service.Snapshot()
	part, found := (workbench.Graph{Components: state.Components}).Component("R1")
	if !found || state.DCSweep == nil || len(state.DCSweep.Operations) != 1 {
		t.Fatal("missing component/capture")
	}
	if part.Value != 2000 || state.DCSweep.Operations[0].SourceUnchanged {
		t.Fatal("UI edit lost or capture not stale")
	}
}
