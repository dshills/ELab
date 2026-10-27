package workbench

import (
	"context"
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func currentResistorGraph(t *testing.T, r float64) Graph {
	t.Helper()
	b := newExampleBuilder()
	b.part("I1", "current-source", "", -200, 0, 0)
	b.part("R1", "resistor", FormatValue(r, "Ω"), 200, 0, 0)
	b.wire("I1.positive", "R1.1")
	b.ground("GI", "I1.negative", -200, 160)
	b.ground("GR", "R1.2", 200, 160)
	return b.graph
}
func TestCurrentSourceNortonComplianceAndLoadDerivatives(t *testing.T) {
	g := currentResistorGraph(t, 100)
	p, _ := g.Component("I1")
	for _, v := range []float64{-5, 0, 5, 11.2, 11.5, 11.8, 12, 15} {
		i, d := currentDeviceLaw(p, v, 0)
		eps := 1e-6
		a, _ := currentDeviceLaw(p, v+eps, 0)
		b, _ := currentDeviceLaw(p, v-eps, 0)
		if !finite(i) || math.Abs(d-(a-b)/(2*eps)) > 1e-7 {
			t.Fatal("source derivative", v, i, d, (a-b)/(2*eps))
		}
	}
	dc, _, err := solveDCOperatingPoint(context.Background(), g, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := .01 / (1./100 + 1e-9)
	if math.Abs(dc.Sources[0].Voltage-want) > 1e-9 || math.Abs(dc.Sources[0].Current-want/100) > 1e-10 || math.Abs(dc.PowerResidual) > 1e-9 {
		t.Fatal("independent Norton", dc)
	}
	g = currentResistorGraph(t, 100000)
	dc, _, err = solveDCOperatingPoint(context.Background(), g, nil)
	if err != nil {
		t.Fatal(err)
	}
	if dc.Sources[0].Voltage <= 11 || dc.Sources[0].Voltage >= 12 || dc.Sources[0].Current >= .001 {
		t.Fatal("compliance", dc.Sources)
	}
	l, _ := MakeComponent("LOAD1", "electronic-load", Point{}, "")
	for _, mode := range []string{"cc", "cp"} {
		l.Device.Load.Mode = mode
		l.Device.Load.Level = .2
		l.Device.Load.MaximumCurrent = .1
		for _, v := range []float64{-1, 0, .1, .3, 1.2, 2.4, 5} {
			i, d := currentDeviceLaw(l, v, 0)
			eps := 1e-6
			a, _ := currentDeviceLaw(l, v+eps, 0)
			b, _ := currentDeviceLaw(l, v-eps, 0)
			if v != 0 && math.Abs(d-(a-b)/(2*eps)) > 1e-7 || i*v < 0 || math.Abs(i) > .100001 {
				t.Fatal("load derivative/passivity", mode, v, i, d, (a-b)/(2*eps))
			}
		}
	}
}
func TestElectronicLoadIndependentCCCPAndRCTransient(t *testing.T) {
	for _, power := range []bool{false, true} {
		g := ElectronicLoadExampleGraph(power)
		for k, p := range g.Components {
			if p.Kind == "electronic-load" {
				g.Components[k].Device.Load.Timing.Pulsed = false
			}
		}
		dc, _, err := solveDCOperatingPoint(context.Background(), g, dcSourceLevels(g))
		if err != nil {
			t.Fatal(err)
		}
		v := 5/(1+10e-9) - 10*.08/(1+10e-9)
		if power { // positive stable root of (1+R*bleed)V² - Vs V + R P = 0.
			v = (5 + math.Sqrt(25-4*(1+10e-9)*10*.25)) / (2 * (1 + 10e-9))
		}
		if len(dc.Loads) != 1 || math.Abs(dc.Loads[0].Voltage-v) > 1e-7 || math.Abs(dc.PowerResidual) > 1e-8 {
			t.Fatal("independent load root", power, dc.Loads, v, err)
		}
		if !power {
			state, tr := initializeLossRC(t, g)
			// CC folds linearly below 1 V, then follows the constant-current RC law.
			// Independent RK4 resolves the piecewise ODE on a 100 ns grid.
			ref := 0.
			at := 0.
			h := 1e-7
			f := func(v float64) float64 {
				load := .08 * math.Min(math.Max(v, 0), 1)
				return ((5-v)/10 - load - v*1e-9) / 10e-6
			}
			for sample := 1; sample <= 100; sample++ {
				end := float64(sample) * .00001
				for at < end-h/2 {
					k1 := f(ref)
					k2 := f(ref + h*k1/2)
					k3 := f(ref + h*k2/2)
					k4 := f(ref + h*k3)
					ref += h * (k1 + 2*k2 + 2*k3 + k4) / 6
					at += h
				}
				state, err = state.advance(tr, end)
				if err != nil {
					t.Fatal(err)
				}
				if math.Abs(state.result.Loads[0].Voltage-ref) > 2e-4 || math.Abs(state.result.PowerResidual) > 1e-7 {
					t.Fatal("independent load RC", sample, state.result.Loads[0], ref)
				}
			}
		}
	}
}
func TestCurrentPulseCornersSharedCaptureAndExamples(t *testing.T) {
	for _, name := range []string{"current-led", "load-step", "constant-power-load"} {
		t.Run(name, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: name}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			settled(t, s)
			duration, step := .11, .00025
			_, op := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: s.Snapshot().Scope.Selected}, "verify")
			s.mu.Lock()
			rec := s.operations[op.Operation.ID].record
			doc := VerificationDocument{Metadata: cloneVerificationMetadata(rec.metadata), Rows: append([]VerificationRow{}, rec.rows...)}
			s.mu.Unlock()
			raw, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = decodeRecording(raw); err != nil {
				t.Fatal("capture", err)
			}
			path := filepath.Join(t.TempDir(), "current.elab")
			call(t, s, "project.save", SaveParams{Path: path}, "save")
			fresh := New()
			defer fresh.Close()
			call(t, fresh, "project.load", LoadParams{Path: path}, "open")
			after := settled(t, fresh)
			if !reflect.DeepEqual(s.Snapshot().Components, after.Components) || after.Document.RecordingCount != 1 {
				t.Fatal("roundtrip")
			}
			svg := string(schematicSVG(s.graph, 1, 1))
			if !strings.Contains(svg, CurrentDeviceModelID) || !strings.Contains(svg, "device_components") {
				t.Fatal("model metadata")
			}
			if name == "current-led" {
				doc.Metadata.Mixed.Sources[0].Current += .01
			} else {
				doc.Metadata.Mixed.Loads[0].Current += .01
			}
			raw, _ = json.Marshal(doc)
			if _, err = decodeRecording(raw); err == nil {
				t.Fatal("forged current accepted")
			}
		})
	}
	g := CurrentLEDExampleGraph()
	c, _, d := g.Compile()
	if d != nil {
		t.Fatal(d)
	}
	n := newMixedNetwork(c)
	tr := &Transient{Mode: "low", Settings: DefaultTransientSettings()}
	if math.Abs(n.nextSourceCorner(tr, 0)-.001) > 1e-12 || n.maximumWaveformStep(tr, 0) > .0005 {
		t.Fatal("independent current pulse corners")
	}
	p, _ := g.Component("I1")
	if math.Abs(currentLevel(p, .02)-.015) > 1e-12 || math.Abs(currentLevel(p, .07)-.002) > 1e-12 {
		t.Fatal("pulse plateau")
	}
}
func TestCurrentConfigurationHistoryAndAdmission(t *testing.T) {
	s := newGraphService(SolveLED, currentResistorGraph(t, 100))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := s.Snapshot().Components
	c := DefaultCurrentConfig()
	c.Current = .02
	call(t, s, "current-source.configure", CurrentParams{ID: "I1", Parameters: c}, "set")
	call(t, s, "edit.undo", struct{}{}, "undo")
	if !reflect.DeepEqual(before, s.Snapshot().Components) {
		t.Fatal("undo")
	}
	c.Low = .1
	refuse(t, s, "current-source.configure", CurrentParams{ID: "I1", Parameters: c}, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot().Components) {
		t.Fatal("invalid changed graph")
	}
	raw := json.RawMessage(`{"id":"I1","parameters":{"version":"elab.current-devices.v1","current_a":0.01,"low_current_a":0,"compliance_v":12,"timing":{"pulsed":false,"high_s":0.05,"low_s":0.05,"rise_s":0.001}}}`)
	s.mu.Lock()
	err := s.applyLocked("current-source.configure", raw)
	s.mu.Unlock()
	if err == nil {
		t.Fatal("missing timing field accepted")
	}
}

