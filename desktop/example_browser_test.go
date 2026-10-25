package main

import (
	"crypto/sha256"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func browserGridNode(t *testing.T, s *screen) ui.Node {
	t.Helper()
	var node ui.Node
	s.app.Do(func() {
		var ok bool
		node, ok = findNode(s.view(), "example-grid")
		if !ok || node.Collection == nil || node.SelectionChange == nil {
			t.Fatal("missing production example collection")
		}
	})
	return node
}

func TestExampleBrowserUserControlsMatchAgentAndProtectEdits(t *testing.T) {
	u, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	a := workbench.New()
	t.Cleanup(a.Close)
	uiAction(t, u, "example.reset", nil)
	agentAction(t, a, "example.load", workbench.ExampleLoadParams{ID: "led"})
	uiAction(t, u, "simulation.pause", nil)
	agentAction(t, a, "simulation.pause", struct{}{})
	before := u.service.Snapshot()
	uiAction(t, u, "examples-menu", nil)
	agentAction(t, a, "examples.visibility.set", workbench.ScopeVisibilityParams{Visible: true})
	uiAction(t, u, "example-category", &ui.ChangeEvent{SelectedID: "timers"})
	agentAction(t, a, "examples.category.set", workbench.PartCategoryParams{Category: "timers"})
	uiAction(t, u, "example-search", &ui.ChangeEvent{Text: "h-bridge"})
	agentAction(t, a, "examples.search.set", workbench.SearchParams{Query: "h-bridge"})
	grid := browserGridNode(t, u)
	u.app.Do(func() {
		grid.SelectionChange(ui.ListSelectionEvent{TargetID: "hbridge-motor", IDs: []string{"hbridge-motor"}})
	})
	agentAction(t, a, "examples.select", workbench.IDParams{ID: "hbridge-motor"})
	user, agent := u.service.Snapshot(), a.Snapshot()
	if !reflect.DeepEqual(user.ExampleBrowser, agent.ExampleBrowser) || !reflect.DeepEqual(before.Components, user.Components) || before.Document != user.Document || before.RunID != user.RunID {
		t.Fatal("browser differs across clients or changed the circuit")
	}
	uiAction(t, u, ui.SheetConfirmID(exampleBrowserID), nil)
	agentAction(t, a, "example.load", workbench.ExampleLoadParams{ID: "hbridge-motor"})
	user, agent = u.service.Snapshot(), a.Snapshot()
	if user.ExampleBrowser.Visible || user.Example == nil || user.Example.Info.ID != "hbridge-motor" || !reflect.DeepEqual(user.Components, agent.Components) || !reflect.DeepEqual(user.Wires, agent.Wires) {
		t.Fatal("user Open did not use the shared template command")
	}
	// Test protected Open separately on an actual edited LED document.
	uiAction(t, u, "example.reset", nil)
	if u.pendingCommand != "" {
		uiAction(t, u, "discard-continue", nil)
	}
	uiAction(t, u, "simulation.pause", nil)
	uiAction(t, u, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	uiAction(t, u, "value-entry", &ui.ChangeEvent{Text: "2k"})
	uiAction(t, u, "apply-value", nil)
	uiAction(t, u, "examples-menu", nil)
	uiAction(t, u, "example-search", &ui.ChangeEvent{Text: "RC charge"})
	grid = browserGridNode(t, u)
	u.app.Do(func() { grid.SelectionChange(ui.ListSelectionEvent{TargetID: "rc", IDs: []string{"rc"}}) })
	uiAction(t, u, ui.SheetConfirmID(exampleBrowserID), nil)
	if u.pendingCommand != "example.load" || u.service.Snapshot().Circuit.Resistance != 2000 {
		t.Fatal("Open bypassed dirty protection")
	}
	uiAction(t, u, "cancel-file-action", nil)
	if !u.service.Snapshot().ExampleBrowser.Visible || u.service.Snapshot().Circuit.Resistance != 2000 {
		t.Fatal("Cancel lost the chooser or edited circuit")
	}
	uiAction(t, u, ui.SheetConfirmID(exampleBrowserID), nil)
	uiAction(t, u, "discard-continue", nil)
	if state := u.service.Snapshot(); state.ExampleBrowser.Visible || state.Example == nil || state.Example.Info.ID != "rc" || state.Document.Dirty {
		t.Fatal("confirmed Open failed")
	}
}

func TestExampleBrowserHeldActionsRefuseAnAgentInterleaving(t *testing.T) {
	for _, kind := range []string{"selection", "open", "collection-open", "clear", "dismiss"} {
		t.Run(kind, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.reset", nil)
			uiAction(t, s, "simulation.pause", nil)
			uiAction(t, s, "examples-menu", nil)
			uiAction(t, s, "example-search", &ui.ChangeEvent{Text: "LED"})
			grid := browserGridNode(t, s)
			var held ui.Node
			s.app.Do(func() {
				switch kind {
				case "clear":
					held, _ = findNode(s.view(), "example-search-clear")
				case "dismiss":
					held, _ = findNode(s.view(), exampleBrowserID)
				default:
					held, _ = findNode(s.view(), ui.SheetConfirmID(exampleBrowserID))
				}
			})
			agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
			before := s.service.Snapshot()
			s.app.Do(func() {
				_ = s.view() // Reuse the retained component with newer callbacks.
				switch kind {
				case "selection":
					grid.SelectionChange(ui.ListSelectionEvent{TargetID: "npn", IDs: []string{"npn"}})
				case "collection-open":
					grid.Collection.Activate(ui.CollectionEvent{ItemID: "npn"})
				case "dismiss":
					held.Dismiss()
				default:
					held.Activate()
				}
			})
			if !reflect.DeepEqual(before, s.service.Snapshot()) || s.pendingCommand != "" {
				t.Fatal("old browser callback borrowed a newer command context")
			}
		})
	}
}

func TestExampleThumbnailsRenderAllExistingCircuits(t *testing.T) {
	var led, rc [32]byte
	for _, info := range workbench.ExampleCatalog() {
		t.Run(info.ID, func(t *testing.T) {
			preview, err := workbench.PreviewExample(info.ID)
			if err != nil {
				t.Fatal(err)
			}
			pixels, err := renderExampleThumbnail(preview)
			if err != nil || pixels == nil {
				t.Fatal("thumbnail render", err)
			}
			ink := 0
			var rgba []byte
			for y := 0; y < pixels.Size().Y; y++ {
				for x := 0; x < pixels.Size().X; x++ {
					p := pixels.NRGBAAt(x, y)
					rgba = append(rgba, p.R, p.G, p.B, p.A)
					if p.R < 210 || p.G < 210 || p.B < 210 {
						ink++
						if x < 2 || y < 2 || x >= pixels.Size().X-2 || y >= pixels.Size().Y-2 {
							t.Fatalf("schematic touches thumbnail clip at %d,%d", x, y)
						}
					}
				}
			}
			if ink < 30 {
				t.Fatal("blank schematic preview", ink)
			}
			if info.ID == "led" {
				led = sha256.Sum256(rgba)
			}
			if info.ID == "rc" {
				rc = sha256.Sum256(rgba)
			}
			if dir := os.Getenv("ELAB_EXAMPLE_THUMBNAIL_DIR"); dir != "" {
				if err := os.MkdirAll(dir, 0o700); err != nil {
					t.Fatal(err)
				}
				f, err := os.Create(filepath.Join(dir, info.ID+".png"))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(f, pixels)
				closeErr := f.Close()
				if err != nil || closeErr != nil {
					t.Fatal(err, closeErr)
				}
			}
		})
	}
	if led == rc || led == ([32]byte{}) || rc == ([32]byte{}) {
		t.Fatal("different circuits rendered the same fallback picture")
	}
}
