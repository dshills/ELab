package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/internal/testutil"
)

func TestSensorResistanceLawsAndBounds(t *testing.T) {
	p, _ := MakeComponent("RT1", "thermistor", Point{}, "")
	// Decimal, 45-digit reference evaluation of the documented beta equation.
	for _, r := range [][2]float64{{-40, 401859.7246250026}, {0, 33620.60372143574}, {25, 10000}, {50, 3588.182581929083}, {100, 697.519772986188}, {125, 358.8338772400339}} {
		setSensorEnvironment(&p, r[0])
		got := sensorResistance(p)
		if math.Abs(got-r[1]) > 1e-7*math.Max(1, r[1]) {
			t.Fatal("NTC reference", r, got)
		}
	}
	p, _ = MakeComponent("LDR1", "ldr", Point{}, "")
	for _, r := range [][2]float64{{0, 1e6}, {.000001, 1e6}, {.1, 251188.643150958}, {1, 50118.7233627272}, {10, 10000}, {100, 1995.26231496888}, {1000, 398.107170553497}, {10000, 100}} {
		setSensorEnvironment(&p, r[0])
		if got := sensorResistance(p); math.Abs(got-r[1]) > 1e-8*math.Max(1, r[1]) {
			t.Fatal("LDR reference", r, got)
		}
	}
	for _, kind := range []string{"thermistor", "ldr"} {
		for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
			if validateSensorEnvironment(kind, v) == nil {
				t.Fatal("nonfinite accepted")
			}
		}
	}
	if validateSensorEnvironment("thermistor", -41) == nil || validateSensorEnvironment("thermistor", 126) == nil || validateSensorEnvironment("ldr", -1) == nil || validateSensorEnvironment("ldr", 10001) == nil {
		t.Fatal("environment domain")
	}
}
func sensorDivider(kind string, environment float64, capacitor bool) Graph {
	b := newExampleBuilder()
	b.part("V1", "dc-source", "5", -300, 0, 0)
	b.ground("GND", "V1.negative", -300, 200)
	b.part("SENSE", kind, "10k", 0, 0, 0)
	setSensorEnvironment(&b.graph.Components[len(b.graph.Components)-1], environment)
	b.part("R1", "resistor", "10k", 200, 100, 0)
	b.wire("V1.positive", "SENSE.1")
	b.wire("SENSE.2", "R1.1")
	b.wire("R1.2", "GND.reference")
	if capacitor {
		b.part("C1", "capacitor", "1u", 0, 300, 0)
		b.wire("C1.1", "R1.1")
		b.wire("C1.2", "GND.reference")
	}
	return b.graph
}
func TestSensorElectricalDividerAndConservation(t *testing.T) {
	for _, kind := range []string{"thermistor", "ldr"} {
		values := []float64{-40, 0, 25, 50, 125}
		if kind == "ldr" {
			values = []float64{0, .1, 1, 10, 100, 10000}
		}
		for _, value := range values {
			g := sensorDivider(kind, value, false)
			c, _, d := g.Compile()
			if d != nil {
				t.Fatal(d)
			}
			k, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
			if err != nil {
				t.Fatal(err)
			}
			r := k.mixed.result
			u := r.Sensors[0]
			wantI := 5 / (u.Resistance + 10000)
			if math.Abs(u.Current-wantI) > 1e-12 || math.Abs(r.Nodes[c.TerminalNet["R1.1"]]-wantI*10000) > 1e-9 || r.KCLResidual > 1e-10 || math.Abs(r.PowerResidual) > 1e-10 || math.Abs(u.Power-u.Current*u.Current*u.Resistance) > 1e-12 {
				t.Fatal("sensor Ohm/Kirchhoff law", kind, value, u, r.PowerResidual)
			}
			if err := validateRecordedSensors(c, g, r); err != nil {
				t.Fatal(err)
			}
		}
	}
}
func TestSensorExampleThresholdsAndRelayStorage(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	for _, example := range []string{"night-light", "thermostat"} {
		s := New()
		func() {
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: example}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			settled(t, s)
			call(t, s, "simulation.step", StepParams{Duration: .01}, "settle")
			a := settled(t, s)
			if example == "night-light" {
				for index, row := range []struct {
					lux float64
					on  bool
				}{{1, true}, {10, true}, {100, false}, {10, false}} {
					before := s.Snapshot()
					call(t, s, "sensor.light.set", LightParams{ID: "LDR1", Illuminance: row.lux}, fmt.Sprintf("lux-%d", index))
					a = settled(t, s)
					if a.Mixed.Controls[0].Memory.High != row.on || a.Mixed.Time != before.Mixed.Time || a.RunID != before.RunID || a.InputWork.Command != "sensor.light.set" {
						t.Fatal("night-light threshold/continuity", row, a.InputWork)
					}
					if (a.Mixed.LEDs[0].Current > .001) != row.on {
						t.Fatal("LED did not follow sensor")
					}
				}
			} else {
				coil := func(a Snapshot) ControlResult {
					for _, u := range a.Mixed.Controls {
						if u.Kind == "relay" {
							return u
						}
					}
					t.Fatal("missing coil")
					return ControlResult{}
				}
				if !coil(a).Memory.High {
					t.Fatal("cold heater relay did not pick up")
				}
				before := a
				call(t, s, "sensor.temperature.set", TemperatureParams{ID: "RT1", Temperature: 30}, "warm")
				a = settled(t, s)
				if coil(a).Current != coil(before).Current || a.Mixed.Time != before.Mixed.Time || a.RunID != before.RunID {
					t.Fatal("temperature reset coil/time/run")
				}
				call(t, s, "simulation.step", StepParams{Duration: .01}, "release")
				a = settled(t, s)
				if coil(a).Memory.High || a.Mixed.LEDs[0].Current > .001 {
					t.Fatal("hot heater failed to release")
				}
				call(t, s, "sensor.temperature.set", TemperatureParams{ID: "RT1", Temperature: 20}, "cold")
				settled(t, s)
				call(t, s, "simulation.step", StepParams{Duration: .01}, "pickup")
				if !coil(settled(t, s)).Memory.High {
					t.Fatal("cold heater failed to recover")
				}
			}
		}()
	}
}
func TestSensorChangePreservesCapacitorThenFollowsRCReference(t *testing.T) {
	s := newGraphService(SolveLED, sensorDivider("ldr", 100, true))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	call(t, s, "simulation.step", StepParams{Duration: .03}, "charge")
	before := settled(t, s)
	call(t, s, "sensor.light.set", LightParams{ID: "SENSE", Illuminance: 1}, "dark")
	a := settled(t, s)
	if a.Mixed.Capacitors[0].Voltage != before.Mixed.Capacitors[0].Voltage || a.Mixed.Time != before.Mixed.Time || a.RunID != before.RunID {
		t.Fatal("sensor movement disturbed stored charge/time")
	}
	call(t, s, "simulation.step", StepParams{Duration: .01}, "decay")
	a = settled(t, s)
	r := 50118.7233627272
	target := 5 * 10000 / (r + 10000)
	tau := r * 10000 / (r + 10000) * 1e-6
	want := target + (before.Mixed.Capacitors[0].Voltage-target)*math.Exp(-.01/tau)
	if math.Abs(a.Mixed.Capacitors[0].Voltage-want) > 1e-4 {
		t.Fatal("changed-environment RC law", a.Mixed.Capacitors[0].Voltage, want)
	}
}
func TestSensorStagedFailureCancellationAndPresentationMerge(t *testing.T) {
	for _, fail := range []bool{true, false} {
		s := New()
		func() {
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: "night-light"}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			before := settled(t, s)
			entered, release := blockInputSolver(t, s, fail)
			defer release()
			promptLiveCommand(t, s, "sensor.light.set", LightParams{ID: "LDR1", Illuminance: 1})
			awaitLiveSampleBlock(t, entered)
			pending := s.Snapshot()
			if pending.InputWork.Light == nil || !reflect.DeepEqual(pending.Components, before.Components) || !reflect.DeepEqual(pending.Mixed, before.Mixed) {
				t.Fatal("staged sensor leaked")
			}
			promptLiveCommand(t, s, "component.move", MoveParams{ID: "LDR1", Position: Point{X: -500, Y: -100}})
			if !fail {
				promptLiveCommand(t, s, "simulation.work.cancel", IDParams{ID: pending.InputWork.ID})
			}
			release()
			status := "failed"
			if !fail {
				status = "canceled"
			}
			a := awaitInputStatus(t, s, status)
			p, _ := s.graph.Component("LDR1")
			if *p.IlluminanceLux != 100 || p.Position.X != -500 || a.Mixed.Time != before.Mixed.Time || !reflect.DeepEqual(a.Mixed, before.Mixed) {
				t.Fatal("failed/canceled input altered accepted state")
			}
		}()
	}
}
func TestSensorCapturePersistenceUndoAndStrictParameters(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "night-light"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	call(t, s, "sensor.light.set", LightParams{ID: "LDR1", Illuminance: 1}, "dark")
	settled(t, s)
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	settled(t, s)
	p, _ := s.graph.Component("LDR1")
	if *p.IlluminanceLux != 100 {
		t.Fatal("sensor undo")
	}
	call(t, s, "edit.redo", HistoryParams{}, "redo")
	settled(t, s)
	refuse(t, s, "sensor.temperature.set", TemperatureParams{ID: "LDR1", Temperature: 25}, "invalid-params")
	refuse(t, s, "sensor.light.set", map[string]any{"id": "LDR1"}, "invalid-params")
	refuse(t, s, "sensor.light.set", LightParams{ID: "LDR1", Illuminance: -1}, "invalid-params")
	duration, step := .01, .0005
	_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:LDR1", "i:LDR1", "v:U1"}}, "capture")
	record := s.operations[view.Operation.ID].record
	raw, _ := json.Marshal(VerificationDocument{Metadata: record.metadata, Rows: record.rows})
	if _, err := decodeRecording(raw); err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	json.Unmarshal(raw, &object)
	sensor := object["metadata"].(map[string]any)["mixed"].(map[string]any)["sensors"].([]any)[0].(map[string]any)
	sensor["resistance_ohm"] = 42
	bad, _ := json.Marshal(object)
	if _, err := decodeRecording(bad); err == nil {
		t.Fatal("forged resistance")
	}
	path := filepath.Join(t.TempDir(), "night.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "reopen")
	a := settled(t, fresh)
	if a.SensorModel == nil || a.Document.Dirty || a.Document.RecordingCount != 1 || a.Mixed.Sensors[0].Environment != 1 {
		t.Fatal("sensor model/environment/capture reopen")
	}
	a.SensorModel.Beta = 1
	for _, p := range a.Components {
		if p.Kind == "ldr" {
			*p.IlluminanceLux = 50
		}
	}
	if fresh.Snapshot().SensorModel.Beta == 1 || fresh.Snapshot().Mixed.Sensors[0].Environment != 1 {
		t.Fatal("sensor model alias")
	}
	doc := s.documentLocked()
	if inspectDocument(doc).SensorModel == nil {
		t.Fatal("recovery model")
	}
	doc.SensorModel = nil
	bad, _ = json.Marshal(doc)
	if _, err := decodeDocument(bad); err == nil {
		t.Fatal("missing model")
	}
	for _, kind := range []string{"thermistor", "ldr"} {
		p, _ := MakeComponent("S", kind, Point{}, "")
		for _, angle := range []int{0, 90, 180, 270} {
			p.Rotation = angle
			svg := string(schematicSVG(Graph{Components: []Component{p}}, 1, 1))
			if !strings.Contains(svg, "sensor_environments") || !strings.Contains(svg, "reference_resistance_ohm") {
				t.Fatal("export lost sensor environment")
			}
			for _, pin := range p.Terminals {
				if _, ok := TerminalPosition(p, pin); !ok {
					t.Fatal("sensor pin")
				}
			}
		}
	}
}

func TestSensorStartupBatchDuplicateAndFixedEnvironmentSweep(t *testing.T) {
	s := New()
	defer s.Close()
	raw, _ := json.Marshal(LightParams{ID: "LDR1", Illuminance: 12.5})
	place, _ := json.Marshal(PlaceParams{ID: "LDR1", Kind: "ldr"})
	call(t, s, "edit.batch", BatchParams{Name: "Configure photocell", Edits: []BatchEdit{{Command: "component.place", Params: place}, {Command: "sensor.light.set", Params: raw}}}, "batch")
	a := s.Snapshot()
	if a.InputWork != nil || *a.Components[0].IlluminanceLux != 12.5 {
		t.Fatal("batch did not configure startup")
	}
	call(t, s, "component.duplicate", DuplicateParams{ID: "LDR1"}, "copy")
	a = s.Snapshot()
	if len(a.Components) != 2 || *a.Components[1].IlluminanceLux != 12.5 || a.Components[0].IlluminanceLux == a.Components[1].IlluminanceLux {
		t.Fatal("sensor duplication lost or aliased environment")
	}
	g := sensorDivider("ldr", 1, false)
	r := sweepForTest(t, g, SweepSettings{ComponentID: "V1", Parameter: "voltage", Start: "0 V", Stop: "5 V", Points: 11, ProbeIDs: []string{"i:SENSE", "v:SENSE"}})
	for _, p := range r.Points {
		want := p.Input / (50118.7233627272 + 10000)
		if math.Abs(p.Values[0]-want) > 1e-11 || math.Abs(p.Values[1]-want*50118.7233627272) > 1e-9 {
			t.Fatal("sensor DC source sweep", p)
		}
	}
}

func TestSensorReturningToCommittedEnvironmentRetiresPendingInput(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "night-light"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := settled(t, s)
	entered, release := blockInputSolver(t, s, false)
	defer release()
	promptLiveCommand(t, s, "sensor.light.set", LightParams{ID: "LDR1", Illuminance: 1})
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "sensor.light.set", LightParams{ID: "LDR1", Illuminance: 100})
	release()
	a := awaitInputStatus(t, s, "superseded")
	if !reflect.DeepEqual(a.Components, before.Components) || !reflect.DeepEqual(a.Mixed, before.Mixed) || !reflect.DeepEqual(a.UndoEntry, before.UndoEntry) {
		t.Fatal("return-to-committed slider position left a pending change")
	}
}
