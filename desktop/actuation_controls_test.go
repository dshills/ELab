package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestActuatorControlsShareHintsAndDistinguishRequestedWiperFromCommittedReading(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	pot, _ := workbench.MakeComponent("P1", "potentiometer", workbench.Point{}, "10k")
	state := workbench.Snapshot{Mixed: &workbench.MixedResult{Potentiometers: []workbench.PotentiometerResult{{ID: "P1", WiperPosition: .5, WiperVoltage: 2.5}}}, InputWork: &workbench.InputWork{ID: "input-1", Status: "running", Wiper: &workbench.WiperParams{ID: "P1", Position: .75}}}
	s.app.Do(func() {
		root := ui.Node{Children: s.potentiometerControls(state, pot)}
		help, ok := findNode(root, "pot-actuation-help")
		if !ok || help.Text != workbench.PartActuationHint("potentiometer") {
			t.Fatal("native wiper explanation diverged from agent hint")
		}
		field, _ := findNode(root, "pot-wiper-exact")
		reading, _ := findNode(root, "pot-reading")
		if field.Number != 75 || !strings.Contains(reading.Text, "Committed") || !strings.Contains(reading.Text, "pending") || !strings.Contains(reading.Text, "2.5 V") {
			t.Fatal("requested wiper is presented as applied physics", field.Number, reading.Text)
		}
		state.Mixed.Stale = true
		root.Children = s.potentiometerControls(state, pot)
		reading, _ = findNode(root, "pot-reading")
		if !strings.HasPrefix(reading.Text, "Stale") {
			t.Fatal("pending label hid stale readings")
		}
		sw, _ := workbench.MakeComponent("S1", "switch", workbench.Point{}, "")
		closed := true
		state.SwitchStates = []workbench.SwitchState{{ID: "S1", DesiredClosed: false, AppliedClosed: &closed, Pending: true}}
		root.Children = s.switchControls(state, sw)
		help, ok = findNode(root, "switch-actuation-help")
		status, _ := findNode(root, "switch-state")
		if !ok || help.Text != workbench.PartActuationHint("switch") || !strings.Contains(status.Text, "applied closed") || !strings.Contains(status.Text, "pending next step") {
			t.Fatal("switch helper lost desired/applied state or shared explanation")
		}
	})
}

func TestExtractedWiperFieldRetainsRenderedRevisionGuard(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "component.place", workbench.PlaceParams{Kind: "potentiometer", ID: "P1"})
	var old ui.Node
	s.app.Do(func() { old, _ = findNode(s.view(), "pot-wiper-exact") })
	agentAction(t, s.service, "potentiometer.wiper.set", workbench.WiperParams{ID: "P1", Position: .75})
	before := s.service.Snapshot()
	s.app.Do(func() { old.Change(ui.ChangeEvent{Number: 25}) })
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) || s.fieldContext != nil {
		t.Fatal("old extracted wiper field overwrote newer agent value or leaked context")
	}
	uiAction(t, s, "pot-wiper-exact", &ui.ChangeEvent{Number: 25})
	if *s.service.Snapshot().Components[0].WiperPosition != .25 || s.fieldContext != nil {
		t.Fatal("fresh extracted field failed")
	}
}

func TestUnchangedDisplayedWiperPreservesExactStoredFraction(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "component.place", workbench.PlaceParams{Kind: "potentiometer", ID: "P1"})
	const exact = .12345678912345
	agentAction(t, s.service, "potentiometer.wiper.set", workbench.WiperParams{ID: "P1", Position: exact})
	before := s.service.Snapshot()
	var field ui.Node
	s.app.Do(func() { field, _ = findNode(s.view(), "pot-wiper-exact") })
	if float64(field.Number)/100 == exact {
		t.Fatal("fixture does not exercise native percentage rounding")
	}
	s.app.Do(func() { field.Change(ui.ChangeEvent{Number: field.Number}) })
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("unchanged displayed percentage changed stored fraction/physics/history", *s.service.Snapshot().Components[0].WiperPosition)
	}
}
