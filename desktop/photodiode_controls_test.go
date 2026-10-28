package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestPhotodiodeInspectorUserAgentParityAndStaleRefusal(t *testing.T) {
	s, done := testScreen(t)
	defer done()
	a := workbench.New()
	defer a.Close()
	for _, service := range []*workbench.Service{s.service, a} {
		agentAction(t, service, "example.load", workbench.ExampleLoadParams{ID: "photodiode-load"})
		agentAction(t, service, "simulation.pause", struct{}{})
		awaitBlockRC(t, service)
	}
	uiAction(t, s, "photo-light", &ui.ChangeEvent{Number: 20})
	agentAction(t, a, "photodiode.light.set", workbench.OpticalLightParams{ID: "PD1", Light: 20})
	awaitBlockRC(t, s.service)
	awaitBlockRC(t, a)
	if s.service.Snapshot().Mixed == nil || a.Snapshot().Mixed == nil {
		t.Fatal("detector solution missing")
	}
	if !reflect.DeepEqual(s.service.Snapshot().Components, a.Snapshot().Components) || !reflect.DeepEqual(s.service.Snapshot().Mixed.Photodiodes, a.Snapshot().Mixed.Photodiodes) {
		t.Fatal("optical light user/agent parity")
	}
	uiAction(t, s, "photo-responsivity", &ui.ChangeEvent{Number: .6})
	part, exists := (workbench.Graph{Components: a.Snapshot().Components}).Component("PD1")
	if !exists || part.Device == nil || part.Device.Photo == nil {
		t.Fatal("detector config missing")
	}
	cfg := *part.Device.Photo
	cfg.Responsivity = .6
	agentAction(t, a, "photodiode.configure", workbench.PhotodiodeParams{ID: "PD1", Parameters: cfg})
	if !reflect.DeepEqual(s.service.Snapshot().Components, a.Snapshot().Components) {
		t.Fatal("detector config user/agent parity")
	}
	held, ok := findNode(s.view(), "photo-light")
	if !ok {
		t.Fatal("light slider absent")
	}
	agentAction(t, s.service, "photodiode.light.set", workbench.OpticalLightParams{ID: "PD1", Light: 30})
	awaitBlockRC(t, s.service)
	before := s.service.Snapshot().Components
	s.app.Do(func() { held.Change(ui.ChangeEvent{Number: 40}) })
	if !reflect.DeepEqual(before, s.service.Snapshot().Components) {
		t.Fatal("stale optical slider accepted")
	}
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
		var viewErr error
		s.app.Do(func() { viewErr = ui.Validate(s.view()) })
		if viewErr != nil {
			t.Fatal(theme, viewErr)
		}
	}
}
