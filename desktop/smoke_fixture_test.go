package main

import (
	"errors"
	"testing"

	"github.com/dshills/vellum/ui"
)

// The bulk smoke changes pages before the native editor tree can mount. Its
// accepted controlled edits must work, while real controls retain settlement.
func TestSmokeBatchControlsDoNotFlushUnmountedEditors(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	called := 0
	stale := func(string, func(error)) error {
		called++
		return errors.New("retired native editor fixture")
	}
	s.settleField, s.settleNumber = stale, stale
	s.app.Do(func() {
		if err := smokeUI(s, "verify-duration", &ui.ChangeEvent{Text: "20 ms"}); err != nil {
			t.Fatal(err)
		}
		if err := smokeUI(s, "verify-start", nil); err != nil {
			t.Fatal(err)
		}
		if called != 0 || s.service.Snapshot().LastOperation == "" {
			t.Fatal("semantic smoke tried to flush an unmounted editor")
		}
		// Going through the actual control after the helper must still settle.
		node, ok := findAction(s.view(), "verify-start")
		if !ok {
			t.Fatal("verification control disappeared")
		}
		node.Activate()
		if called != 1 || s.errorMessage != "retired native editor fixture" {
			t.Fatal("smoke helper changed ordinary native settlement", called, s.errorMessage)
		}
	})
}
