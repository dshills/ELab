package workbench

func compileNetwork(out Compiled, g Graph, groups map[string][]string, root func(string) string) (Compiled, string, *Diagnostic) {
	out.Analysis = MixedAnalysis
	fail := func(status, code, msg, id string) (Compiled, string, *Diagnostic) {
		return out, status, &Diagnostic{code, msg, id}
	}
	inductors := []Component{}
	mos := []Component{}
	opamps := []Component{}
	timers := []Component{}
	sources, caps, qs, rs, leds, grounds, switches, diodes, pots := []Component{}, []Component{}, []Component{}, []Component{}, []Component{}, []Component{}, []Component{}, []Component{}, []Component{}
	for _, p := range g.Components {
		switch p.Kind {
		case "dc-source", "pulse-source", "sine-source", "finite-pulse-source":
			sources = append(sources, p)
		case "capacitor":
			caps = append(caps, p)
		case "inductor":
			inductors = append(inductors, p)
		case "nmos", "pmos":
			mos = append(mos, p)
		case "timer555":
			timers = append(timers, p)
		case "opamp":
			opamps = append(opamps, p)
		case "npn", "pnp":
			qs = append(qs, p)
		case "switch":
			switches = append(switches, p)
		case "resistor":
			rs = append(rs, p)
		case "led":
			leds = append(leds, p)
		case "diode", "zener", "schottky":
			diodes = append(diodes, p)
		case "potentiometer":
			pots = append(pots, p)
		case "junction", "net-label":
			continue
		case "ground":
			grounds = append(grounds, p)
		default:
			return fail("unsupported", "unsupported-model", "The network solver supports sources, resistors, SPST switches, potentiometers, capacitors, inductors, diodes, red LEDs, NPN/PNP transistors, N/P-channel MOSFETs generic powered op-amps and 555 timers.", p.ID)
		}
	}
	generalBipolar := hasBipolarPolarity(qs, "pnp")
	if len(grounds) > 0 {
		for _, q := range qs {
			generalBipolar = generalBipolar || out.TerminalNet[q.ID+".emitter"] != out.TerminalNet[grounds[0].ID+".reference"]
		}
	}
	if len(opamps) > 0 || len(mos) > 0 || len(pots) > 0 || len(diodes) > 0 || len(switches) > 0 || len(caps) == 0 || len(qs) == 0 || generalBipolar {
		out.Analysis = NetworkDCAnalysis
		if len(caps) > 0 {
			out.Analysis = NetworkTransientAnalysis
		} else {
			for _, source := range sources {
				if source.Kind == "pulse-source" || IsWaveformSource(source.Kind) {
					out.Analysis = NetworkTransientAnalysis
				}
			}
		}
	}
	for _, source := range sources {
		if IsWaveformSource(source.Kind) {
			out.Analysis = WaveformAnalysis
			break
		}
	}

	if len(inductors) > 0 {
		out.Analysis = InductorAnalysis
	}
	if len(timers) > 0 {
		out.Analysis = Timer555Analysis
	}
	if len(timers) > MaxNetworkTimers555 {
		return fail("unsupported", "mixed-limit", "Network simulation supports at most two 555 timers.", "")
	}
	if len(opamps) > MaxNetworkOpAmps {
		return fail("unsupported", "mixed-limit", "Network simulation supports at most two generic powered op-amps within the existing total part, node and source limits.", "")
	}
	if len(inductors) > MaxNetworkInductors || len(mos) > MaxNetworkMOSFETs || SimulatedComponentCount(g) > MaxSimulatedComponents || len(pots) > MaxNetworkPotentiometers || len(diodes) > MaxNetworkDiodes || len(switches) > MaxNetworkSwitches || len(sources) > MaxNetworkSources || len(caps) > MaxNetworkCapacitors || len(qs) > MaxNetworkTransistors || len(rs) > MaxNetworkResistors || len(leds) > MaxNetworkLEDs || len(out.Nets) > MaxElectricalNodes {
		return fail("unsupported", "mixed-limit", networkLimitMessage(), "")
	}
	if len(grounds) == 0 || len(sources) == 0 || len(rs)+len(switches)+len(diodes)+len(pots)+len(opamps)+len(timers) == 0 {
		return fail("incomplete", "missing-parts", "Add ground, a source and a resistor, switch, diode, potentiometer, op-amp or 555 timer to complete the network.", "")
	}
	net := func(p Component, t string) string { return out.TerminalNet[p.ID+"."+t] }
	out.Ground = net(grounds[0], "reference")
	adjacency := map[string][]string{}
	for _, p := range g.Components {
		if p.Kind == "ground" || IsConnectivityPrimitive(p.Kind) {
			continue
		}
		if p.Kind == "potentiometer" {
			connected := 0
			unique := map[string]bool{}
			for _, pin := range p.Terminals {
				if len(physicalTerminals(g, groups[root(p.ID+"."+pin)])) >= 2 {
					connected++
				}
				unique[net(p, pin)] = true
			}
			if connected < 2 {
				return fail("incomplete", "open-terminal", "Connect at least two potentiometer pins for a divider or rheostat.", p.ID)
			}
			if len(unique) < 2 {
				return fail("error", "shorted-component", "Remove the short across all potentiometer pins.", p.ID)
			}
		}

		if p.Kind == "timer555" && net(p, "vcc") == net(p, "gnd") {
			return fail("error", "shorted-component", "Remove the short across 555 supply pins.", p.ID)
		}
		for i, a := range p.Terminals {
			ref := p.ID + "." + a
			if p.Kind != "potentiometer" && !(p.Kind == "opamp" && a == "out") && !(p.Kind == "timer555" && (a == "control" || a == "out" || a == "discharge")) && len(physicalTerminals(g, groups[root(ref)])) < 2 {
				return fail("incomplete", "open-terminal", "Connect terminal "+ref+" to complete the circuit.", ref)
			}
			for _, b := range p.Terminals[i+1:] {
				u, v := net(p, a), net(p, b)
				if u == v && p.Kind != "opamp" && p.Kind != "timer555" && p.Kind != "potentiometer" && !(IsBipolarKind(p.Kind) && a == "base" && b == "collector") && !(IsMOSFETKind(p.Kind) && (a == "gate" || b == "gate")) {
					return fail("error", "shorted-component", "Remove the short across component "+p.ID+".", p.ID)
				}
				if networkReferencePair(p, a, b, len(opamps) > 0) {
					adjacency[u] = append(adjacency[u], v)
					adjacency[v] = append(adjacency[v], u)
				}
			}
		}
	}
	for _, v := range sources {
		if v.Kind != "dc-source" && net(v, "negative") != out.Ground {
			return fail("unsupported", "unsupported-topology", "Waveform sources must have their negative terminal grounded; DC sources may float in a ground-referenced network.", v.ID+".negative")
		}
		for _, c := range caps {
			a, b := net(c, "1"), net(c, "2")
			positive, negative := net(v, "positive"), net(v, "negative")
			if a == positive && b == negative || b == positive && a == negative {
				return fail("unsupported", "ideal-source-capacitor", "Insert resistance between an ideal source and capacitor; direct voltage jumps require an impulse model.", c.ID)
			}
		}
	}

	// Voltage-source and capacitor constraints must be independent at startup.
	// Redundant ideal constraints have indeterminate branch currents; reject them
	// explicitly rather than manufacturing a gmin or an initial current split.
	constraint := map[string]string{}
	for _, node := range out.Nets {
		constraint[node.ID] = node.ID
	}
	var cr func(string) string
	cr = func(id string) string {
		if constraint[id] != id {
			constraint[id] = cr(constraint[id])
		}
		return constraint[id]
	}
	join := func(a, b string) bool {
		a, b = cr(a), cr(b)
		if a == b {
			return false
		}
		constraint[b] = a
		return true
	}
	for _, p := range sources {
		if !join(net(p, "positive"), net(p, "negative")) {
			return fail("unsupported", "ideal-constraint-loop", "A loop of ideal voltage sources has indeterminate or inconsistent constraints; remove the redundant source or add series resistance.", p.ID)
		}
	}
	for _, p := range caps {
		if !join(net(p, "1"), net(p, "2")) {
			return fail("unsupported", "ideal-constraint-loop", "A capacitor/source constraint loop has indeterminate startup currents; remove the redundant capacitor or add resistance.", p.ID)
		}
	}
	seen := map[string]bool{out.Ground: true}
	queue := []string{out.Ground}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		for _, v := range adjacency[n] {
			if !seen[v] {
				seen[v] = true
				queue = append(queue, v)
			}
		}
	}
	for _, n := range out.Nets {
		if !seen[n.ID] {
			return fail("incomplete", "floating-island", "Connect every circuit island to the ground reference.", n.ID)
		}
	}
	out.MixedGraph = g.Clone()
	out.Source = sources[0]
	for _, v := range sources {
		if v.Kind == "pulse-source" {
			out.Source = v
			break
		}
	}
	for _, v := range sources {
		if IsWaveformSource(v.Kind) {
			out.Source = v
			break
		}
	}

	if len(caps) > 0 {
		out.Capacitor = caps[0]
	}
	if len(qs) > 0 {
		out.Transistor = qs[0]
	}
	if len(leds) > 0 {
		out.LED = leds[0]
	}
	out.Output = out.Supply
	if len(qs) > 0 {
		out.Output = net(qs[0], "collector")
	} else if len(mos) > 0 {
		out.Output = net(mos[0], "drain")
	} else if len(timers) > 0 {
		out.Output = net(timers[0], "out")
	} else if len(opamps) > 0 {
		out.Output = net(opamps[0], "out")
	} else if len(caps) > 0 {
		out.Output = net(caps[0], "1")
		if out.Output == out.Ground {
			out.Output = net(caps[0], "2")
		}
	} else if len(leds) > 0 {
		out.Output = net(leds[0], "anode")
	} else if len(pots) > 0 {
		out.Output = net(pots[0], "wiper")
	} else if len(diodes) > 0 {
		out.Output = net(diodes[0], "anode")
	} else if len(rs) > 0 {
		out.Output = net(rs[0], "2")
	}
	resistance := 1.
	if len(rs) > 0 {
		resistance = rs[0].Value
		out.Resistor = rs[0]
	}
	out.Supply = net(out.Source, "positive")
	if out.Output == "" {
		out.Output = out.Supply
	}
	out.Circuit = Circuit{out.Source.Value, resistance, true}
	return out, "", nil
}
