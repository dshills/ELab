package workbench

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExampleDrawingPreviewPreservesDocumentExports(t *testing.T) {
	// Recorded from the original report renderer before adding drawing-only
	// previews. A chooser must not change ordinary schematic export bytes.
	want := map[string]string{
		"led":           "385417a71433c373a10cf610952901e83c83cf362ec47ddd6fd79160a87b756a",
		"rc":            "26fd7d010c2f259def665e9073bf7cfd894f6cd48d56306d65c06a42462be7a0",
		"hbridge-motor": "621340378f841713e6ca3bac09c12550e990779c2e95b92b2959d5f9b8b000ea",
	}
	for id, expected := range want {
		d, _ := findExample(id)
		display := CircuitDisplay{Preferences: DefaultDisplayPreferences()}
		report := schematicSVGWithAnnotations(d.graph, 0, 0, display, configuredSourceAnnotations(d.graph, d.settings), d.settings)
		hash := sha256.Sum256(report)
		if hex.EncodeToString(hash[:]) != expected {
			t.Fatal("ordinary document export changed", id)
		}
		preview, err := PreviewExample(id)
		if err != nil || strings.Contains(preview.SVG, "Components · generic") || strings.Contains(preview.SVG, "ELab · schematic · electrical revision") {
			t.Fatal("chooser contains document-report furniture", id, err)
		}
	}
}

func TestExampleBrowserCategoriesSearchSelectionAndPhysics(t *testing.T) {
	s := NewExample()
	t.Cleanup(s.Close)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := s.Snapshot()
	b := before.ExampleBrowser
	if b.Visible || b.Category != "start" || b.SelectedID != "led" || len(b.MatchingIDs) != 6 {
		t.Fatal("default chooser", b)
	}
	seen := map[string]bool{}
	for _, category := range b.Categories {
		call(t, s, "examples.category.set", PartCategoryParams{Category: category.ID}, category.ID)
		b = s.Snapshot().ExampleBrowser
		if category.Count != len(b.MatchingIDs) {
			t.Fatal("category count differs", category, b)
		}
		if category.ID != "all" && category.ID != "start" {
			for _, id := range b.MatchingIDs {
				if seen[id] || exampleCategory(id) != category.ID {
					t.Fatal("duplicate/misclassified example", id)
				}
				seen[id] = true
			}
		}
	}
	if len(seen) != len(ExampleCatalog()) || len(seen) != 78 {
		t.Fatal("catalog coverage", len(seen))
	}
	call(t, s, "examples.category.set", PartCategoryParams{Category: "timers"}, "timers-again")
	call(t, s, "examples.search.set", SearchParams{Query: "  REVERSE pwm "}, "search")
	b = s.Snapshot().ExampleBrowser
	if b.Category != "timers" || !reflect.DeepEqual(b.MatchingIDs, []string{"hbridge-motor"}) {
		t.Fatal("global word search", b)
	}
	call(t, s, "examples.select", IDParams{ID: "hbridge-motor"}, "select")
	call(t, s, "examples.visibility.set", ScopeVisibilityParams{Visible: true}, "show")
	call(t, s, "examples.visibility.set", ScopeVisibilityParams{Visible: false}, "hide")
	after := s.Snapshot()
	after.ExampleBrowser = before.ExampleBrowser
	after.Revision, after.CommandRevision, after.LastActor = before.Revision, before.CommandRevision, before.LastActor
	if !reflect.DeepEqual(before, after) {
		t.Fatal("browsing changed document, history or physical state")
	}
	call(t, s, "examples.search.set", SearchParams{Query: "no-circuit-has-this-name"}, "empty")
	if b = s.Snapshot().ExampleBrowser; len(b.MatchingIDs) != 0 || b.SelectedID != "" {
		t.Fatal("empty results retained an eligible Open target", b)
	}
	refuse(t, s, "examples.select", IDParams{ID: "led"}, "invalid-params")
	call(t, s, "examples.search.set", SearchParams{Query: "  "}, "clear")
	if len(s.Snapshot().ExampleBrowser.MatchingIDs) != 3 {
		t.Fatal("clearing search did not restore category")
	}
	refuse(t, s, "examples.category.set", PartCategoryParams{Category: "missing"}, "invalid-params")
	refuse(t, s, "examples.search.set", SearchParams{Query: strings.Repeat("µ", 65)}, "invalid-params")
	refuse(t, s, "examples.search.set", SearchParams{Query: "a\x00b"}, "invalid-params")
}

