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

func mountedIntervalField(t *testing.T, s *screen, id string, recorded bool) *mountedControlWindow {
	t.Helper()
	return newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		nodes := s.scopeWindowControls(state)
		if recorded {
			nodes = s.recordingInspector(state)
		}
		field, ok := findNode(ui.Node{Children: nodes}, id)
		if !ok || field.Disabled {
			t.Fatal("missing enabled interval field", id)
		}
		field.MinWidth, field.MinHeight = 190, 40
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: id, Sequence: 1}, Children: []ui.Node{field}}
		peerID := "window-start"
		if id == peerID {
			peerID = "window-end"
		}
		if recorded {
			peerID = "recorded-window-start"
			if id == peerID {
				peerID = "recorded-window-end"
			}
		}
		peer, ok := findNode(ui.Node{Children: nodes}, peerID)
		if !ok {
			t.Fatal("missing interval peer fixture", peerID)
		}
		peer.MinWidth, peer.MinHeight = 190, 40
		root.Children = append(root.Children, peer)
		resetID := "scope-live-window"
		if recorded {
			resetID = "recorded-whole-window"
		}
		reset, ok := findNode(ui.Node{Children: nodes}, resetID)
		if !ok {
			t.Fatal("missing interval reset fixture", resetID)
		}
		root.Children = append(root.Children, reset)
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
}

func TestMountedScopeCursorIntervalCorrectionPreservesRejectedPeer(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		name, startID, endID := "live", "window-start", "window-end"
		if recorded {
			name, startID, endID = "recorded", "recorded-window-start", "recorded-window-end"
		}
		t.Run(name, func(t *testing.T) {
			s, cleanup := scopeContextRC(t)
			t.Cleanup(cleanup)
			if recorded {
				recordedCursorSource(t, s.service)
			} else {
				agentAction(t, s.service, "scope.window.set", workbench.ScopeWindow{Mode: "fixed", End: .02})
			}
			m := mountedIntervalField(t, s, startID, recorded)
			for _, id := range []string{startID, endID} {
				typeMountedCursor(t, m, id, "oops")
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
				m.frame()
			}
			before := s.service.Snapshot()
			if len(s.intervalTimeRejected) != 2 {
				t.Fatal("fixture requires two independently rejected fields")
			}
			typeMountedCursor(t, m, startID, "0 s")
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			m.frame()
			after := s.service.Snapshot()
			if mountedControlElement(t, m, endID).Value != "oops" || len(s.intervalTimeRejected) != 1 || after.CommandRevision != before.CommandRevision+1 {
				t.Fatal("correcting one endpoint erased its peer's unchanged private entry")
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
		})
	}
}

func TestMountedScopeCursorIntervalExplicitResetClearsRejectedText(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		name, id, reset := "live", "window-end", "scope-live-window"
		if recorded {
			name, id, reset = "recorded", "recorded-window-end", "recorded-whole-window"
		}
		t.Run(name, func(t *testing.T) {
			s, cleanup := scopeContextRC(t)
			t.Cleanup(cleanup)
			if recorded {
				recordedCursorSource(t, s.service)
			}
			m := mountedIntervalField(t, s, id, recorded)
			typeMountedCursor(t, m, id, "oops")
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			m.frame()
			before := s.service.Snapshot()
			if len(s.intervalTimeRejected) != 1 || s.errorMessage == "" {
				t.Fatal("fixture needs a retained refusal")
			}
			clickMountedControl(t, m, reset)
			m.frame()
			after := s.service.Snapshot()
			if mountedControlElement(t, m, id).Value != "20 ms" || len(s.intervalTimeRejected) != 0 || s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 {
				t.Fatal("explicit reset retained a rejected interval entry", s.errorMessage)
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
		})
	}
}

