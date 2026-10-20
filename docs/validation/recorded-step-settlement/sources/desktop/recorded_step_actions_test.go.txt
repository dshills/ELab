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

func TestRecordedStepWaitsForFieldsAndIgnoresOtherTransportAdmissions(t *testing.T) {
	for _, direction := range []string{"next", "previous"} {
		t.Run(direction, func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			index := 8
			agentAction(t, s.service, "recording.seek", workbench.RecordingSeekParams{Index: &index})
			before := s.service.Snapshot()
			finish := map[string]func(error){}
			var order []string
			settle := func(id string, done func(error)) error { order = append(order, id); finish[id] = done; return nil }
			s.settleField, s.settleNumber = settle, settle
			step := recordedPlayControl(t, s, "recorded-"+direction)
			other := "previous"
			if direction == "previous" {
				other = "next"
			}
			opposite := recordedPlayControl(t, s, "recorded-"+other)
			play := recordedPlayControl(t, s, "recorded-play")
			s.app.Do(func() { step.Activate(); opposite.Activate(); play.Activate(); step.Activate() })
			if !reflect.DeepEqual(before, s.service.Snapshot()) || !reflect.DeepEqual(order, []string{"recorded-index"}) {
				t.Fatal("transport action failed to wait once", order)
			}
			for _, id := range []string{"recorded-next", "recorded-previous", "recorded-play"} {
				if !recordedPlayControl(t, s, id).Disabled {
					t.Fatal("pending transport left another admission enabled", id)
				}
			}
			for _, field := range []struct {
				id    string
				event ui.ChangeEvent
			}{{"recorded-index", ui.ChangeEvent{Number: 2}}, {"recorded-time", ui.ChangeEvent{Text: "3 ms"}}, {"recorded-speed", ui.ChangeEvent{Number: .0078125}}} {
				uiAction(t, s, field.id, &field.event)
				if finish[field.id] == nil {
					t.Fatal("field barrier missing", field.id)
				}
				s.app.Do(func() { finish[field.id](nil); finish[field.id](nil) })
			}
			after := s.service.Snapshot()
			want := 4
			if direction == "previous" {
				want = 2
			}
			if !reflect.DeepEqual(order, []string{"recorded-index", "recorded-time", "recorded-speed"}) || s.pendingRecordedPlayback != nil || s.errorMessage != "" || after.Recording.Index != want || after.Recording.Playing || after.Recording.Speed != .0078125 || after.CommandRevision != before.CommandRevision+4 {
				t.Fatal("transport did not use its own fields once", order, after.Recording, s.errorMessage)
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
		})
	}
}

func TestRecordedStepRefusesInterveningWork(t *testing.T) {
	for _, direction := range []string{"next", "previous"} {
		for _, scenario := range []string{"agent-before-field", "agent-after-field", "other-user-field", "capture-replaced", "document-replaced", "wrong-field"} {
			t.Run(direction+"/"+scenario, func(t *testing.T) {
				s, cleanup := recordedCursorRC(t)
				t.Cleanup(cleanup)
				index := 8
				agentAction(t, s.service, "recording.seek", workbench.RecordingSeekParams{Index: &index})
				var finish func(error)
				settle := func(_ string, done func(error)) error { finish = done; return nil }
				s.settleField, s.settleNumber = settle, settle
				step := recordedPlayControl(t, s, "recorded-"+direction)
				s.app.Do(step.Activate)
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
				if !reflect.DeepEqual(before, s.service.Snapshot()) || s.errorMessage == "" || s.pendingRecordedPlayback != nil {
					t.Fatal("deferred Step borrowed intervening work", s.errorMessage)
				}
			})
		}
	}
}

func TestRecordedStepRetainsInvalidPositionUntilCorrection(t *testing.T) {
	for _, direction := range []string{"next", "previous"} {
		t.Run(direction, func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			field := recordedPlayControl(t, s, "recorded-time")
			s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "bad"}) })
			message := s.errorMessage
			uiAction(t, s, "recorded-speed", &ui.ChangeEvent{Number: .0078125})
			before := s.service.Snapshot()
			step := recordedPlayControl(t, s, "recorded-"+direction)
			s.app.Do(step.Activate)
			if !reflect.DeepEqual(before, s.service.Snapshot()) || s.errorMessage != message {
				t.Fatal("another field's success erased the refusal", s.errorMessage)
			}
			uiAction(t, s, "recorded-time", &ui.ChangeEvent{Text: "3 ms"})
			uiAction(t, s, "recorded-"+direction, nil)
			want := 4
			if direction == "previous" {
				want = 2
			}
			if s.service.Snapshot().Recording.Index != want || len(s.recordedPlaybackRejected) != 0 {
				t.Fatal("Step failed after position correction")
			}
		})
	}
}

func TestRecordedStepAfterOwnSpeedAndNaturalPlaybackEnd(t *testing.T) {
	for _, intervening := range []bool{false, true} {
		t.Run(map[bool]string{false: "ticker-only", true: "agent-after-ticker"}[intervening], func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			agentAction(t, s.service, "recording.play", workbench.RecordingPlayParams{Speed: .001})
			step := recordedPlayControl(t, s, "recorded-previous")
			uiAction(t, s, "recorded-speed", &ui.ChangeEvent{Number: .125})
			accepted := s.service.Snapshot()
			if !accepted.Recording.Playing || accepted.Recording.Speed != .125 {
				t.Fatal("missing own playing-speed response")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			for s.service.Snapshot().Recording.Playing {
				state := s.service.Snapshot()
				timeout := 250
				raw, err := json.Marshal(workbench.EventPollParams{AfterRevision: state.Revision, TimeoutMS: &timeout})
				if err != nil {
					t.Fatal(err)
				}
				response := s.service.Dispatch(ctx, workbench.Request{Version: 1, ID: "viewer-progress", Command: "events.poll", Params: raw}, "observer")
				if response.Error != nil || ctx.Err() != nil {
					t.Fatal("viewer did not finish through its normal ticker", response.Error, ctx.Err())
				}
			}
			ended := s.service.Snapshot()
			if ended.CommandRevision != accepted.CommandRevision || ended.Recording.Index != 20 {
				t.Fatal("natural end changed command ownership or missed the final stored row")
			}
			if intervening {
				agentAction(t, s.service, "recording.speed.set", workbench.RecordingPlayParams{Speed: .25})
			}
			before := s.service.Snapshot()
			s.app.Do(step.Activate)
			after := s.service.Snapshot()
			if intervening {
				if !reflect.DeepEqual(before, after) || s.errorMessage == "" {
					t.Fatal("ticker continuation borrowed later agent speed", s.errorMessage)
				}
			} else if after.Recording.Index != 19 || after.Recording.Time != .019 || after.Recording.Playing || after.Recording.Speed != .125 || after.CommandRevision != before.CommandRevision+1 || s.errorMessage != "" {
				t.Fatal("Step mistook normal viewer progress for unrelated command work", after.Recording, s.errorMessage)
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
		})
	}
}
