package workbench

import (
	"fmt"
	"math"
)

type inductorCutEdge struct {
	index int
	sign  float64
}

type inductorCut struct {
	row   int
	edges []inductorCutEdge
}

// At startup an ideal inductor imposes current, not voltage. A group whose
// only external current paths are inductors has a redundant summed KCL row.
// Differentiate that sum: Σ(sign * V/L) = 0 determines its voltage without
// adding a fictitious leakage resistor or advancing simulation time.
//
// Group all non-L current paths (including ideal source/C constraints). The
// insulated MOS gate is not a current path. Internal branch currents cancel
// in the group sum regardless of their nonlinear voltage dependence.
func (n *mixedNetwork) inductorStartupCuts() []inductorCut {
	if len(n.inductors) == 0 {
		return nil
	}
	parent := make([]int, len(n.nodes)+1) // Ground (-1) is shifted to zero.
	for i := range parent {
		parent[i] = i
	}
	var root func(int) int
	root = func(i int) int {
		if parent[i] != i {
			parent[i] = root(parent[i])
		}
		return parent[i]
	}
	join := func(p Component, one, two string) {
		a, b := root(n.pin(p, one)+1), root(n.pin(p, two)+1)
		parent[b] = a
	}
	for _, p := range n.c.MixedGraph.Components {
		switch p.Kind {
		case "optocoupler":
			join(p, "anode", "cathode")
			join(p, "collector", "emitter")
		case "linear-regulator", "ldo-regulator":
			join(p, "in", "gnd")
			join(p, "out", "gnd")
		case "relay":
			join(p, "common", "no")
			join(p, "common", "nc")
		case "comparator":
			for _, pin := range p.Terminals {
				if pin != "gnd" {
					join(p, pin, "gnd")
				}
			}
		case "bcd-decoder", "seven-segment":
			ground := "gnd"
			if p.Kind == "seven-segment" {
				ground = "common"
			}
			for _, pin := range p.Terminals {
				if pin != ground {
					join(p, pin, ground)
				}
			}
		case "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor":
			for _, pin := range p.Terminals {
				if pin != "gnd" {
					join(p, pin, "gnd")
				}
			}
		case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter":
			for _, pin := range p.Terminals {
				if pin != "gnd" {
					join(p, pin, "gnd")
				}
			}
		case "timer555":
			for _, pin := range timer555Pins[1:] {
				join(p, pin, "gnd")
			}
		case "thermistor", "ldr", "pushbutton", "bouncing-pushbutton", "resistor", "switch", "capacitor":
			join(p, "1", "2")
		case "potentiometer":
			join(p, "1", "wiper")
			join(p, "2", "wiper")
		case "battery", "dc-source", "pulse-source", "sine-source", "finite-pulse-source":
			join(p, "positive", "negative")
		case "led", "diode", "zener", "schottky":
			join(p, "anode", "cathode")
		case "npn", "pnp":
			join(p, "base", "emitter")
			join(p, "collector", "emitter")
		case "nmos", "pmos":
			join(p, "drain", "source")
		}
	}
	ground := root(0)
	groups := make(map[int]int)
	cuts := []inductorCut{}
	for i := range n.nodes {
		group := root(i + 1)
		if group == ground {
			continue
		}
		if _, exists := groups[group]; !exists {
			groups[group] = len(cuts)
			cuts = append(cuts, inductorCut{row: i})
		}
	}
	for k, p := range n.inductors {
		one, two := root(n.pin(p, "1")+1), root(n.pin(p, "2")+1)
		if one == two {
			continue
		}
		if i, ok := groups[one]; ok {
			cuts[i].edges = append(cuts[i].edges, inductorCutEdge{k, 1})
		}
		if i, ok := groups[two]; ok {
			cuts[i].edges = append(cuts[i].edges, inductorCutEdge{k, -1})
		}
	}
	// A fully floating group without any L path remains a normal rank error.
	result := cuts[:0]
	for _, cut := range cuts {
		if len(cut.edges) > 0 {
			result = append(result, cut)
		}
	}
	return result
}

func (n *mixedNetwork) checkInductorStartup(old []float64) error {
	for _, cut := range n.inductorCuts {
		current, magnitude := 0., 0.
		for _, edge := range cut.edges {
			i := old[len(n.caps)+edge.index]
			current += edge.sign * i
			magnitude += math.Abs(i)
		}
		if !finite(current) || math.Abs(current) > inductorKCLAbsolute+inductorKCLRelative*magnitude {
			return fmt.Errorf("incompatible inductor startup currents at %s; currents entering and leaving an inductor-only cut must balance", n.nodes[cut.row])
		}
	}
	return nil
}

func (n *mixedNetwork) stampInductorStartup(x []float64, a [][]float64, f []float64) {
	for _, cut := range n.inductorCuts {
		row := cut.row
		clear(a[row])
		f[row] = 0
		scale := 0.
		for _, edge := range cut.edges {
			scale += 1 / n.inductors[edge.index].Value
		}
		for _, edge := range cut.edges {
			p := n.inductors[edge.index]
			one, two := n.pin(p, "1"), n.pin(p, "2")
			weight := edge.sign / (p.Value * scale)
			f[row] += weight * (voltage(x, one) - voltage(x, two) - inductorWindingResistance(p)*x[n.inductorIndex(edge.index)])
			a[row][n.inductorIndex(edge.index)] -= weight * inductorWindingResistance(p)
			if one >= 0 {
				a[row][one] += weight
			}
			if two >= 0 {
				a[row][two] -= weight
			}
		}
	}
}
