package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func TestNativeVerifyModesFollowSharedInventory(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	for _, example := range []string{"example.rc", "example.npn-pulse", "example.rectifier"} {
		agentAction(t, s.service, example, workbench.ReplaceParams{DiscardChanges: true})
		agentAction(t, s.service, "simulation.pause", struct{}{})
		cfg := s.service.Snapshot().VerificationSettings
		cfg.InputMode = "charge"
		agentAction(t, s.service, "verification.settings.set", cfg)
		uiAction(t, s, "inspector-section-simulation", nil)
		uiAction(t, s, "simulation-tool", &ui.ChangeEvent{SelectedID: "recordings"})
		state := s.service.Snapshot()
		s.app.Do(func() {
			node, ok := findAction(s.view(), "verify-mode")
			if !ok {
				t.Fatal("missing transient mode selector", example)
			}
			options := node.Options
			if choices, native := node.Custom.(menuChoices); native {
				options = choices.Options
			}
			var enabled []string
			for _, option := range options {
				if !option.Disabled {
					enabled = append(enabled, option.ID)
				}
			}
			if !reflect.DeepEqual(enabled, state.VerificationInputModes) {
				t.Fatal("native mode list differs", example, enabled, state.VerificationInputModes)
			}
			if example != "example.rc" {
				last := options[len(options)-1]
				if node.SelectedID != "charge" || last.ID != "charge" || !last.Disabled {
					t.Fatal("incompatible stored mode silently hidden")
				}
			}
		})
		uiAction(t, s, "verify-project-startup", nil)
	}
}

func TestNativeDCVerifyCanResetIncompatibleDefaults(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned verification recovery", Width: 1140, Height: 860, View: s.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "simulation.pause", nil)
	settings := s.service.Snapshot().VerificationSettings
	initial := 1.
	settings.InitialVoltage = &initial
	settings.InputMode = "charge"
	settings.Step = .002
	agentAction(t, s.service, "verification.settings.set", settings)
	uiAction(t, s, "inspector-section-simulation", nil)
	uiAction(t, s, "simulation-tool", &ui.ChangeEvent{SelectedID: "recordings"})
	s.app.Do(func() {
		if _, ok := findAction(s.view(), "verify-project-startup"); !ok {
			t.Fatal("DC workflow hides the reset needed to clear incompatible retained startup defaults")
		}
	})
	before := s.service.Snapshot()
	uiAction(t, s, "verify-project-startup", nil)
	after := s.service.Snapshot()
	if after.VerificationSettings.InitialVoltage != nil || after.VerificationSettings.InputMode != "auto" || after.VerificationSettings.Step != 0 || len(after.VerificationSettings.Capacitors) != 0 {
		t.Fatal("reset left incompatible defaults")
	}
	if !reflect.DeepEqual(before.Components, after.Components) || before.Document != after.Document || before.RunID != after.RunID || !reflect.DeepEqual(before.Result, after.Result) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("reset changed live circuit/history")
	}
	uiAction(t, s, "verify-start", nil)
	view := completedVerification(t, s.service, s.service.Snapshot().LastOperation)
	if view.Metadata.Duration != 0 || view.Metadata.SampleCount != 1 || view.Metadata.InputMode != "auto" {
		t.Fatal("DC verification not recovered")
	}
}
