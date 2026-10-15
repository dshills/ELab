package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func nmosDCGraph() Graph {
	g := NMOSExampleGraph()
	for i := range g.Components {
		if g.Components[i].ID == "VIN" {
			c, _ := MakeComponent("VIN", "dc-source", g.Components[i].Position, "5 V")
			g.Components[i] = c
		}
	}
	return g
}

// Independent current-domain load line: invert the MOS triode quadratic and
// the LED Shockley relation. These constants pin the public v1 model.
func nmosLEDReference(gate, source float64) float64 {
	if gate-source <= 1.5 {
		return 1e-9
	}
	lo, hi := 0., (5-source)/1000
	for k := 0; k < 100; k++ {
		i := (lo + hi) / 2
		led := 2*1.380649e-23*300.15/1.602176634e-19*math.Log1p(i/1e-20) + i
		drain := 5 - source - 1000*i - led
		u := gate - source - 1.5
		channel := 0.
		if drain > 0 {
			if drain < u {
				channel = .1 * (u*drain - .5*drain*drain) * (1 + .02*drain)
			} else {
				channel = .05 * u * u * (1 + .02*drain)
			}
		}
		if channel+1e-9 > i {
			lo = i
		} else {
			hi = i
		}
	}
	return (lo + hi) / 2
}

func TestNMOSChannelRegionsReverseAndAnalyticJacobian(t *testing.T) {
	for _, tc := range []struct{ g, d, want float64 }{{0, 5, 0}, {1.5, 5, 0}, {2.5, .5, .037875}, {2.5, 2, .052}, {2.5, 1, .051}} {
		i, _, _ := nmosChannel(tc.g, tc.d)
		if math.Abs(i-tc.want) > 1e-14 {
			t.Fatalf("channel (%g,%g): %g != %g", tc.g, tc.d, i, tc.want)
		}
	}
	for _, v := range [][2]float64{{0, 5}, {2.5, .5}, {2.5, 2}, {5, -.2}, {0, -.8}, {-.5, -.1}, {2.5, 0}} {
		i, gm, gd := nmosLinear(v[0], v[1])
		const h = 1e-6
		plus, _, _ := nmosLinear(v[0]+h, v[1])
		minus, _, _ := nmosLinear(v[0]-h, v[1])
		ng := (plus - minus) / (2 * h)
		plus, _, _ = nmosLinear(v[0], v[1]+h)
		minus, _, _ = nmosLinear(v[0], v[1]-h)
		nd := (plus - minus) / (2 * h)
		if !finite(i) || math.Abs(ng-gm) > 1e-6 || math.Abs(nd-gd) > 1e-6 {
			t.Fatalf("Jacobian %v: %g/%g != %g/%g", v, gm, gd, ng, nd)
		}
		if i*v[1] < -1e-14 {
			t.Fatal("passive drain/source branch generated power")
		}
	}
	i, _, _ := nmosLinear(-2, -.8)
	// Body diode: inverse Shockley, distinct from the channel implementation.
	nv := 1.7 * 1.380649e-23 * 300.15 / 1.602176634e-19
	if i >= 0 || math.Abs(nv*math.Log1p(-i/1e-9)-i-.8) > 1e-12 {
		t.Fatal("body diode orientation/drop", i)
	}
	if nmosState(5, .01) != "triode" || nmosState(2, 5) != "saturation" || nmosState(0, 5) != "cutoff" || nmosState(0, -1) != "reverse" {
		t.Fatal("regions")
	}
}

func TestNMOSDCLoadLineAndNonGroundedSource(t *testing.T) {
	for _, source := range []float64{0, .5} {
		for _, gate := range []float64{0, 1, 1.6, 2, 3.3, 5, 12} {
			g := nmosDCGraph()
			for i := range g.Components {
				if g.Components[i].ID == "VIN" {
					g.Components[i].Value = gate
				}
			}
			if source != 0 {
				bias, _ := MakeComponent("VS", "dc-source", Point{400, 200}, "0.5 V")
				g.Components = append(g.Components, bias)
				for i := range g.Wires {
					if g.Wires[i].From == "M1.source" {
						g.Wires[i].To = "VS.positive"
					}
				}
				g.Wires = append(g.Wires, Wire{ID: "WB", From: "VS.negative", To: "GND1.reference", Connected: true})
			}
			s := newGraphService(SolveLED, g)
			state := settled(t, s)
			s.Close()
			if state.Mixed == nil || len(state.Mixed.MOSFETs) != 1 {
				t.Fatal(state.Diagnostics)
			}
			m := state.Mixed.MOSFETs[0]
			want := nmosLEDReference(gate*100000/100100, source)
			if state.Analysis != NetworkDCAnalysis || math.Abs(m.DrainCurrent-want) > 2e-9 || m.GateCurrent != 0 || m.SourceCurrent != m.DrainCurrent || math.Abs(m.VGS-(gate*100000/100100-source)) > 1e-10 || state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
				t.Fatalf("gate/source %g/%g: %+v want %g", gate, source, m, want)
			}
		}
	}
}

