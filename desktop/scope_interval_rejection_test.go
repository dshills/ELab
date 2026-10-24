package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestIntervalParserRefusalCannotRetainAnotherOwnersText(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		for _, change := range []string{"peer-endpoint", "new-owner", "document"} {
			name := "live/" + change
			if recorded {
				name = "recorded/" + change
			}
			t.Run(name, func(t *testing.T) {
				s, cleanup := scopeContextRC(t)
				t.Cleanup(cleanup)
				id := "window-end"
				if recorded {
					recordedCursorSource(t, s.service)
					id = "recorded-window-end"
				}
				var original ui.Node
				s.app.Do(func() {
					state := s.service.Snapshot()
					nodes := s.scopeWindowControls(state)
					if recorded {
						nodes = s.recordingInspector(state)
					}
					root := ui.Node{Children: nodes}
					s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
					var ok bool
					original, ok = findNode(root, id)
					if !ok {
						t.Fatal("missing interval fixture")
					}
				})
				switch change {
				case "peer-endpoint":
					if recorded {
						agentAction(t, s.service, "recording.window.set", workbench.RecordingWindowParams{Start: .004, End: .02})
					} else {
						agentAction(t, s.service, "scope.window.set", workbench.ScopeWindow{Mode: "fixed", Start: .004, End: .02})
					}
				case "new-owner":
					if recorded {
						duration := .02
						agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration})
						completedVerification(t, s.service, s.service.Snapshot().LastOperation)
					} else {
						agentAction(t, s.service, "simulation.restart", struct{}{})
					}
				case "document":
					agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
				}
				before := s.service.Snapshot()
				s.app.Do(func() {
					original.Change(ui.ChangeEvent{Text: "oops"})
					if len(s.intervalTimeRejected) != 0 {
						t.Fatal("stale interval parser failure acquired a new owner")
					}
				})
				if !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("stale interval callback changed accepted state")
				}
			})
		}
	}
}

func TestIntervalResetUsesOnlyItsOwnFieldContinuation(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		for _, intervening := range []bool{false, true} {
			name, id, reset := "live", "window-start", "scope-live-window"
			if recorded {
				name, id, reset = "recorded", "recorded-window-start", "recorded-whole-window"
			}
			if intervening {
				name += "/agent"
			} else {
				name += "/own-field"
			}
			t.Run(name, func(t *testing.T) {
				s, cleanup := scopeContextRC(t)
				t.Cleanup(cleanup)
				if recorded {
					recordedCursorSource(t, s.service)
				}
				var original ui.Node
				s.app.Do(func() {
					state := s.service.Snapshot()
					nodes := s.scopeWindowControls(state)
					if recorded {
						nodes = s.recordingInspector(state)
					}
					root := ui.Node{Children: nodes}
					s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
					field, fieldOK := findNode(root, id)
					button, buttonOK := findNode(root, reset)
					if !fieldOK || !buttonOK {
						t.Fatal("missing original interval controls")
					}
					original = button
					field.Change(ui.ChangeEvent{Text: "4 ms"})
				})
				if s.errorMessage != "" || s.intervalFieldAccepted == nil {
					t.Fatal("fixture requires the field's own accepted continuation")
				}
				if intervening {
					if recorded {
						agentAction(t, s.service, "recording.window.set", workbench.RecordingWindowParams{Start: .006, End: .018})
					} else {
						agentAction(t, s.service, "scope.window.set", workbench.ScopeWindow{Mode: "fixed", Start: .006, End: .018})
					}
				}
				before := s.service.Snapshot()
				s.app.Do(original.Activate)
				after := s.service.Snapshot()
				if intervening {
					if !reflect.DeepEqual(before, after) || s.errorMessage == "" {
						t.Fatal("old reset continued through an intervening agent command")
					}
				} else {
					window := after.Scope.Window
					if recorded {
						window = after.Recording.Scope.Window
					}
					if window.Start != 0 || window.End != .02 || after.CommandRevision != before.CommandRevision+1 || s.errorMessage != "" {
						t.Fatal("reset did not continue its own field", window, s.errorMessage)
					}
					assertCursorMeasurementPreservesPhysics(t, before, after)
				}
			})
		}
	}
}
