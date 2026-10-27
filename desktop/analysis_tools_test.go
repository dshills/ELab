package main

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestLoopSweepInspectorUsesSharedAgentWorkflow(t *testing.T) {
	u, done := testScreen(t)
	defer done()
	a := workbench.New()
	defer a.Close()
	for _, s := range []*workbench.Service{u.service, a} {
		agentAction(t, s, "example.load", workbench.ExampleLoadParams{ID: "feedback-loop"})
		agentAction(t, s, "simulation.pause", struct{}{})
		agentAction(t, s, "simulation.restart", struct{}{})
		awaitInductorCircuit(t, s)
	}
	if u.service.Snapshot().DCSweep.Settings.Parameter != "loop-frequency" {
		t.Fatal("loop example defaults")
	}
	uiAction(t, u, "sweep-start", nil)
	cfg := a.Snapshot().DCSweep.Settings
	agentAction(t, a, "sweep.start", workbench.SweepStartParams{Settings: &cfg})
	left := awaitNativeSweep(t, u)
	right := awaitDesktopLoopAgent(t, a)
	if left.DCSweep.Result.Loop == nil || !reflect.DeepEqual(left.DCSweep.Result, right.DCSweep.Result) {
		t.Fatal("loop user/agent result parity")
	}
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		agentAction(t, u.service, "appearance.theme.set", map[string]string{"theme": theme})
		if err := ui.Validate(u.view()); err != nil {
			t.Fatal(theme, err)
		}
		if _, ok := findNode(u.view(), "loop-margins"); !ok {
			t.Fatal("loop margins not visible")
		}
	}
}
func awaitDesktopLoopAgent(t *testing.T, s *workbench.Service) workbench.Snapshot {
	t.Helper() // Poll the public immutable snapshot with the same bounded deadline as the UI fixture.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		state := s.Snapshot()
		if state.DCSweep != nil {
			for _, op := range state.DCSweep.Operations {
				if op.ID == state.DCSweep.Selected {
					if op.Status == "completed" {
						return state
					}
					if op.Status == "failed" {
						t.Fatalf("agent loop sweep failed: %+v", op.Error)
					}
				}
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("agent loop sweep did not finish")
	return workbench.Snapshot{}
}
func TestRecordedStepInspectorQueryParityAndStaleGuard(t *testing.T) {
	s, done := testScreen(t)
	defer done()
	agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "step-response"})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	duration, step := .02, .00002
	agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:C1"}})
	completedVerification(t, s.service, s.service.Snapshot().LastOperation)
	uiAction(t, s, "recording-show", nil)
	before := s.service.Snapshot()
	uiAction(t, s, "step-response-probe", &ui.ChangeEvent{SelectedID: "v:C1"})
	uiAction(t, s, "step-response-final", &ui.ChangeEvent{Number: 5})
	after := s.service.Snapshot()
	result := after.Recording.StepResponse
	if result == nil || result.Settling == nil || before.Document != after.Document || before.RunID != after.RunID || !reflect.DeepEqual(before.Transient, after.Transient) {
		t.Fatal("step measurement missing or live changed", result)
	}
	raw, err := json.Marshal(workbench.StepResponseParams(result.Settings))
	if err != nil {
		t.Fatal(err)
	}
	r := s.service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "step-query", Command: "recording.step-response", Params: raw}, "agent")
	if r.Error != nil || !reflect.DeepEqual(result, r.Result) {
		t.Fatal("step UI/agent query mismatch", r.Error)
	}
	held, ok := findNode(s.view(), "step-response-final")
	if !ok || held.Change == nil {
		t.Fatal("step field missing")
	}
	cfg := workbench.StepResponseParams(result.Settings)
	cfg.Tolerance = .05
	agentAction(t, s.service, "recording.step-response.set", cfg)
	current := s.service.Snapshot()
	s.app.Do(func() { held.Change(ui.ChangeEvent{Number: 4}) })
	if !reflect.DeepEqual(current, s.service.Snapshot()) {
		t.Fatal("stale step reference overwrote agent")
	}
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
		if err := ui.Validate(s.view()); err != nil {
			t.Fatal(theme, err)
		}
	}
}