func TestNMOSPulseTimingRCGateAndOwnedProbes(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "nmos"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	first := settled(t, s)
	if first.Analysis != NetworkTransientAnalysis || first.Mixed.MOSFETs[0].State != "triode" || first.NMOSModel == nil || len(first.Scope.Selected) != 5 {
		t.Fatal("NMOS template", first.Diagnostics)
	}
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"v:M1", "v:M1.gate-source", "i:M1.drain", "i:M1.gate", "i:M1.source"}}, "probes")
	call(t, s, "simulation.step", StepParams{Duration: .2}, "fall")
	off := s.Snapshot()
	if off.Mixed.MOSFETs[0].State != "cutoff" || off.Mixed.LEDs[0].Current > 2e-9 || math.Abs(off.Transient.Sample.Time-.2) > 1e-12 {
		t.Fatal("pulse falling edge", off.Diagnostics, off.Mixed)
	}
	call(t, s, "simulation.step", StepParams{Duration: .3}, "rise")
	if s.Snapshot().Mixed.LEDs[0].Current < .0028 {
		t.Fatal("pulse rising edge")
	}
	copied := s.Snapshot()
	copied.Mixed.MOSFETs[0].Model.Beta = 1
	copied.NMOSModel.Beta = 2
	if s.Snapshot().Mixed.MOSFETs[0].Model != NMOSDeviceModel() || *s.Snapshot().NMOSModel != NMOSDeviceModel() {
		t.Fatal("snapshot aliases")
	}
	// An explicit gate capacitor gives analytic first-order timing. Use 10k RG,
	// 100k RPD and 1uF => tau=9.0909ms, target=4.54545V.
	call(t, s, "component.set", ValueParams{ID: "RG", Value: "10k"}, "rg")
	call(t, s, "component.place", PlaceParams{Kind: "capacitor", ID: "CG", Position: Point{-200, 220}, Value: "1uF"}, "cap")
	call(t, s, "wire.connect", ConnectParams{From: "CG.1", To: "M1.gate"}, "cg1")
	call(t, s, "wire.connect", ConnectParams{From: "CG.2", To: "GND1.reference"}, "cg2")
	call(t, s, "simulation.restart", struct{}{}, "rc-restart")
	initial := settled(t, s)
	if initial.Mixed.MOSFETs[0].State != "cutoff" || initial.Mixed.Capacitors[0].Voltage != 0 {
		t.Fatal("gate capacitor startup", initial.Diagnostics)
	}
	call(t, s, "simulation.step", StepParams{Duration: .01}, "rc-charge")
	charged := s.Snapshot()
	want := 5. * 100000 / 110000 * (1 - math.Exp(-.01/(1./(1./10000+1./100000)*1e-6)))
	if math.Abs(charged.Mixed.MOSFETs[0].VGS-want) > .003 || charged.Mixed.LEDs[0].Current < .0028 || charged.Mixed.KCLResidual > 1e-9 {
		t.Fatal("external gate RC reference", charged.Mixed, want)
	}
	call(t, s, "simulation.step", StepParams{Duration: .19}, "rc-edge")
	edge := s.Snapshot()
	if edge.Mixed.MOSFETs[0].VGS < 4.5 {
		t.Fatal("gate capacitor charge erased at pulse edge")
	}
	call(t, s, "simulation.step", StepParams{Duration: .01}, "rc-discharge")
	discharged := s.Snapshot()
	want = 5. * 100000 / 110000 * math.Exp(-.01/(1./(1./10000+1./100000)*1e-6))
	if math.Abs(discharged.Mixed.MOSFETs[0].VGS-want) > .003 {
		t.Fatal("gate discharge", discharged.Mixed.MOSFETs[0].VGS, want)
	}
}

