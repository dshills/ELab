//go:build offscreen

package main

import (
	"image"
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

// Keep the real assembled Sheet and its separate modal scene. This exercises
// the driver's modal routes, without claiming physical macOS queue acceptance.
func mountedExampleSheet(t *testing.T, s *screen, focus string) *mountedControlWindow {
	t.Helper()
	return newMountedControlWindowSize(t, s, func() ui.Node {
		full := s.view()
		sheet, ok := findNode(full, exampleBrowserID)
		if !ok || !sheet.Sheet {
			t.Fatal("actual assembled sheet missing")
		}
		opener, _ := findNode(full, "examples-menu")
		search, _ := findNode(full, "part-search")
		root := ui.Node{ID: "root", Kind: ui.Column, Grow: true, Theme: full.Theme, Children: []ui.Node{opener, search, sheet}}
		if sheet.Open {
			root.Focus = ui.FocusRequest{ID: focus, Sequence: 1}
		}
		return root
	}, 1000, 900, inspectorLayoutTestTheme())
}

func TestMountedExampleSheetActionsAcrossAgentEdits(t *testing.T) {
	for _, action := range []string{"escape", "confirm", "cancel"} {
		for _, interleave := range []string{"draft", "replace", "component"} {
			t.Run(action+"/"+interleave, func(t *testing.T) {
				s, cleanup := testScreen(t)
				defer cleanup()
				agentAction(t, s.service, "example.reset", workbench.ReplaceParams{})
				agentAction(t, s.service, "simulation.pause", struct{}{})
				agentAction(t, s.service, "examples.visibility.set", workbench.ScopeVisibilityParams{Visible: true})
				seedExampleBrowserImage(t, s)
				focus := ui.SheetConfirmID(exampleBrowserID)
				code := key.CodeReturnEnter
				if action == "cancel" {
					focus = ui.SheetCancelID(exampleBrowserID)
				} else if action == "escape" {
					focus, code = "example-search", key.CodeEscape
				}
				m := mountedExampleSheet(t, s, focus)
				// Shown/default focus belongs to the actual modal scene. Reach
				// the target through real Tab routing after that initialization.
				for n := 0; m.snapshot().FocusedID != focus && n < 8; n++ {
					m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeTab})
				}
				if m.snapshot().FocusedID != focus {
					t.Fatal("fixture could not Tab to actual modal target", m.snapshot().FocusedID, focus)
				}
				before := s.service.Snapshot()
				if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.KeyChord, Code: code}); err != nil {
					t.Fatal(err)
				}
				step, err := m.manual.Step(m.ctx, 1)
				if err != nil || step.Events != 1 {
					t.Fatal("modal input boundary unavailable", step, err)
				}
				atBoundary := s.service.Snapshot()
				synchronous := !reflect.DeepEqual(before, atBoundary)
				if !synchronous && s.app.Pending().Ordinary == 0 {
					t.Fatal("no immediate effect or deferred modal action", step)
				}
				t.Logf("boundary synchronous=%v events=%d callbacks=%d pending=%d revision=%d->%d", synchronous, step.Events, step.OwnerCallbacks, s.app.Pending().Ordinary, before.CommandRevision, atBoundary.CommandRevision)
				if interleave == "draft" {
					agentAction(t, s.service, "editor.value.set", workbench.DraftParams{Value: "2k"})
				} else if interleave == "component" {
					agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
				} else {
					agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
				}
				agent := s.service.Snapshot()
				if err := s.app.Invalidate("main"); err != nil {
					t.Fatal(err)
				}
				if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.MouseMove, Position: image.Pt(980, 880)}); err != nil {
					t.Fatal(err)
				}
				m.settle()
				after := s.service.Snapshot()
				if !reflect.DeepEqual(agent, after) || s.pendingCommand != "" {
					t.Fatalf("old modal %s changed agent context (synchronous=%v): revision %d->%d generation %d->%d browser %v->%v draft %q->%q pending=%q", action, synchronous, agent.CommandRevision, after.CommandRevision, agent.Document.Generation, after.Document.Generation, agent.ExampleBrowser.Visible, after.ExampleBrowser.Visible, agent.Drawing.Value, after.Drawing.Value, s.pendingCommand)
				}
				// Fresh explicit user intent must still work after an older
				// callback refuses, or after the earlier synchronous dismissal.
				if !after.ExampleBrowser.Visible {
					agentAction(t, s.service, "examples.visibility.set", workbench.ScopeVisibilityParams{Visible: true})
					m.frame()
				}
				for n := 0; m.snapshot().FocusedID != focus && n < 8; n++ {
					m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeTab})
				}
				if m.snapshot().FocusedID != focus {
					t.Fatal("fresh retry could not reach its modal control")
				}
				freshBefore := s.service.Snapshot()
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: code})
				fresh := s.service.Snapshot()
				if action == "confirm" {
					if interleave == "component" {
						if s.pendingCommand != "example.load" || !reflect.DeepEqual(freshBefore, fresh) {
							t.Fatal("fresh confirmation lost dirty-document protection")
						}
					} else if fresh.ExampleBrowser.Visible || fresh.Document.Generation != freshBefore.Document.Generation+1 || fresh.Example == nil || fresh.LastActor != "user" {
						t.Fatal("fresh confirmation failed to load its selected example")
					}
				} else if fresh.ExampleBrowser.Visible || fresh.CommandRevision != freshBefore.CommandRevision+1 || fresh.Document != freshBefore.Document || fresh.Drawing != freshBefore.Drawing || !reflect.DeepEqual(fresh.Components, freshBefore.Components) || !reflect.DeepEqual(fresh.Wires, freshBefore.Wires) || fresh.LastActor != "user" {
					t.Fatal("fresh dismissal failed or changed circuit/document/draft")
				}
			})
		}
	}
}
