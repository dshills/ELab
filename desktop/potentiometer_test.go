package main

import (
	"math"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativePotentiometerSliderAndAgentParity(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	if err := smokeConstructPotentiometer(user); err != nil {
		t.Fatal(err)
	}
	agent := workbench.New()
	defer agent.Close()
	agentAction(t, agent, "example.reset", workbench.ReplaceParams{})
	agentAction(t, agent, "object.delete", workbench.IDParams{ID: "R1"})
	agentAction(t, agent, "component.place", workbench.PlaceParams{Kind: "potentiometer", Position: workbench.Point{X: 0, Y: -110}})
	id := agent.Snapshot().Selected
	for _, pair := range [][2]string{{"V1.positive", id + ".1"}, {id + ".2", "GND.reference"}, {id + ".wiper", "D1.anode"}} {
		agentAction(t, agent, "wire.connect", workbench.ConnectParams{From: pair[0], To: pair[1]})
	}
	for _, percent := range []float32{0, 25, 75, 100} {
		uiAction(t, user, "pot-wiper", &ui.ChangeEvent{Number: percent})
		agentAction(t, agent, "potentiometer.wiper.set", workbench.WiperParams{ID: id, Position: float64(percent) / 100})
		u, a := user.service.Snapshot(), agent.Snapshot()
		if !reflect.DeepEqual(u.Components, a.Components) || !reflect.DeepEqual(u.Mixed, a.Mixed) || !reflect.DeepEqual(u.PotentiometerModel, a.PotentiometerModel) {
			t.Fatal("native slider/agent pot differs")
		}
	}
	uiAction(t, user, "pot-wiper-exact", &ui.ChangeEvent{Number: 50})
	if math.Abs(user.service.Snapshot().Mixed.Potentiometers[0].WiperPosition-.5) > 1e-12 {
		t.Fatal("exact wiper percent failed")
	}
	uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: "20k"})
	uiAction(t, user, "apply-value", nil)
	if user.service.Snapshot().Mixed.Potentiometers[0].Resistance != 20000 {
		t.Fatal("native resistance edit failed")
	}
	node, ok := findNode(user.view(), "pot-wiper")
	if !ok || node.Kind != ui.Slider || node.Accessibility.Name == "" {
		t.Fatal("pot slider/accessibility absent")
	}
	if _, ok := findNode(user.view(), "pot-reading"); !ok {
		t.Fatal("pot V/I/power absent")
	}
	if err := ui.Validate(user.view()); err != nil {
		t.Fatal(err)
	}
	if mesh, err := user.mesh(*user.drawing.View); err != nil || mesh == nil {
		t.Fatal("pot symbol mesh absent")
	}
}
