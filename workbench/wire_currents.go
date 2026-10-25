package workbench

import "math"

// Device injections use conventional current into each terminal. Ideal-wire
// cycles are underdetermined; only graph bridges have a unique cut current.
// Net-label aliases are virtual edges, so labels never fabricate physical flow.
func uniqueWireCurrents(g Graph, c readingContext) map[string]float64 {
	result := map[string]float64{}
	if c.stale || c.sample.frame == nil {
		return result
	}
	injection := map[string]float64{}
	known := map[string]bool{}
	for _, p := range g.Components {
		for _, pin := range p.Terminals {
			ref := p.ID + "." + pin
			injection[ref] = 0
			known[ref] = true
		}
		set := func(pin, id string, sign float64) {
			ref := p.ID + "." + pin
			value, ok := c.sample.frame.values["i:"+id]
			injection[ref] = sign * value
			known[ref] = ok
		}
		switch p.Kind {
		case "shift-register":
			for _, pin := range shiftPins {
				set(pin, p.ID+"."+pin, 1)
			}
		case "piezo", "active-buzzer":
			set("positive", p.ID, 1)
			set("negative", p.ID, -1)
		case "pwm-controller":
			for _, pin := range pwmPins {
				set(pin, p.ID+"."+pin, 1)
			}
		case "hbridge":
			for _, pin := range hbridgePins {
				set(pin, p.ID+"."+pin, 1)
			}
		case "transformer":
			for _, pin := range transformerPins {
				set(pin, p.ID+"."+pin, 1)
			}
		case "optocoupler":
			for _, pin := range optoPins {
				set(pin, p.ID+"."+pin, 1)
			}
		case "dc-motor":
			set("positive", p.ID, 1)
			set("negative", p.ID, -1)
		case "battery", "dc-source", "pulse-source", "sine-source", "finite-pulse-source":
			set("positive", p.ID, -1)
			set("negative", p.ID, 1)
		case "nmos", "pmos":
			set("gate", p.ID+".gate", 1)
			set("drain", p.ID+".drain", 1)
			set("source", p.ID+".source", -1)
		case "npn", "pnp":
			set("base", p.ID+".base", 1)
			set("collector", p.ID+".collector", 1)
			set("emitter", p.ID+".emitter", -1)
		case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter":
			for _, pin := range p.Terminals {
				sign := 1.
				if sequentialOutput(pin) {
					sign = -1
				}
				set(pin, p.ID+"."+pin, sign)
			}
		case "bcd-decoder", "seven-segment":
			for _, pin := range p.Terminals {
				sign := 1.
				if p.Kind == "bcd-decoder" && len(pin) == 1 {
					sign = -1
				}
				set(pin, p.ID+"."+pin, sign)
			}
		case "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor", "timer555":
			for _, pin := range p.Terminals {
				sign := 1.
				if pin == "out" {
					sign = -1
				}
				set(pin, p.ID+"."+pin, sign)
			}
		case "linear-regulator", "ldo-regulator":
			for _, pin := range p.Terminals {
				sign := 1.
				if pin == "out" {
					sign = -1
				}
				set(pin, p.ID+"."+pin, sign)
			}
		case "relay":
			for _, pin := range p.Terminals {
				set(pin, p.ID+"."+pin, 1)
			}
		case "comparator":
			for _, pin := range p.Terminals {
				sign := 1.
				if pin == "out" {
					sign = -1
				}
				set(pin, p.ID+"."+pin, sign)
			}
		case "opamp":
			for _, pin := range p.Terminals {
				sign := 1.
				if pin == "out" {
					sign = -1
				}
				set(pin, p.ID+"."+pin, sign)
			}
		case "potentiometer":
			for _, pin := range p.Terminals {
				set(pin, p.ID+"."+pin, 1)
			}
		case "thermistor", "ldr", "pushbutton", "bouncing-pushbutton", "resistor", "capacitor", "inductor", "switch":
			set("1", p.ID, 1)
			set("2", p.ID, -1)
		case "led", "diode", "zener", "schottky":
			set("anode", p.ID, 1)
			set("cathode", p.ID, -1)
		case "ground", "junction", "net-label":
		default:
			for _, pin := range p.Terminals {
				known[p.ID+"."+pin] = false
			}
		}
	}
	type edge struct{ from, to, id string }
	edges := []edge{}
	for _, w := range g.Wires {
		if w.Connected {
			edges = append(edges, edge{w.From, w.To, w.ID})
		}
	}
	labels := map[string]string{}
	for _, p := range g.Components {
		if p.Kind == "net-label" {
			ref := p.ID + ".node"
			if first, ok := labels[p.Label]; ok {
				edges = append(edges, edge{first, ref, ""})
			} else {
				labels[p.Label] = ref
			}
		}
	}
	type neighbor struct {
		to   string
		edge int
	}
	adj := map[string][]neighbor{}
	for i, e := range edges {
		adj[e.from] = append(adj[e.from], neighbor{e.to, i})
		adj[e.to] = append(adj[e.to], neighbor{e.from, i})
	}
	discovered, low := map[string]int{}, map[string]int{}
	clock := 0
	var visit func(string, int, map[string]float64) (float64, bool)
	visit = func(at string, parent int, candidates map[string]float64) (float64, bool) {
		clock++
		discovered[at] = clock
		low[at] = clock
		sum, valid := injection[at], known[at]
		for _, n := range adj[at] {
			if n.edge == parent {
				continue
			}
			if discovered[n.to] != 0 {
				low[at] = min(low[at], discovered[n.to])
				continue
			}
			child, ok := visit(n.to, n.edge, candidates)
			sum += child
			valid = valid && ok
			low[at] = min(low[at], low[n.to])
			if low[n.to] > discovered[at] && edges[n.edge].id != "" {
				// A subtree consumes child amperes. Flow from parent into that
				// subtree is child; convert to the wire's declared From → To.
				flow := child
				if edges[n.edge].from == n.to {
					flow = -flow
				}
				candidates[edges[n.edge].id] = flow
			}
		}
		return sum, valid
	}
	// Component order fixes traversal/roundoff so identical frames give identical
	// arrows. A missing/unbalanced injection refuses that entire wire group.
	for _, p := range g.Components {
		for _, pin := range p.Terminals {
			ref := p.ID + "." + pin
			if discovered[ref] != 0 {
				continue
			}
			candidates := map[string]float64{}
			sum, valid := visit(ref, -1, candidates)
			if valid && math.Abs(sum) <= 1e-8 {
				for id, current := range candidates {
					result[id] = current
				}
			}
		}
	}
	return result
}
