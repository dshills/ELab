package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeSiliconConstructionAndAgentParity(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	if err := smokeConstructDiode(user); err != nil {
		t.Fatal(err)
	}
	agent := workbench.New()
	defer agent.Close()
	agentAction(t, agent, "example.reset", workbench.ReplaceParams{})
	agentAction(t, agent, "object.delete", workbench.IDParams{ID: "D1"})
	agentAction(t, agent, "component.place", workbench.PlaceParams{Kind: "diode", Position: workbench.Point{X: 200, Y: 0}})
	id := agent.Snapshot().Selected
	for _, pair := range [][2]string{{"R1.2", id + ".anode"}, {id + ".cathode", "GND.reference"}} {
		agentAction(t, agent, "wire.connect", workbench.ConnectParams{From: pair[0], To: pair[1]})
	}
	u, a := user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(u.Components, a.Components) || !reflect.DeepEqual(u.Wires, a.Wires) || !reflect.DeepEqual(u.Mixed, a.Mixed) || !reflect.DeepEqual(u.DiodeModel, a.DiodeModel) {
		t.Fatal("native/agent silicon construction differs")
	}
	if _, ok := findNode(user.view(), "diode-model"); !ok {
		t.Fatal("diode limits absent")
	}
	if _, ok := findNode(user.view(), "mixed-selected-reading"); !ok {
		t.Fatal("diode voltage/current/power absent")
	}
	if err := ui.Validate(user.view()); err != nil {
		t.Fatal(err)
	}
	if mesh, err := user.mesh(*user.drawing.View); err != nil || mesh == nil {
		t.Fatal("silicon symbol mesh absent")
	}
	// Reverse the same diode with shared graph commands and verify signed leakage.
	for _, service := range []*workbench.Service{user.service, agent} {
		state := service.Snapshot()
		for _, wire := range state.Wires {
			if wire.From == id+".cathode" || wire.To == id+".anode" {
				agentAction(t, service, "object.delete", workbench.IDParams{ID: wire.ID})
			}
		}
		agentAction(t, service, "wire.connect", workbench.ConnectParams{From: "R1.2", To: id + ".cathode"})
		agentAction(t, service, "wire.connect", workbench.ConnectParams{From: id + ".anode", To: "GND.reference"})
	}
	u, a = user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(u.Mixed, a.Mixed) || u.Mixed.Diodes[0].Current >= 0 {
		t.Fatal("native reverse leakage differs")
	}
}