func TestNMOSCapturePersistenceSweepAndModelGuards(t *testing.T) {
	s := newGraphService(SolveLED, nmosDCGraph())
	defer s.Close()
	original := settled(t, s)
	_, record := verify(t, s, VerifyParams{ProbeIDs: []string{"v:M1", "v:M1.gate-source", "i:M1.drain", "i:M1.gate", "i:M1.source"}}, "capture")
	if record.Metadata.NMOSModel == nil || record.Metadata.Mixed.MOSFETs[0].Model != NMOSDeviceModel() {
		t.Fatal("capture omitted NMOS model")
	}
	path := filepath.Join(t.TempDir(), "nmos.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	loaded := settled(t, fresh)
	if loaded.Document.RecordingCount != 1 || !reflect.DeepEqual(loaded.Mixed.MOSFETs, original.Mixed.MOSFETs) || loaded.NMOSModel == nil {
		t.Fatal("NMOS save/reopen")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"missing", "tampered"} {
		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		if bad == "missing" {
			delete(doc, "nmos_model")
		} else {
			doc["nmos_model"].(map[string]any)["threshold_v"] = 99
		}
		raw, _ := json.Marshal(doc)
		if _, err := decodeDocument(raw); err == nil {
			t.Fatal("accepted", bad, "NMOS parameters")
		}
	}
	result := sweepForTest(t, nmosDCGraph(), SweepSettings{ComponentID: "VIN", Parameter: "voltage", Start: "0 V", Stop: "5 V", Points: 11, ProbeIDs: []string{"i:M1.drain", "v:M1.gate-source"}})
	for _, p := range result.Points {
		if math.Abs(p.Values[0]-nmosLEDReference(p.Input*100000/100100, 0)) > 2e-9 {
			t.Fatal("gate sweep", p)
		}
	}
	c, _, d := nmosDCGraph().Compile()
	if d != nil {
		t.Fatal(d)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n := newMixedNetwork(c)
	if _, err := n.solveContext(ctx, nil, nil, 0, &Transient{}, 0); err == nil {
		t.Fatal("canceled NMOS solve accepted")
	}
}

func TestNMOSConstructionAndCapacity(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "parts.search.set", SearchParams{Query: "NMOS"}, "search")
	if p := s.Snapshot().MatchingParts; len(p) != 1 || p[0].Kind != "nmos" {
		t.Fatal("search discoverability", p)
	}
	call(t, s, "component.place", PlaceParams{Kind: "nmos", Position: Point{}}, "place")
	if _, ok := (Graph{Components: s.Snapshot().Components}).Component("M1"); !ok {
		t.Fatal("auto ID")
	}
	g := nmosDCGraph()
	for i := 2; i <= 3; i++ {
		m, _ := MakeComponent(fmt.Sprintf("M%d", i), "nmos", Point{float64(i) * 100, 300}, "")
		g.Components = append(g.Components, m)
		for _, pin := range m.Terminals {
			g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("WM%d%s", i, pin), From: m.ID + "." + pin, To: "M1." + pin, Connected: true})
		}
	}
	if _, _, d := g.Compile(); d == nil || d.Code != "mixed-limit" {
		t.Fatal("missing NMOS capacity diagnostic", d)
	}
}

func TestNMOSBodyDiodeWithGateTiedToSource(t *testing.T) {
	g := EmptyGraph()
	for _, p := range []struct {
		id, kind, value string
		point           Point
	}{{"V1", "dc-source", "5 V", Point{-200, 0}}, {"R1", "resistor", "1k", Point{0, -100}}, {"M1", "nmos", "", Point{150, 0}}, {"GND", "ground", "", Point{0, 100}}} {
		c, _ := MakeComponent(p.id, p.kind, p.point, p.value)
		g.Components = append(g.Components, c)
	}
	for i, p := range [][2]string{{"V1.positive", "M1.source"}, {"M1.gate", "M1.source"}, {"M1.drain", "R1.1"}, {"R1.2", "GND.reference"}, {"V1.negative", "GND.reference"}} {
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", i+1), From: p[0], To: p[1], Connected: true})
	}
	s := newGraphService(SolveLED, g)
	defer s.Close()
	state := settled(t, s)
	if state.Mixed == nil {
		t.Fatal(state.Diagnostics)
	}
	m := state.Mixed.MOSFETs[0]
	want := referenceSiliconCurrent(5, 1000)
	if m.State != "reverse" || math.Abs(m.DrainCurrent+want) > 1e-10 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
		t.Fatal("body diode reverse load", m, want)
	}
}
