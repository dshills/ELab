package workbench

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func pmosDCGraph(gate float64) Graph {
	g := PMOSExampleGraph()
	for i := range g.Components {
		if g.Components[i].ID == "VIN" {
			c, _ := MakeComponent("VIN", "dc-source", g.Components[i].Position, FormatValue(gate, "V"))
			g.Components[i] = c
		}
	}
	return g
}

func TestPMOSPolarityRegionsBodyDiodeAndJacobian(t *testing.T) {
	for _, p := range []struct{ g, d, want float64 }{{0, -5, 0}, {-1.5, -5, 0}, {-2.5, -.5, -.037875}, {-2.5, -2, -.052}} {
		i, _, _ := mosfetLinear("pmos", p.g, p.d)
		want := p.want + 1e-9*math.Expm1(p.d/(1.7*1.380649e-23*300.15/1.602176634e-19))
		if math.Abs(i-want) > 1e-14 {
			t.Fatalf("PMOS %g/%g: %g != %g", p.g, p.d, i, p.want)
		}
	}
	for _, p := range [][2]float64{{0, -5}, {-2.5, -.5}, {-2.5, -2}, {-5, .2}, {2, .8}, {-2.5, 0}} {
		i, gm, gd := mosfetLinear("pmos", p[0], p[1])
		const h = 1e-6
		a, _, _ := mosfetLinear("pmos", p[0]+h, p[1])
		b, _, _ := mosfetLinear("pmos", p[0]-h, p[1])
		dg := (a - b) / (2 * h)
		a, _, _ = mosfetLinear("pmos", p[0], p[1]+h)
		b, _, _ = mosfetLinear("pmos", p[0], p[1]-h)
		dd := (a - b) / (2 * h)
		if math.Abs(gm-dg) > 1e-6 || math.Abs(gd-dd) > 1e-6 || i*p[1] < -1e-14 {
			t.Fatal("PMOS derivative/passivity", p, gm, gd, dg, dd, i)
		}
	}
	i, _, _ := mosfetLinear("pmos", 2, .8)
	nv := 1.7 * 1.380649e-23 * 300.15 / 1.602176634e-19
	if i <= 0 || math.Abs(nv*math.Log1p(i/1e-9)+i-.8) > 1e-12 {
		t.Fatal("PMOS body diode does not flow drain to source", i)
	}
	for _, p := range []struct {
		g, d  float64
		state string
	}{{0, -5, "cutoff"}, {-5, -.01, "triode"}, {-2, -5, "saturation"}, {0, 1, "reverse"}} {
		if mosfetState("pmos", p.g, p.d) != p.state {
			t.Fatal("PMOS region", p)
		}
	}
}

func TestPMOSIndependentLoadLineAndSignedProbes(t *testing.T) {
	for _, gate := range []float64{0, 1, 2, 3.3, 3.5, 4, 5, 12} {
		s := newGraphService(SolveLED, pmosDCGraph(gate))
		state := settled(t, s)
		s.Close()
		if state.Mixed == nil || len(state.Mixed.MOSFETs) != 1 || state.PMOSModel == nil || state.NMOSModel != nil {
			t.Fatal("PMOS metadata", state.Diagnostics)
		}
		m := state.Mixed.MOSFETs[0]
		want := nmosLEDReference((5-gate)*100000/100100, 0)
		if math.Abs(m.DrainCurrent+want) > 2e-9 || math.Abs(state.Mixed.LEDs[0].Current-want) > 2e-9 || math.Abs(m.VGS-(gate-5)*100000/100100) > 1e-10 || m.GateCurrent != 0 || m.SourceCurrent != m.DrainCurrent || state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
			t.Fatal("high-side load/reference", gate, m, want, state.Diagnostics)
		}
	}
}

