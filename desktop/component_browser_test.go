package main

import (
	"reflect"
	"testing"

	"cogentcore.org/core/core"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestComponentBrowserUIAgentParity(t *testing.T) {
	user, closeUser := testScreen(t)
	defer closeUser()
	agent, closeAgent := testScreen(t)
	defer closeAgent()
	if user.service.Snapshot().PartCategory != "common" || matchingPart(user.view(), "nmos") {
		t.Fatal("default Common view absent")
	}
	for _, category := range []string{"semiconductors", "sources", "all", "common"} {
		uiAction(t, user, "part-category", &ui.ChangeEvent{SelectedID: category})
		agentAction(t, agent.service, "parts.category.set", workbench.PartCategoryParams{Category: category})
		if !sameBrowserState(user.service.Snapshot(), agent.service.Snapshot()) {
			t.Fatal("category UI/agent divergence", category)
		}
	}
	uiAction(t, user, "part-search", &ui.ChangeEvent{Text: "op-amp"})
	agentAction(t, agent.service, "parts.search.set", workbench.SearchParams{Query: "op-amp"})
	if !matchingPart(user.view(), "opamp") || !sameBrowserState(user.service.Snapshot(), agent.service.Snapshot()) {
		t.Fatal("global search parity")
	}
	uiAction(t, user, "part-list", &ui.ChangeEvent{SelectedID: "opamp"})
	agentAction(t, agent.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "opamp"})
	if !sameBrowserState(user.service.Snapshot(), agent.service.Snapshot()) {
		t.Fatal("placement parity")
	}
	uiAction(t, user, "part-category", &ui.ChangeEvent{SelectedID: "passives"})
	if user.service.Snapshot().PartSearch != "" || user.service.Snapshot().Drawing.Kind != "opamp" {
		t.Fatal("category did not clear query or changed tool")
	}
}

func TestComponentSymbolsDecorateOnlyCatalogRows(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	scene := core.NewScene("symbols")
	list := core.NewFrame(scene)
	list.SetName("part-list")
	symbols := browserSymbols()
	if len(symbols) != 18 {
		t.Fatal("incomplete symbol catalog")
	}
	unique := map[string]bool{}
	for _, part := range workbench.Library() {
		symbol := symbols[part.Kind]
		if symbol == "" || unique[string(symbol)] {
			t.Fatal("missing/duplicate symbol", part.Kind)
		}
		unique[string(symbol)] = true
		row := core.NewSwitch(list)
		row.SetName("choice:" + part.Kind)
		if !s.styleNativeChrome(scene) || row.IconOn != symbol || row.IconOff != symbol {
			t.Fatal("row symbol absent", part.Kind)
		}
		// Reconciliation resets native Switch artwork. Restore both states.
		row.SetType(core.SwitchSegmentedButton)
		if !s.styleNativeChrome(scene) || row.IconOff != symbol {
			t.Fatal("symbol lost on redraw")
		}
	}
	other := core.NewSwitch(scene)
	other.SetName("choice:resistor")
	if styleComponentRow(other, other.AsWidget()) {
		t.Fatal("unrelated choice decorated")
	}
}

func sameBrowserState(a, b workbench.Snapshot) bool {
	return a.PartCategory == b.PartCategory && a.PartSearch == b.PartSearch && reflect.DeepEqual(a.PartCategories, b.PartCategories) && reflect.DeepEqual(a.MatchingParts, b.MatchingParts) && reflect.DeepEqual(a.Drawing, b.Drawing) && reflect.DeepEqual(a.Components, b.Components) && reflect.DeepEqual(a.Document, b.Document) && a.CommandRevision == b.CommandRevision && a.ElectricalRevision == b.ElectricalRevision && a.RunID == b.RunID
}
