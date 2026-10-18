//go:build offscreen

package main

import (
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func mountedLiveWindowField(t *testing.T, s *screen, id string) *mountedControlWindow {
	t.Helper()
	return newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		controls := ui.Node{ID: "window-controls", Kind: ui.Column, Children: s.scopeWindowControls(state)}
		field, ok := findNode(controls, id)
		if !ok || field.Disabled {
			t.Fatal("missing enabled live interval field", id)
		}
		field.MinWidth, field.MinHeight = 190, 40
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: id, Sequence: 1}, Children: []ui.Node{field}}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
}

func TestMountedScopeCursorWindowDraftBelongsToLiveRun(t *testing.T) {
	for _, side := range []string{"start", "end"} {
		for _, change := range []string{"repaint", "sample", "restart"} {
			t.Run(side+"/"+change, func(t *testing.T) {
				s, cleanup := scopeContextRC(t)
				t.Cleanup(cleanup)
				agent := workbench.New()
				t.Cleanup(agent.Close)
				agentAction(t, agent, "example.rc", workbench.ReplaceParams{})
				agentAction(t, agent, "simulation.pause", struct{}{})
				agentAction(t, agent, "simulation.restart", struct{}{})
				agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .02})
				window := s.service.Snapshot().Scope.Window
				window.Mode = "fixed"
				for _, service := range []*workbench.Service{s.service, agent} {
					agentAction(t, service, "scope.window.set", window)
				}
				id := "window-" + side
				m := mountedLiveWindowField(t, s, id)
				original := mountedControlElement(t, m, id)
				before := s.service.Snapshot()
				typeMountedCursor(t, m, id, "8 ms")
				if !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("fixture lacks private uncommitted interval input")
				}
				for _, service := range []*workbench.Service{s.service, agent} {
					switch change {
					case "sample":
						agentAction(t, service, "simulation.step", workbench.StepParams{Duration: .001})
					case "restart":
						agentAction(t, service, "simulation.restart", struct{}{})
						agentAction(t, service, "simulation.step", workbench.StepParams{Duration: .02})
					}
				}
				current := s.service.Snapshot()
				value := func(state workbench.Snapshot) float64 {
					if side == "start" {
						return state.Scope.Window.Start
					}
					return state.Scope.Window.End
				}
				if (current.RunID != before.RunID) != (change == "restart") || current.Document.Generation != before.Document.Generation || value(current) != value(before) {
					t.Fatal("fixture changed controlled text or the wrong owner identity")
				}
				m.frame()
				field := mountedControlElement(t, m, id)
				t.Logf("run %d→%d; incarnation %d→%d; field=%v", before.RunID, current.RunID, original.Incarnation, field.Incarnation, field.Value)
				clickMountedControl(t, m, id)
				if m.snapshot().FocusedID != id {
					t.Fatal("interval field did not receive focus before Enter", id)
				}
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
				after := s.service.Snapshot()
				if change == "restart" {
					if !reflect.DeepEqual(current, after) {
						t.Fatalf("old private interval text changed a restarted run: field=%s run=%d→%d interval=%+v revision=%d→%d error=%s", id, before.RunID, current.RunID, after.Scope.Window, current.CommandRevision, after.CommandRevision, s.errorMessage)
					}
					if s.errorMessage != "" {
						t.Fatal("discarding the old-run draft reported an error", s.errorMessage)
					}
					if field.Incarnation == original.Incarnation || field.Value != timeText(value(current)) {
						t.Fatal("new run retained old private interval state", field)
					}
					typeMountedCursor(t, m, id, "6 ms")
					m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
					after = s.service.Snapshot()
				} else if field.Incarnation != original.Incarnation {
					t.Fatal("same-run repaint or sample remounted unfinished input")
				}
				want := .008
				if change == "restart" {
					want = .006
				}
				window = agent.Snapshot().Scope.Window
				window.Mode = "fixed"
				if side == "start" {
					window.Start = want
				} else {
					window.End = want
				}
				agentAction(t, agent, "scope.window.set", window)
				if value(after) != want || after.CommandRevision != current.CommandRevision+1 || s.errorMessage != "" || !reflect.DeepEqual(after.Scope, agent.Snapshot().Scope) {
					t.Fatal("fresh interval input differs from public agent", s.errorMessage, after.Scope.Window)
				}
				assertCursorMeasurementPreservesPhysics(t, current, after)
			})
		}
	}
}
