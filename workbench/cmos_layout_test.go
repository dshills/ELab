package workbench

import (
	"math"
	"testing"
)

func TestCMOSInputAndOutputRoutesDoNotShareVisibleSegments(t *testing.T) {
	g := CMOSExampleGraph()
	c, _, diagnostic := g.Compile()
	if diagnostic != nil {
		t.Fatal(diagnostic)
	}
	input, output := c.TerminalNet["MN.gate"], c.TerminalNet["MN.drain"]
	for _, a := range g.Wires {
		if c.TerminalNet[a.From] != input {
			continue
		}
		for _, b := range g.Wires {
			if c.TerminalNet[b.From] != output {
				continue
			}
			pa, pb := WirePath(g, a), WirePath(g, b)
			for i := 1; i < len(pa); i++ {
				for j := 1; j < len(pb); j++ {
					x, y, u, v := pa[i-1], pa[i], pb[j-1], pb[j]
					// Orthogonal crossings are allowed; a shared length would
					// depict a connection absent from the electrical graph.
					horizontal := x.Y == y.Y && u.Y == v.Y && x.Y == u.Y && math.Min(math.Max(x.X, y.X), math.Max(u.X, v.X))-math.Max(math.Min(x.X, y.X), math.Min(u.X, v.X)) > 1e-9
					vertical := x.X == y.X && u.X == v.X && x.X == u.X && math.Min(math.Max(x.Y, y.Y), math.Max(u.Y, v.Y))-math.Max(math.Min(x.Y, y.Y), math.Min(u.Y, v.Y)) > 1e-9
					if horizontal || vertical {
						t.Fatalf("input %s and output %s visually share a segment: %v–%v / %v–%v", a.ID, b.ID, x, y, u, v)
					}
				}
			}
		}
	}
}
