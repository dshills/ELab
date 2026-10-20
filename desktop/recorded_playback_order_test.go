package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// Model managed text focus-loss arriving after button admission but before the
// first asynchronous settlement completes. The native SDK can publish this
// Change independently of a NumberField barrier; the manual driver does not
// exercise the macOS native text bridge. This is a scheduling regression, not
// a substitute for rechecking the original native failure.
func TestRecordedTransportManagedTimeBeforeFirstSettlement(t *testing.T) {
	for _, direction := range []string{"", "next", "previous"} {
		for _, intervention := range []string{"none", "agent", "document", "stale-origin", "wrong-field"} {
			t.Run(direction+"/"+intervention, func(t *testing.T) {
				s, cleanup := recordedCursorRC(t)
				t.Cleanup(cleanup)
				index := 4
				agentAction(t, s.service, "recording.seek", workbench.RecordingSeekParams{Index: &index})
				field := recordedPlayControl(t, s, "recorded-time")
				if intervention == "stale-origin" {
					agentAction(t, s.service, "recording.speed.set", workbench.RecordingPlayParams{Speed: .00390625})
				}
				id := "recorded-play"
				if direction != "" {
					id = "recorded-" + direction
				}
				action := recordedPlayControl(t, s, id)
				var finish []func(error)
				var order []string
				settle := func(id string, done func(error)) error {
					order = append(order, id)
					finish = append(finish, done)
					return nil
				}
				s.settleField, s.settleNumber = settle, settle
				s.app.Do(action.Activate)
				if s.pendingRecordedPlayback == nil || len(finish) != 1 {
					t.Fatal("missing deferred settlement")
				}
				switch intervention {
				case "agent":
					agentAction(t, s.service, "recording.speed.set", workbench.RecordingPlayParams{Speed: .0078125})
				case "document":
					agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
				case "wrong-field":
					field = recordedPlayControl(t, s, "recorded-speed")
				}
				before := s.service.Snapshot()
				s.app.Do(func() {
					if intervention == "wrong-field" {
						field.Change(ui.ChangeEvent{Number: .0078125})
					} else {
						field.Change(ui.ChangeEvent{Text: "3 ms"})
					}
					// Each completion may admit the next barrier. Repeated old completions
					// must never advance that barrier or dispatch the action twice.
					for i := 0; i < len(finish); i++ {
						finish[i](nil)
						finish[i](nil)
					}
				})
				after := s.service.Snapshot()
				if s.pendingRecordedPlayback != nil {
					t.Fatal("settlement stranded")
				}
				if intervention != "none" {
					if !reflect.DeepEqual(before, after) || s.errorMessage == "" {
						t.Fatal("transport borrowed unrelated or stale field work", s.errorMessage)
					}
					return
				}
				want := 3
				if direction == "next" {
					want++
				}
				if direction == "previous" {
					want--
				}
				if s.errorMessage != "" || after.Recording.Index != want || after.Recording.Playing != (direction == "") || after.CommandRevision != before.CommandRevision+2 {
					t.Fatalf("owned managed-time callback failed: order=%v index=%d error=%q revision=%d→%d", order, after.Recording.Index, s.errorMessage, before.CommandRevision, after.CommandRevision)
				}
				assertCursorMeasurementPreservesPhysics(t, before, after)
			})
		}
	}
}
