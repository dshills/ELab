package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func recordedPlayControl(t *testing.T, s *screen, id string) ui.Node {
	t.Helper()
	var node ui.Node
	s.app.Do(func() {
		var ok bool
		node, ok = findNode(s.view(), id)
		if !ok {
			t.Fatalf("missing recorded control %s", id)
		}
	})
	return node
}

func TestRecordedPlaySettlesFieldsInOrderOnce(t *testing.T) {
	s, cleanup := recordedCursorRC(t)
	t.Cleanup(cleanup)
	agentAction(t, s.service, "recording.speed.set", workbench.RecordingPlayParams{Speed: .00390625})
	before := s.service.Snapshot()
	finish := map[string]func(error){}
	var order []string
	settle := func(id string, done func(error)) error {
		order = append(order, id)
		finish[id] = done
		return nil
	}
	s.settleField, s.settleNumber = settle, settle
	play := recordedPlayControl(t, s, "recorded-play")
	s.app.Do(play.Activate)
	s.app.Do(play.Activate)
	if !reflect.DeepEqual(order, []string{"recorded-index"}) || !reflect.DeepEqual(before, s.service.Snapshot()) || !recordedPlayControl(t, s, "recorded-play").Disabled {
		t.Fatal("Play did not wait once for index settlement", order)
	}
	for _, field := range []struct {
		id    string
		event ui.ChangeEvent
	}{{"recorded-index", ui.ChangeEvent{Number: 2}}, {"recorded-time", ui.ChangeEvent{Text: "3 ms"}}, {"recorded-speed", ui.ChangeEvent{Number: .0078125}}} {
		uiAction(t, s, field.id, &field.event)
		if finish[field.id] == nil || s.service.Snapshot().Recording.Playing {
			t.Fatal("Play skipped a field barrier", field.id, order)
		}
		s.app.Do(func() { finish[field.id](nil); finish[field.id](nil) })
	}
	after := s.service.Snapshot()
	if !reflect.DeepEqual(order, []string{"recorded-index", "recorded-time", "recorded-speed"}) || s.pendingRecordedPlay != nil || s.errorMessage != "" || !after.Recording.Playing || after.Recording.Speed != .0078125 || after.Recording.Index < 3 || after.CommandRevision != before.CommandRevision+4 {
		t.Fatal("Play failed to use its exact accepted fields once", order, after.Recording, s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestRecordedPlayRefusesInterveningWork(t *testing.T) {
	for _, scenario := range []string{"agent-before-field", "agent-after-field", "other-user-field", "capture-replaced", "document-replaced", "wrong-field"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			var finish func(error)
			settle := func(_ string, done func(error)) error { finish = done; return nil }
			s.settleField, s.settleNumber = settle, settle
			play := recordedPlayControl(t, s, "recorded-play")
			s.app.Do(play.Activate)
			if scenario == "agent-after-field" {
				uiAction(t, s, "recorded-index", &ui.ChangeEvent{Number: 3})
			}
			switch scenario {
			case "capture-replaced":
				duration := .01
				agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration})
				completedVerification(t, s.service, s.service.Snapshot().LastOperation)
			case "document-replaced":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			case "other-user-field":
				uiAction(t, s, "pdf-margin", &ui.ChangeEvent{Number: 11})
			case "wrong-field":
			default:
				agentAction(t, s.service, "recording.speed.set", workbench.RecordingPlayParams{Speed: .0078125})
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				if scenario == "agent-before-field" || scenario == "other-user-field" || scenario == "wrong-field" {
					id := "recorded-index"
					if scenario == "wrong-field" {
						id = "recorded-speed"
					}
					field, _ := findNode(s.view(), id)
					field.Change(ui.ChangeEvent{Number: 4})
				}
				for range 3 {
					finish(nil)
				}
			})
			if s.pendingRecordedPlay != nil || s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("old Play borrowed intervening work", s.errorMessage)
			}
		})
	}
}

func TestRecordedPlayRetainedButtonContinuesOnlyOwnFields(t *testing.T) {
	for _, intervening := range []string{"none", "agent", "other-field", "viewport"} {
		t.Run(intervening, func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			play := recordedPlayControl(t, s, "recorded-play")
			uiAction(t, s, "recorded-index", &ui.ChangeEvent{Number: 3})
			uiAction(t, s, "recorded-speed", &ui.ChangeEvent{Number: .00390625})
			switch intervening {
			case "agent":
				agentAction(t, s.service, "recording.speed.set", workbench.RecordingPlayParams{Speed: .0078125})
			case "other-field":
				uiAction(t, s, "pdf-margin", &ui.ChangeEvent{Number: 11})
			case "viewport":
				from := *s.recordedPlayAccepted
				agentAction(t, s.service, "view.viewport.set", workbench.ViewportParams{Width: 900, Height: 600})
				accepted := s.service.Snapshot()
				// This helper is called only by our accepted viewport publisher.
				s.app.Do(func() { s.continueRecordedPlayViewport(from.to, accepted) })
			}
			before := s.service.Snapshot()
			s.app.Do(play.Activate)
			after := s.service.Snapshot()
			if intervening == "agent" || intervening == "other-field" {
				if s.errorMessage == "" || !reflect.DeepEqual(before, after) {
					t.Fatal("retained Play adopted unrelated context", s.errorMessage)
				}
			} else if s.errorMessage != "" || !after.Recording.Playing || after.Recording.Speed != .00390625 || after.CommandRevision != before.CommandRevision+1 {
				t.Fatal("own field/viewport continuation was lost", s.errorMessage)
			}
		})
	}
}

