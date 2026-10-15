package workbench

import (
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func bufferedLEDGraph() Graph {
	g := ExampleGraph()
	c, _ := MakeComponent("C1", "capacitor", Point{X: 360}, "100uF")
	g.Components = append(g.Components, c)
	g.Wires = append(g.Wires, Wire{ID: "W5", From: "C1.1", To: "D1.anode", Connected: true}, Wire{ID: "W6", From: "C1.2", To: "GND.reference", Connected: true})
	return g
}

func highPassRCGraph() Graph {
	g := RCExampleGraph()
	g.Components[0].Kind = "dc-source"
	g.Components[0].Model = "ideal-dc.v1"
	g.Wires = []Wire{{ID: "W1", From: "V1.positive", To: "C1.1", Connected: true}, {ID: "W2", From: "C1.2", To: "R1.1", Connected: true}, {ID: "W3", From: "R1.2", To: "GND.reference", Connected: true}, {ID: "W4", From: "V1.negative", To: "GND.reference", Connected: true}}
	return g
}

func TestGeneralRCNetworksDoNotFallIntoLegacyTopologyRejection(t *testing.T) {
	for name, graph := range map[string]Graph{"capacitor-buffered LED": bufferedLEDGraph(), "series capacitor high-pass": highPassRCGraph()} {
		t.Run(name, func(t *testing.T) {
			compiled, status, diagnostic := graph.Compile()
			if status != "" || diagnostic != nil || compiled.Analysis != NetworkTransientAnalysis {
				t.Fatalf("supported network rejected by dispatch: %s, %+v, %s", status, diagnostic, compiled.Analysis)
			}
		})
	}
}

// Independent scalar capacitor ODE and inverse current-domain LED equation;
// this does not invoke ELab's nodal Newton or backward-Euler implementation.
func fadeReferenceCurrent(voltage float64) float64 {
	if voltage <= 0 {
		return 0
	}
	nv := 2 * 1.380649e-23 * 300.15 / 1.602176634e-19
	lo, hi := 0., voltage
	for i := 0; i < 70; i++ {
		current := (lo + hi) / 2
		if nv*math.Log1p(current/1e-20)+current > voltage {
			hi = current
		} else {
			lo = current
		}
	}
	return (lo + hi) / 2
}

func TestBufferedLEDChargeAgainstIndependentODEAndConservation(t *testing.T) {
	s := newGraphService(SolveLED, bufferedLEDGraph())
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause-led-cap")
	call(t, s, "simulation.restart", struct{}{}, "restart-led-cap")
	initial := settled(t, s)
	if initial.Mixed == nil || initial.Mixed.Capacitors[0].Voltage != 0 || initial.Mixed.LEDs[0].Current > 1e-12 {
		t.Fatal("capacitor-buffered LED startup", initial.Diagnostics)
	}
	reference := 0.
	slope := func(voltage float64) float64 { return ((5-voltage)/1000 - fadeReferenceCurrent(voltage)) / 100e-6 }
	for sample := 1; sample <= 10; sample++ {
		const dt = 20e-6
		for i := 0; i < 500; i++ {
			a := slope(reference)
			b := slope(reference + dt*a/2)
			c := slope(reference + dt*b/2)
			d := slope(reference + dt*c)
			reference += dt * (a + 2*b + 2*c + d) / 6
		}
		call(t, s, "simulation.step", StepParams{Duration: .01}, fmt.Sprintf("led-cap-step-%d", sample))
		state := s.Snapshot()
		got := state.Mixed.Capacitors[0].Voltage
		if math.Abs(state.Transient.Sample.Time-float64(sample)*.01) > 1e-12 || math.Abs(got-reference) > .003 || math.Abs(state.Mixed.LEDs[0].Current-fadeReferenceCurrent(got)) > 1e-9 || state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
			t.Fatalf("LED capacitor at %g: MNA %g, scalar %g, KCL/power %g/%g", state.Transient.Sample.Time, got, reference, state.Mixed.KCLResidual, state.Mixed.PowerResidual)
		}
	}
	before := s.Snapshot()
	duration := .1
	_, capture := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1", "i:D1"}}, "verify-led-cap")
	if math.Abs(capture.Metadata.Mixed.Capacitors[0].Voltage-before.Mixed.Capacitors[0].Voltage) > 1e-9 || !reflect.DeepEqual(before.Transient, s.Snapshot().Transient) {
		t.Fatal("Verify disagrees or changes live capacitor state")
	}
}