func TestPMOSPulseGateRCAndCapturedModels(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "pmos"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	first := settled(t, s)
	if first.Mixed == nil || first.Mixed.MOSFETs[0].State != "cutoff" {
		t.Fatal("gate high should turn PMOS off", first.Diagnostics)
	}
	probes := []string{"v:M1", "v:M1.gate-source", "i:M1.drain", "i:M1.gate", "i:M1.source", "i:D1"}
	call(t, s, "scope.probes.set", ProbeParams{IDs: probes}, "probes")
	call(t, s, "simulation.step", StepParams{Duration: .2}, "fall")
	on := s.Snapshot()
	if on.Mixed.MOSFETs[0].State != "triode" || on.Mixed.MOSFETs[0].DrainCurrent > -.0028 || on.Mixed.LEDs[0].Current < .0028 {
		t.Fatal("low gate failed to turn high-side on", on.Mixed)
	}
	call(t, s, "simulation.step", StepParams{Duration: .3}, "rise")
	if s.Snapshot().Mixed.MOSFETs[0].State != "cutoff" {
		t.Fatal("high gate failed to turn PMOS off")
	}
	duration := .5
	_, record := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: probes}, "verify")
	if record.Metadata.PMOSModel == nil || record.Metadata.NMOSModel != nil || record.Metadata.Mixed.MOSFETs[0].Model != PMOSDeviceModel() {
		t.Fatal("PMOS capture model")
	}
	stored := s.operations[record.Operation.ID].record
	data, err := encodeVerificationExport(stored, "json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeRecording(data); err != nil {
		t.Fatal("valid PMOS capture refused", err)
	}
	for _, bad := range []string{"missing-metadata", "wrong-polarity", "duplicate-id", "unknown-id"} {
		var doc VerificationDocument
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		switch bad {
		case "missing-metadata":
			doc.Metadata.PMOSModel = nil
		case "wrong-polarity":
			doc.Metadata.Mixed.MOSFETs[0].Model = NMOSDeviceModel()
		case "duplicate-id":
			doc.Metadata.Mixed.MOSFETs = append(doc.Metadata.Mixed.MOSFETs, doc.Metadata.Mixed.MOSFETs[0])
		case "unknown-id":
			doc.Metadata.Mixed.MOSFETs[0].ID = "Foreign"
		}
		raw, _ := json.Marshal(doc)
		if _, err := decodeRecording(raw); err == nil {
			t.Fatal("accepted corrupted PMOS capture", bad)
		}
	}
	path := filepath.Join(t.TempDir(), "pmos.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "reopen")
	loaded := settled(t, fresh)
	if loaded.PMOSModel == nil || *loaded.PMOSModel != PMOSDeviceModel() || loaded.Document.RecordingCount != 1 || loaded.Transient.Sample.Time != 0 || loaded.Mixed.MOSFETs[0].State != "cutoff" {
		t.Fatal("PMOS portable capture/startup", loaded.Diagnostics)
	}
	copy := loaded
	copy.PMOSModel.Beta = 99
	copy.Mixed.MOSFETs[0].Model.Beta = 88
	if *fresh.Snapshot().PMOSModel != PMOSDeviceModel() || fresh.Snapshot().Mixed.MOSFETs[0].Model != PMOSDeviceModel() {
		t.Fatal("PMOS snapshot model aliases")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"missing", "tampered"} {
		var doc map[string]any
		_ = json.Unmarshal(raw, &doc)
		if bad == "missing" {
			delete(doc, "pmos_model")
		} else {
			doc["pmos_model"].(map[string]any)["threshold_v"] = 1.5
		}
		invalid, _ := json.Marshal(doc)
		if _, err := decodeDocument(invalid); err == nil {
			t.Fatal("invalid PMOS project accepted", bad)
		}
	}
	call(t, s, "component.set", ValueParams{ID: "RG", Value: "10k"}, "rg")
	call(t, s, "component.place", PlaceParams{ID: "CG", Kind: "capacitor", Position: Point{520, -220}, Value: "1uF"}, "cg")
	call(t, s, "wire.connect", ConnectParams{From: "CG.1", To: "M1.gate"}, "cg1")
	call(t, s, "wire.connect", ConnectParams{From: "CG.2", To: "M1.source"}, "cg2")
	call(t, s, "simulation.restart", struct{}{}, "rc-restart")
	initial := settled(t, s)
	if math.Abs(initial.Mixed.MOSFETs[0].VGS) > 1e-12 {
		t.Fatal("PMOS gate capacitor startup")
	}
	call(t, s, "simulation.step", StepParams{Duration: .2}, "rc-fall")
	if math.Abs(s.Snapshot().Mixed.MOSFETs[0].VGS) > 1e-12 {
		t.Fatal("capacitor charge discontinuity at falling edge")
	}
	call(t, s, "simulation.step", StepParams{Duration: .01}, "rc-on")
	charged := s.Snapshot()
	tau := 1e-6 / (1./10000 + 1./100000)
	want := -5. * 100000 / 110000 * (1 - math.Exp(-.01/tau))
	if math.Abs(charged.Mixed.MOSFETs[0].VGS-want) > .003 || charged.Mixed.LEDs[0].Current < .0028 {
		t.Fatal("PMOS gate RC charge", charged.Mixed, want)
	}
	call(t, s, "simulation.step", StepParams{Duration: .29}, "rc-rise")
	edge := s.Snapshot()
	if edge.Mixed.MOSFETs[0].VGS > -4.5 {
		t.Fatal("PMOS gate RC lost stored charge")
	}
	call(t, s, "simulation.step", StepParams{Duration: .01}, "rc-off")
	discharged := s.Snapshot()
	want = -5. * 100000 / 110000 * (1 - math.Exp(-.3/tau)) * math.Exp(-.01/tau)
	if math.Abs(discharged.Mixed.MOSFETs[0].VGS-want) > .003 || discharged.Mixed.KCLResidual > 1e-9 || math.Abs(discharged.Mixed.PowerResidual) > 1e-8 {
		t.Fatal("PMOS RC discharge", discharged.Mixed, want)
	}
}