func TestRecordedPlayRejectionSurvivesOtherFieldUntilCorrection(t *testing.T) {
	for _, id := range []string{"recorded-index", "recorded-time"} {
		t.Run(id, func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			bad, good := ui.ChangeEvent{Number: 1.5}, ui.ChangeEvent{Number: 3}
			if id == "recorded-time" {
				bad, good = ui.ChangeEvent{Text: "bad"}, ui.ChangeEvent{Text: "3 ms"}
			}
			field := recordedPlayControl(t, s, id)
			s.app.Do(func() { field.Change(bad) })
			message := s.errorMessage
			if message == "" {
				t.Fatal("missing field refusal")
			}
			uiAction(t, s, "recorded-speed", &ui.ChangeEvent{Number: .00390625})
			before := s.service.Snapshot()
			play := recordedPlayControl(t, s, "recorded-play")
			s.app.Do(play.Activate)
			if !reflect.DeepEqual(before, s.service.Snapshot()) || s.errorMessage != message || s.pendingRecordedPlay != nil {
				t.Fatal("other field cleared refusal and played old data", s.errorMessage)
			}
			uiAction(t, s, id, &good)
			uiAction(t, s, "recorded-play", nil)
			if !s.service.Snapshot().Recording.Playing || len(s.recordedPlayRejected) != 0 {
				t.Fatal("correction did not recover Play")
			}
		})
	}
}

func TestRecordedPlaySettlementFailurePreservesState(t *testing.T) {
	for _, admission := range []bool{false, true} {
		t.Run(map[bool]string{false: "completion", true: "admission"}[admission], func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			var finish func(error)
			s.settleNumber = func(_ string, done func(error)) error {
				finish = done
				if admission {
					return errors.New("field retired")
				}
				return nil
			}
			before := s.service.Snapshot()
			play := recordedPlayControl(t, s, "recorded-play")
			s.app.Do(func() { play.Activate(); finish(errors.New("composition unfinished")) })
			if s.pendingRecordedPlay != nil || s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("failed settlement played stale state", s.errorMessage)
			}
		})
	}
}

func TestRecordedPlayViewportCannotBorrowAnotherOwnerOrValue(t *testing.T) {
	for _, scenario := range []string{"revision", "generation", "capture", "index", "speed"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "recorded-index", &ui.ChangeEvent{Number: 3})
			before := *s.recordedPlayAccepted
			accepted := s.service.Snapshot()
			from := before.to
			accepted.CommandRevision++
			switch scenario {
			case "revision":
				from.revision--
			case "generation":
				accepted.Document.Generation++
			case "capture":
				accepted.Recording.ID = "another-capture"
			case "index":
				accepted.Recording.Index++
			case "speed":
				accepted.Recording.Speed = .0078125
			}
			s.app.Do(func() { s.continueRecordedPlayViewport(from, accepted) })
			if *s.recordedPlayAccepted != before {
				t.Fatal("viewport advanced an unrelated context or value", scenario)
			}
		})
	}
}

func TestRecordedPlayRejectionContinuesViewportAndExpiresWithOwner(t *testing.T) {
	s, cleanup := recordedCursorRC(t)
	t.Cleanup(cleanup)
	field := recordedPlayControl(t, s, "recorded-time")
	s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "bad"}) })
	message := s.errorMessage
	play := recordedPlayControl(t, s, "recorded-play")
	from := s.recordedPlayAccepted.to
	agentAction(t, s.service, "view.viewport.set", workbench.ViewportParams{Width: 900, Height: 600})
	accepted := s.service.Snapshot()
	s.app.Do(func() { s.continueRecordedPlayViewport(from, accepted); play.Activate() })
	if !reflect.DeepEqual(accepted, s.service.Snapshot()) || s.errorMessage != message {
		t.Fatal("viewport hid rejection or played", s.errorMessage)
	}
	uiAction(t, s, "recorded-time", &ui.ChangeEvent{Text: "3 ms"})
	uiAction(t, s, "recorded-play", nil)
	if !s.service.Snapshot().Recording.Playing {
		t.Fatal("viewport refusal could not recover")
	}
	agentAction(t, s.service, "recording.pause", struct{}{})
	field = recordedPlayControl(t, s, "recorded-time")
	s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "bad"}) })
	agentAction(t, s.service, "recording.view.set", workbench.RecordingViewParams{Mode: "live"})
	s.app.Do(func() { s.view() })
	if len(s.recordedPlayRejected) != 0 {
		t.Fatal("hidden viewer inherited stale rejection")
	}
}
