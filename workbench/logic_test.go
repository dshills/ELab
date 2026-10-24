package workbench

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var logicKinds = []string{"logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor"}

func logicTruth(kind string, a, b bool) bool {
	switch kind {
	case "logic-not":
		return !a
	case "logic-and":
		return a && b
	case "logic-or":
		return a || b
	case "logic-nand":
		return !(a && b)
	case "logic-nor":
		return !(a || b)
	case "logic-xor":
		return a != b
	case "logic-xnor":
		return a == b
	}
	panic(kind)
}
func logicFixture(t *testing.T, kind string, a, b, supply, offset, load float64) *opAmpNetworkFixture {
	f := newOpAmpNetworkFixture()
	f.add(t, "VP", "dc-source", fmt.Sprint(supply), "vcc", "local")
	f.add(t, "VG", "dc-source", fmt.Sprint(offset), "local", "0")
	f.add(t, "VA", "dc-source", fmt.Sprint(a), "a", "local")
	f.add(t, "VB", "dc-source", fmt.Sprint(b), "b", "local")
	if kind == "logic-not" {
		f.add(t, "U1", kind, "", "a", "output", "vcc", "local")
	} else {
		f.add(t, "U1", kind, "", "a", "b", "output", "vcc", "local")
	}
	f.add(t, "RL", "resistor", fmt.Sprint(load), "output", "local")
	return f
}
func TestLogicTruthTablesLoadedRailReferencesAndPower(t *testing.T) {
	for _, kind := range logicKinds {
		for _, supply := range []float64{3.3, 5, 12} {
			for _, offset := range []float64{-3, 0, 2} {
				for bits := 0; bits < 4; bits++ {
					a, b := bits&1 != 0, bits&2 != 0
					av, bv := 0., 0.
					if a {
						av = supply
					}
					if b {
						bv = supply
					}
					n := logicFixture(t, kind, av, bv, supply, offset, 1000).network()
					n.dc = true
					state, err := n.solveContext(context.Background(), nil, nil, 0, &Transient{Mode: "auto", Settings: DefaultTransientSettings()}, 0)
					if err != nil {
						t.Fatal(kind, bits, err)
					}
					got := state.result.LogicGates[0]
					want := 0.
					if logicTruth(kind, a, b) {
						m := LogicDeviceModel()
						e := supply * supply / (supply*supply + m.SupplySoftVoltage*m.SupplySoftVoltage)
						want = supply / (1 + m.OutputResistance/e*(1./1000+1/m.OffResistance))
					}
					if math.Abs(got.OutputVoltage-want) > 1e-8 || math.Abs(state.result.PowerResidual) > 1e-8 || state.result.KCLResidual > 1e-8 {
						t.Fatal(kind, bits, got, want, state.result)
					}
					if got.State != map[bool]string{true: "high", false: "low"}[logicTruth(kind, a, b)] {
						t.Fatal("truth state", got)
					}
				}
			}
		}
	}
}
func TestLogicJacobianConservationAndIntermediateInputs(t *testing.T) {
	for _, kind := range logicKinds {
		for _, v := range [][5]float64{{2.1, 2.8, 3, 5, 0}, {-1, -.5, 1, 2, -3}, {5, 0, 2, 5, 0}, {0, 0, 1, 0, 0}, {1, 2, 3, -2, 0}} {
			e := logicCurrents(kind, v)
			sum, power := 0., 0.
			for k, i := range e.Current {
				sum += i
				power += i * (v[k] - v[4])
			}
			if math.Abs(sum) > 1e-12 || power < -1e-12 {
				t.Fatal("current/power conservation", kind, e, power)
			}
			for col := range v {
				plus, minus := v, v
				h := 1e-7
				plus[col] += h
				minus[col] -= h
				p, m := logicCurrents(kind, plus), logicCurrents(kind, minus)
				for row := range v {
					want := (p.Current[row] - m.Current[row]) / (2 * h)
					if math.Abs(want-e.Jacobian[row][col]) > 2e-7 {
						t.Fatal("logic Jacobian", kind, v, row, col, want, e.Jacobian[row][col])
					}
				}
			}
		}
	}
	n := logicFixture(t, "logic-not", 2.5, 0, 5, 0, 100000).network()
	n.dc = true
	state, err := n.solve(nil, nil, 0, &Transient{Mode: "auto", Settings: DefaultTransientSettings()}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if u := state.result.LogicGates[0]; u.InputA != "intermediate" || u.State != "intermediate" {
		t.Fatal("invented digital level", u)
	}
	for _, supply := range []float64{0, -5} {
		n = logicFixture(t, "logic-not", 0, 0, supply, 0, 1000).network()
		n.dc = true
		state, err = n.solve(nil, nil, 0, &Transient{Mode: "auto", Settings: DefaultTransientSettings()}, 0)
		if err != nil {
			t.Fatal(err)
		}
		if u := state.result.LogicGates[0]; u.State != "unpowered" || math.Abs(u.OutputVoltage) > 1e-9 {
			t.Fatal("unpowered output", u)
		}
	}
}
func TestLogicExamplesTransientTruthAndLiveSwitches(t *testing.T) {
	for _, id := range []string{"logic-inverter", "logic-switches", "logic-half-adder"} {
		t.Run(id, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: id}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			initial := settled(t, s)
			if initial.LogicModel == nil || initial.Document.Dirty || initial.Mixed == nil {
				t.Fatal("logic example startup", initial.Diagnostics)
			}
			if warnings := waveformSamplingWarnings(Graph{Components: initial.Components, Wires: initial.Wires}, initial.TransientSettings); len(warnings) != 0 {
				t.Fatal("logic example undersamples its own edges", warnings)
			}
			if id == "logic-switches" {
				for _, bits := range []int{0, 1, 3, 2, 0} {
					for k, name := range []string{"SA", "SB"} {
						call(t, s, "switch.set", SwitchParams{ID: name, Closed: bits&(1<<k) != 0}, fmt.Sprint(bits, k))
					}

					call(t, s, "simulation.restart", struct{}{}, fmt.Sprintf("apply-%d", bits))
					r := settled(t, s)
					want := "low"
					if bits == 3 {
						want = "high"
					}
					if r.Mixed.LogicGates[0].State != want {
						t.Fatal("manual AND", bits, r.Mixed.LogicGates, r.SwitchStates, r.Mixed.Switches, r.Status, r.Diagnostics)
					}
				}
				return
			}
			duration, step := 1.6, .005
			_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: initial.Scope.Selected}, "capture")
			_ = view
			seen := map[string]bool{}
			for i := 0; i < 320; i++ {
				call(t, s, "simulation.step", StepParams{Duration: step}, fmt.Sprint(i))
				state := s.Snapshot()
				for _, u := range state.Mixed.LogicGates {
					seen[u.ID+":"+u.State] = true
					if u.InputA == "intermediate" || u.InputB == "intermediate" {
						continue
					}
					a, b := u.InputA == "high", u.InputB == "high"
					want := map[bool]string{true: "high", false: "low"}[logicTruth(u.Kind, a, b)]
					if u.State != want {
						t.Fatal("transient truth", state.Mixed.Time, u)
					}
				}
			}
			if !seen["U1:high"] || !seen["U1:low"] || id == "logic-half-adder" && (!seen["U2:high"] || !seen["U2:low"]) {
				t.Fatal("missing switching", seen)
			}
			if s.Snapshot().Mixed.Time < duration-1e-10 {
				t.Fatal("did not advance")
			}
		})
	}
}
func TestLogicCapturePersistenceStrictImportAndProbeSigns(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "logic-half-adder"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	duration, step := .2, .005
	_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:U1", "i:U1", "i:U1.vcc", "i:U1.gnd"}}, "record")
	record := s.operations[view.Operation.ID].record
	raw, _ := json.Marshal(VerificationDocument{Metadata: record.metadata, Rows: record.rows})
	decoded, err := decodeRecording(raw)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.metadata.LogicModel == nil || recordingModelHashes(decoded.metadata)[LogicModelID] == "" {
		t.Fatal("logic provenance")
	}
	path := filepath.Join(t.TempDir(), "logic.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	r := settled(t, fresh)
	if r.LogicModel == nil || r.Document.Dirty || r.Document.RecordingCount != 1 {
		t.Fatal("logic project roundtrip")
	}
	data, _ := os.ReadFile(path)
	d, err := decodeDocument(data)
	if err != nil || inspectDocument(d).LogicModel == nil {
		t.Fatal("logic recovery", err)
	}
	var doc map[string]any
	json.Unmarshal(data, &doc)
	delete(doc, "logic_model")
	bad, _ := json.Marshal(doc)
	if _, err = decodeDocument(bad); err == nil {
		t.Fatal("missing logic model accepted")
	}
	var capture map[string]any
	json.Unmarshal(raw, &capture)
	u := capture["metadata"].(map[string]any)["mixed"].(map[string]any)["logic_gates"].([]any)[0].(map[string]any)
	for _, field := range []string{"kind", "state", "input_a", "input_b", "model", "pin_currents_a"} {
		saved := u[field]
		delete(u, field)
		bad, _ = json.Marshal(capture)
		if _, err = decodeRecording(bad); err == nil {
			t.Fatal("missing logic field accepted", field)
		}
		u[field] = saved
	}
	savedPins := u["pin_currents_a"]
	u["pin_currents_a"] = savedPins.([]any)[:4]
	bad, _ = json.Marshal(capture)
	if _, err = decodeRecording(bad); err == nil {
		t.Fatal("short logic pin-current array accepted")
	}
	u["pin_currents_a"] = savedPins
	u["state"] = "wrong"
	bad, _ = json.Marshal(capture)
	if _, err = decodeRecording(bad); err == nil {
		t.Fatal("corrupt logic accepted")
	}
	state := s.Snapshot()
	state.LogicModel.OutputResistance = 1
	state.Mixed.LogicGates[0].Model.OutputResistance = 2
	if s.Snapshot().LogicModel.OutputResistance != 50 || s.Snapshot().Mixed.LogicGates[0].Model.OutputResistance != 50 {
		t.Fatal("logic snapshot alias")
	}
	clone := cloneVerificationMetadata(decoded.metadata)
	clone.LogicModel.OutputResistance = 3
	if reflect.DeepEqual(clone.LogicModel, decoded.metadata.LogicModel) {
		t.Fatal("record clone alias")
	}
	c, _, _ := LogicHalfAdderExampleGraph().Compile()
	state = s.Snapshot()
	sample := mixedSample(c, state.Mixed, 0)
	gate := state.Mixed.LogicGates[0]
	for _, probe := range probeInventory(c.MixedGraph) {
		if probe.ID == "i:U1" || probe.ID == "i:U1.out" {
			v, ok := frameValue(sample, probe)
			if !ok || v != -gate.PinCurrents[2] {
				t.Fatal("output-current sign", probe)
			}
		}
	}
}

func TestLogicCascadeAndExternalRCTiming(t *testing.T) {
	for _, input := range []float64{0, 5} {
		f := logicFixture(t, "logic-not", input, 5, 5, 0, 1000)
		f.add(t, "U2", "logic-and", "", "output", "b", "carry", "vcc", "local")
		f.add(t, "R2", "resistor", "1k", "carry", "local")
		n := f.network()
		n.dc = true
		state, err := n.solve(nil, nil, 0, &Transient{Mode: "auto", Settings: DefaultTransientSettings()}, 0)
		if err != nil {
			t.Fatal(err)
		}
		want := "high"
		if input == 5 {
			want = "low"
		}
		for _, u := range state.result.LogicGates {
			if u.State != want {
				t.Fatal("gate cascade", u)
			}
		}
	}
	f := logicFixture(t, "logic-not", 0, 0, 5, 0, 1000)
	f.add(t, "C1", "capacitor", "100u", "output", "local")
	n := f.network()
	trace := &Transient{Mode: "auto", Settings: DefaultTransientSettings()}
	initial, err := n.solve(nil, []float64{0}, 0, trace, 0)
	if err != nil {
		t.Fatal(err)
	}
	next, err := initial.advanceBudget(trace, .01, newIntegrationBudget())
	if err != nil {
		t.Fatal(err)
	}
	m := LogicDeviceModel()
	g := 25 / (25 + m.SupplySoftVoltage*m.SupplySoftVoltage) / m.OutputResistance
	total := g + 1./1000 + 1/m.OffResistance
	want := 5 * g / total * (1 - math.Exp(-.01*total/100e-6))
	if math.Abs(next.result.Capacitors[0].Voltage-want) > 2e-5 {
		t.Fatal("logic RC reference", next.result.Capacitors[0].Voltage, want)
	}
}
func TestLogicWiringValidationAndTimerIntegration(t *testing.T) {
	for _, kind := range logicKinds {
		f := logicFixture(t, kind, 0, 5, 5, 0, 1000)
		f.graph.Components = append(f.graph.Components[:1], f.graph.Components[2:]...)
		for ref, net := range f.pins {
			if net == "local" {
				f.pins[ref] = "0"
			}
		}
		if kind == "logic-not" {
			for k, p := range f.graph.Components {
				if p.ID == "VB" {
					f.graph.Components = append(f.graph.Components[:k], f.graph.Components[k+1:]...)
					break
				}
			}
		}
		g := f.wiredGraph(t)
		if _, _, d := g.Compile(); d != nil {
			t.Fatal("wired logic", kind, d)
		}
		for _, pin := range logicPins(kind) {
			if pin == "out" {
				continue
			}
			broken := g.Clone()
			broken.Wires = nil
			for _, w := range g.Wires {
				if w.From != "U1."+pin && w.To != "U1."+pin {
					broken.Wires = append(broken.Wires, w)
				}
			}
			if _, _, d := broken.Compile(); d == nil {
				t.Fatal("floating logic pin accepted", kind, pin)
			}
		}
	}
	g := Timer555FlasherExampleGraph()
	p, _ := MakeComponent("U2", "logic-not", Point{500, -200}, "")
	g.Components = append(g.Components, p)
	for _, pair := range [][2]string{{"U1.out", "U2.a"}, {"U1.vcc", "U2.vcc"}, {"U1.gnd", "U2.gnd"}} {
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", len(g.Wires)+1), From: pair[0], To: pair[1], Connected: true})
	}
	s := New()
	defer s.Close()
	for _, p := range g.Components {
		value := ""
		if p.Unit != "" {
			value = FormatValue(p.Value, p.Unit)
		}
		call(t, s, "component.place", PlaceParams{ID: p.ID, Kind: p.Kind, Position: p.Position, Value: value}, "p-"+p.ID)
	}
	for _, w := range g.Wires {
		call(t, s, "wire.connect", ConnectParams{From: w.From, To: w.To}, w.ID)
	}
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	duration, step := 1.6, .005
	_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:U1", "v:U2"}}, "capture")
	r := s.operations[view.Operation.ID].record
	high, low := false, false
	for _, row := range r.rows {
		if row.Values[0] > 3.5 {
			high = true
			if row.Values[1] > .01 {
				t.Fatal("555 high not inverted", row)
			}
		}
		if row.Values[0] < 1.5 {
			low = true
			if row.Values[1] < 4.9 {
				t.Fatal("555 low not inverted", row)
			}
		}
	}
	if !high || !low {
		t.Fatal("timer never switched")
	}
}

func TestLogicVanishingSupplyHasFiniteStamp(t *testing.T) {
	for _, span := range []float64{1e-20, 1e-160, 1e-300, 1e-320} {
		for _, kind := range logicKinds {
			e := logicCurrents(kind, [5]float64{.5 * span, .6 * span, 1, span, 0})
			for row, i := range e.Current {
				if !finite(i) {
					t.Fatal("nonfinite vanishing-supply current", span)
				}
				for _, d := range e.Jacobian[row] {
					if !finite(d) {
						t.Fatal("nonfinite vanishing-supply derivative", kind, span)
					}
				}
			}
		}
	}
}

func TestLogicSymbolExportsAreWellFormed(t *testing.T) {
	for _, kind := range logicKinds {
		p, _ := MakeComponent("U1", kind, Point{}, "")
		for _, rotation := range []int{0, 90, 180, 270} {
			p.Rotation = rotation
			data := schematicSVG(Graph{Components: []Component{p}}, 1, 1)
			d := xml.NewDecoder(bytes.NewReader(data))
			for {
				_, err := d.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal("logic SVG", kind, rotation, err)
				}
			}
		}
	}
}