func cmosTestGraph(t *testing.T, gate float64) Graph {
	t.Helper()
	g := EmptyGraph()
	for _, p := range []struct {
		id, kind, value string
		point           Point
	}{{"VCC", "dc-source", "5 V", Point{-200, -100}}, {"VIN", "dc-source", FormatValue(gate, "V"), Point{-200, 100}}, {"MP", "pmos", "", Point{100, -100}}, {"MN", "nmos", "", Point{100, 100}}, {"RL", "resistor", "10k", Point{250, 0}}, {"GND", "ground", "", Point{300, 200}}} {
		c, err := MakeComponent(p.id, p.kind, p.point, p.value)
		if err != nil {
			t.Fatal(err)
		}
		g.Components = append(g.Components, c)
	}
	for i, p := range [][2]string{{"VCC.positive", "MP.source"}, {"VIN.positive", "MP.gate"}, {"MP.gate", "MN.gate"}, {"MP.drain", "MN.drain"}, {"MN.source", "GND.reference"}, {"MN.drain", "RL.1"}, {"RL.2", "GND.reference"}, {"VCC.negative", "GND.reference"}, {"VIN.negative", "GND.reference"}} {
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", i+1), From: p[0], To: p[1], Connected: true})
	}
	return g
}

func TestComplementaryMOSInverterConservationCaptureAndCombinedLimit(t *testing.T) {
	for _, gate := range []float64{0, 2.5, 5} {
		g := cmosTestGraph(t, gate)
		s := newGraphService(SolveLED, g)
		state := settled(t, s)
		if state.Mixed == nil || len(state.Mixed.MOSFETs) != 2 || state.PMOSModel == nil || state.NMOSModel == nil {
			t.Fatal("complementary models", state.Diagnostics)
		}
		mp, mn := state.Mixed.MOSFETs[0], state.Mixed.MOSFETs[1]
		voltage := mn.VDS
		if gate == 0 && voltage < 4.99 || gate == 5 && voltage > 1e-6 || gate == 2.5 && (voltage < 2 || voltage > 3) {
			t.Fatal("inverter levels", gate, voltage)
		}
		if math.Abs(mp.DrainCurrent+mn.DrainCurrent+voltage/10000) > 1e-9 || state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
			t.Fatal("inverter conservation", state.Mixed)
		}
		_, view := verify(t, s, VerifyParams{ProbeIDs: []string{"i:MP.drain", "i:MN.drain", "v:MN"}}, "verify")
		raw, err := encodeVerificationExport(s.operations[view.Operation.ID].record, "json")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeRecording(raw); err != nil {
			t.Fatal("mixed MOS capture", err)
		}
		s.Close()
		for _, id := range []string{"M3", "M4", "M5"} {
			extra, _ := MakeComponent(id, "pmos", Point{400, -100}, "")
			g.Components = append(g.Components, extra)
			for _, pin := range extra.Terminals {
				g.Wires = append(g.Wires, Wire{ID: "W" + id + pin, From: id + "." + pin, To: "MP." + pin, Connected: true})
			}
		}
		if _, _, d := g.Compile(); d == nil || d.Code != "mixed-limit" {
			t.Fatal("N/P combined limit missing", d)
		}
	}
}

