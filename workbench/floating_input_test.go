package workbench

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

func floatingRCInputGraph(t *testing.T, returned, switched bool) Graph {
	t.Helper()
	g := RCExampleGraph()
	g.Components[0].Kind, g.Components[0].Model = "dc-source", "ideal-dc.v1"
	g.Wires = []Wire{{"W1", "V1.positive", "R1.1", true}, {"W2", "R1.2", "C1.1", true}, {"W3", "C1.2", "V1.negative", true}, {"W4", "C1.1", "GND.reference", true}}
	if returned {
		g.Wires = []Wire{{"W1", "V1.positive", "C1.1", true}, {"W2", "C1.2", "GND.reference", true}, {"W3", "V1.negative", "R1.1", true}, {"W4", "R1.2", "GND.reference", true}}
	}
	if switched {
		part, err := MakeComponent("S1", "switch", Point{X: -100, Y: -60}, "")
		if err != nil {
			t.Fatal(err)
		}
		closed := true
		part.SwitchClosed = &closed
		g.Components = append(g.Components, part)
		g.Wires[0].From = "S1.2"
		g.Wires = append(g.Wires, Wire{"WS", "V1.positive", "S1.1", true})
	}
	return g
}

func TestFloatingRCInputReportsSourceDifferenceAcrossStepsAndEvents(t *testing.T) {
	for _, returned := range []bool{false, true} {
		for _, switched := range []bool{false, true} {
			t.Run(fmt.Sprintf("return-resistor=%t/switch=%t", returned, switched), func(t *testing.T) {
				s := newGraphService(SolveLED, floatingRCInputGraph(t, returned, switched))
				defer s.Close()
				call(t, s, "simulation.pause", struct{}{}, "pause")
				call(t, s, "simulation.restart", struct{}{}, "restart")
				settled(t, s)
				call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"v:V1", "v:V1.positive", "v:C1"}}, "probes")
				before := s.Snapshot()
				call(t, s, "simulation.step", StepParams{Duration: .02}, "charge")
				charged := s.Snapshot()
				resistance := 10000.
				if switched {
					resistance++ // Existing closed-switch model is one ohm.
				}
				wantCap := 5 * -math.Expm1(-.02/(resistance*1e-6))
				if math.Abs(charged.Mixed.Capacitors[0].Voltage-wantCap) > .003 {
					t.Fatal("floating reference changed RC charging law")
				}
				if charged.RunID != before.RunID || charged.Document != before.Document || charged.ElectricalRevision != before.ElectricalRevision || !reflect.DeepEqual(charged.UndoEntry, before.UndoEntry) {
					t.Fatal("stepping changed circuit identity/history")
				}
				assertFloatingInputRows(t, charged, 5)
				// The positive terminal genuinely differs from source voltage.
				positive := charged.Scope.Traces[1].Points
				if math.Abs(positive[len(positive)-1].Value-5) < .1 {
					t.Fatal("fixture does not distinguish terminal and differential voltage")
				}
				if switched {
					call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: false}, "open")
					call(t, s, "simulation.step", StepParams{Duration: .001}, "accept-switch")
					opened := s.Snapshot()
					assertFloatingInputRows(t, opened, 5)
					if opened.SwitchStates[0].Pending || *opened.SwitchStates[0].AppliedClosed {
						t.Fatal("switch event did not apply")
					}
				}
				at := s.Snapshot().Transient.Sample.Time
				old := s.Snapshot()
				call(t, s, "simulation.input.set", InputParams{Mode: "low"}, "queue-off")
				call(t, s, "simulation.step", StepParams{Duration: .001}, "power-off")
				off := s.Snapshot()
				// Duplicate event rows retain the old left-hand value at 'at'.
				for i := range old.Transient.Samples {
					if old.Transient.Samples[i] != off.Transient.Samples[i] {
						t.Fatal("power-off rewrote accepted history")
					}
				}
				for _, sample := range off.Transient.Samples[len(old.Transient.Samples):] {
					if sample.Time < at || math.Abs(sample.Input) > 1e-12 {
						t.Errorf("off source reports positive terminal: t=%g input=%g; want 0 V", sample.Time, sample.Input)
					}
				}
			})
		}
	}
}

func assertFloatingInputRows(t *testing.T, state Snapshot, expected float64) {
	t.Helper()
	if len(state.Scope.Traces[0].Points) != len(state.Transient.Samples) {
		t.Fatal("scope/source sample counts differ")
	}
	for i, sample := range state.Transient.Samples {
		want := expected
		if math.Abs(state.Scope.Traces[0].Points[i].Value-want) > 1e-12 {
			t.Fatal("source-difference probe is wrong before scalar comparison")
		}
		if math.Abs(sample.Input-want) > 1e-12 {
			t.Errorf("t=%g input=%g; want source difference %g V", sample.Time, sample.Input, want)
			break
		}
	}
}
