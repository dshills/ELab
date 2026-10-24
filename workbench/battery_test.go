package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func batteryLoadGraph(resistance string) Graph {
	b := newExampleBuilder()
	b.part("B1", "battery", "9", -200, 0, 0)
	b.ground("GND", "B1.negative", -200, 180)
	b.part("R1", "resistor", resistance, 100, 0, 0)
	b.wire("B1.positive", "R1.1")
	b.wire("R1.2", "GND.reference")
	return b.graph
}
func batteryKernel(t *testing.T, g Graph) *simulationKernel {
	t.Helper()
	c, _, d := g.Compile()
	if d != nil {
		t.Fatal(d)
	}
	cfg := DefaultTransientSettings()
	cfg.StepSeconds = .01
	k, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: cfg, mode: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	return k
}
func TestBatterySagDischargeAndEnergyReference(t *testing.T) {
	g := batteryLoadGraph("100")
	g.Components[0].Power.Battery = &BatteryConfig{.0001, 10, .8}
	k := batteryKernel(t, g)
	initial := k.mixed.result.Batteries[0]
	q := .8 * 9
	capacitance := .0001 * 3600 / 9
	if math.Abs(initial.Current-q/110) > 1e-10 || math.Abs(initial.Voltage-q*100/110) > 1e-9 {
		t.Fatal("Thevenin sag", initial)
	}
	trace := k.trace
	for i := 0; i < 100; i++ {
		next, updated, err := advanceMixed(k.mixed, trace, float64(i+1)*.01)
		if err != nil {
			t.Fatal(err)
		}
		k.mixed, trace = next, updated
	}
	r := k.mixed.result
	b := r.Batteries[0]
	want := q * math.Exp(-1/(110*capacitance))
	if math.Abs(b.OpenCircuitVoltage-want) > 2e-6 || math.Abs(b.Current-want/110) > 1e-8 {
		t.Fatal("charge integral differs from analytic reservoir", b, want)
	}
	if math.Abs(r.PowerResidual) > 1e-8 || r.KCLResidual > 1e-9 || math.Abs(b.Energy-.5*capacitance*want*want) > 1e-6 || b.InternalHeat <= 0 {
		t.Fatal("energy/KCL", r)
	}
	if len(r.Capacitors) != 0 || len(r.Resistors) != 1 || len(r.Sources) != 1 || len(r.Nodes) != len(k.compiled.Nets) {
		t.Fatal("private storage leaked into authored inventory", r)
	}
	if err := validateRecordedPower(k.compiled, g, r); err != nil {
		t.Fatal(err)
	}
	altered := cloneMixed(r)
	altered.Batteries[0].Charge += .1
	if validateRecordedPower(k.compiled, g, altered) == nil {
		t.Fatal("forged charge accepted")
	}
}
func TestBatteryFrozenDCAndEmptyStartup(t *testing.T) {
	g := batteryLoadGraph("100")
	g.Components[0].Power.Battery = &BatteryConfig{.1, 10, .5}
	r, _, err := solveDCOperatingPoint(t.Context(), g, nil)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(r.Batteries[0].Voltage-4.5*100/110) > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
		t.Fatal(r)
	}
	g.Components[0].Power.Battery.Charge = 0
	k := batteryKernel(t, g)
	if b := k.mixed.result.Batteries[0]; b.Current != 0 || b.State != "empty" {
		t.Fatal(b)
	}
}
func TestBatteryCommandsHistoryPersistenceAndOwnership(t *testing.T) {
	s := New()
	defer s.Close()
	batteryCall(t, s, "simulation.pause", struct{}{})
	g := batteryLoadGraph("100")
	s.mu.Lock()
	s.graph = g
	s.mu.Unlock()
	batteryCall(t, s, "battery.configure", BatteryParams{"B1", .0001, 10, .75})
	batteryCall(t, s, "simulation.restart", struct{}{})
	s.Wait(t.Context())
	before := s.Snapshot()
	if len(before.Mixed.Batteries) != 1 {
		t.Fatal(before)
	}
	path := filepath.Join(t.TempDir(), "battery.elab")
	batteryCall(t, s, "project.save", SaveParams{Path: path})
	s.Wait(t.Context())
	fresh := New()
	defer fresh.Close()
	batteryCall(t, fresh, "project.load", LoadParams{Path: path})
	fresh.Wait(t.Context())
	after := fresh.Snapshot()
	if after.PowerModel == nil || !reflect.DeepEqual(before.Components, after.Components) || math.Abs(after.Mixed.Batteries[0].Charge-.75) > 1e-12 {
		t.Fatal("roundtrip", after)
	}
	after.Components[0].Power.Battery.Charge = 0
	if s.Snapshot().Components[0].Power.Battery.Charge != .75 || fresh.Snapshot().Components[0].Power.Battery.Charge != .75 {
		t.Fatal("snapshot aliases configuration")
	}
	for _, bad := range []BatteryParams{{"B1", 0, 1, 1}, {"B1", 1, 0, 1}, {"B1", 1, 1, 2}, {"R1", 1, 1, 1}} {
		state := s.Snapshot()
		raw, _ := json.Marshal(bad)
		reply := s.Dispatch(t.Context(), Request{Version: 1, ID: "bad", Key: fmt.Sprintf("bad-%g-%g-%g-%s", bad.CapacityAh, bad.Resistance, bad.Charge, bad.ID), Command: "battery.configure", Params: raw, ExpectedCommandRevision: &state.CommandRevision, ExpectedDocumentGeneration: &state.Document.Generation}, "test")
		if reply.Error == nil {
			t.Fatal("invalid accepted", bad)
		}
		if !reflect.DeepEqual(state.Components, s.Snapshot().Components) {
			t.Fatal("invalid mutated graph")
		}
	}
}

