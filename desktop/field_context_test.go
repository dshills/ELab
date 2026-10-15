package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestRenderedSineFieldCannotOverwriteInterveningAgentWaveform(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: "V1", Kind: "sine-source"})
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "V1"})
	var captured ui.Node
	s.app.Do(func() {
		var ok bool
		captured, ok = findNode(s.view(), "sine-frequency")
		if !ok {
			t.Fatal("frequency field missing")
		}
	})
	waveform := workbench.DefaultSineWaveform()
	waveform.Amplitude = 1
	agentAction(t, s.service, "source.sine.set", workbench.SineParams{ID: "V1", Waveform: waveform})
	before := s.service.Snapshot()
	s.app.Do(func() { captured.Change(ui.ChangeEvent{Text: "20"}) })
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("old frequency field replaced newer amplitude")
	}
	uiAction(t, s, "sine-frequency", &ui.ChangeEvent{Text: "20"})
	state := s.service.Snapshot()
	if state.Components[0].Sine.Amplitude != 1 || state.Components[0].Sine.Frequency != 20 || s.fieldContext != nil {
		t.Fatal("fresh field lost agent values or leaked callback context")
	}
}

func TestRenderedFieldContextProtectsDocumentAndRestoresContext(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	state := s.service.Snapshot()
	var captured ui.Node
	s.app.Do(func() {
		captured, _ = findNode(s.view(), "value-entry")
	})
	// The current document/revision accepts the event and restores owner context.
	s.app.Do(func() { captured.Change(ui.ChangeEvent{Text: "2k"}) })
	if s.service.Snapshot().Drawing.Value != "2k" || s.fieldContext != nil {
		t.Fatal("current field rejected or context leaked")
	}
	// Direct binding isolates generation from command revision: an old generation
	// cannot act even if its command revision was refreshed after replacement.
	agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
	before := s.service.Snapshot()
	s.app.Do(func() {
		node := ui.Node{ID: "old-draft", Kind: ui.TextField, Change: func(e ui.ChangeEvent) { s.invoke("editor.value.set", workbench.DraftParams{Value: e.Text}) }}
		s.bindFieldContext(&node, actionContext{before.CommandRevision, state.Document.Generation})
		node.Change(ui.ChangeEvent{Text: "3k"})
	})
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) || s.fieldContext != nil {
		t.Fatal("field context accepted another document or leaked after refusal")
	}
}