func TestCurrentDevicesNoStoragePulsingAndCapacitorDrive(t *testing.T) {
	g := ElectronicLoadExampleGraph(false)
	parts := []Component{}
	for _, p := range g.Components {
		if p.ID != "C1" && p.ID != "GC" {
			parts = append(parts, p)
		}
	}
	g.Components = parts
	wires := []Wire{}
	for _, w := range g.Wires {
		if !strings.HasPrefix(w.From, "C1.") && !strings.HasPrefix(w.To, "C1.") && !strings.HasPrefix(w.From, "GC.") && !strings.HasPrefix(w.To, "GC.") {
			wires = append(wires, w)
		}
	}
	g.Wires = wires
	c, _, d := g.Compile()
	if d != nil || !IsNPNTransientAnalysis(c.Analysis) {
		t.Fatal("storage-free pulsed load classification", c.Analysis, d)
	}
	state, tr := initializeLossRC(t, g)
	var err error
	state, err = state.advance(tr, .02)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(state.result.Loads[0].Current-.08) > 1e-7 {
		t.Fatal("storage-free pulse", state.result.Loads)
	}
	// A current source imposes no redundant voltage constraint across storage.
	g = currentResistorGraph(t, 1000)
	cap, _ := MakeComponent("C1", "capacitor", Point{400, 0}, "1u")
	g.Components = append(g.Components, cap)
	g.Wires = append(g.Wires, Wire{Connected: true, ID: "WC1", From: "I1.positive", To: "C1.1"}, Wire{Connected: true, ID: "WC2", From: "I1.negative", To: "C1.2"})
	state, tr = initializeLossRC(t, g)
	for k := 1; k <= 20; k++ {
		at := float64(k) * .0001
		state, err = state.advance(tr, at)
		if err != nil {
			t.Fatal(err)
		}
		target := .01 / (.001 + 1e-9)
		want := target * (1 - math.Exp(-at*(.001+1e-9)/1e-6))
		if math.Abs(state.result.Capacitors[0].Voltage-want) > 1e-4 {
			t.Fatal("current driven capacitor", state.result.Capacitors, want)
		}
	}
}

