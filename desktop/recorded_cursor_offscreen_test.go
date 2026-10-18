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

func TestMountedScopeCursorRecordedStartupMatchesPublicAgent(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	agent := workbench.New()
	t.Cleanup(agent.Close)
	for _, service := range []*workbench.Service{s.service, agent} {
		recordedCursorSource(t, service)
	}
	m := mountedRecordedCursorControls(t, s)
	before := s.service.Snapshot()
	if before.Recording.Scope.A != nil || before.Recording.Scope.B != nil {
		t.Fatal("fixture already has a cursor")
	}
	typeMountedCursor(t, m, "recorded-cursor-a", "0 s")
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("private typing changed recorded measurements")
	}
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
	agentAction(t, agent, "recording.cursor.set", workbench.RecordingCursorParams{OperationID: agent.Snapshot().Recording.ID, Cursor: "a", Time: 0})
	after := s.service.Snapshot()
	if after.Recording.Scope.A == nil || !after.Recording.Scope.A.Valid || after.Recording.Scope.A.Time != 0 || after.CommandRevision != before.CommandRevision+1 || !reflect.DeepEqual(after.Recording.Scope, agent.Snapshot().Recording.Scope) {
		t.Fatalf("typing startup time did not place the same cursor as the agent: cursor=%+v revision=%d→%d error=%s", after.Recording.Scope.A, before.CommandRevision, after.CommandRevision, s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func mountedRecordedCursorControls(t *testing.T, s *screen) *mountedControlWindow {
	t.Helper()
	m := newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		controls := s.recordingCursors(state)
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: "recorded-cursor-a", Sequence: 1}}
		for i := 0; i < 6; i += 3 {
			row := ui.Node{ID: controls.Children[i].ID + "-row", Kind: ui.Row}
			for _, node := range controls.Children[i : i+3] {
				if node.Kind == ui.TextField {
					node.MinWidth, node.MinHeight = 190, 40
				}
				row.Children = append(row.Children, node)
			}
			root.Children = append(root.Children, row)
		}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
	s.settleField = func(id string, done func(error)) error {
		return cogent.SettleTextField(m.ctx, s.app, "main", id, done)
	}
	return m
}

func TestMountedScopeCursorRecordedFocusLossClearUsesOneClick(t *testing.T) {
	for _, action := range []struct{ name, button, command, cursor string }{
		{"clear", "recorded-cursor-clear-a", "recording.cursor.clear", "a"},
		{"select", "recorded-cursor-select-b", "recording.cursor.select", "b"},
	} {
		t.Run(action.name, func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			agent := workbench.New()
			t.Cleanup(agent.Close)
			recordedCursorSource(t, agent)
			for _, service := range []*workbench.Service{s.service, agent} {
				agentAction(t, service, "recording.cursor.set", workbench.RecordingCursorParams{OperationID: service.Snapshot().Recording.ID, Cursor: "a", Time: .004})
			}
			m := mountedRecordedCursorControls(t, s)
			before := s.service.Snapshot()
			typeMountedCursor(t, m, "recorded-cursor-a", "8 ms")
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("private typing changed captured state")
			}
			clickMountedControl(t, m, action.button)
			agentAction(t, agent, "recording.cursor.set", workbench.RecordingCursorParams{OperationID: agent.Snapshot().Recording.ID, Cursor: "a", Time: .008})
			agentAction(t, agent, action.command, workbench.CursorSelectParams{Cursor: action.cursor})
			after := s.service.Snapshot()
			if after.CommandRevision != before.CommandRevision+2 || s.errorMessage != "" || !reflect.DeepEqual(after.Recording.Scope, agent.Snapshot().Recording.Scope) {
				t.Fatalf("one %s click did not settle and match the agent: cursor=%+v active=%s revision=%d→%d error=%s", action.name, after.Recording.Scope.A, after.Recording.Scope.ActiveCursor, before.CommandRevision, after.CommandRevision, s.errorMessage)
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
		})
	}
}

