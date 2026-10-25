package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestHBridgeJacobianPassivityAndLocalReference(t *testing.T) {
	// Exercise all derivatives including intermediate inputs, supply startup,
	// body diode conduction, regeneration and shifted local ground.
	for _, v := range [][7]float64{{5, 0, 5, 11, .5, 12, 0}, {0, 5, 0, 12.9, -.8, 12, 0}, {1.1, 1.7, 1.2, 4, 2, 2.5, 0}, {-1.9, -1.3, -1.8, 1, -1, -.5, -3}, {0, 0, 0, 2, -1, 0, 0}, {5, 5, 5, 13, 13, 12, 0}} {
		e := hbridgeCurrents(v)
		sum, power := 0., 0.
		for k, i := range e.Current {
			sum += i
			power += (v[k] - v[6]) * i
		}
		if math.Abs(sum) > 1e-12 || power < -1e-12 {
			t.Fatal("non-passive driver", v, e, power)
		}
		for col := range v {
			a, b := v, v
			h := 1e-7
			a[col] += h
			b[col] -= h
			ea, eb := hbridgeCurrents(a), hbridgeCurrents(b)
			for row := range v {
				want := (ea.Current[row] - eb.Current[row]) / (2 * h)
				if math.Abs(want-e.Jacobian[row][col]) > 1e-6 {
					t.Fatal("Jacobian", v, row, col, want, e.Jacobian[row][col])
				}
			}
		}
		shifted := v
		for k := range shifted {
			shifted[k] += 1.25
		}
		m := hbridgeCurrents(shifted)
		for k := range e.Current {
			if math.Abs(m.Current[k]-e.Current[k]) > 1e-12 {
				t.Fatal("ground reference changed law")
			}
		}
	}
}

func hbridgeDCFixture(t *testing.T, bits int, enabled bool, supply, offset float64) *mixedNetwork {
	t.Helper()
	f := newOpAmpNetworkFixture()
	f.add(t, "VM", "dc-source", fmt.Sprint(supply), "vm", "gnd")
	f.add(t, "VG", "dc-source", fmt.Sprint(offset), "gnd", "0")
	a, b, en := 0., 0., 0.
	if bits&1 != 0 {
		a = 5
	}
	if bits&2 != 0 {
		b = 5
	}
	if enabled {
		en = 5
	}
	f.add(t, "VA", "dc-source", fmt.Sprint(a), "a", "gnd")
	f.add(t, "VB", "dc-source", fmt.Sprint(b), "b", "gnd")
	f.add(t, "VE", "dc-source", fmt.Sprint(en), "en", "gnd")
	f.add(t, "U1", "hbridge", "", "a", "b", "en", "o1", "o2", "vm", "gnd")
	f.add(t, "RL", "resistor", "10", "o1", "o2")
	n := f.network()
	n.dc = true
	return n
}
func TestHBridgeTruthTableLoadedOutputAndConservation(t *testing.T) {
	for _, supply := range []float64{3.3, 5, 12} {
		for _, offset := range []float64{-3, 0, 2} {
			for _, enabled := range []bool{false, true} {
				for bits := 0; bits < 4; bits++ {
					n := hbridgeDCFixture(t, bits, enabled, supply, offset)
					s, err := n.solve(nil, nil, 0, &Transient{Mode: "auto", Settings: DefaultTransientSettings()}, 0)
					if err != nil {
						t.Fatal(bits, enabled, err)
					}
					u := s.result.HBridges[0]
					want, state := 0., "coast"
					if enabled {
						switch bits {
						case 0:
							state = "brake low"
						case 1:
							state = "forward"
							want = supply * 10 / 10.5
						case 2:
							state = "reverse"
							want = -supply * 10 / 10.5
						case 3:
							state = "brake high"
						}
					}
					if u.State != state || math.Abs(u.OutputVoltage-want) > 1e-7 || math.Abs(s.result.PowerResidual) > 1e-7 || s.result.KCLResidual > 1e-9 {
						t.Fatal(bits, enabled, offset, u, want, s.result.PowerResidual)
					}
				}
			}
		}
	}
}