func TestCurrentDevicesPreserveCombinedSourceBudgetAndInitialReadout(t *testing.T) {
	g := currentResistorGraph(t, 1000)
	for k := 2; k <= MaxNetworkSources+1; k++ {
		p, _ := MakeComponent("I"+string(rune('0'+k)), "current-source", Point{X: float64(k) * 100}, "")
		g.Components = append(g.Components, p)
		g.Wires = append(g.Wires, Wire{ID: "WP" + p.ID, From: p.ID + ".positive", To: "I1.positive", Connected: true}, Wire{ID: "WN" + p.ID, From: p.ID + ".negative", To: "I1.negative", Connected: true})
	}
	_, _, d := g.Compile()
	if d == nil || d.Code != "mixed-limit" {
		t.Fatal("combined current/voltage source budget", d)
	}
	s := newGraphService(SolveLED, CurrentLEDExampleGraph())
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause-initial")
	call(t, s, "simulation.restart", struct{}{}, "restart-initial")
	state := settled(t, s)
	if state.Transient == nil || state.Mixed == nil {
		t.Fatal("missing initial current transient")
	}
	if math.Abs(state.Transient.Sample.Input-state.Mixed.Sources[0].Voltage) > 1e-10 {
		t.Fatal("initial current-source voltage trace")
	}
	if len(state.LiveInputModes) != 0 {
		t.Fatal("current-only circuit offers voltage overrides")
	}
}

func TestCurrentDevicePrimaryVoltageSelectionAndMissingPresentation(t *testing.T) {
	g := ElectronicLoadExampleGraph(false)
	p, _ := MakeComponent("I1", "current-source", Point{}, "")
	g.Components = append([]Component{p}, g.Components...)
	g.Wires = append(g.Wires, Wire{ID: "ICPOS", From: "I1.positive", To: "LOAD1.positive", Connected: true}, Wire{ID: "ICNEG", From: "I1.negative", To: "V1.negative", Connected: true})
	c, _, d := g.Compile()
	if d != nil || c.Source.ID != "V1" {
		t.Fatal("mixed primary voltage", c.Source.ID, d)
	}
	for _, kind := range []string{"current-source", "electronic-load"} {
		p := Component{Kind: kind, ID: "BROKEN"}
		if currentDeviceName(p) == "" {
			t.Fatal("missing config presentation")
		}
		_ = configuredSourceAnnotations(Graph{Components: []Component{p}}, DefaultTransientSettings())
		_ = sourceSamplingAdvice(Graph{Components: []Component{p}}, DefaultTransientSettings())
	}
}
