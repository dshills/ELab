package main

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func awaitInductorCircuit(t *testing.T, service *workbench.Service) workbench.Snapshot {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := service.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	state := service.Snapshot()
	if state.Mixed == nil || state.Mixed.Stale {
		t.Fatal("inductor circuit did not initialize", state.Diagnostics)
	}
	return state
}

func TestInductorBlankConstructionStartupAndProbeParity(t *testing.T) {
	for _, fixture := range []workbench.Graph{workbench.RLExampleGraph(), workbench.RLCExampleGraph(), workbench.FlybackExampleGraph()} {
		t.Run(fmt.Sprint(len(fixture.Components)), func(t *testing.T) {
			user, agent, ids := buildNetworkFixtureUIAndAgent(t, fixture)
			id := ids["L1"]
			uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: id})
			for _, probe := range []string{"v:" + id, "i:" + id} {
				if _, ok := findNode(user.view(), "plot-"+probe); !ok {
					t.Fatal("missing coil probe action", probe)
				}
			}
			uiAction(t, user, "inductor-initial", &ui.ChangeEvent{Text: "-20 mA"})
			agentAction(t, agent, "inductor.initial.set", workbench.ValueParams{ID: id, Value: "-20 mA"})
			// A negative current is valid for the passive examples. For the
			// MOS switch it can require a voltage beyond the model envelope;
			// reset before running rather than manufacturing a return path.
			if len(fixture.Components) > 5 {
				uiAction(t, user, "inductor-initial-default", nil)
				agentAction(t, agent, "inductor.initial.set", workbench.ValueParams{ID: id, Value: "default"})
			}
			uiAction(t, user, "simulation.restart", nil)
			agentAction(t, agent, "simulation.restart", struct{}{})
			left, right := awaitInductorCircuit(t, user.service), awaitInductorCircuit(t, agent)
			if !reflect.DeepEqual(left.Mixed.Inductors, right.Mixed.Inductors) || !reflect.DeepEqual(left.Components, right.Components) {
				t.Fatal("native startup differs from shared agent commands", left.Diagnostics, right.Diagnostics)
			}
			for sample := 0; sample < 5; sample++ {
				uiAction(t, user, "transient-step", nil)
				agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .001})
			}
			left, right = user.service.Snapshot(), agent.Snapshot()
			if left.Mixed.Stale || right.Mixed.Stale || !reflect.DeepEqual(left.Mixed.Inductors, right.Mixed.Inductors) || math.Abs(left.Transient.Sample.Time-.005) > 1e-12 {
				t.Fatal("native coil stepping diverged", left.Diagnostics, right.Diagnostics)
			}
			if _, ok := findNode(user.view(), "inductor-reading"); !ok {
				t.Fatal("selected coil lacks signed current/voltage/energy reading")
			}
		})
	}
}

func TestInductorStartupTextRetainsPrecisionAndRejectsStaleCommits(t *testing.T) {
	user, _, ids := buildNetworkFixtureUIAndAgent(t, workbench.RLExampleGraph())
	id := ids["L1"]
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: id})
	uiAction(t, user, "inductor-initial", &ui.ChangeEvent{Text: "-0.012345678901234567 A"})
	uiAction(t, user, "simulation.restart", nil)
	before := awaitInductorCircuit(t, user.service)
	part, _ := (workbench.Graph{Components: before.Components}).Component(id)
	field, ok := findNode(user.view(), "inductor-initial")
	if !ok || field.Kind != ui.TextField {
		t.Fatal("missing unit-aware startup text field")
	}
	value, err := workbench.ParseValue(field.Text, "A")
	if err != nil || math.Float64bits(value) != math.Float64bits(*part.InitialCurrent) {
		t.Fatal("startup field rounded float64 intent", field.Text, err)
	}
	uiAction(t, user, "inductor-initial", &ui.ChangeEvent{Text: field.Text})
	after := user.service.Snapshot()
	if before.ElectricalRevision != after.ElectricalRevision || before.RunID != after.RunID || before.Document != after.Document || !reflect.DeepEqual(before.Mixed, after.Mixed) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("unchanged startup commit changed physics/document/history")
	}
	field, _ = findNode(user.view(), "inductor-initial")
	agentAction(t, user.service, "inductor.initial.set", workbench.ValueParams{ID: id, Value: "30 mA"})
	before = user.service.Snapshot()
	user.app.Do(func() { field.Change(ui.ChangeEvent{Text: "10 mA"}) })
	if user.errorMessage == "" || !reflect.DeepEqual(before, user.service.Snapshot()) {
		t.Fatal("stale startup field replaced intervening agent intent")
	}
	field, _ = findNode(user.view(), "inductor-initial")
	user.app.Do(func() { field.Change(ui.ChangeEvent{Text: "20 mV"}) })
	if user.errorMessage == "" || !reflect.DeepEqual(before, user.service.Snapshot()) {
		t.Fatal("invalid startup unit changed state")
	}
}

func TestInductorSummaryNamesCoilAndIncludesCapacitorEnergy(t *testing.T) {
	for _, id := range []string{"rl", "rlc", "flyback"} {
		t.Run(id, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			uiAction(t, s, "example."+id, nil)
			uiAction(t, s, "simulation.pause", nil)
			uiAction(t, s, "simulation.restart", nil)
			initial := awaitInductorCircuit(t, s.service)
			for range int(math.Round(.005 / initial.TransientSettings.StepSeconds)) {
				uiAction(t, s, "transient-step", nil)
			}
			state := s.service.Snapshot()
			if state.Mixed == nil || state.Mixed.Stale || math.Abs(state.Transient.Sample.Time-.005) > 1e-12 {
				t.Fatal("fixture did not advance", state.Diagnostics)
			}
			coil := state.Mixed.Inductors[0]
			energy := coil.Energy
			voltageID, voltage := coil.ID, fmt.Sprintf("%.5g V", coil.Voltage)
			if id == "rlc" {
				capacitor := state.Mixed.Capacitors[0]
				if capacitor.Energy <= coil.Energy {
					t.Fatal("fixture must distinguish total energy from coil-only energy")
				}
				energy += capacitor.Energy
				voltageID, voltage = capacitor.ID, fmt.Sprintf("%.3f V", capacitor.Voltage)
			}
			want := map[string]string{
				"current-title": coil.ID + " CURRENT", "current-value": fmt.Sprintf("%.5g mA", coil.Current*1000),
				"voltage-title": voltageID + " VOLTAGE", "voltage-value": voltage,
				"power-title": "L + C STORED ENERGY", "power-value": fmt.Sprintf("%.3g µJ", energy*1e6),
			}
			// Selecting a resistor must not silently retarget these named
			// whole-network summaries away from the first coil/capacitor.
			uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: state.Mixed.Resistors[0].ID})
			s.app.Do(func() {
				root := s.view()
				for nodeID, text := range want {
					node, ok := findNode(root, nodeID)
					if !ok || node.Text != text {
						t.Errorf("%s: got %q, want %q", nodeID, node.Text, text)
					}
				}
			})
		})
	}
}
