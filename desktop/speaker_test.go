package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/ui"
)

func TestSpeakerNativeExampleControlsDrawingAndAgentParity(t *testing.T) {
	u, cleanup := testScreen(t)
	defer cleanup()
	a := workbench.New()
	defer a.Close()
	uiAction(t, u, "examples-menu", nil)
	uiAction(t, u, "example-search", &ui.ChangeEvent{Text: "loudspeaker"})
	grid := browserGridNode(t, u)
	u.app.Do(func() {
		grid.SelectionChange(ui.ListSelectionEvent{TargetID: "speaker-amplifier", IDs: []string{"speaker-amplifier"}})
	})
	uiAction(t, u, ui.SheetConfirmID(exampleBrowserID), nil)
	agentAction(t, a, "example.load", workbench.ExampleLoadParams{ID: "speaker-amplifier"})
	for _, s := range []*workbench.Service{u.service, a} {
		agentAction(t, s, "simulation.pause", struct{}{})
		agentAction(t, s, "simulation.restart", struct{}{})
		awaitInductorCircuit(t, s)
	}
	uiAction(t, u, "speaker-fs", &ui.ChangeEvent{Number: 150})
	cfg := workbench.DefaultSpeakerConfig()
	cfg.Resonance = 150
	agentAction(t, a, "speaker.configure", workbench.SpeakerParams{ID: "SP1", Parameters: cfg})
	if !reflect.DeepEqual(u.service.Snapshot().Components, a.Snapshot().Components) {
		t.Fatal("speaker configure parity")
	}
	for _, s := range []*workbench.Service{u.service, a} {
		agentAction(t, s, "simulation.restart", struct{}{})
		awaitInductorCircuit(t, s)
	}
	uiAction(t, u, "transient-step", nil)
	agentAction(t, a, "simulation.step", workbench.StepParams{Duration: .0001})
	if !reflect.DeepEqual(u.service.Snapshot().Mixed, a.Snapshot().Mixed) {
		t.Fatal("speaker step parity")
	}
	for _, id := range []string{"speaker-reading", "audio-render", "speaker-coil-r", "speaker-coil-l", "speaker-motional-r", "speaker-motional-l"} {
		if _, ok := findNode(u.view(), id); !ok {
			t.Fatal("missing speaker control", id)
		}
	}
	held, ok := findNode(u.view(), "speaker-fs")
	if !ok {
		t.Fatal("missing resonance control")
	}
	agentAction(t, u.service, "speaker.configure", workbench.SpeakerParams{ID: "SP1", Parameters: workbench.DefaultSpeakerConfig()})
	before := u.service.Snapshot()
	u.app.Do(func() { held.Change(ui.ChangeEvent{Number: 200}) })
	if !reflect.DeepEqual(before.Components, u.service.Snapshot().Components) {
		t.Fatal("stale speaker edit replaced agent settings")
	}
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		agentAction(t, u.service, "appearance.theme.set", map[string]string{"theme": theme})
		if err := ui.Validate(u.view()); err != nil {
			t.Fatal(theme, err)
		}
		mesh, err := u.mesh(canvas.Viewport{Width: 1000, Height: 700, Zoom: .6})
		if err != nil || mesh == nil {
			t.Fatal("speaker schematic mesh", err)
		}
	}
}

func TestSpeakerNativeNumericBoundariesRemainExact(t *testing.T) {
	u, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, u.service, "example.load", workbench.ExampleLoadParams{ID: "speaker-amplifier"})
	agentAction(t, u.service, "simulation.pause", struct{}{})
	awaitInductorCircuit(t, u.service)
	for _, f := range []struct {
		id    string
		value float32
		read  func(workbench.SpeakerConfig) float64
		want  float64
	}{{"speaker-coil-l", .01, func(c workbench.SpeakerConfig) float64 { return c.Inductance }, 1e-5}, {"speaker-coil-l", 100, func(c workbench.SpeakerConfig) float64 { return c.Inductance }, .1}, {"speaker-motional-l", .1, func(c workbench.SpeakerConfig) float64 { return c.MotionalInductance }, .0001}, {"speaker-fs", 20, func(c workbench.SpeakerConfig) float64 { return c.Resonance }, 20}, {"speaker-fs", 1000, func(c workbench.SpeakerConfig) float64 { return c.Resonance }, 1000}, {"speaker-coil-r", 1, func(c workbench.SpeakerConfig) float64 { return c.Resistance }, 1}} {
		uiAction(t, u, f.id, &ui.ChangeEvent{Number: f.value})
		state := u.service.Snapshot()
		found := false
		for _, p := range state.Components {
			if p.ID == "SP1" {
				found = true
				if p.Device == nil || p.Device.Speaker == nil || f.read(*p.Device.Speaker) != f.want {
					t.Fatal("UI unit conversion crossed a domain boundary", f.id, p.Device)
				}
			}
		}
		if !found {
			t.Fatal("missing speaker")
		}
	}
}
