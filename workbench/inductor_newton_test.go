package workbench

import (
	"math"
	"testing"
)

func TestInductorNewtonLimitsJunctionExcursionsWithoutRampingLinearNodes(t *testing.T) {
	state, _ := initializeInductorTest(t, FlybackExampleGraph())
	n := state.net
	var mos Component
	for _, p := range n.c.MixedGraph.Components {
		if p.ID == "M1" {
			mos = p
		}
	}
	drain, gate := n.pin(mos, "drain"), n.pin(mos, "gate")
	for _, stages := range []int{1, 3} {
		for _, sample := range []struct {
			name                       string
			drain, drainMove, gateMove float64
			want                       float64
		}{
			{"reverse diode toward conduction", .1, 5, 0, 1},
			{"clamp diode forward excursion", .1, 10, 0, .51},
			{"gate-source increment", .1, 0, 3, .2 / 3},
			{"body diode forward excursion", 5, -10, 0, .52},
		} {
			x, delta := make([]float64, stages*len(state.x)), make([]float64, stages*len(state.x))
			for k := range stages {
				copy(x[k*len(state.x):], state.x)
			}
			// The last stage alone must constrain a multi-stage proposal.
			offset := (stages - 1) * len(state.x)
			x[offset+drain] = sample.drain
			delta[offset+drain], delta[offset+gate] = sample.drainMove, sample.gateMove
			if scale := n.inductorNewtonScale(x, delta, stages); math.Abs(scale-sample.want) > 1e-12 {
				t.Fatal(sample.name, stages, scale, sample.want)
			}
		}
	}
}
