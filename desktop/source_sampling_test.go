package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeSourceMinimumWarningAndFitMatchAgent(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	for _, service := range []*workbench.Service{s.service, agent} {
		agentAction(t, service, "component.place", workbench.PlaceParams{ID: "Ramp", Kind: "finite-pulse-source"})
		agentAction(t, service, "simulation.pause", struct{}{})
		wave := workbench.DefaultPulseWaveform()
		wave.RiseSeconds = 1e-6
		agentAction(t, service, "source.pulse.set", workbench.PulseParams{ID: "Ramp", Waveform: wave})
		agentAction(t, service, "component.select", workbench.IDParams{ID: "Ramp"})
	}
	uiAction(t, s, "fit-waveform-sampling", nil)
	agentAction(t, agent, "simulation.sampling.fit", struct{}{})
	u, a := s.service.Snapshot(), agent.Snapshot()
	if u.TransientSettings != a.TransientSettings || !reflect.DeepEqual(u.SourceSampling, a.SourceSampling) || !u.SourceSampling[0].MinimumLimited || u.SourceSampling[0].CanFit {
		t.Fatal("native and agent minimum sampling disagree")
	}
	s.app.Do(func() {
		root := s.view()
		if err := ui.Validate(root); err != nil {
			t.Fatal(err)
		}
		if _, ok := findAction(root, "fit-waveform-sampling"); ok {
			t.Fatal("ineffective source Fit remains visible at minimum")
		}
		warning, ok := findAction(root, "source-sampling-warning")
		if !ok || !strings.Contains(warning.Text, "minimum") || !strings.Contains(warning.Text, "cannot meet") {
			t.Fatal("native minimum explanation disappeared")
		}
		if root.Focus.ID != "setting-step" {
			t.Fatal("disappearing source Fit lost keyboard continuation", root.Focus)
		}

		if _, ok := ui.FocusTarget(root, "setting-step"); !ok {
			t.Fatal("Fit names an unavailable focus continuation")
		}
	})
	u.Recording = &workbench.RecordingView{Viewing: true}
	nodes := s.waveformControls(u, u.Components[0])
	found := false
	for _, node := range nodes {
		if node.ID == "source-sampling-recorded-note" {
			found = strings.Contains(node.Text, "original samples")
		}
	}
	if !found {
		t.Fatal("live advice suggests fitting a saved capture")
	}
}

func TestNativeSourceSamplingFitKeepsFocusAndRefusesRetainedAction(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := smokeConstructWaveform(s, "finite-pulse-source"); err != nil {
		t.Fatal(err)
	}
	agentAction(t, s.service, "simulation.pause", struct{}{})
	var retained ui.Node
	s.app.Do(func() {
		var ok bool
		retained, ok = findAction(s.view(), "fit-waveform-sampling")
		if !ok {
			t.Fatal("missing coarse fit")
		}
	})
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "20k"})
	before := s.service.Snapshot()
	s.app.Do(func() { retained.Activate() })
	after := s.service.Snapshot()
	if before.TransientSettings != after.TransientSettings || before.ElectricalRevision != after.ElectricalRevision || s.errorMessage == "" {
		t.Fatal("retained Fit ignored intervening source/circuit context")
	}
	uiAction(t, s, "fit-waveform-sampling", nil)
	s.app.Do(func() {
		root := s.view()
		if root.Focus.ID != "setting-step" {
			t.Fatal("valid source Fit did not continue at output interval", root.Focus)
		}
	})
}
