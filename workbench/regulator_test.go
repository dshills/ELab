package workbench

import (
	"encoding/json"
	"math"
	"path/filepath"
	"strings"
	"testing"
)

func regulatorFixture(kind string, vin, load float64) Graph {
	b := newExampleBuilder()
	b.part("V1", "dc-source", FormatValue(vin, "V"), -300, 0, 0)
	b.ground("GND", "V1.negative", -300, 200)
	b.part("U1", kind, "5", 0, 0, 0)
	b.wire("V1.positive", "U1.in")
	b.wire("GND.reference", "U1.gnd")
	b.part("RL", "resistor", FormatValue(load, "Ω"), 300, 0, 0)
	b.wire("U1.out", "RL.1")
	b.wire("RL.2", "GND.reference")
	return b.graph
}

// Invert the pass-device load line, independently of the stamp implementation.
func expectedRegulatorVoltage(target, load float64) float64 {
	lo, hi := 0., math.Min(target, .1/(1/load+1e-9))
	for range 100 {
		mid := (lo + hi) / 2
		i := mid * (1/load + 1e-9)
		required := mid + .2*i/math.Sqrt(1-i*i/.01)
		if required > target {
			hi = mid
		} else {
			lo = mid
		}
	}
	return (lo + hi) / 2
}
func TestRegulatorLoadDropoutCurrentLimitAndConservation(t *testing.T) {
	for _, kind := range []string{"linear-regulator", "ldo-regulator"} {
		for _, vin := range []float64{0, 1, 4, 5.5, 6, 9, 12} {
			for _, load := range []float64{10, 100, 1000, 100000} {
				g := regulatorFixture(kind, vin, load)
				c, _, d := g.Compile()
				if d != nil {
					t.Fatal(d)
				}
				kernel, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
				if err != nil {
					t.Fatal(kind, vin, load, err)
				}
				r := kernel.mixed.result
				u := r.Regulators[0]
				target := math.Max(0, math.Min(5, vin-regulatorDropout(kind)))
				want := expectedRegulatorVoltage(target, load)
				if math.Abs(u.OutputVoltage-want) > 1e-7 {
					t.Fatal("load line", kind, vin, load, u.OutputVoltage, want)
				}
				if u.Current > .100000001 || u.Power < 0 || r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-7 {
					t.Fatal("regulator conservation/limit", u, r.PowerResidual)
				}
				if err := validateRecordedRegulators(c, g, r); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
func TestRegulatorJacobianTranslationAndNoSinking(t *testing.T) {
	for _, kind := range []string{"linear-regulator", "ldo-regulator"} {
		p, _ := MakeComponent("U1", kind, Point{}, "5")
		for _, ground := range []float64{-3, 0, 4} {
			for _, pair := range [][2]float64{{9, 4.999}, {6, 3.999}, {9, 1}, {9, 6}, {0, 0}, {-2, 0}} {
				v := [3]float64{pair[0] + ground, pair[1] + ground, ground}
				e := regulatorCurrents(p, v)
				sum, power := 0., 0.
				for k, i := range e.Current {
					sum += i
					power += i * v[k]
				}
				if math.Abs(sum) > 1e-12 || power < -1e-12 {
					t.Fatal("stamp conservation", e, power)
				}
				if pair[1] > 5 && e.PassCurrent != 0 {
					t.Fatal("regulator sinks externally held output")
				}
				for k := range 3 {
					plus, minus := v, v
					plus[k] += 1e-7
					minus[k] -= 1e-7
					a, b := regulatorCurrents(p, plus), regulatorCurrents(p, minus)
					for j := range 3 {
						got := (a.Current[j] - b.Current[j]) / 2e-7
						want := e.Jacobian[j][k]
						if math.Abs(got-want) > 2e-6*math.Max(1, math.Abs(want)) {
							t.Fatal("Jacobian", kind, pair, j, k, got, want)
						}
					}
				}
			}
		}
	}
}
func TestRegulatorSupplyTransientLoadStepAndDropoutExamples(t *testing.T) {
	for _, id := range []string{"regulator-supply", "regulator-dropout"} {
		s := New()
		func() {
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: id}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			settled(t, s)
			if id == "regulator-supply" {
				call(t, s, "simulation.step", StepParams{Duration: .02}, "charge")
				a := settled(t, s)
				if a.Mixed.Regulators[0].OutputVoltage < 4.99 {
					t.Fatal("output capacitor failed to charge", a.Mixed)
				}
				call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: true}, "load-step")
				a = settled(t, s)
				call(t, s, "simulation.step", StepParams{Duration: .02}, "settle-load")
				a = settled(t, s)
				if v := a.Mixed.Regulators[0].OutputVoltage; v < 4.98 || v > 4.991 {
					t.Fatal("finite load droop", v)
				}
				call(t, s, "component.set", ValueParams{ID: "RLOAD", Value: "10 Ω"}, "overload")
				call(t, s, "simulation.restart", struct{}{}, "restart")
				settled(t, s)
				call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: true}, "overload-on")
				settled(t, s)
				call(t, s, "simulation.step", StepParams{Duration: .02}, "overload-settle")
				a = settled(t, s)
				u := a.Mixed.Regulators[0]
				if u.State != "current limited" || u.OutputVoltage > 1.10 {
					t.Fatal("overload", u)
				}
			} else {
				for range 20 {
					call(t, s, "simulation.step", StepParams{Duration: .05}, "step")
					a := settled(t, s)
					vin := 6.5 + 2.5*math.Sin(2*math.Pi*a.Mixed.Time)
					for _, u := range a.Mixed.Regulators {
						want := expectedRegulatorVoltage(math.Max(0, math.Min(5, vin-regulatorDropout(u.Kind))), 1000)
						if math.Abs(u.OutputVoltage-want) > 1e-6 {
							t.Fatal("dropout cycle", u, want)
						}
					}
				}
			}
		}()
	}
}
func TestRegulatorStrictCaptureModelPersistenceAndDomains(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "regulator-supply"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	call(t, s, "component.set", ValueParams{ID: "U1", Value: "3.3 V"}, "setpoint")
	refuse(t, s, "component.set", ValueParams{ID: "U1", Value: "0 V"}, "invalid-params")
	duration, step := .02, .0005
	_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:U1", "i:U1", "i:U1.in"}}, "capture")
	record := s.operations[view.Operation.ID].record
	raw, _ := json.Marshal(VerificationDocument{Metadata: record.metadata, Rows: record.rows})
	if _, err := decodeRecording(raw); err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	json.Unmarshal(raw, &object)
	u := object["metadata"].(map[string]any)["mixed"].(map[string]any)["regulators"].([]any)[0].(map[string]any)
	for _, field := range []string{"model", "state", "output_voltage_v", "pin_currents_a", "power_w"} {
		saved := u[field]
		delete(u, field)
		bad, _ := json.Marshal(object)
		if _, err := decodeRecording(bad); err == nil {
			t.Fatal("missing regulator field", field)
		}
		u[field] = saved
	}
	u["current_a"] = .5
	bad, _ := json.Marshal(object)
	if _, err := decodeRecording(bad); err == nil {
		t.Fatal("forged output current")
	}
	path := filepath.Join(t.TempDir(), "supply.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "reopen")
	a := settled(t, fresh)
	if a.RegulatorModel == nil || a.Document.Dirty || a.Document.RecordingCount != 1 {
		t.Fatal("regulator reopen")
	}
	a.RegulatorModel.CurrentLimit = 1
	if fresh.Snapshot().RegulatorModel.CurrentLimit == 1 {
		t.Fatal("model alias")
	}
	doc := s.documentLocked()
	if inspectDocument(doc).RegulatorModel == nil {
		t.Fatal("recovery model")
	}
	doc.RegulatorModel = nil
	bad, _ = json.Marshal(doc)
	if _, err := decodeDocument(bad); err == nil {
		t.Fatal("missing model")
	}
	for _, kind := range []string{"linear-regulator", "ldo-regulator"} {
		p, _ := MakeComponent("U1", kind, Point{}, "")
		for _, angle := range []int{0, 90, 180, 270} {
			p.Rotation = angle
			for _, pin := range p.Terminals {
				if _, ok := TerminalPosition(p, pin); !ok {
					t.Fatal("pin geometry")
				}
			}
			svg := string(schematicSVG(Graph{Components: []Component{p}}, 1, 1))
			if !strings.Contains(svg, ">IN</text>") || !strings.Contains(svg, ">OUT</text>") || !strings.Contains(svg, ">GND</text>") {
				t.Fatal("missing exported regulator pin labels")
			}
			if !strings.Contains(string(schematicSVG(Graph{Components: []Component{p}}, 1, 1)), "regulator_model") {
				t.Fatal("export provenance")
			}
		}
	}
}
func TestRegulatorDCSweep(t *testing.T) {
	g := regulatorFixture("linear-regulator", 9, 1000)
	r := sweepForTest(t, g, SweepSettings{ComponentID: "V1", Parameter: "voltage", Start: "0 V", Stop: "12 V", Points: 25, ProbeIDs: []string{"v:U1", "i:U1"}})
	for _, p := range r.Points {
		want := expectedRegulatorVoltage(math.Max(0, math.Min(5, p.Input-2)), 1000)
		if math.Abs(p.Values[0]-want) > 1e-7 {
			t.Fatal("input sweep", p.Input, p.Values[0], want)
		}
	}
}

func TestRegulatorFeedsInductiveLoad(t *testing.T) {
	g := regulatorFixture("linear-regulator", 9, 1000)
	l, _ := MakeComponent("L1", "inductor", Point{200, 100}, "100 mH")
	g.Components = append(g.Components, l)
	for k := range g.Wires {
		if g.Wires[k].From == "U1.out" && g.Wires[k].To == "RL.1" {
			g.Wires[k].To = "L1.1"
		}
	}
	g.Wires = append(g.Wires, Wire{ID: "WL", From: "L1.2", To: "RL.1", Connected: true})
	c, _, d := g.Compile()
	if d != nil {
		t.Fatal(d)
	}
	kernel, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	for k := 1; k <= 100; k++ {
		if err := kernel.advance(t.Context(), float64(k)*.0001); err != nil {
			t.Fatal(k, err)
		}
		r := kernel.mixed.result
		if r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-7 {
			t.Fatal("regulated inductive load balance", r.PowerResidual)
		}
	}
	r := kernel.mixed.result
	if math.Abs(r.Inductors[0].Current-expectedRegulatorVoltage(5, 1000)/1000) > 1e-8 {
		t.Fatal("inductive load did not settle", r.Inductors)
	}
}
