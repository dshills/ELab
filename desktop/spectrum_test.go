package main

import (
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/ui"
)

func TestSpectrumInspectorAndAgentUseTheSameRecordedAnalysis(t *testing.T) {
	u, cleanup := testScreen(t)
	defer cleanup()
	a := workbench.New()
	defer a.Close()
	for _, s := range []*workbench.Service{u.service, a} {
		agentAction(t, s, "example.load", workbench.ExampleLoadParams{ID: "amplifier-distortion"})
		agentAction(t, s, "simulation.pause", struct{}{})
		agentAction(t, s, "simulation.restart", struct{}{})
		awaitInductorCircuit(t, s)
		duration, step := .2, .0005
		agentAction(t, s, "simulation.verify", workbench.VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:VIN.positive", "v:U1"}})
		deadline := time.Now().Add(5 * time.Second)
		for s.Snapshot().Verification == nil && time.Now().Before(deadline) {
			time.Sleep(5 * time.Millisecond)
		}
		if s.Snapshot().Verification == nil {
			t.Fatal("capture did not complete")
		}
	}
	cfg := u.service.Snapshot().Spectrum.Settings
	cfg.ProbeID = "v:U1"
	uiAction(t, u, "spectrum-probe", &ui.ChangeEvent{SelectedID: "v:U1"})
	agentAction(t, a, "spectrum.settings.set", cfg)
	before := u.service.Snapshot().Mixed
	uiAction(t, u, "spectrum-analyze", nil)
	agentConfig := cfg
	agentConfig.ID = a.Snapshot().LastOperation
	agentAction(t, a, "spectrum.settings.set", agentConfig)
	agentAction(t, a, "spectrum.analyze", agentConfig)
	user, agent := u.service.Snapshot(), a.Snapshot()
	if user.Spectrum == nil || agent.Spectrum == nil || agent.Spectrum.Result == nil || agent.Spectrum.Result.THD == nil || user.Spectrum.Result == nil || user.Spectrum.Result.THD == nil || !reflect.DeepEqual(user.Spectrum.Result.Bins, agent.Spectrum.Result.Bins) || *user.Spectrum.Result.THD != *agent.Spectrum.Result.THD || !reflect.DeepEqual(before, user.Mixed) {
		t.Fatal("recorded spectrum parity/live separation")
	}
	if *user.Spectrum.Result.THD < 1 {
		t.Fatal("overdriven amplifier should expose harmonic distortion")
	}
	for _, id := range []string{"spectrum-title", "spectrum-plot", "spectrum-thd", "spectrum-export-json", "spectrum-window", "spectrum-harmonics"} {
		if _, ok := findNode(u.view(), id); !ok {
			t.Fatal("missing spectrum control", id)
		}
	}
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		agentAction(t, u.service, "appearance.theme.set", map[string]string{"theme": theme})
		if err := ui.Validate(u.view()); err != nil {
			t.Fatal(err)
		}
		mesh, err := u.spectrumMesh(canvas.Viewport{Width: 300, Height: 190, Zoom: 1})
		if err != nil || mesh == nil {
			t.Fatal("spectrum plot", err)
		}
	}
}