func TestMountedScopeCursorIntervalResetAppliesOnFirstClickWithPrivateEntry(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		for _, text := range []string{"4 ms", "oops"} {
			name, id, reset := "live/"+text, "window-start", "scope-live-window"
			if recorded {
				name, id, reset = "recorded/"+text, "recorded-window-start", "recorded-whole-window"
			}
			t.Run(name, func(t *testing.T) {
				s, cleanup := scopeContextRC(t)
				t.Cleanup(cleanup)
				if recorded {
					recordedCursorSource(t, s.service)
				}
				m := mountedIntervalField(t, s, id, recorded)
				before := s.service.Snapshot()
				typeMountedCursor(t, m, id, text)
				if !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("fixture requires a private uncommitted interval entry")
				}
				clickMountedControl(t, m, reset)
				m.frame()
				after := s.service.Snapshot()
				window := after.Scope.Window
				if recorded {
					window = after.Recording.Scope.Window
				}
				if s.errorMessage != "" || len(s.intervalTimeRejected) != 0 || window.Start != 0 || window.End != .02 || mountedControlElement(t, m, id).Value != "0 s" {
					t.Fatal("reset required another click after private interval entry", window, s.errorMessage)
				}
				assertCursorMeasurementPreservesPhysics(t, before, after)
			})
		}
	}
}

func TestMountedScopeCursorIntervalHeldResetCannotCrossAnAgentEdit(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		name, id, reset := "live", "window-start", "scope-live-window"
		if recorded {
			name, id, reset = "recorded", "recorded-window-start", "recorded-whole-window"
		}
		t.Run(name, func(t *testing.T) {
			s, cleanup := scopeContextRC(t)
			t.Cleanup(cleanup)
			if recorded {
				recordedCursorSource(t, s.service)
			}
			m := mountedIntervalField(t, s, id, recorded)
			typeMountedCursor(t, m, id, "4 ms")
			before := s.service.Snapshot()
			button := mountedControlElement(t, m, reset)
			at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
			t.Logf("revision before/after press: %d/%d; do not assume blur precedes release", before.CommandRevision, s.service.Snapshot().CommandRevision)
			if recorded {
				agentAction(t, s.service, "recording.window.set", workbench.RecordingWindowParams{Start: .006, End: .018})
			} else {
				agentAction(t, s.service, "scope.window.set", workbench.ScopeWindow{Mode: "fixed", Start: .006, End: .018})
			}
			current := s.service.Snapshot()
			m.frame()
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
			if !reflect.DeepEqual(current, s.service.Snapshot()) {
				t.Fatal("held interval reset silently replaced the agent's interval")
			}
			m.frame()
			clickMountedControl(t, m, reset)
			after := s.service.Snapshot()
			window := after.Scope.Window
			if recorded {
				window = after.Recording.Scope.Window
			}
			if s.errorMessage != "" || after.CommandRevision != current.CommandRevision+1 || window.Start != 0 || window.End != .02 {
				t.Fatal("fresh reset after refusal did not work", window, s.errorMessage)
			}
			assertCursorMeasurementPreservesPhysics(t, current, after)
		})
	}
}

