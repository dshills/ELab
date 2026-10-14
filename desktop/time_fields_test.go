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
