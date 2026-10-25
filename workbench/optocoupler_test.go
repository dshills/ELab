package workbench

import (
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func optoFixture(load string, ctr float64) Graph {
	b := newExampleBuilder()
	b.part("VIN", "dc-source", "5", -300, 0, 0)
	b.ground("GIN", "VIN.negative", -300, 150)
	b.part("RIN", "resistor", "1k", -100, -120, 0)
	b.wire("VIN.positive", "RIN.1")
	b.part("OC1", "optocoupler", "", 100, 0, 0)
	b.graph.Components[len(b.graph.Components)-1].Power.Optocoupler.CTR = ctr
	b.wire("RIN.2", "OC1.anode")
	b.ground("GI", "OC1.cathode", 0, 150)
	b.part("VOUT", "dc-source", "5", 500, 0, 0)
	b.ground("GOUT", "VOUT.negative", 500, 150)
	b.part("RL", "resistor", load, 300, -120, 0)
	b.wire("VOUT.positive", "RL.1")
	b.wire("RL.2", "OC1.collector")
	b.ground("GE", "OC1.emitter", 220, 150)
	return b.graph
}
func TestOptocouplerCTRHeadroomAndPortConservation(t *testing.T) {
	for _, load := range []string{"100", "10k"} {
		for _, ctr := range []float64{.1, 1, 5} {
			g := optoFixture(load, ctr)
			r, _, err := solveDCOperatingPoint(t.Context(), g, nil)
			if err != nil {
				t.Fatal(load, ctr, err)
			}
			u := r.Optocouplers[0]
			// Independent resistor load line and analytic smooth-headroom relation.
			R, _ := ParseValue(load, "Ω")
			want := ctr*u.InputCurrent*u.OutputVoltage/math.Hypot(u.OutputVoltage, .1) + u.OutputVoltage/1e9
			if math.Abs(u.OutputCurrent-want) > 1e-10 || math.Abs((5-u.OutputVoltage)/R-u.OutputCurrent) > 1e-10 || math.Abs(u.InputCurrent-(5-u.InputVoltage)/1000) > 1e-10 {
				t.Fatal("CTR/load law", u)
			}
			if load == "100" && u.EffectiveCTR < ctr*.99 {
				t.Fatal("active CTR", u)
			}
			if load == "10k" && ctr >= 1 && u.OutputVoltage > .1 {
				t.Fatal("output failed to saturate", u)
			}
			if r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 || u.InputPower < 0 || u.OutputPower < 0 {
				t.Fatal("port power", u, r.PowerResidual)
			}
			if err := validateRecordedOptocouplers(func() Compiled { c, _, _ := g.Compile(); return c }(), g, r); err != nil {
				t.Fatal(err)
			}
		}
	}
}
func TestOptocouplerAnalyticJacobianAndNoConductiveBarrier(t *testing.T) {
	for _, vce := range []float64{-.5, .01, .5, 5} {
		vf, ctr := 2., .8
		i, o, gi, gm, goff := optoCurrents(vf, vce, ctr)
		h := 1e-6
		ip, op, _, _, _ := optoCurrents(vf+h, vce, ctr)
		im, om, _, _, _ := optoCurrents(vf-h, vce, ctr)
		_, vp, _, _, _ := optoCurrents(vf, vce+h, ctr)
		_, vm, _, _, _ := optoCurrents(vf, vce-h, ctr)
		for _, x := range [][2]float64{{gi, (ip - im) / (2 * h)}, {gm, (op - om) / (2 * h)}, {goff, (vp - vm) / (2 * h)}} {
			if math.Abs(x[0]-x[1]) > 1e-9+math.Abs(x[0])*1e-6 {
				t.Fatal("opto derivative", vce, x)
			}
		}
		if !finite(i) || !finite(o) {
			t.Fatal("nonfinite")
		}
	}
	p, _ := MakeComponent("OC1", "optocoupler", Point{}, "")
	if networkReferencePair(p, "anode", "collector", false) || !networkReferencePair(p, "anode", "cathode", false) || !networkReferencePair(p, "collector", "emitter", false) {
		t.Fatal("optical barrier became conductor")
	}

	g := optoFixture("1k", 1)
	parts := []Component{}
	for _, p := range g.Components {
		if p.ID != "GOUT" && p.ID != "GE" {
			parts = append(parts, p)
		}
	}
	g.Components = parts
	wires := []Wire{}
	for _, w := range g.Wires {
		if w.To != "GOUT.reference" && w.To != "GE.reference" {
			wires = append(wires, w)
		}
	}
	g.Wires = append(wires, Wire{ID: "RETURN", From: "VOUT.negative", To: "OC1.emitter", Connected: true})
	if _, _, d := g.Compile(); d == nil || d.Code != "floating-island" {
		t.Fatal("floating output was referenced through optical control", d)
	}

}
func TestOptocouplerSharedConfigureCapturePersistenceAndSwitch(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "opto-switch"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	a := settled(t, s)
	if a.Mixed == nil || a.Mixed.Optocouplers[0].InputCurrent > 1e-6 {
		t.Fatal("off example", a.Diagnostics)
	}
	call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: true}, "on")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	a = settled(t, s)
	if a.Mixed.LEDs[0].Current < .001 {
		t.Fatal("isolated switch did not light output", a.Mixed)
	}
	call(t, s, "optocoupler.configure", OptocouplerParams{"OC1", .5}, "ctr")
	call(t, s, "simulation.restart", struct{}{}, "restart2")
	settled(t, s)
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	settled(t, s)
	p, _ := s.graph.Component("OC1")
	if p.Power.Optocoupler.CTR != 1 {
		t.Fatal("CTR undo")
	}
	call(t, s, "edit.redo", HistoryParams{}, "redo")
	settled(t, s)
	call(t, s, "simulation.restart", struct{}{}, "restart3")
	settled(t, s)
	duration := 0.
	_, v := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:OC1.output", "i:OC1.collector", "i:OC1.anode"}}, "capture")
	rec := s.operations[v.Operation.ID].record
	raw, _ := json.Marshal(VerificationDocument{Metadata: rec.metadata, Rows: rec.rows})
	if _, err := decodeRecording(raw); err != nil {
		t.Fatal("opto capture", err)
	}
	var obj map[string]any
	json.Unmarshal(raw, &obj)
	obj["metadata"].(map[string]any)["mixed"].(map[string]any)["optocouplers"].([]any)[0].(map[string]any)["collector_current_a"] = .123
	raw, _ = json.Marshal(obj)
	if _, err := decodeRecording(raw); err == nil {
		t.Fatal("forged optical transfer")
	}
	path := filepath.Join(t.TempDir(), "isolated.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	before := settled(t, s)
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	after := settled(t, fresh)
	if !reflect.DeepEqual(before.Components, after.Components) || after.Document.Dirty || after.Document.RecordingCount != 1 {
		t.Fatal("opto project roundtrip")
	}
	refuse(t, s, "optocoupler.configure", OptocouplerParams{"OC1", 0}, "invalid-params")
	refuse(t, s, "optocoupler.configure", map[string]any{"id": "OC1"}, "invalid-params")
}
func TestPowerSymbolWireAnchorsAtLeadEnds(t *testing.T) {
	for _, kind := range []string{"battery", "dc-motor"} {
		p, _ := MakeComponent("X", kind, Point{}, "")
		one, _ := TerminalPosition(p, "positive")
		two, _ := TerminalPosition(p, "negative")
		if one.Y != -55 || two.Y != 55 {
			t.Fatal("wire misses the new symbol lead end", kind, one, two)
		}
	}
}

