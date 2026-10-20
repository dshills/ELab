//go:build offscreen

package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func mountedRecordedStep(t *testing.T, s *screen, focus string) *mountedControlWindow {
	t.Helper()
	m := newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		panel := s.recordingPanel(state)
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: focus, Sequence: 1}}
		fields := ui.Node{ID: "fields", Kind: ui.Row}
		for _, id := range []string{"recorded-index", "recorded-time", "recorded-speed"} {
			node, ok := findNode(panel, id)
			if !ok {
				t.Fatalf("missing recorded control %s", id)
			}
			node.MinWidth, node.MinHeight = 90, 40
			fields.Children = append(fields.Children, node)
		}
		actions := ui.Node{ID: "actions", Kind: ui.Row}
		for _, id := range []string{"recorded-previous", "recorded-next"} {
			node, ok := findNode(panel, id)
			if !ok {
				t.Fatal("step missing", id)
			}
			actions.Children = append(actions.Children, node)
		}
		root.Children = []ui.Node{fields, actions}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
	s.settleField = func(id string, done func(error)) error { return cogent.SettleTextField(m.ctx, s.app, "main", id, done) }
	s.settleNumber = func(id string, done func(error)) error {
		return cogent.SettleNumberField(m.ctx, s.app, "main", id, done)
	}
	return m
}

func TestMountedNumericOwnerRecordedStepAfterPrivateEntry(t *testing.T) {
	for _, direction := range []string{"next", "previous"} {
		for _, fixture := range []struct {
			name, field, text string
			index             int
			speed             float64
			refused           bool
		}{
			{"no-edit", "recorded-index", "", 8, .00390625, false},
			{"index", "recorded-index", "3", 3, .00390625, false},
			{"time", "recorded-time", "3 ms", 3, .00390625, false},
			{"speed", "recorded-speed", "0.0078125", 8, .0078125, false},
			{"invalid-index", "recorded-index", "1.5", 8, .00390625, true},
			{"invalid-time", "recorded-time", "bad", 8, .00390625, true},
		} {
			t.Run(direction+"/"+fixture.name, func(t *testing.T) {
				s, cleanup := recordedCursorRC(t)
				defer cleanup()
				index := 8
				agentAction(t, s.service, "recording.seek", workbench.RecordingSeekParams{Index: &index})
				agentAction(t, s.service, "recording.speed.set", workbench.RecordingPlayParams{Speed: .00390625})
				m := mountedRecordedStep(t, s, fixture.field)
				before := s.service.Snapshot()
				if fixture.text != "" {
					typeRecordedPlay(t, m, fixture.text)
				}
				if !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("private entry committed early")
				}
				clickMountedControl(t, m, "recorded-"+direction)
				after := s.service.Snapshot()
				t.Logf("input=%q index=%d speed=%g revision=%d→%d error=%q", fixture.text, after.Recording.Index, after.Recording.Speed, before.CommandRevision, after.CommandRevision, s.errorMessage)
				if fixture.refused {
					if !reflect.DeepEqual(before, after) || s.errorMessage == "" {
						t.Fatal("invalid entry stepped from old position or lost refusal")
					}
					clickMountedControl(t, m, fixture.field)
					correction := "3"
					if fixture.field == "recorded-time" {
						correction = "3 ms"
					}
					typeRecordedPlay(t, m, correction)
					clickMountedControl(t, m, "recorded-"+direction)
					recovered := s.service.Snapshot()
					want := 4
					if direction == "previous" {
						want = 2
					}
					if recovered.Recording.Index != want || recovered.CommandRevision != before.CommandRevision+2 || s.errorMessage != "" {
						t.Fatal("fresh correction did not recover first-click Step", s.errorMessage)
					}
					assertCursorMeasurementPreservesPhysics(t, before, recovered)
				} else {
					expected := fixture.index + 1
					if direction == "previous" {
						expected = fixture.index - 1
					}
					commands := uint64(1)
					if fixture.text != "" {
						commands++
					}
					if after.Recording.Playing || after.Recording.Index != expected || after.Recording.Speed != fixture.speed || after.CommandRevision != before.CommandRevision+commands || s.errorMessage != "" {
						t.Fatal("first step failed to use unfinished entry exactly once")
					}
					assertCursorMeasurementPreservesPhysics(t, before, after)
				}
			})
		}
	}
}

