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
)

// Route actual retained text-field Return events, stopping before the posted
// owner callback. This covers parser origin through the real driver queue;
// it does not establish macOS native event-queue or hardware-rate acceptance.
func TestMountedParserQueueCannotInstallPreAgentRefusal(t *testing.T) {
	entries := append([]struct{ id, correction string }{}, analysisEntryCases...)
	entries = append(entries, struct{ id, correction string }{"recorded-time", "3 ms"})
	for _, entry := range entries {
		t.Run(entry.id, func(t *testing.T) {
			var s *screen
			if entry.id == "recorded-time" {
				var cleanup func()
				s, cleanup = recordedCursorRC(t)
				t.Cleanup(cleanup)
			} else {
				s = analysisEntryScreen(t, entry.id)
			}
			if _, ok := findNode(s.view(), entry.id); !ok {
				uiAction(t, s, "inspector-section-simulation", nil)
			}
			m := mountedAssembledDiscreteControl(t, s, entry.id)
			typeMountedCursor(t, m, entry.id, "oops")
			before := s.service.Snapshot()
			if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.KeyChord, Code: key.CodeReturnEnter}); err != nil {
				t.Fatal(err)
			}
			step, err := m.manual.Step(m.ctx, 1)
			if err != nil || step.Events != 1 || step.OwnerCallbacks != 0 || s.app.Pending().Ordinary == 0 {
				t.Fatal("parser commit was not isolated at an actual queue boundary", step, err)
			}
			if !reflect.DeepEqual(before, s.service.Snapshot()) || analysisEntryRejections(s) != 0 || len(s.recordedPlaybackRejected) != 0 {
				t.Fatal("fixture parsed before the withheld owner delivery")
			}
			t.Logf("actual Return boundary: events=%d owner_callbacks=%d pending=%d", step.Events, step.OwnerCallbacks, s.app.Pending().Ordinary)
			agentAction(t, s.service, "editor.value.set", workbench.DraftParams{Value: "new agent draft"})
			agent := s.service.Snapshot()
			if err := s.app.Invalidate("main"); err != nil {
				t.Fatal(err)
			}
			if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.MouseMove, Position: image.Pt(680, 480)}); err != nil {
				t.Fatal(err)
			}
			m.settle()
			if !reflect.DeepEqual(agent, s.service.Snapshot()) || analysisEntryRejections(s) != 0 || len(s.recordedPlaybackRejected) != 0 || s.recordedPlaybackAccepted != nil {
				t.Fatal("queued pre-agent parser installed a refusal/continuation or changed newer work", s.errorMessage)
			}
			typeMountedCursor(t, m, entry.id, entry.correction)
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			after := s.service.Snapshot()
			if s.errorMessage != "" || after.CommandRevision != agent.CommandRevision+1 || after.Drawing != agent.Drawing || analysisEntryRejections(s) != 0 || len(s.recordedPlaybackRejected) != 0 {
				t.Fatal("fresh explicit correction failed or overwrote the agent draft", s.errorMessage)
			}
			assertCursorMeasurementPreservesPhysics(t, agent, after)
		})
	}
}