func TestHighPassRCAnalyticResponseAndLegacyLowPassPreserved(t *testing.T) {
	legacy, status, diagnostic := RCExampleGraph().Compile()
	if status != "" || diagnostic != nil || legacy.Analysis != "RC transient" {
		t.Fatal("legacy exact low-pass path changed")
	}
	s := newGraphService(SolveLED, highPassRCGraph())
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause-highpass")
	call(t, s, "simulation.restart", struct{}{}, "restart-highpass")
	initial := settled(t, s)
	if initial.Mixed == nil || initial.Mixed.Capacitors[0].Voltage != 0 || math.Abs(initial.Mixed.Resistors[0].Voltage-5) > 1e-12 {
		t.Fatal("high-pass startup constraint", initial.Diagnostics)
	}
	for i := 1; i <= 5; i++ {
		call(t, s, "simulation.step", StepParams{Duration: .01}, fmt.Sprintf("highpass-step-%d", i))
		state := s.Snapshot()
		want := 5 * math.Exp(-state.Transient.Sample.Time/.01)
		if math.Abs(state.Transient.Sample.Time-float64(i)*.01) > 1e-12 || math.Abs(state.Mixed.Resistors[0].Voltage-want) > .003 || math.Abs(state.Mixed.Capacitors[0].Voltage-(5-want)) > .003 || state.Mixed.KCLResidual > 1e-9 {
			t.Fatal("high-pass exponential reference", state.Mixed)
		}
	}
}

func TestLEDFadeExamplePulseContinuityTailAndPersistence(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "led-fade"}, "load-fade")
	call(t, s, "simulation.pause", struct{}{}, "pause-fade")
	call(t, s, "simulation.restart", struct{}{}, "restart-fade")
	initial := settled(t, s)
	if initial.Selected != "C1" || initial.Example.Modified || len(initial.Scope.Selected) != 3 {
		t.Fatal("fade example configuration")
	}
	call(t, s, "simulation.step", StepParams{Duration: .2}, "fade-edge")
	edge := s.Snapshot()

	if edge.Mixed.Capacitors[0].Voltage < 2 || edge.Mixed.LEDs[0].Current < .002 || edge.Transient.Sample.Input != 0 {
		t.Fatal("falling edge erased stored capacitor voltage/current", edge.Mixed)
	}
	before, after, found := 0., 0., false
	for _, trace := range edge.Scope.Traces {
		if trace.Probe.ID != "v:C1" {
			continue
		}
		for i, point := range trace.Points {
			if i > 0 && point.Time == trace.Points[i-1].Time && math.Abs(point.Time-.2) < 1e-12 {
				before, after, found = trace.Points[i-1].Value, point.Value, true
			}
		}
	}
	if !found || before != after {
		t.Fatal("pulse edge did not preserve capacitor voltage", before, after)
	}
	call(t, s, "simulation.step", StepParams{Duration: .01}, "fade-tail")
	tail := s.Snapshot()
	if tail.Mixed.LEDs[0].Current <= 1e-8 || tail.Mixed.LEDs[0].Current >= edge.Mixed.LEDs[0].Current || tail.Mixed.Capacitors[0].Voltage >= edge.Mixed.Capacitors[0].Voltage {
		t.Fatal("stored-charge LED tail absent", tail.Mixed)
	}
	path := filepath.Join(t.TempDir(), "fade.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save-fade")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "reopen-fade")
	loaded := settled(t, fresh)
	if loaded.Example.Info.ID != "led-fade" || loaded.Example.Modified || !reflect.DeepEqual(loaded.Components, initial.Components) || !reflect.DeepEqual(loaded.Wires, initial.Wires) || loaded.Transient.Sample.Time != 0 || loaded.Mixed.Capacitors[0].Voltage != 0 {
		t.Fatal("fade document did not reopen as fresh configured physics")
	}
}