func TestHBridgeClampPathsAndUnpoweredBridge(t *testing.T) {
	m := HBridgeDeviceModel()
	// A disabled winding current can enter either VM clamp and leave either low
	// clamp. Active power-off does not remove passive flyback paths.
	for _, vm := range []float64{0, 12} {
		v := [7]float64{0, 0, 0, vm + 1, -1, vm, 0}
		e := hbridgeCurrents(v)
		if math.Abs(e.Clamps[0]-(1-m.DiodeDrop)/m.DiodeResistance) > 1e-12 || e.Clamps[3] <= 0 || e.Current[5] >= 0 {
			t.Fatal("missing regeneration/flyback", vm, e)
		}
	}
	n := hbridgeDCFixture(t, 1, true, 0, 0)
	s, err := n.solve(nil, nil, 0, &Transient{Settings: DefaultTransientSettings()}, 0)
	if err != nil || s.result.HBridges[0].State != "unpowered / coast" || math.Abs(s.result.HBridges[0].OutputVoltage) > 1e-10 {
		t.Fatal("unpowered active drive", s, err)
	}
}

func hbridgeConstantGraph() Graph {
	g := HBridgeExampleGraph()
	for k, p := range g.Components {
		if p.ID == "PWM" {
			g.Components[k].Pulse.HighVoltage = 5
			g.Components[k].Pulse.LowVoltage = 5
		}
	}
	return g
}
func TestHBridgeMotorForwardReverseAndStallReference(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, locked := range []bool{false, true} {
			g := hbridgeConstantGraph()
			for k, p := range g.Components {
				if p.ID == "S1" || p.ID == "S2" {
					v := (p.ID == "S1") != reverse
					g.Components[k].SwitchClosed = &v
				}
				if p.ID == "MTR1" {
					g.Components[k].Power.Motor.Locked = locked
				}
			}
			k := batteryKernel(t, g)
			tr := k.trace
			for j := 1; j <= 400; j++ {
				next, trace, err := advanceMixed(k.mixed, tr, float64(j)*.0005)
				if err != nil {
					t.Fatal(j, reverse, locked, err)
				}
				k.mixed, tr = next, trace
			}
			r := k.mixed.result
			m := r.Motors[0]
			sign := 1.
			if reverse {
				sign = -1
			}
			if locked {
				want := sign * 12 / 8.5
				if math.Abs(m.Current-want) > 1e-5 || m.RPM != 0 {
					t.Fatal("stall reference", m, want)
				}
			} else {
				want := sign * .02 * 12 / (8.5*1e-5 + .02*.02)
				if math.Abs(m.Omega-want) > .5 {
					t.Fatal("loaded speed reference", m, want)
				}
			}
			if r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-7 {
				t.Fatal("conservation", r)
			}
		}
	}
}

func TestHBridgePWMExampleResolvesDriveCoastAndFlyback(t *testing.T) {
	speeds := []float64{}
	for _, duty := range []float64{.25, .75} {
		g := HBridgeExampleGraph()
		for k, p := range g.Components {
			if p.ID == "PWM" {
				g.Components[k].Pulse.HighSeconds = .01 * duty
				g.Components[k].Pulse.LowSeconds = .01 * (1 - duty)
			}
		}
		k := batteryKernel(t, g)
		tr := k.trace
		drive, coast, clamp := false, false, false
		for j := 1; j <= 1200; j++ {
			next, trace, err := advanceMixed(k.mixed, tr, float64(j)*.00005)
			if err != nil {
				t.Fatal(duty, j, err)
			}
			k.mixed, tr = next, trace
			r := next.result
			u := r.HBridges[0]
			drive = drive || u.State == "forward"
			coast = coast || u.State == "coast"
			for _, i := range u.ClampCurrents {
				clamp = clamp || i > .001
			}
			if r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-6 {
				t.Fatal("PWM conservation", duty, j, r.PowerResidual)
			}
		}
		if !drive || !coast || !clamp {
			t.Fatal("missing PWM mode/current path", drive, coast, clamp)
		}
		speeds = append(speeds, k.mixed.result.Motors[0].RPM)
	}
	if speeds[1] < speeds[0]*1.5 {
		t.Fatal("PWM failed to change speed", speeds)
	}
}