func TestOptocouplerTransientPulsesAndOutputCannotBackDriveInput(t *testing.T) {
	g := optoFixture("1k", 1)
	pulse, _ := MakeComponent("VIN", "pulse-source", g.Components[0].Position, "5")
	g.Components[0] = pulse
	k := batteryKernel(t, g)
	trace := k.trace
	high, low := false, false
	for j := 1; j <= 24; j++ {
		next, updated, err := advanceMixed(k.mixed, trace, float64(j)*.005)
		if err != nil {
			t.Fatal(j, err)
		}
		k.mixed, trace = next, updated
		u := next.result.Optocouplers[0]
		high = high || u.OutputCurrent > .001
		low = low || u.OutputCurrent < 1e-7
		if math.Abs(next.result.PowerResidual) > 1e-8 || next.result.KCLResidual > 1e-9 {
			t.Fatal("opto pulse conservation")
		}
	}
	if !high || !low {
		t.Fatal("optical input failed to switch separate port")
	}
	g = optoFixture("1k", 1)
	first, _, err := solveDCOperatingPoint(t.Context(), g, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range g.Components {
		if p.ID == "VOUT" {
			g.Components[i].Value = 12
		}
	}
	second, _, err := solveDCOperatingPoint(t.Context(), g, nil)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(first.Optocouplers[0].InputCurrent-second.Optocouplers[0].InputCurrent) > 1e-12 {
		t.Fatal("output supply back-fed input LED")
	}
}

func TestOptocouplerInductorStartupUsesSeparateConductivePorts(t *testing.T) {
	g := optoFixture("100", 1)
	for i, p := range g.Components {
		if p.ID == "RL" {
			q, _ := MakeComponent("RL", "inductor", p.Position, "100mH")
			g.Components[i] = q
		}
	}
	b := exampleBuilder{graph: g}
	b.part("LIMIT", "resistor", "10k", 300, -250, 0)
	for i, w := range b.graph.Wires {
		if w.From == "VOUT.positive" && w.To == "RL.1" {
			b.graph.Wires[i].To = "LIMIT.1"
		}
	}
	b.wire("LIMIT.2", "RL.1")
	k := batteryKernel(t, b.graph)
	trace := k.trace
	for j := 1; j <= 10; j++ {
		next, updated, err := advanceMixed(k.mixed, trace, float64(j)*.00001)
		if err != nil {
			t.Fatal(j, err)
		}
		k.mixed, trace = next, updated
		if next.result.KCLResidual > 1e-9 || math.Abs(next.result.PowerResidual) > 1e-8 {
			t.Fatal("opto coil conservation")
		}
	}
	r := k.mixed.result
	if math.Abs(r.Inductors[0].Current-r.Optocouplers[0].OutputCurrent) > 1e-9 || r.Inductors[0].Current < .0004 || r.Inductors[0].Current > .0006 {
		t.Fatal("opto coil current/port startup", r.Inductors, r.Optocouplers)
	}
	assertLabelMasksLeaveWiresContinuous(t, OptocouplerExampleGraph(), []string{"RIN", "RL", "D1", "OC1"})
}
