package uiaudit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInventoryTracksLiteralDynamicCallbacksAndUnknownRoutes(t *testing.T) {
	dir := t.TempDir()
	source := `package main
func view(s *screen, route string) {
 s.action("pause", "Label mentioning fake.command", "simulation.pause", nil)
 s.action("dynamic", "Dynamic", route, nil)
 s.button("file.open", "Open")
 s.commandMenu("menu", "Choices", [][2]string{{"simulation.resume", "Resume"}, {"appearance.dark", "Dark"}})
 node := ui.Node{ID:"draft", Kind:ui.TextField, Change:func(e ui.ChangeEvent){s.invoke("editor.value.set", e.Text)}}
 node.Activate = func(){s.invokeAtCommandRevision("missing.command", nil, 1)}
}
`
	path := filepath.Join(dir, "view.go")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	// A fixture/smoke command must not invent shipped action coverage.
	if err := os.WriteFile(filepath.Join(dir, "fixture_smoke.go"), []byte(`package main; func smoke(){s.invoke("ignored.command", nil)}`), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := Audit(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Issues) != 1 || !strings.Contains(r.Issues[0], "missing.command") || len(r.Sources) != 1 {
		t.Fatal("unknown route or excluded fixture handling", r.Issues)
	}
	kinds := map[string]bool{}
	dynamic := false
	for _, site := range r.Sites {
		kinds[site.Kind] = true
		if site.Kind == "action-factory" && site.Status == "requires-binding-review" {
			dynamic = true
		}
	}
	if !dynamic || !kinds["node-Change"] || !kinds["callback-assignment"] || !kinds["command-menu"] {
		t.Fatal("dynamic or callback surfaces omitted", kinds)
	}
	before := r.Sources["desktop/view.go"]
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(source, "missing.command", "simulation.restart")), 0600); err != nil {
		t.Fatal(err)
	}
	r, err = Audit(dir)
	if err != nil || len(r.Issues) != 0 || r.Sources["desktop/view.go"] == before {
		t.Fatal("repaired route/source change not detected", r.Issues, err)
	}
}

func TestNativeInventoryMatchesReviewedLedger(t *testing.T) {
	root := filepath.Join("..", "..")
	r, err := Audit(filepath.Join(root, "desktop"))
	if err != nil || len(r.Issues) > 0 {
		t.Fatal("native source route errors", r.Issues, err)
	}
	want, err := os.ReadFile(filepath.Join(root, "research", "13-native-command-inventory.json"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(append(data, '\n')) {
		t.Fatal("native sources/routes changed; review and regenerate the source inventory")
	}
}
