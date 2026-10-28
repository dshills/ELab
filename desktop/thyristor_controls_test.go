package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestThyristorInspectorUserAgentParityAndStaleRefusal(t *testing.T) {
	s, done := testScreen(t)
	defer done()
	a := workbench.New()
	defer a.Close()
	for _, service := range []*workbench.Service{s.service, a} {
		agentAction(t, service, "example.load", workbench.ExampleLoadParams{ID: "triac-phase"})
		agentAction(t, service, "simulation.pause", struct{}{})
		awaitBlockRC(t, service)
	}
	uiAction(t, s, "thyristor-holding", &ui.ChangeEvent{Number: 1})
	cfg := workbench.DefaultThyristorConfig()
	cfg.Holding = .001
	agentAction(t, a, "thyristor.configure", workbench.ThyristorParams{ID: "TR1", Parameters: cfg})
	if !reflect.DeepEqual(s.service.Snapshot().Components, a.Snapshot().Components) {
		t.Fatal("thyristor model user/agent parity")
	}
	var held ui.Node
	var ok bool
	s.app.Do(func() { held, ok = findNode(s.view(), "thyristor-holding") })
	if !ok {
		t.Fatal("holding field absent")
	}
	cfg.Holding = .0015
	agentAction(t, s.service, "thyristor.configure", workbench.ThyristorParams{ID: "TR1", Parameters: cfg})
	before := s.service.Snapshot().Components
	s.app.Do(func() { held.Change(ui.ChangeEvent{Number: .5}) })
	if !reflect.DeepEqual(before, s.service.Snapshot().Components) {
		t.Fatal("stale thyristor field accepted")
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
