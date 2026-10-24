package workbench

import (
	"context"
	"reflect"
	"testing"
)

func TestThemeIsSharedSessionPresentationWithoutCircuitChanges(t *testing.T) {
	s := newGraphService(SolveLED, RCExampleGraph())
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	if _, err := s.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	before := s.Snapshot()
	for _, mode := range []string{"dark", "system", "light", "apollo"} {
		call(t, s, "appearance.theme.set", map[string]string{"theme": mode}, mode)
		after := s.Snapshot()
		if after.Theme != mode || after.RunID != before.RunID || after.ElectricalRevision != before.ElectricalRevision || !reflect.DeepEqual(after.Transient, before.Transient) || !reflect.DeepEqual(after.Document, before.Document) || after.UndoName != before.UndoName {
			t.Fatal("theme changed circuit, physical state, history, or saved preferences")
		}
	}
	current := s.Snapshot()
	refuse(t, s, "appearance.theme.set", map[string]string{"theme": "sepia"}, "invalid-params")
	if !reflect.DeepEqual(s.Snapshot(), current) {
		t.Fatal("invalid theme altered state")
	}
	call(t, s, "appearance.theme.set", map[string]string{"theme": "apollo"}, "apollo-final")
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "clear")
	if s.Snapshot().Theme != "apollo" {
		t.Fatal("project replacement reset session appearance")
	}
}

func TestApolloThemeIsAdvertisedToAgents(t *testing.T) {
	for _, mode := range inputProperty(navigationSchema(t, "appearance.theme.set"), "theme")["enum"].([]any) {
		if mode == "apollo" {
			return
		}
	}
	t.Fatal("Apollo is missing from the public command schema")
}