func mountedOwnedRecordedField(t *testing.T, s *screen, id string) *mountedControlWindow {
	t.Helper()
	return newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		var controls ui.Node
		switch id {
		case "recorded-cursor-a":
			controls = s.recordingCursors(state)
		case "recorded-time":
			controls = s.recordingPanel(state)
		default:
			controls = ui.Node{ID: "recorded-inspector", Kind: ui.Column, Children: s.recordingInspector(state)}
		}
		field, ok := findNode(controls, id)
		if !ok || field.Disabled {
			t.Fatal("missing enabled recorded field", id)
		}
		field.MinWidth, field.MinHeight = 190, 40
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: id, Sequence: 1}, Children: []ui.Node{field}}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
}

func TestMountedScopeCursorRecordedOwnerDropsOnlyAnotherCapturesDraft(t *testing.T) {
	for _, id := range []string{"recorded-cursor-a", "recorded-time", "recorded-window-start"} {
		t.Run(id, func(t *testing.T) {
			for _, replace := range []bool{false, true} {
				name := "same-capture-repaint"
				if replace {
					name = "agent-selected-capture"
				}
				t.Run(name, func(t *testing.T) {
					s, cleanup := recordedCursorRC(t)
					t.Cleanup(cleanup)
					first := s.service.Snapshot().Recording.ID
					duration := .02
					agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration})
					completedVerification(t, s.service, s.service.Snapshot().LastOperation)
					second := s.service.Snapshot().Recording.ID
					if first == second {
						t.Fatal("fixture did not create two captures")
					}
					agentAction(t, s.service, "operation.select", workbench.IDParams{ID: first})
					m := mountedOwnedRecordedField(t, s, id)
					original := mountedControlElement(t, m, id)
					before := s.service.Snapshot()
					typeMountedCursor(t, m, id, "8 ms")
					if !reflect.DeepEqual(before, s.service.Snapshot()) {
						t.Fatal("fixture lacks private uncommitted input")
					}
					if replace {
						agentAction(t, s.service, "operation.select", workbench.IDParams{ID: second})
					}
					current := s.service.Snapshot()
					m.frame()
					field := mountedControlElement(t, m, id)
					t.Logf("capture %s→%s; incarnation %d→%d; field=%v", first, current.Recording.ID, original.Incarnation, field.Incarnation, field.Value)
					clickMountedControl(t, m, id)
					m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
					after := s.service.Snapshot()
					check := func(state workbench.Snapshot, at float64) bool {
						switch id {
						case "recorded-cursor-a":
							return state.Recording.Scope.A != nil && state.Recording.Scope.A.Requested == at
						case "recorded-time":
							return state.Recording.Time == at
						default:
							return state.Recording.Scope.Window.Start == at
						}
					}
					if !replace {
						if field.Incarnation != original.Incarnation || !check(after, .008) || after.CommandRevision != before.CommandRevision+1 {
							t.Fatal("same-capture repaint lost its unfinished edit", s.errorMessage)
						}
						assertCursorMeasurementPreservesPhysics(t, before, after)
						return
					}
					if !reflect.DeepEqual(current, after) {
						t.Fatalf("old private draft changed a different recording: field=%s old=%s selected=%s cursor=%+v index=%d start=%g revision=%d→%d error=%s", id, first, second, after.Recording.Scope.A, after.Recording.Index, after.Recording.Scope.Window.Start, current.CommandRevision, after.CommandRevision, s.errorMessage)
					}
					if field.Incarnation == original.Incarnation || field.Value != "0 s" {
						t.Fatal("recording replacement retained private native state", field)
					}
					typeMountedCursor(t, m, id, "6 ms")
					m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
					after = s.service.Snapshot()
					if after.Recording.ID != second || !check(after, .006) || after.CommandRevision != current.CommandRevision+1 || s.errorMessage != "" {
						t.Fatal("fresh input did not work on the new recording", s.errorMessage)
					}
					assertCursorMeasurementPreservesPhysics(t, current, after)
				})
			}
		})
	}
}
