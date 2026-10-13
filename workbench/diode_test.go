package workbench

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func siliconSeriesGraph(t *testing.T, reverse bool) Graph {
	t.Helper()
	g := ExampleGraph()
	diode, err := MakeComponent("D1", "diode", Point{200, 0}, "")
	if err != nil {
		t.Fatal(err)
	}
	g.Components[2] = diode
	if reverse {
		g.Wires[1].To = "D1.cathode"
		g.Wires[2].From = "D1.anode"
	}
	return g
}
func siliconRCGraph(t *testing.T) Graph {
	t.Helper()
	g := RCExampleGraph()
	diode, err := MakeComponent("D1", "diode", Point{100, -60}, "")
	if err != nil {
		t.Fatal(err)
	}
	g.Components = append(g.Components, diode)
	for i := range g.Wires {
		if g.Wires[i].From == "R1.2" {
			g.Wires[i].From = "D1.cathode"
		}
	}
	g.Wires = append(g.Wires, Wire{ID: "WD", From: "R1.2", To: "D1.anode", Connected: true})
	return g
}

// Independent current-domain reference for a forward series junction. It uses
// the inverse Shockley equation plus the external resistance, not the MNA code.
func referenceSiliconCurrent(v, r float64) float64 {
	if v <= 0 {
		return -1e-9
	}
	nv := 1.7 * 1.380649e-23 * 300.15 / 1.602176634e-19
	lo, hi := 0., v/(r+1)
	for k := 0; k < 80; k++ {
		i := (lo + hi) / 2
		if i*(r+1)+nv*math.Log1p(i/1e-9) > v {
			hi = i
		} else {
			lo = i
		}
	}
	return (lo + hi) / 2
}
func TestSiliconDiodeForwardReverseAndConservation(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		s := newGraphService(SolveLED, siliconSeriesGraph(t, reverse))
		state := settled(t, s)
		if state.Analysis != NetworkDCAnalysis || state.DiodeModel == nil || state.Mixed == nil || len(state.Mixed.Diodes) != 1 {
			t.Fatalf("missing silicon result: %+v", state.Diagnostics)
		}
		result := state.Mixed.Diodes[0]
		want := referenceSiliconCurrent(5, 1000)
		if reverse {
			want = -1e-9
		}
		if math.Abs(result.Current-want) > 1e-11 || state.Mixed.KCLResidual > 1e-10 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
			t.Fatalf("silicon reverse=%v: %+v, want %g", reverse, result, want)
		}
		if reverse && result.Voltage > -4.99 || !reverse && (result.Voltage < .6 || result.Voltage > .8) {
			t.Fatalf("diode polarity: %+v", result)
		}
		_, record := verify(t, s, VerifyParams{ProbeIDs: []string{"i:D1", "v:D1"}}, "verify")
		if record.Metadata.DiodeModel == nil || record.Metadata.Mixed.Diodes[0].Current != result.Current {
			t.Fatal("silicon recording lacks model/probes")
		}
		s.Close()
	}
}
func TestSiliconDiodeRCChargeAndReverseHold(t *testing.T) {
	s := newGraphService(SolveLED, siliconRCGraph(t))
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	call(t, s, "simulation.input.set", InputParams{Mode: "high"}, "high")
	call(t, s, "simulation.step", StepParams{Duration: .01}, "charge")
	actual := s.Snapshot()
	// RK4 with 2 microsecond steps independently integrates the scalar RC ODE.
	ref, dt := 0., 2e-6
	slope := func(v float64) float64 { return referenceSiliconCurrent(5-v, 10000) / 1e-6 }
	for k := 0; k < 5000; k++ {
		a := slope(ref)
		b := slope(ref + dt*a/2)
		c := slope(ref + dt*b/2)
		d := slope(ref + dt*c)
		ref += dt * (a + 2*b + 2*c + d) / 6
	}
	got := actual.Mixed.Capacitors[0].Voltage
	if math.Abs(got-ref) > .003 {
		t.Fatalf("nonlinear RC got %g reference %g", got, ref)
	}
	call(t, s, "simulation.input.set", InputParams{Mode: "low"}, "low")
	call(t, s, "simulation.step", StepParams{Duration: .01}, "hold")
	after := s.Snapshot()
	if math.Abs(after.Mixed.Capacitors[0].Voltage-(got-1e-5)) > 2e-8 || math.Abs(after.Mixed.Diodes[0].Current+1e-9) > 1e-12 {
		t.Fatal("reverse-biased diode failed capacitor hold/leakage")
	}
	duration := .01
	_, record := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"i:D1", "v:C1"}}, "verify")
	if record.Metadata.Engine != NetworkEngineID || record.Metadata.DiodeModel == nil {
		t.Fatal("nonlinear RC verification failed")
	}
}
func TestSiliconDiodeDerivativeAndMonotonicity(t *testing.T) {
	previous := -math.MaxFloat64
	for _, v := range []float64{-12, -1, -.1, -.01, 0, .1, .5, .7, 1, 12} {
		i, g := siliconDiodeCurrent(v)
		if !finite(i) || !finite(g) || g < 0 || i < previous {
			t.Fatalf("nonfinite/nonmonotone diode at %g", v)
		}
		previous = i
		step := 1e-6
		right, _ := siliconDiodeCurrent(v + step)
		left, _ := siliconDiodeCurrent(v - step)
		derivative := (right - left) / (2 * step)
		if math.Abs(g-derivative) > 1e-8+1e-4*math.Abs(g) {
			t.Fatalf("diode derivative at %g: %g != %g", v, g, derivative)
		}
	}
}
func TestSiliconDiodePortableModelAndStrictImport(t *testing.T) {
	s := newGraphService(SolveLED, siliconSeriesGraph(t, false))
	defer s.Close()
	original := settled(t, s)
	verify(t, s, VerifyParams{}, "verify")
	path := filepath.Join(t.TempDir(), "diode.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	loaded := settled(t, fresh)
	if loaded.DiodeModel == nil || *loaded.DiodeModel != SiliconDiodeModel() || loaded.Document.RecordingCount != 1 || loaded.Mixed.Diodes[0].Current != original.Mixed.Diodes[0].Current {
		t.Fatal("silicon project/record changed")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err = json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	document["diode_model"].(map[string]any)["is_a"] = 1
	damaged, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(t.TempDir(), "bad.elab")
	if err = os.WriteFile(bad, damaged, 0600); err != nil {
		t.Fatal(err)
	}
	before := fresh.Snapshot()
	refuse(t, fresh, "project.load", LoadParams{Path: bad}, "invalid-document")
	if fresh.Snapshot().Document.Generation != before.Document.Generation {
		t.Fatal("failed diode import replaced project")
	}
}

func TestSiliconDiodeFiniteSeriesResistanceWithoutExternalResistor(t *testing.T) {
	g := siliconSeriesGraph(t, false)
	g.Components = append(g.Components[:1], g.Components[2:]...)
	g.Wires = append([]Wire{{ID: "VD", From: "V1.positive", To: "D1.anode", Connected: true}}, g.Wires[2:]...)
	s := newGraphService(SolveLED, g)
	defer s.Close()
	state := settled(t, s)
	if state.Mixed == nil || math.Abs(state.Mixed.Diodes[0].Current-referenceSiliconCurrent(5, 0)) > 1e-9 {
		t.Fatalf("internal diode series resistance: %+v", state.Diagnostics)
	}
}

func TestDiodeAndSwitchCSVRetainsCapturedModelAndValues(t *testing.T) {
	g := switchLEDGraph(t, true)
	diode, err := MakeComponent("D1", "diode", Point{200, 0}, "")
	if err != nil {
		t.Fatal(err)
	}
	g.Components[2] = diode
	s := newGraphService(SolveLED, g)
	defer s.Close()
	settled(t, s)
	_, record := verify(t, s, VerifyParams{ProbeIDs: []string{"i:D1"}}, "verify")
	call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: false}, "open")
	path := filepath.Join(t.TempDir(), "diode-switch.csv")
	call(t, s, "verification.export", VerificationExportParams{ID: record.Operation.ID, Path: path, Format: "csv"}, "export")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil || len(rows) != 2 {
		t.Fatalf("CSV rows: %v", err)
	}
	values := map[string]string{}
	for i, name := range rows[0] {
		values[name] = rows[1][i]
	}
	current, err := strconv.ParseFloat(values["i:D1 [A]"], 64)
	if err != nil || current != record.Metadata.Mixed.Diodes[0].Current || values["diode_model"] != DiodeModelID || values["switch_model"] != SwitchModelID {
		t.Fatal("CSV omitted captured models or substituted the live open-switch result")
	}
}
