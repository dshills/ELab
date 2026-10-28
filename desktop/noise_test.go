package main

import (
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNoiseSweepUserAgentParityAndTheme(t *testing.T) {
	s, done := testScreen(t)
	defer done()
	a := workbench.New()
	defer a.Close()
	for _, service := range []*workbench.Service{s.service, a} {
		agentAction(t, service, "example.load", workbench.ExampleLoadParams{ID: "noise-rc"})
		agentAction(t, service, "simulation.pause", struct{}{})
		awaitBlockRC(t, service)
	}
	before := s.service.Snapshot()
	cfg := before.DCSweep.Settings
	uiAction(t, s, "sweep-start", nil)
	state := awaitNativeSweep(t, s)
	agentAction(t, a, "sweep.start", workbench.SweepStartParams{Settings: &cfg})
	deadline := time.Now().Add(3 * time.Second)
	for a.Snapshot().DCSweep.Result == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if a.Snapshot().DCSweep.Result == nil {
		t.Fatal("agent noise sweep timed out")
	}
	if state.DCSweep.Result == nil || state.DCSweep.Result.Noise == nil || !reflect.DeepEqual(state.DCSweep.Result, a.Snapshot().DCSweep.Result) {
		t.Fatal("noise user/agent parity")
	}
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
		var viewErr error
		s.app.Do(func() {
			viewErr = ui.Validate(s.view())
			if viewErr == nil {
				_, viewErr = s.sweepMesh(*s.sweepDrawing.View)
			}
		})
		if viewErr != nil {
			t.Fatal(theme, viewErr)
		}
	}
	nodes := noiseReadouts(state.DCSweep.Result)
	if len(nodes) < 5 {
		t.Fatal("noise results absent")
	}
}