func TestPMOSSearchSweepAndLegacyNMOSModelHash(t *testing.T) {
	model := NMOSDeviceModel()
	raw, _ := json.Marshal(&model)
	h := sha256.Sum256(raw)
	// Actual model dependency hash captured by 0.88.0 /103; type refactoring
	// must not reinterpret the already versioned NMOS model or saved recordings.
	if hex.EncodeToString(h[:]) != "7d0606fec7e453a3dde0026b6fca0e4d6f5794bb774e008e1e3ee046d2626331" {
		t.Fatal("legacy NMOS model changed without a version")
	}
	s := New()
	defer s.Close()
	call(t, s, "parts.search.set", SearchParams{Query: "MOSFET"}, "search")
	got := map[string]bool{}
	for _, p := range s.Snapshot().MatchingParts {
		got[p.Kind] = true
	}
	if !reflect.DeepEqual(got, map[string]bool{"nmos": true, "pmos": true}) {
		t.Fatal("MOSFET search", got)
	}
	for _, kind := range []string{"pmos", "nmos"} {
		call(t, s, "component.place", PlaceParams{Kind: kind, Position: Point{X: float64(len(s.Snapshot().Components)) * 100}}, kind)
	}
	if s.Snapshot().Components[0].ID != "M1" || s.Snapshot().Components[1].ID != "M2" {
		t.Fatal("shared lifetime M IDs")
	}
	result := sweepForTest(t, pmosDCGraph(0), SweepSettings{ComponentID: "VIN", Parameter: "voltage", Start: "0 V", Stop: "5 V", Points: 11, ProbeIDs: []string{"i:M1.drain", "v:M1.gate-source"}})
	for _, p := range result.Points {
		want := -nmosLEDReference((5-p.Input)*100000/100100, 0)
		if math.Abs(p.Values[0]-want) > 2e-9 {
			t.Fatal("PMOS sweep reference", p, want)
		}
	}
}

func TestTransistorExportLabelDoesNotCoverNamedPins(t *testing.T) {
	for _, kind := range []string{"nmos", "pmos", "npn"} {
		for _, rotation := range []int{0, 90, 180, 270} {
			part, err := MakeComponent("M1", kind, Point{100, 0}, "")
			if err != nil {
				t.Fatal(err)
			}
			part.Rotation = rotation
			x, y, text := schematicPartLabel(part)
			for _, pin := range part.Terminals {
				at, ok := TerminalPosition(part, pin)
				if !ok {
					t.Fatal("missing pin")
				}
				if at.X >= x-3 && at.X <= x-3+schematicLabelWidth(text) && at.Y >= y-15 && at.Y <= y+4 {
					t.Fatalf("%s rotated %d hides named %s terminal behind label background", kind, rotation, pin)
				}
			}
		}
	}
}
