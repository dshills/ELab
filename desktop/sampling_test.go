package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func prepareLCSamplingScreen(t *testing.T, s *screen, minimum bool) {
	t.Helper()
	agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "rlc"})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	coil, cap := "10 mH", "1 µF"
	if minimum {
		coil, cap = "1 µH", "1 nF"
	}
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "L1", Value: coil})
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "C1", Value: cap})
	agentAction(t, s.service, "component.select", workbench.IDParams{})
}

func TestNativeLCSamplingFitMatchesAgentWithoutSelectedComponent(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	prepareLCSamplingScreen(t, user, false)
	agent := workbench.New()
	defer agent.Close()
	for _, call := range []struct {
		command string
		params  any
	}{
		{"example.load", workbench.ExampleLoadParams{ID: "rlc"}},
		{"simulation.pause", struct{}{}},
		{"component.set", workbench.ValueParams{ID: "L1", Value: "10 mH"}},
		{"component.set", workbench.ValueParams{ID: "C1", Value: "1 µF"}},
		{"component.select", workbench.IDParams{}},
	} {
		agentAction(t, agent, call.command, call.params)
	}
	uiAction(t, user, "fit-circuit-sampling", nil)
	agentAction(t, agent, "simulation.sampling.fit", struct{}{})
	u, a := user.service.Snapshot(), agent.Snapshot()
	if u.Selected != "" || u.TransientSettings != a.TransientSettings || !reflect.DeepEqual(u.LCSampling, a.LCSampling) || u.LCSampling.Unresolved || !u.Paused || !reflect.DeepEqual(u.Components, a.Components) {
		t.Fatal("native fit differs from the shared agent action")
	}
	user.app.Do(func() {
		root := user.view()
		if err := ui.Validate(root); err != nil {
			t.Fatal(err)
		}
		if _, found := findAction(root, "fit-circuit-sampling"); found {
			t.Fatal("resolved hint retained a redundant fit button")
		}
		if root.Focus.ID != "setting-step" {
			t.Fatal("fit lost keyboard focus when its button disappeared", root.Focus)
		}
		if _, ok := ui.FocusTarget(root, "setting-step"); !ok {
			t.Fatal("sampling focus continuation names an unavailable field")
		}
	})
}

func TestNativeLCSamplingRetainedButtonRefusesAgentChanges(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	prepareLCSamplingScreen(t, s, false)
	var retained ui.Node
	s.app.Do(func() {
		var found bool
		retained, found = findAction(s.view(), "fit-circuit-sampling")
		if !found {
			t.Fatal("missing original fit action")
		}
	})
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "L1", Value: "20 mH"})
	before := s.service.Snapshot()
	focus := s.keyboardFocus
	s.app.Do(func() { retained.Activate() })
	after := s.service.Snapshot()
	if after.TransientSettings != before.TransientSettings || after.ElectricalRevision != before.ElectricalRevision || s.errorMessage == "" || s.keyboardFocus != focus {
		t.Fatal("stale sampling action changed the agent's experiment or stole focus")
	}
}

func TestNativeLCSamplingMinimumAndRecordedContextAreExplicit(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	prepareLCSamplingScreen(t, s, true)
	uiAction(t, s, "fit-circuit-sampling", nil)
	state := s.service.Snapshot()
	if !state.LCSampling.MinimumLimited || !state.LCSampling.Unresolved || state.LCSampling.CanFit {
		t.Fatal("minimum warning falsely resolved")
	}
	s.app.Do(func() {
		root := s.view()
		if _, found := findAction(root, "fit-circuit-sampling"); found {
			t.Fatal("minimum step retained an ineffective fit button")
		}
		text, found := findAction(root, "lc-sampling-summary")
		if !found || !strings.Contains(text.Text, "cannot fully resolve") {
			t.Fatal("native minimum limitation is missing")
		}
	})
	state.Recording = &workbench.RecordingView{Viewing: true}
	nodes := s.lcSamplingControls(state)
	if nodes[len(nodes)-1].ID != "lc-sampling-recorded-note" || !strings.Contains(nodes[len(nodes)-1].Text, "original samples") {
		t.Fatal("live fit was presented as repairing a saved recording")
	}
}

func TestNativeLCAndSourceHintsOfferOneFitAction(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	prepareLCSamplingScreen(t, s, false)
	agentAction(t, s.service, "object.delete", workbench.IDParams{ID: "V1"})
	agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: "VSINE", Kind: "sine-source", Position: workbench.Point{X: -240, Y: 0}})
	cfg := workbench.DefaultSineWaveform()
	cfg.Frequency = 1000
	agentAction(t, s.service, "source.sine.set", workbench.SineParams{ID: "VSINE", Waveform: cfg})
	s.app.Do(func() {
		root := s.view()
		if _, found := findAction(root, "fit-waveform-sampling"); found {
			t.Fatal("combined hints duplicated the same fit action")
		}
		if _, found := findAction(root, "fit-circuit-sampling"); !found {
			t.Fatal("combined source/LC fit is unavailable")
		}
	})
}
