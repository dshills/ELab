package main

import (
	"math"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestTimeTextFormatsRoundTripWithoutFloat32Loss(t *testing.T) {
	for _, seconds := range []float64{0, 1e-12, 1e-6, .00025, .001, .0035, .01, .1, .3333333333333333, 1, 10, 12345.6789} {
		text := timeText(seconds)
		got, err := workbench.ParseTimeSeconds(text)
		if err != nil || got != seconds {
			t.Fatalf("%.17g → %q → %.17g, %v", seconds, text, got, err)
		}
	}
}

func TestCursorTimeFieldsDistinguishUnsetFromStartup(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "simulation.restart", nil)
	agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .01})
	for _, id := range []string{"a", "b"} {
		field, ok := findNode(s.view(), "cursor-time-"+id)
		clear, _ := findNode(s.view(), "cursor-clear-"+id)
		if !ok || field.Text != "" || field.Disabled || !clear.Disabled {
			t.Fatalf("unset cursor %s: text %q, disabled %v, clear disabled %v", id, field.Text, field.Disabled, clear.Disabled)
		}
	}
	// Typing zero must change the initially empty native field, so its normal
	// commit event places a cursor instead of accepting an unchanged fake zero.
	uiAction(t, s, "cursor-time-a", &ui.ChangeEvent{Text: "0 s"})
	uiAction(t, s, "cursor-time-b", &ui.ChangeEvent{Text: "10 ms"})
	state := s.service.Snapshot()
	if state.Scope.A == nil || !state.Scope.A.Valid || state.Scope.A.Requested != 0 || state.Scope.DeltaSeconds == nil || math.Abs(*state.Scope.DeltaSeconds-.01) > 1e-12 {
		t.Fatal("startup cursor and timing interval unavailable", state.Scope)
	}
	field, _ := findNode(s.view(), "cursor-time-a")
	clear, _ := findNode(s.view(), "cursor-clear-a")
	if field.Text != "0 s" || clear.Disabled {
		t.Fatal("placed zero cursor is indistinguishable from an unset cursor")
	}
	uiAction(t, s, "cursor-clear-a", nil)
	field, _ = findNode(s.view(), "cursor-time-a")
	other, _ := findNode(s.view(), "cursor-time-b")
	if field.Text != "" || other.Text != "10 ms" || s.service.Snapshot().Scope.A != nil {
		t.Fatal("clearing a cursor keeps a misleading time or clears the other cursor")
	}
}

func TestCommittedTimeFieldsRejectInvalidAndInterveningChanges(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, user, "example.rc", nil)
	uiAction(t, user, "simulation.pause", nil)
	uiAction(t, user, "simulation.restart", nil)
	agentAction(t, user.service, "simulation.step", workbench.StepParams{Duration: .02})
	cursor, ok := findNode(user.view(), "cursor-time-a")
	if !ok || cursor.Kind != ui.TextField {
		t.Fatal("cursor time is not a unit-aware text field")
	}
	// A rendered callback must not commit stale intent after another actor edits.
	agentAction(t, user.service, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	before := user.service.Snapshot()
	user.app.Do(func() { cursor.Change(ui.ChangeEvent{Text: "10 ms"}) })
	if user.errorMessage == "" || !reflect.DeepEqual(before, user.service.Snapshot()) {
		t.Fatal("stale rendered time field overwrites intervening state")
	}
	cursor, _ = findNode(user.view(), "cursor-time-a")
	user.app.Do(func() { cursor.Change(ui.ChangeEvent{Text: "10 Hz"}) })
	if user.errorMessage == "" || !reflect.DeepEqual(before, user.service.Snapshot()) {
		t.Fatal("invalid committed unit changes state")
	}
	uiAction(t, user, "cursor-time-a", &ui.ChangeEvent{Text: "10000 μs"})
	cursorResult := user.service.Snapshot().Scope.A
	if cursorResult == nil || cursorResult.Requested != .01 || math.Abs(cursorResult.Time-.01) > 1e-12 {
		t.Fatal("native unit text fails to select exact accepted time")
	}
	uiAction(t, user, "setting-step", &ui.ChangeEvent{Text: "40 us"})
	changed := user.service.Snapshot()
	if changed.TransientSettings.StepSeconds != .00004 || changed.UndoName != "Change experiment settings" || !changed.Transient.Stale {
		t.Fatal("time setting bypasses normal history/staleness rules")
	}
}
