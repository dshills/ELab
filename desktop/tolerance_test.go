package main

import (
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
	"reflect"
	"testing"
	"time"
)

func waitNativeTolerance(t *testing.T, s *workbench.Service) *workbench.ToleranceState {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		v := s.Snapshot().Tolerance
		if v != nil && v.Status == "completed" {
			return v
		}
		if v != nil && v.Status == "failed" {
			t.Fatal(v.Failure)
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("tolerance run did not complete")
	return nil
}
func TestToleranceNativeControlsAndAgentParity(t *testing.T) {
	u, cleanup := testScreen(t)
	defer cleanup()
	a := workbench.New()
	defer a.Close()
	for _, s := range []*workbench.Service{u.service, a} {
		agentAction(t, s, "example.load", workbench.ExampleLoadParams{ID: "rc-tolerance"})
		agentAction(t, s, "simulation.pause", struct{}{})
		agentAction(t, s, "simulation.restart", struct{}{})
		state, err := s.Wait(t.Context())
		if err != nil || state.Transient == nil {
			t.Fatal("RC circuit did not initialize", err, state.Diagnostics)
		}
	}
	uiAction(t, u, "tolerance-resistance", &ui.ChangeEvent{Number: 10})
	p := a.Snapshot().Tolerance.Settings
	p.Resistance = .1
	agentAction(t, a, "tolerance.settings.set", p)
	before := u.service.Snapshot()
	uiAction(t, u, "tolerance-run", nil)
	agentAction(t, a, "tolerance.run", struct{}{})
	left, right := waitNativeTolerance(t, u.service), waitNativeTolerance(t, a)
	if left.Result == nil || right.Result == nil || !reflect.DeepEqual(left.Result.Trials, right.Result.Trials) || !reflect.DeepEqual(left.Result.Statistics, right.Result.Statistics) || !reflect.DeepEqual(before.Transient, u.service.Snapshot().Transient) {
		t.Fatal("shared tolerance workflow/live state")
	}
	held, ok := findNode(u.view(), "tolerance-run")
	if !ok {
		t.Fatal("missing Run")
	}
	uiAction(t, u, "tolerance-seed", &ui.ChangeEvent{Text: "9007199254740991"})
	if u.service.Snapshot().Tolerance.Settings.Seed != 9007199254740991 {
		t.Fatal("seed text lost exact integer")
	}
	beforeStale := u.service.Snapshot()
	u.app.Do(func() { held.Activate() })
	afterStale := u.service.Snapshot()
	if !reflect.DeepEqual(beforeStale.Tolerance, afterStale.Tolerance) {
		t.Fatal("stale Run replaced newer settings")
	}
	beforeInvalid := u.service.Snapshot().Tolerance
	entry, ok := findNode(u.view(), "tolerance-trials")
	if !ok {
		t.Fatal("missing trial entry")
	}
	u.app.Do(func() { entry.Change(ui.ChangeEvent{Text: "15.9"}) })
	if !reflect.DeepEqual(beforeInvalid, u.service.Snapshot().Tolerance) {
		t.Fatal("fractional trial count silently truncated")
	}
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		agentAction(t, u.service, "appearance.theme.set", map[string]string{"theme": theme})
		if err := ui.Validate(u.view()); err != nil {
			t.Fatal(theme, err)
		}
	}
}