func TestMountedNumericOwnerRecordedStepBoundaryDraft(t *testing.T) {
	for _, direction := range []string{"previous", "next"} {
		for _, field := range []string{"recorded-index", "recorded-time"} {
			t.Run(direction+"/"+field, func(t *testing.T) {
				s, cleanup := recordedCursorRC(t)
				defer cleanup()
				if direction == "next" {
					index := 20
					agentAction(t, s.service, "recording.seek", workbench.RecordingSeekParams{Index: &index})
				}
				m := mountedRecordedStep(t, s, field)
				before := s.service.Snapshot()
				text := "3"
				if field == "recorded-time" {
					text = "3 ms"
				}
				typeRecordedPlay(t, m, text)
				clickMountedControl(t, m, "recorded-"+direction)
				after := s.service.Snapshot()
				want := 2
				if direction == "next" {
					want = 4
				}
				t.Logf("field=%s index=%d expected=%d revision=%d→%d error=%q", field, after.Recording.Index, want, before.CommandRevision, after.CommandRevision, s.errorMessage)
				if after.Recording.Index != want || after.CommandRevision != before.CommandRevision+2 || s.errorMessage != "" {
					t.Fatal("disabled boundary step did not apply fresh position and step once")
				}
			})
		}
	}
}

func TestMountedNumericOwnerRecordedStepWithoutFocusLoss(t *testing.T) {
	for _, direction := range []string{"next", "previous"} {
		for _, field := range []struct{ id, text string }{{"recorded-index", "3"}, {"recorded-time", "3 ms"}, {"recorded-speed", "0.0078125"}, {"recorded-index", "1.5"}, {"recorded-time", "bad"}} {
			t.Run(direction+"/"+field.id+"/"+field.text, func(t *testing.T) {
				s, cleanup := recordedCursorRC(t)
				t.Cleanup(cleanup)
				index := 8
				agentAction(t, s.service, "recording.seek", workbench.RecordingSeekParams{Index: &index})
				agentAction(t, s.service, "recording.speed.set", workbench.RecordingPlayParams{Speed: .00390625})
				m := mountedRecordedStep(t, s, field.id)
				before := s.service.Snapshot()
				typeRecordedPlay(t, m, field.text)
				if !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("draft committed before activation")
				}
				step := recordedPlayControl(t, s, "recorded-"+direction)
				s.app.Do(step.Activate)
				m.settle()
				after := s.service.Snapshot()
				if field.text == "bad" || field.text == "1.5" {
					if !reflect.DeepEqual(before, after) || s.errorMessage == "" || s.pendingRecordedPlayback != nil {
						t.Fatal("activation ignored invalid private input", s.errorMessage)
					}
					return
				}
				speed, want := .00390625, 3
				if field.id == "recorded-speed" {
					speed, want = .0078125, 8
				}
				if direction == "next" {
					want++
				} else {
					want--
				}
				if after.Recording.Playing || after.Recording.Index != want || after.Recording.Speed != speed || after.CommandRevision != before.CommandRevision+2 || s.errorMessage != "" || s.pendingRecordedPlayback != nil {
					t.Fatal("activation stepped before private entry settled", after.Recording, s.errorMessage)
				}
				assertCursorMeasurementPreservesPhysics(t, before, after)
			})
		}
	}
}

func TestMountedNumericOwnerRecordedStepBoundaryRefusesWithoutDraft(t *testing.T) {
	for _, direction := range []string{"previous", "next"} {
		t.Run(direction, func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			if direction == "next" {
				index := 20
				agentAction(t, s.service, "recording.seek", workbench.RecordingSeekParams{Index: &index})
			}
			m := mountedRecordedStep(t, s, "recorded-index")
			before := s.service.Snapshot()
			control := recordedPlayControl(t, s, "recorded-"+direction)
			if control.Disabled || control.Accessibility.Help == "" {
				t.Fatal("transient boundary must allow settling a private position and explain the boundary")
			}
			clickMountedControl(t, m, "recorded-"+direction)
			if !reflect.DeepEqual(before, s.service.Snapshot()) || s.errorMessage != "Recorded sample boundary reached" || s.pendingRecordedPlayback != nil {
				t.Fatal("unchanged boundary moved/wrapped or lost shared feedback", s.errorMessage)
			}
		})
	}
}

func TestMountedNumericOwnerRecordedStepPrivateEntryRefusesAgent(t *testing.T) {
	for _, direction := range []string{"next", "previous"} {
		t.Run(direction, func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			index := 8
			agentAction(t, s.service, "recording.seek", workbench.RecordingSeekParams{Index: &index})
			m := mountedRecordedStep(t, s, "recorded-index")
			step := recordedPlayControl(t, s, "recorded-"+direction)
			typeRecordedPlay(t, m, "3")
			agentAction(t, s.service, "recording.speed.set", workbench.RecordingPlayParams{Speed: .0078125})
			before := s.service.Snapshot()
			s.app.Do(step.Activate)
			m.settle()
			if !reflect.DeepEqual(before, s.service.Snapshot()) || s.errorMessage == "" || s.pendingRecordedPlayback != nil {
				t.Fatal("retained private Step overwrote agent work", s.errorMessage)
			}
		})
	}
}
