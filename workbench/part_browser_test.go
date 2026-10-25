package workbench

import (
	"reflect"
	"testing"
)

func TestPartBrowserGroupsAndGlobalSearch(t *testing.T) {
	s := New()
	defer s.Close()
	before := s.Snapshot()
	if before.PartCategory != "common" || len(before.MatchingParts) != 6 || len(Library()) != 53 {
		t.Fatal("default catalog", before.PartCategory, len(before.MatchingParts))
	}
	expected := map[string]int{"common": 6, "all": 53, "sources": 5, "motors": 3, "passives": 9, "semiconductors": 10, "amplifiers": 4, "logic": 15, "switches": 4, "connections": 3}
	seen := map[string]bool{}
	for _, category := range before.PartCategories {
		call(t, s, "parts.category.set", PartCategoryParams{category.ID}, "category-"+category.ID)
		state := s.Snapshot()
		if len(state.MatchingParts) != expected[category.ID] || category.Count != expected[category.ID] {
			t.Fatalf("wrong %s count", category.ID)
		}
		for _, part := range state.MatchingParts {
			if category.ID != "common" && category.ID != "all" {
				if seen[part.Kind] || part.Category != category.ID {
					t.Fatal("duplicate/misclassified", part.Kind)
				}
				seen[part.Kind] = true
			}
			if category.ID == "common" && !part.Common {
				t.Fatal("noncommon", part.Kind)
			}
		}
	}
	if len(seen) != 53 {
		t.Fatal("missing parts", seen)
	}
	call(t, s, "parts.category.set", PartCategoryParams{"passives"}, "passives")
	call(t, s, "parts.search.set", SearchParams{" MOSFET "}, "global")
	if state := s.Snapshot(); len(state.MatchingParts) != 2 || state.PartCategory != "passives" {
		t.Fatal("search must escape browsing category")
	}
	call(t, s, "parts.search.set", SearchParams{"  "}, "clear")
	if len(s.Snapshot().MatchingParts) != 9 {
		t.Fatal("empty query did not restore category")
	}
	call(t, s, "parts.search.set", SearchParams{"battery"}, "alias")
	call(t, s, "parts.category.set", PartCategoryParams{"common"}, "common")
	after := s.Snapshot()
	if after.PartSearch != "" || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Document, after.Document) || before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || before.UndoName != after.UndoName {
		t.Fatal("browser changed document or physics")
	}
	refuse(t, s, "parts.category.set", PartCategoryParams{"not-a-category"}, "invalid-params")
	if !reflect.DeepEqual(after, s.Snapshot()) {
		t.Fatal("invalid category mutated state")
	}
	// Exported catalogs and retained response snapshots must own their metadata.
	copy := cloneSnapshot(after)
	copy.PartCategories[0].Name = "corrupt"
	copy.MatchingParts[0].Aliases[0] = "corrupt"
	if s.Snapshot().PartCategories[0].Name != "Common" || s.Snapshot().MatchingParts[0].Aliases[0] == "corrupt" {
		t.Fatal("shared browser metadata")
	}
}
