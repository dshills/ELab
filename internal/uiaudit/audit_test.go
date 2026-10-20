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

func TestInventoryTracksCursorActionRouteArgument(t *testing.T) {
	dir := t.TempDir()
	source := `package main
func view(s *screen, route string) {
 s.cursorAction(state, "select-b", "Label fake.command", "scope.cursor.select", "b")
 s.cursorAction(state, "dynamic", "Dynamic", route, "a")
 s.cursorAction(state, "unknown", "Clear", "missing.command", "a")
 s.recordedCursorAction(state, "recorded", "Label fake.command", "recording.cursor.clear", "a")
 s.cursorActionFor(state, "relay", "Dynamic", route, "a", "capture-1")
}`
	if err := os.WriteFile(filepath.Join(dir, "view.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := Audit(dir)
	if err != nil || len(r.Issues) != 1 || !strings.Contains(r.Issues[0], "missing.command") || len(r.Sites) != 5 {
		t.Fatal("cursor factory route/unknown command detection failed", r.Issues, err)
	}
	if len(r.Sites[0].Commands) != 1 || r.Sites[0].Commands[0] != "scope.cursor.select" || r.Sites[1].Status != "requires-binding-review" || len(r.Sites[1].Commands) != 0 {
		t.Fatal("cursor command argument confused with label or dynamic route", r.Sites)
	}
	if len(r.Sites[3].Commands) != 1 || r.Sites[3].Commands[0] != "recording.cursor.clear" || r.Sites[4].Status != "requires-binding-review" {
		t.Fatal("recorded cursor factory or captured-recording relay lost", r.Sites)
	}
}

func TestInventoryTracksPlacementFieldRelay(t *testing.T) {
	dir := t.TempDir()
	source := `package main
func view(s *screen,route string) {
 s.changePlacementField("drawing.position.set",params)
 s.changePlacementField(route,params)
 s.changePlacementField("missing.command",params)
}`
	if err := os.WriteFile(filepath.Join(dir, "view.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := Audit(dir)
	if err != nil || len(r.Issues) != 1 || !strings.Contains(r.Issues[0], "missing.command") || len(r.Sites) != 3 {
		t.Fatal("placement relay inventory failed", r.Issues, err)
	}
	if len(r.Sites[0].Commands) != 1 || r.Sites[0].Commands[0] != "drawing.position.set" || r.Sites[1].Status != "requires-binding-review" {
		t.Fatal("placement relay lost literal or dynamic route", r.Sites)
	}
}

func TestInventoryTracksFileDecisionCallbackFactory(t *testing.T) {
	dir := t.TempDir()
	source := `package main
func view(s *screen) {
 s.withFileDecision(decision,true,func(){ s.invoke("project.clear",params) })
}`
	if err := os.WriteFile(filepath.Join(dir, "view.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	report, err := Audit(dir)
	if err != nil || len(report.Issues) != 0 || len(report.Sites) != 2 || report.Sites[0].Kind != "callback-factory" || report.Sites[0].Status != "requires-binding-review" || len(report.Sites[1].Commands) != 1 || report.Sites[1].Commands[0] != "project.clear" {
		t.Fatal("decision callback factory or nested ingress lost", report.Sites, report.Issues, err)
	}
}

func TestInventoryTracksValueSettlementFactories(t *testing.T) {
	dir := t.TempDir()
	source := `package main
func view(s *screen) {
 s.valueApplyAction(state, "R1", "Apply fake.command")
 s.valuePresetAction(state, "r-2k", "Preset missing.command", params)
}`
	if err := os.WriteFile(filepath.Join(dir, "view.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := Audit(dir)
	if err != nil || len(r.Issues) != 0 || len(r.Sites) != 2 {
		t.Fatal("value settlement factories omitted or labels mistaken for routes", r.Sites, r.Issues, err)
	}
	for _, site := range r.Sites {
		if site.Kind != "callback-factory" || site.Status != "requires-binding-review" || len(site.Commands) != 0 {
			t.Fatal("factory presence must not invent literal or runtime coverage", site)
		}
	}
}

func TestInventoryTracksVerificationSettlementFactory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "view.go"), []byte(`package main
 func view(s *screen) { s.verificationAction(state,false); s.verificationAction(state,true) }`), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := Audit(dir)
	if err != nil || len(r.Issues) != 0 || len(r.Sites) != 2 {
		t.Fatal("verification factories omitted", r.Sites, r.Issues, err)
	}
	for _, site := range r.Sites {
		if site.Kind != "callback-factory" || site.Status != "requires-binding-review" || len(site.Commands) != 0 {
			t.Fatal("factory must not invent routes or runtime coverage", site)
		}
	}
}

func TestInventoryTracksComponentStartupDefaultFactory(t *testing.T) {
	dir := t.TempDir()
	source := `package main; func view(s *screen) { s.componentStartupDefault(state, part) }`
	if err := os.WriteFile(filepath.Join(dir, "view.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	report, err := Audit(dir)
	if err != nil || len(report.Issues) != 0 || len(report.Sites) != 1 {
		t.Fatal(report, err)
	}
	site := report.Sites[0]
	if site.Kind != "callback-factory" || site.Status != "requires-binding-review" || len(site.Commands) != 0 {
		t.Fatal("factory must not invent routes or runtime coverage", site)
	}
}

func TestInventoryTracksRecordedPlaySettlementFactory(t *testing.T) {
	dir := t.TempDir()
	source := `package main; func view(s *screen) { s.recordedPlayAction(state) }`
	if err := os.WriteFile(filepath.Join(dir, "view.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	report, err := Audit(dir)
	if err != nil || len(report.Issues) != 0 || len(report.Sites) != 1 {
		t.Fatal(report, err)
	}
	site := report.Sites[0]
	if site.Kind != "callback-factory" || site.Status != "requires-binding-review" || len(site.Commands) != 0 {
		t.Fatal("factory must not invent routes or runtime coverage", site)
	}
}

func TestInventoryTracksRelatedValueActionRoute(t *testing.T) {
	directory := t.TempDir()
	source := `package main
func view(s *screen,route string) {
 s.valueRelatedAction(state,"copy","Label fake.command",params,true,"component.duplicate",encode)
 s.valueRelatedAction(state,"dynamic","Dynamic",params,false,route,encode)
 s.valueRelatedAction(state,"unknown","Rotate",params,true,"missing.command",encode)
}`
	if err := os.WriteFile(filepath.Join(directory, "view.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	report, err := Audit(directory)
	if err != nil || len(report.Issues) != 1 || !strings.Contains(report.Issues[0], "missing.command") || len(report.Sites) != 3 {
		t.Fatal("related value route tracking failed", report.Sites, report.Issues, err)
	}
	if len(report.Sites[0].Commands) != 1 || report.Sites[0].Commands[0] != "component.duplicate" || report.Sites[1].Status != "requires-binding-review" || len(report.Sites[1].Commands) != 0 {
		t.Fatal("route confused with text or dynamic value action", report.Sites)
	}
}