func TestExamplePreviewsAreExactPureOwnedAndDoNotLoad(t *testing.T) {
	s := NewExample()
	t.Cleanup(s.Close)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := s.Snapshot()
	for _, info := range ExampleCatalog() {
		raw, _ := json.Marshal(IDParams{ID: info.ID})
		r := s.Dispatch(t.Context(), Request{Version: 1, ID: info.ID, Command: "example.preview", Params: raw}, "reader")
		if r.Error != nil {
			t.Fatal(r.Error)
		}
		p := r.Result.(ExamplePreview)
		d, ok := findExample(info.ID)
		hash := contentHash(d.graph, d.settings)
		if !ok || p.Info != info || p.Category == "" || p.SourceSHA256 != hex.EncodeToString(hash[:]) || !strings.Contains(p.SVG, "<svg") || len(p.SVG) > 256*1024 {
			t.Fatal("incorrect/unbounded preview", info.ID)
		}
		if strings.Contains(p.SVG, "<script") || strings.Contains(p.SVG, "<image") {
			t.Fatal("preview unexpectedly uses an external/active resource", info.ID)
		}
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("query loaded/changed a document")
	}
	copy := s.Snapshot()
	copy.ExampleBrowser.MatchingIDs[0] = "corrupt"
	copy.ExampleBrowser.Categories[0].Name = "corrupt"
	if fresh := s.Snapshot(); fresh.ExampleBrowser.MatchingIDs[0] == "corrupt" || fresh.ExampleBrowser.Categories[0].Name == "corrupt" {
		t.Fatal("borrowed browser metadata")
	}
	for _, raw := range []string{`{}`, `{"id":null}`, `{"id":"led","extra":true}`} {
		r := s.Dispatch(t.Context(), Request{Version: 1, ID: "bad", Command: "example.preview", Params: json.RawMessage(raw)}, "reader")
		if r.Error == nil || r.Error.Code != "invalid-params" {
			t.Fatal("invalid preview params admitted", raw, r.Error)
		}
	}
	r := s.Dispatch(t.Context(), Request{Version: 1, ID: "missing", Command: "example.preview", Params: json.RawMessage(`{"id":"../missing"}`)}, "reader")
	if r.Error == nil || r.Error.Code != "not-found" {
		t.Fatal(r.Error)
	}
}

func TestExampleBrowserLoadingRetainsUnsavedProtectionAndSessionScope(t *testing.T) {
	s := NewExample()
	t.Cleanup(s.Close)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "edit")
	call(t, s, "examples.visibility.set", ScopeVisibilityParams{Visible: true}, "show")
	refuse(t, s, "example.load", ExampleLoadParams{ID: "rc"}, "unsaved-changes")
	if !s.Snapshot().ExampleBrowser.Visible {
		t.Fatal("refused load closed the chooser")
	}
	call(t, s, "example.load", ExampleLoadParams{ID: "rc", DiscardChanges: true}, "load")
	if s.Snapshot().ExampleBrowser.Visible || s.Snapshot().Document.Dirty {
		t.Fatal("accepted template did not close cleanly")
	}
	for _, id := range []string{"examples.visibility.set", "examples.category.set", "examples.search.set", "examples.select", "example.preview"} {
		schema := navigationSchema(t, id)
		if schema["x-command-contract"] == nil {
			t.Fatal("missing public contract", id)
		}
	}
}

func TestExampleBrowserIsNotSavedInSchematics(t *testing.T) {
	s := New()
	t.Cleanup(s.Close)
	call(t, s, "example.load", ExampleLoadParams{ID: "led"}, "template")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "examples.category.set", PartCategoryParams{Category: "motors"}, "category")
	call(t, s, "examples.search.set", SearchParams{Query: "reverse"}, "search")
	call(t, s, "examples.visibility.set", ScopeVisibilityParams{Visible: true}, "show")
	path := filepath.Join(t.TempDir(), "Browser.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	data, err := os.ReadFile(path)
	if err != nil || strings.Contains(string(data), "example_browser") {
		t.Fatal("presentation state leaked into the document", err)
	}
	fresh := New()
	t.Cleanup(fresh.Close)
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	b := fresh.Snapshot().ExampleBrowser
	if b.Visible || b.Category != "start" || b.Query != "" || b.SelectedID != "led" {
		t.Fatal("new session inherited a file's chooser state", b)
	}
}
