package workbench

import (
	"fmt"
	"math"
	"testing"
)

func TestInductorFastLCEndpointRetainsShortFinalInterval(t *testing.T) {
	const inductance, capacitance, initial = 1e-6, 1e-9, .9
	for _, intervals := range []int{1, 317} {
		t.Run(fmt.Sprint(intervals), func(t *testing.T) {
			capValue, current := capacitance, initial
			g := inductorTestGraph(t, 0, 100, inductance, &capValue)
			g.Wires[2].To = "GND1.reference"
			g.Wires[3].To = "GND1.reference"
			g.Wires = append(g.Wires, Wire{ID: "WL", From: "L1.1", To: "C1.1", Connected: true})
			g.Components[2].InitialCurrent = &current
			state, trace := initializeInductorTest(t, g)
			// The second target is above the public 1 µs sample floor. Both
			// leave a real 50 fs remainder after whole LC-ceiling intervals.
			end := float64(intervals)*state.net.maximumInductorStep() + 5e-14
			next, err := state.advance(trace, end)
			if err != nil {
				t.Fatal(err)
			}
			phase := end / math.Sqrt(inductance*capacitance)
			wantV := -initial * math.Sqrt(inductance/capacitance) * math.Sin(phase)
			wantI := initial * math.Cos(phase)
			v, i := next.result.Capacitors[0].Voltage, next.result.Inductors[0].Current
			// Preserve the declared C-network absolute/relative accuracy;
			// checking only the reported timestamp would miss lost physics.
			if math.Abs(v-wantV) > 5e-8+5e-7*math.Abs(wantV) || math.Abs(i-wantI) > 1e-10+5e-7*math.Abs(wantI) {
				t.Fatalf("endpoint %.17g s: voltage error %.9g V, current error %.9g A", end, v-wantV, i-wantI)
			}
			t.Logf("endpoint %.17g s: voltage error %.9g V, current error %.9g A", end, v-wantV, i-wantI)
		})
	}
}

func TestInductorPulseCornerRetainsDistinctOutputTime(t *testing.T) {
	g := inductorTestGraph(t, 5, 100, 1e-6, nil)
	g.Components[0], _ = MakeComponent("V1", "pulse-source", g.Components[0].Position, "5 V")
	state, trace := initializeInductorTest(t, g)
	// A zero-current RL state is exact throughout the preceding low phase.
	// Start just before its next rising edge rather than simulate 100 ms of
	// settled history merely to isolate the source/output boundary policy.
	const corner = .1
	var err error
	state, err = state.net.solve(state.x, state.capV, 0, trace, corner-1e-9)
	if err != nil || state.result.Inductors[0].Current != 0 {
		t.Fatal("inconsistent low-phase fixture", err)
	}
	end := corner + 5e-14
	next, recorded, err := advanceMixed(state, trace, end)
	if err != nil {
		t.Fatal(err)
	}
	want := .05 * -math.Expm1(-(end-corner)/1e-8)
	if difference := next.result.Inductors[0].Current - want; math.Abs(difference) > 5e-9 {
		t.Fatalf("distinct post-edge interval lost: current error %.9g A at %.17g s", difference, end)
	}
	if len(recorded.Events) != 1 || recorded.Events[0].Time != corner {
		t.Fatal("source event moved to the later output time", recorded.Events)
	}
	t.Logf("distinct 50 fs post-edge interval: current error %.9g A; event %.17g s", next.result.Inductors[0].Current-want, recorded.Events[0].Time)
}