func TestMountedScopeCursorIntervalRefusalRetainsTextAndRecovers(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		for _, side := range []string{"start", "end"} {
			name, id := "live/"+side, "window-"+side
			if recorded {
				name, id = "recorded/"+side, "recorded-window-"+side
			}
			t.Run(name, func(t *testing.T) {
				s, cleanup := scopeContextRC(t)
				t.Cleanup(cleanup)
				if recorded {
					recordedCursorSource(t, s.service)
				}
				m := mountedIntervalField(t, s, id, recorded)
				before := s.service.Snapshot()
				bad, corrected := "-1 ms", "0 s"
				if side == "end" {
					bad, corrected = "30 ms", "20 ms"
				}
				typeMountedCursor(t, m, id, bad)
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
				m.frame()
				if !reflect.DeepEqual(before, s.service.Snapshot()) || s.errorMessage == "" {
					t.Fatal("invalid interval changed accepted state or was not refused", s.errorMessage)
				}
				if field := mountedControlElement(t, m, id); field.Value != bad {
					t.Fatalf("rejected interval %q disappeared before correction: field=%q", bad, field.Value)
				}
				// Correcting back to the accepted value must still submit once;
				// otherwise the earlier refusal stays visible indefinitely.
				typeMountedCursor(t, m, id, corrected)
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
				after := s.service.Snapshot()
				if s.errorMessage != "" || s.requestHelp != nil || after.CommandRevision != before.CommandRevision+1 {
					t.Fatal("first corrected commit did not recover", s.errorMessage, before.CommandRevision, after.CommandRevision)
				}
				assertCursorMeasurementPreservesPhysics(t, before, after)
				agent := workbench.New()
				t.Cleanup(agent.Close)
				agentAction(t, agent, "example.rc", workbench.ReplaceParams{})
				agentAction(t, agent, "simulation.pause", struct{}{})
				agentAction(t, agent, "simulation.restart", struct{}{})
				agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .02})
				if recorded {
					recordedCursorSource(t, agent)
				}
				text, start, end := "8 ms", .008, .02
				if side == "end" {
					text, start, end = "12 ms", 0, .012
				}
				typeMountedCursor(t, m, id, text)
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
				current := s.service.Snapshot()
				if recorded {
					agentAction(t, agent, "recording.window.set", workbench.RecordingWindowParams{Start: start, End: end})
					if !reflect.DeepEqual(current.Recording.Scope, agent.Snapshot().Recording.Scope) {
						t.Fatal("corrected captured interval differs from the public agent")
					}
				} else {
					agentAction(t, agent, "scope.window.set", workbench.ScopeWindow{Mode: "fixed", Start: start, End: end})
					if !reflect.DeepEqual(current.Scope, agent.Snapshot().Scope) {
						t.Fatal("corrected live interval differs from the public agent")
					}
				}
				if s.errorMessage != "" || current.CommandRevision != after.CommandRevision+1 || len(s.intervalTimeRejected) != 0 {
					t.Fatal("fresh interval entry did not apply exactly once", s.errorMessage)
				}
				assertCursorMeasurementPreservesPhysics(t, after, current)
			})
		}
	}
}

func TestMountedScopeCursorIntervalRefusalStaysWithItsOwner(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		for _, side := range []string{"start", "end"} {
			for _, change := range []string{"repaint", "samples", "peer-endpoint", "restart", "new-capture"} {
				if !recorded && change == "new-capture" {
					continue
				}
				name, id := "live/"+side+"/"+change, "window-"+side
				if recorded {
					name, id = "recorded/"+side+"/"+change, "recorded-window-"+side
				}
				t.Run(name, func(t *testing.T) {
					s, cleanup := scopeContextRC(t)
					t.Cleanup(cleanup)
					if recorded {
						recordedCursorSource(t, s.service)
					}
					m := mountedIntervalField(t, s, id, recorded)
					typeMountedCursor(t, m, id, "oops")
					m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
					m.frame()
					switch change {
					case "samples":
						agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .001})
					case "peer-endpoint":
						start, end := 0., .016
						if side == "end" {
							start, end = .004, .02
						}
						if recorded {
							agentAction(t, s.service, "recording.window.set", workbench.RecordingWindowParams{Start: start, End: end})
						} else {
							agentAction(t, s.service, "scope.window.set", workbench.ScopeWindow{Mode: "fixed", Start: start, End: end})
						}
					case "restart":
						agentAction(t, s.service, "simulation.restart", struct{}{})
					case "new-capture":
						duration := .02
						agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration})
						completedVerification(t, s.service, s.service.Snapshot().LastOperation)
					}
					before := s.service.Snapshot()
					m.frame()
					retain := change == "repaint" || change == "samples" || recorded && change == "restart"
					if field := mountedControlElement(t, m, id); (field.Value == "oops") != retain || (len(s.intervalTimeRejected) == 1) != retain {
						t.Fatal("rejected interval followed another owner or lost same-owner input", retain, field.Value)
					}
					if !reflect.DeepEqual(before, s.service.Snapshot()) {
						t.Fatal("rendering interval rejection changed accepted state")
					}
				})
			}
		}
	}
}
