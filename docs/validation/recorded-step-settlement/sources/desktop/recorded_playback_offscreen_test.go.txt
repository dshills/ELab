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

func mountedRecordedPlay(t *testing.T, s *screen, focus string) *mountedControlWindow {
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
		play, _ := findNode(panel, "recorded-play")
		root.Children = []ui.Node{fields, play}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
	s.settleField = func(id string, done func(error)) error { return cogent.SettleTextField(m.ctx, s.app, "main", id, done) }
	s.settleNumber = func(id string, done func(error)) error {
		return cogent.SettleNumberField(m.ctx, s.app, "main", id, done)
	}
	return m
}

func typeRecordedPlay(t *testing.T, m *mountedControlWindow, text string) {
	t.Helper()
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: key.Meta})
	for _, r := range text {
		code := key.CodeUnknown
		switch {
		case r == '0':
			code = key.Code0
		case r >= '1' && r <= '9':
			code = key.Code1 + key.Codes(r-'1')
		case r == '.':
			code = key.CodeFullStop
		}
		m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r, Code: code})
	}
}

func TestMountedNumericOwnerRecordedPlayAfterPrivateEntry(t *testing.T) {
	for _, fixture := range []struct {
		name, field, text string
		index             int
		speed             float64
		refused           bool
	}{
		{"no-edit", "recorded-index", "", 0, .00390625, false},
		{"index", "recorded-index", "3", 3, .00390625, false},
		{"speed", "recorded-speed", "0.0078125", 0, .0078125, false},
		{"time", "recorded-time", "3 ms", 3, .00390625, false},
		{"invalid-index", "recorded-index", "1.5", 0, .00390625, true},
		{"invalid-time", "recorded-time", "bad", 0, .00390625, true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			defer cleanup()
			agentAction(t, s.service, "recording.speed.set", workbench.RecordingPlayParams{Speed: .00390625})
			m := mountedRecordedPlay(t, s, fixture.field)
			before := s.service.Snapshot()
			if fixture.text != "" {
				typeRecordedPlay(t, m, fixture.text)
				if !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("private entry changed public state")
				}
			}
			clickMountedControl(t, m, "recorded-play")
			after := s.service.Snapshot()
			t.Logf("field=%s input=%q index=%d speed=%.9g playing=%t revision=%d→%d error=%q", fixture.field, fixture.text, after.Recording.Index, after.Recording.Speed, after.Recording.Playing, before.CommandRevision, after.CommandRevision, s.errorMessage)
			if fixture.refused {
				if !reflect.DeepEqual(before, after) || s.errorMessage == "" {
					t.Fatal("invalid recorded entry played old data or lost refusal")
				}
				clickMountedControl(t, m, fixture.field)
				correction := "3"
				if fixture.field == "recorded-time" {
					correction = "3 ms"
				}
				typeRecordedPlay(t, m, correction)
				clickMountedControl(t, m, "recorded-play")
				recovered := s.service.Snapshot()
				if !recovered.Recording.Playing || recovered.Recording.Index < 3 || recovered.CommandRevision != before.CommandRevision+2 || s.errorMessage != "" {
					t.Fatal("fresh correction did not recover first-click Play", s.errorMessage)
				}
				assertCursorMeasurementPreservesPhysics(t, before, recovered)
			} else {
				commands := uint64(1)
				if fixture.text != "" {
					commands++
				}
				if !after.Recording.Playing || after.Recording.Speed != fixture.speed || after.Recording.Index < fixture.index || after.CommandRevision != before.CommandRevision+commands || s.errorMessage != "" {
					t.Fatal("first Play did not apply its private entry and start once")
				}
				assertCursorMeasurementPreservesPhysics(t, before, after)
			}
		})
	}
}

// Direct activation does not blur the field. The real SDK settlement barriers
// must read its private buffer before dispatching the public Play command.
func TestMountedNumericOwnerRecordedPlayWithoutFocusLoss(t *testing.T) {
	for _, field := range []struct{ id, text string }{{"recorded-index", "3"}, {"recorded-time", "3 ms"}, {"recorded-speed", "0.0078125"}, {"recorded-index", "1.5"}, {"recorded-time", "bad"}} {
		t.Run(field.id+"/"+field.text, func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			agentAction(t, s.service, "recording.speed.set", workbench.RecordingPlayParams{Speed: .00390625})
			m := mountedRecordedPlay(t, s, field.id)
			before := s.service.Snapshot()
			typeRecordedPlay(t, m, field.text)
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("draft committed before activation")
			}
			play := recordedPlayControl(t, s, "recorded-play")
			s.app.Do(play.Activate)
			m.settle()
			after := s.service.Snapshot()
			if field.text == "bad" || field.text == "1.5" {
				if !reflect.DeepEqual(before, after) || s.errorMessage == "" || s.pendingRecordedPlayback != nil {
					t.Fatal("activation discarded invalid private input", s.errorMessage)
				}
				return
			}
			speed, index := .00390625, 3
			if field.id == "recorded-speed" {
				speed, index = .0078125, 0
			}
			if !after.Recording.Playing || after.Recording.Index < index || after.Recording.Speed != speed || after.CommandRevision != before.CommandRevision+2 || s.errorMessage != "" || s.pendingRecordedPlayback != nil {
				t.Fatal("activation played before private input settled", after.Recording, s.errorMessage)
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
		})
	}
}

func TestMountedNumericOwnerRecordedPlayPrivateEntryRefusesAgent(t *testing.T) {
	s, cleanup := recordedCursorRC(t)
	t.Cleanup(cleanup)
	m := mountedRecordedPlay(t, s, "recorded-index")
	play := recordedPlayControl(t, s, "recorded-play")
	typeRecordedPlay(t, m, "3")
	agentAction(t, s.service, "recording.speed.set", workbench.RecordingPlayParams{Speed: .0078125})
	before := s.service.Snapshot()
	// Retain the original activation, as with an already queued native action.
	s.app.Do(play.Activate)
	m.settle()
	if !reflect.DeepEqual(before, s.service.Snapshot()) || s.errorMessage == "" || s.pendingRecordedPlayback != nil {
		t.Fatal("private entry/Play overwrote intervening agent work", s.errorMessage)
	}
}
