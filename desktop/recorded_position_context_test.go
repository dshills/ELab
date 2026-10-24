package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestRecordedPositionParserRefusalCannotInstallAnOldDraft(t *testing.T) {
	for _, change := range []string{"agent-cursor", "agent-speed", "live-component"} {
		t.Run(change, func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			field := recordedPlayControl(t, s, "recorded-time")
			switch change {
			case "agent-cursor":
				agentAction(t, s.service, "recording.cursor.set", workbench.CursorParams{Cursor: "b", Time: .01})
			case "agent-speed":
				agentAction(t, s.service, "recording.speed.set", workbench.RecordingPlayParams{Speed: .5})
			case "live-component":
				agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "20k"})
			}
			before := s.service.Snapshot()
			s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "oops"}) })
			current := recordedPlayControl(t, s, "recorded-time")
			if current.Text == "oops" || len(s.recordedPlaybackRejected) != 0 || s.recordedPlaybackAccepted != nil {
				t.Fatal("stale parser failure installed private text in the newer editor", current.Text)
			}
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("stale position refusal changed accepted state")
			}
			s.app.Do(func() { current.Change(ui.ChangeEvent{Text: "3 ms"}) })
			after := s.service.Snapshot()
			if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 || after.Recording.ID != before.Recording.ID || after.Recording.Index != 3 || !reflect.DeepEqual(before.Recording.Scope, after.Recording.Scope) {
				t.Fatal("fresh recorded-position correction failed or changed other measurements", s.errorMessage)
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
		})
	}
}
