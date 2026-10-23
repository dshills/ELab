package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestUntouchedExampleMenuSwitchesWithoutSavePrompt(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	for _, tc := range []struct{ menu, id string }{
		{"example.reset", "led"}, {"example.rc", "rc"}, {"example.npn", "npn"}, {"example.nmos", "nmos"},
	} {
		uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: tc.menu})
		agentAction(t, agent, "example.load", workbench.ExampleLoadParams{ID: tc.id})
		state, want := s.service.Snapshot(), agent.Snapshot()
		if s.pendingCommand != "" || state.Document.Dirty || state.Example.Info.ID != tc.id {
			t.Fatalf("untouched example switch prompted or retained previous circuit: pending=%s document=%+v", s.pendingCommand, state.Document)
		}
		if !reflect.DeepEqual(state.Components, want.Components) || !reflect.DeepEqual(state.Wires, want.Wires) || state.Document.Dirty != want.Document.Dirty {
			t.Fatal("menu and agent example loading differ")
		}
		title, ok := findNode(s.view(), "title")
		if !ok || title.Text != state.Example.Info.Title {
			t.Fatalf("example title missing: %q", title.Text)
		}
	}
}

func TestDocumentNameUsesExampleTitleAndSavedFilename(t *testing.T) {
	s := &screen{}
	for _, info := range workbench.ExampleCatalog() {
		state := workbench.Snapshot{Example: &workbench.ExampleContext{Info: info}}
		for _, edited := range []bool{false, true} {
			state.Document.Dirty = edited
			state.Example.Modified = edited
			title, _ := findNode(s.documentTitle(state), "title")
			if title.Text != info.Title {
				t.Fatalf("example %s title=%q", info.ID, title.Text)
			}
			subtitle, _ := findNode(s.documentTitle(state), "document-subtitle")
			if edited && subtitle.Text != "Edited" || !edited && subtitle.Text == "Edited" {
				t.Fatal("title dirty indicator differs from document state")
			}
		}
		state.Document.Path = "/tmp/My saved circuit.elab"
		title, _ := findNode(s.documentTitle(state), "title")
		if title.Text != "My saved circuit.elab" {
			t.Fatal("example origin hides saved filename")
		}
	}
	if got := documentName(workbench.Snapshot{}); got != "Untitled circuit" {
		t.Fatal("blank document no longer has its fallback name", got)
	}
}