func batteryCall(t *testing.T, s *Service, cmd string, p any) Response {
	t.Helper()
	return call(t, s, cmd, p, fmt.Sprintf("battery-%s-%d", cmd, s.Snapshot().CommandRevision))
}

func TestBatteryExamplesCaptureAndSwitchContinuity(t *testing.T) {
	for _, id := range []string{"battery-led", "battery-regulator"} {
		t.Run(id, func(t *testing.T) {
			s := New()
			defer s.Close()
			batteryCall(t, s, "example.load", ExampleLoadParams{ID: id})
			batteryCall(t, s, "simulation.pause", struct{}{})
			batteryCall(t, s, "simulation.restart", struct{}{})
			start := settled(t, s)
			initialLED := 0.
			if len(start.Mixed.LEDs) > 0 {
				initialLED = start.Mixed.LEDs[0].Current
			}
			if id == "battery-led" {
				batteryCall(t, s, "switch.set", SwitchParams{ID: "S1", Closed: true})
				batteryCall(t, s, "simulation.step", StepParams{Duration: .005})
				settled(t, s)
				after := s.Snapshot()
				if after.Mixed.Batteries[0].Charge >= 1 || after.Mixed.Batteries[0].Voltage >= start.Mixed.Batteries[0].Voltage*.8 {
					t.Fatal("added load did not sag battery", after.Mixed)
				}
			}
			duration, step := .05, .005
			_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:B1", "i:B1"}}, "capture-"+id)
			record := s.operations[view.Operation.ID].record
			raw, _ := json.Marshal(VerificationDocument{Metadata: record.metadata, Rows: record.rows})
			if _, err := decodeRecording(raw); err != nil {
				t.Fatal("battery capture", err)
			}
			var object map[string]any
			json.Unmarshal(raw, &object)
			object["metadata"].(map[string]any)["mixed"].(map[string]any)["batteries"].([]any)[0].(map[string]any)["stored_energy_j"] = 42
			bad, _ := json.Marshal(object)
			if _, err := decodeRecording(bad); err == nil {
				t.Fatal("forged battery energy accepted")
			}
			if initialLED < .001 && id == "battery-led" {
				t.Fatal("LED example did not light")
			}
			svg := string(schematicSVG(Graph{Components: s.Snapshot().Components, Wires: s.Snapshot().Wires}, 1, 1))
			if !strings.Contains(svg, "power_model") || !strings.Contains(svg, "startup") || !strings.Contains(svg, "ESR") {
				t.Fatal("export omitted battery settings")
			}
		})
	}
}
func TestBatteryLargeCapacitySmallStepAndFailedWorkPreserveCharge(t *testing.T) {
	g := batteryLoadGraph("100")
	g.Components[0].Power.Battery = &BatteryConfig{100, .01, 1}
	k := batteryKernel(t, g)
	next, _, err := advanceMixed(k.mixed, k.trace, 1e-6)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(next.result.PowerResidual) > 1e-7 || next.result.KCLResidual > 1e-9 {
		t.Fatal("large-capacity current/energy balance", next.result)
	}
	before, _ := json.Marshal(k.mixed.result)
	budget := newIntegrationBudget()
	budget.elapsed = func() time.Duration { return MaxIntegrationWallTime + 1 }
	_, _, err = advanceMixedBudget(k.mixed, k.trace, .01, budget)
	after, _ := json.Marshal(k.mixed.result)
	if err == nil || string(after) != string(before) {
		t.Fatal("failed sample consumed battery charge")
	}
}

func TestBatteryExampleDischargeDimsLEDAndCausesDropout(t *testing.T) {
	for _, regulator := range []bool{false, true} {
		g := BatteryExampleGraph(regulator)
		k := batteryKernel(t, g)
		trace := k.trace
		initial := k.mixed.result
		for j := 0; j < 81; j++ {
			end := .005 + float64(j)*.1
			next, updated, err := advanceMixed(k.mixed, trace, end)
			if err != nil {
				t.Fatal("discharge example", regulator, j, err)
			}
			k.mixed, trace = next, updated
		}
		final := k.mixed.result
		if final.Batteries[0].Charge >= initial.Batteries[0].Charge*.8 {
			t.Fatal("example failed to consume visible charge", regulator, final.Batteries)
		}
		if regulator {
			if final.Regulators[0].State != "dropout" || final.Regulators[0].OutputVoltage > 4.5 {
				t.Fatal("battery failed to cause dropout", final.Regulators)
			}
		} else if final.LEDs[0].Current >= initial.LEDs[0].Current*.5 {
			t.Fatal("LED failed to fade", final.LEDs)
		}
	}
}