func TestHBridgeSwitchModesPreserveMotorStorageAndBrakeFasterThanCoast(t *testing.T) {
	speeds := map[string]float64{}
	for _, mode := range []string{"brake low", "brake high", "coast", "reverse"} {
		s := newGraphService(SolveLED, hbridgeConstantGraph())
		call(t, s, "simulation.pause", struct{}{}, "pause")
		settled(t, s)
		call(t, s, "simulation.step", StepParams{Duration: .06}, "spin")
		before := settled(t, s)
		edits := []SwitchParams{}
		switch mode {
		case "brake low":
			edits = append(edits, SwitchParams{ID: "S1", Closed: false})
		case "brake high":
			edits = append(edits, SwitchParams{ID: "S2", Closed: true})
		case "coast":
			edits = append(edits, SwitchParams{ID: "SEN", Closed: false})
		case "reverse":
			edits = append(edits, SwitchParams{ID: "S1", Closed: false}, SwitchParams{ID: "S2", Closed: true})
		}
		for j, e := range edits {
			call(t, s, "switch.set", e, fmt.Sprint("mode", j))
		}
		if !reflect.DeepEqual(before.Mixed, s.Snapshot().Mixed) {
			t.Fatal("queued change consumed accepted physics")
		}
		// Paused switch requests apply at the next accepted output boundary.
		// Advance a private unchanged reference to that boundary first: the
		// event must preserve BOTH armature current and mechanical storage.
		s.mu.Lock()
		refState, refTrace := cloneMixedState(s.mixedState), cloneTransient(s.transient)
		s.mu.Unlock()
		reference, _, err := advanceMixed(refState, refTrace, before.Mixed.Time+.00005)
		if err != nil {
			t.Fatal(err)
		}
		call(t, s, "simulation.step", StepParams{Duration: .00005}, "edge")
		after := settled(t, s)
		m, want := after.Mixed.Motors[0], reference.result.Motors[0]
		if math.Abs(m.Current-want.Current) > 1e-12 || math.Abs(m.Omega-want.Omega) > 1e-12 || after.RunID != before.RunID || after.Mixed.HBridges[0].State != mode {
			t.Fatal("mode boundary changed stored current/speed", mode, m, want, after.Mixed.HBridges)
		}

		call(t, s, "simulation.step", StepParams{Duration: .04}, "decay")
		after = settled(t, s)
		speeds[mode] = after.Mixed.Motors[0].RPM
		if after.Mixed.Stale || after.Mixed.Time <= before.Mixed.Time || after.Mixed.KCLResidual > 1e-9 || math.Abs(after.Mixed.PowerResidual) > 1e-6 {
			t.Fatal("failed mode continuation", mode, after.Diagnostics)
		}
		s.Close()
	}
	if speeds["brake low"] > speeds["coast"]*.5 || speeds["brake high"] > speeds["coast"]*.5 || speeds["reverse"] >= 0 {
		t.Fatal("brake/coast/reverse", speeds)
	}
}

func TestHBridgeCaptureProjectProvenanceAndProbeSigns(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "hbridge-motor"}, "example")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	settled(t, s)
	duration, step := .01, .00005
	_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:U1", "i:U1.vm", "i:U1.out1", "i:U1.out2", "i:MTR1"}}, "verify")
	rec := s.operations[view.Operation.ID].record
	raw, _ := json.Marshal(VerificationDocument{Metadata: rec.metadata, Rows: rec.rows})
	decoded, err := decodeRecording(raw)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.metadata.HBridgeModel == nil || recordingModelHashes(decoded.metadata)[HBridgeModelID] == "" {
		t.Fatal("missing provenance")
	}
	var obj map[string]any
	json.Unmarshal(raw, &obj)
	device := obj["metadata"].(map[string]any)["mixed"].(map[string]any)["hbridges"].([]any)[0].(map[string]any)
	for _, field := range []string{"state", "pin_currents_a", "clamp_currents_a", "model", "dissipation_w"} {
		old := device[field]
		delete(device, field)
		bad, _ := json.Marshal(obj)
		if _, e := decodeRecording(bad); e == nil {
			t.Fatal("missing field accepted", field)
		}
		device[field] = old
	}
	device["state"] = "forged"
	bad, _ := json.Marshal(obj)
	if _, e := decodeRecording(bad); e == nil {
		t.Fatal("forged state")
	}
	path := filepath.Join(t.TempDir(), "bridge.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	r := settled(t, fresh)
	if r.HBridgeModel == nil || r.Document.Dirty || r.Document.RecordingCount != 1 {
		t.Fatal("project roundtrip")
	}
	data, _ := os.ReadFile(path)
	d, e := decodeDocument(data)
	if e != nil || inspectDocument(d).HBridgeModel == nil {
		t.Fatal("recovery provenance", e)
	}
	var doc map[string]any
	json.Unmarshal(data, &doc)
	delete(doc, "hbridge_model")
	bad, _ = json.Marshal(doc)
	if _, e := decodeDocument(bad); e == nil {
		t.Fatal("missing model")
	}
	state := s.Snapshot()
	state.HBridgeModel.OnResistance = 9
	state.Mixed.HBridges[0].PinCurrents[0] = 42
	if s.Snapshot().HBridgeModel.OnResistance != .25 || s.Snapshot().Mixed.HBridges[0].PinCurrents[0] == 42 {
		t.Fatal("snapshot alias")
	}
	clone := cloneVerificationMetadata(decoded.metadata)
	clone.HBridgeModel.OnResistance = 9
	if reflect.DeepEqual(clone.HBridgeModel, decoded.metadata.HBridgeModel) {
		t.Fatal("capture alias")
	}
	call(t, fresh, "simulation.step", StepParams{Duration: .002}, "probe-current")
	r = settled(t, fresh)
	if r.Mixed.Motors[0].Current < .1 || r.Mixed.HBridges[0].PinCurrents[3] >= 0 || math.Abs(r.Mixed.HBridges[0].PinCurrents[3]+r.Mixed.Motors[0].Current) > 1e-9 {
		t.Fatal("driver/motor output signs", r.Mixed.HBridges, r.Mixed.Motors)
	}
	c, _, _ := HBridgeExampleGraph().Compile()
	frame := mixedSample(c, r.Mixed, 0)
	for _, p := range probeInventory(c.MixedGraph) {
		if p.ID == "i:U1.out1" {
			v, ok := frameValue(frame, p)
			if !ok || v != r.Mixed.HBridges[0].PinCurrents[3] {
				t.Fatal("output current sign")
			}
		}
	}
}

func TestHBridgeRequiredControlsSupplyGuardAndOptionalOutputs(t *testing.T) {
	g := HBridgeExampleGraph()
	drop := func(g Graph, ref string) Graph {
		out := g.Clone()
		out.Wires = nil
		for _, w := range g.Wires {
			if w.From != ref && w.To != ref {
				out.Wires = append(out.Wires, w)
			}
		}
		return out
	}
	if _, status, diagnostic := drop(g, "U1.in1").Compile(); status != "incomplete" || diagnostic == nil || diagnostic.Code != "open-terminal" {
		t.Fatal("unwired control accepted", status, diagnostic)
	}
	short := g.Clone()
	short.Wires = append(short.Wires, Wire{ID: "SHORT", From: "U1.vm", To: "U1.gnd", Connected: true})
	if _, status, diagnostic := short.Compile(); status != "error" || diagnostic == nil || diagnostic.Code != "shorted-component" {
		t.Fatal("shorted supply accepted", status, diagnostic)
	}
	open := drop(drop(g, "U1.out1"), "U1.out2")
	// Remove the now-unused motor; the driver's explicit off paths determine open
	// output voltage without demanding an invented external bleed resistor.
	parts := open.Components[:0]
	for _, p := range open.Components {
		if p.ID != "MTR1" {
			parts = append(parts, p)
		}
	}
	open.Components = parts
	if _, status, diagnostic := open.Compile(); status != "" || diagnostic != nil {
		t.Fatal("open bridge outputs refused", status, diagnostic)
	}
}
