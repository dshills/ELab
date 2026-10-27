package workbench

func compileNetwork(out Compiled, g Graph, groups map[string][]string, root func(string) string) (Compiled, string, *Diagnostic) {
	out.Analysis = MixedAnalysis
	fail := func(status, code, msg, id string) (Compiled, string, *Diagnostic) {
		return out, status, &Diagnostic{code, msg, id}
	}
	motors := []Component{}
	optos := []Component{}
	transformers := []Component{}
	inductors := []Component{}
	mos := []Component{}
	opamps := []Component{}
	timers := []Component{}
	logic := []Component{}
	displays := []Component{}
	controls := []Component{}
	regulators := []Component{}
	bridges := []Component{}
	sequential := []Component{}
	sources, caps, qs, rs, leds, grounds, switches, diodes, pots := []Component{}, []Component{}, []Component{}, []Component{}, []Component{}, []Component{}, []Component{}, []Component{}, []Component{}
	for _, p := range g.Components {
		if err := validateDeviceParameters(p); err != nil {
			return fail("error", "invalid-parameters", err.Error(), p.ID)
		}
		switch p.Kind {
		case "shift-register":
			sequential = append(sequential, p)
		case "electronic-load", "speaker", "piezo", "active-buzzer":
			bridges = append(bridges, p)
		case "pwm-controller":
			bridges = append(bridges, p)
		case "hbridge":
			bridges = append(bridges, p)
		case "transformer":
			transformers = append(transformers, p)
		case "optocoupler":
			optos = append(optos, p)
		case "dc-motor":
			motors = append(motors, p)
		case "current-source", "battery", "dc-source", "pulse-source", "sine-source", "finite-pulse-source":
			sources = append(sources, p)
		case "capacitor":
			caps = append(caps, p)
		case "inductor":
			inductors = append(inductors, p)
		case "nmos", "pmos":
			mos = append(mos, p)
		case "linear-regulator", "ldo-regulator":
			regulators = append(regulators, p)
		case "relay":
			controls = append(controls, p)
			inductors = append(inductors, p)
		case "comparator", "pushbutton", "bouncing-pushbutton":
			controls = append(controls, p)
		case "bcd-decoder", "seven-segment":
			displays = append(displays, p)
		case "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor":
			logic = append(logic, p)
		case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter":
			sequential = append(sequential, p)
		case "timer555":
			timers = append(timers, p)
		case "opamp":
			opamps = append(opamps, p)
		case "npn", "pnp":
			qs = append(qs, p)
		case "switch":
			switches = append(switches, p)
		case "thermistor", "ldr", "resistor":
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
			return fail("unsupported", "unsupported-model", "The network solver supports sources, resistors, SPST switches, potentiometers, capacitors, inductors, diodes, red LEDs, NPN/PNP transistors, N/P-channel MOSFETs generic powered op-amps, 555 timers, logic gates, flip-flops and counters.", p.ID)
		}
	}
	for _, p := range opamps {
		if hasDynamics(p) && out.Analysis == NetworkDCAnalysis {
			out.Analysis = NetworkTransientAnalysis
		}
	}

	generalBipolar := hasBipolarPolarity(qs, "pnp")
	if len(grounds) > 0 {
		for _, q := range qs {
			generalBipolar = generalBipolar || hasTransistorProfile(q) || out.TerminalNet[q.ID+".emitter"] != out.TerminalNet[grounds[0].ID+".reference"]
		}
	}
	if len(logic) > 0 || len(opamps) > 0 || len(mos) > 0 || len(pots) > 0 || len(diodes) > 0 || len(switches) > 0 || len(caps) == 0 || len(qs) == 0 || generalBipolar {
		out.Analysis = NetworkDCAnalysis
		if len(caps) > 0 {
			out.Analysis = NetworkTransientAnalysis
		} else {
			for _, source := range sources {
				if source.Kind == "current-source" && source.Device.Current.Timing.Pulsed || source.Kind == "pulse-source" || IsWaveformSource(source.Kind) {
					out.Analysis = NetworkTransientAnalysis
				}
			}
		}
	}
	for _, p := range qs {
		if hasTransistorProfile(p) && (p.Device.Bipolar.CBE > 0 || p.Device.Bipolar.CBC > 0) && out.Analysis == NetworkDCAnalysis {
			out.Analysis = NetworkTransientAnalysis
		}
	}
	for _, source := range sources {
		if IsWaveformSource(source.Kind) {
			out.Analysis = WaveformAnalysis
			break
		}
	}

	for _, p := range sources {
		if p.Kind == "battery" {
			out.Analysis = NetworkTransientAnalysis
		}
	}
	if hasSpeakers(g) || len(inductors)+len(motors)+len(transformers) > 0 {
		out.Analysis = InductorAnalysis
	}
	for _, p := range bridges {
		if p.Kind == "electronic-load" && p.Device.Load.Timing.Pulsed && !hasSpeakers(g) && len(inductors)+len(motors)+len(transformers) == 0 {
			out.Analysis = WaveformAnalysis
		}
		if p.Kind == "pwm-controller" && !hasSpeakers(g) && len(inductors)+len(motors)+len(transformers) == 0 {
			out.Analysis = WaveformAnalysis
		}
	}
	if len(timers) > 0 {
		out.Analysis = Timer555Analysis
	}
	if len(sequential) > 0 {
		out.Analysis = SequentialAnalysis
	}
	if len(controls) > 0 {
		out.Analysis = ControlAnalysis
	}
	for _, p := range opamps {
		if hasDynamics(p) && out.Analysis == NetworkDCAnalysis {
			out.Analysis = NetworkTransientAnalysis
		}
	}
	for _, p := range bridges {
		if p.Kind == "piezo" && out.Analysis == NetworkDCAnalysis {
			out.Analysis = NetworkTransientAnalysis
		}
	}
	if len(timers) > MaxNetworkTimers555 {
		return fail("unsupported", "mixed-limit", "Network simulation supports at most two 555 timers.", "")
	}
	if len(opamps) > MaxNetworkOpAmps {
		return fail("unsupported", "mixed-limit", "Network simulation supports at most two generic powered op-amps within the existing total part, node and source limits.", "")
	}

	usage := powerResources(g)
	if SimulatedComponentCount(g)+usage.parts > MaxSimulatedComponents || len(caps)+usage.caps > MaxNetworkCapacitors || len(rs)+usage.resistors > MaxNetworkResistors || len(inductors)+usage.inductors > MaxNetworkInductors || len(sources)+usage.sources > MaxNetworkSources || len(out.Nets)+usage.nodes > MaxElectricalNodes {
		return fail("unsupported", "mixed-limit", networkLimitMessage(), "")
	}

	if len(inductors) > MaxNetworkInductors || len(mos) > MaxNetworkMOSFETs || SimulatedComponentCount(g) > MaxSimulatedComponents || len(pots) > MaxNetworkPotentiometers || len(diodes)+len(optos) > MaxNetworkDiodes || len(switches) > MaxNetworkSwitches || len(sources) > MaxNetworkSources || len(caps) > MaxNetworkCapacitors || len(qs)+len(optos) > MaxNetworkTransistors || len(rs) > MaxNetworkResistors || len(leds) > MaxNetworkLEDs || len(out.Nets) > MaxElectricalNodes {
		return fail("unsupported", "mixed-limit", networkLimitMessage(), "")
	}
	if len(grounds) == 0 || len(sources) == 0 || len(rs)+len(switches)+len(diodes)+len(pots)+len(opamps)+len(timers)+len(logic)+len(sequential)+len(displays)+len(controls)+len(regulators)+len(motors)+len(optos)+len(transformers)+len(bridges) == 0 && !hasPassiveDissipation(g) && !hasCurrentDevices(g) {
		return fail("incomplete", "missing-parts", "Add ground, a source and a resistor, switch, diode, potentiometer, op-amp, 555 timer, logic gate, flip-flop counter, or saved capacitor/inductor loss to complete the network.", "")
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

		if p.Kind == "hbridge" && net(p, "vm") == net(p, "gnd") {
			return fail("error", "shorted-component", "Remove the short across H-bridge supply pins.", p.ID)
		}
		if (p.Kind == "shift-register" || p.Kind == "pwm-controller" || p.Kind == "timer555" || IsLogicKind(p.Kind) || IsSequentialKind(p.Kind) || p.Kind == "bcd-decoder" || p.Kind == "comparator") && net(p, "vcc") == net(p, "gnd") {
			return fail("error", "shorted-component", "Remove the short across IC supply pins.", p.ID)
		}
		if IsDisplayKind(p.Kind) {
			for _, pin := range p.Terminals {
				required := pin == "common" || p.Kind == "bcd-decoder" && (pin == "vcc" || pin == "gnd" || pin[0] == 'b' && len(pin) == 2)
				if required && len(physicalTerminals(g, groups[root(p.ID+"."+pin)])) < 2 {
					return fail("incomplete", "open-terminal", "Connect terminal "+p.ID+"."+pin, p.ID+"."+pin)
				}
			}
		}
		for i, a := range p.Terminals {
			ref := p.ID + "." + a
			if !(p.Kind == "shift-register" && (a == "serial" || len(a) == 2 && a[0] == 'q')) && !(p.Kind == "pwm-controller" && a == "out") && !(p.Kind == "hbridge" && (a == "out1" || a == "out2")) && p.Kind != "transformer" && !(IsRegulatorKind(p.Kind) && a == "out") && !(p.Kind == "relay" && (a == "no" || a == "nc")) && !(p.Kind == "comparator" && a == "out") && !IsDisplayKind(p.Kind) && p.Kind != "potentiometer" && !(IsSequentialKind(p.Kind) && sequentialOutput(a)) && !((p.Kind == "opamp" || IsLogicKind(p.Kind)) && a == "out") && !(p.Kind == "timer555" && (a == "control" || a == "out" || a == "discharge")) && len(physicalTerminals(g, groups[root(ref)])) < 2 {
				return fail("incomplete", "open-terminal", "Connect terminal "+ref+" to complete the circuit.", ref)
			}
			for _, b := range p.Terminals[i+1:] {
				u, v := net(p, a), net(p, b)
				if u == v && p.Kind != "shift-register" && p.Kind != "pwm-controller" && p.Kind != "hbridge" && p.Kind != "transformer" && p.Kind != "optocoupler" && !IsRegulatorKind(p.Kind) && p.Kind != "relay" && p.Kind != "comparator" && !IsDisplayKind(p.Kind) && !IsSequentialKind(p.Kind) && !IsLogicKind(p.Kind) && p.Kind != "opamp" && p.Kind != "timer555" && p.Kind != "potentiometer" && !(IsBipolarKind(p.Kind) && a == "base" && b == "collector") && !(IsMOSFETKind(p.Kind) && (a == "gate" || b == "gate")) {
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
		if v.Kind != "current-source" && v.Kind != "battery" && v.Kind != "dc-source" && net(v, "negative") != out.Ground {
			return fail("unsupported", "unsupported-topology", "Waveform sources must have their negative terminal grounded; DC sources may float in a ground-referenced network.", v.ID+".negative")
		}
		if v.Kind == "battery" || v.Kind == "current-source" {
			continue
		}
		for _, c := range caps {
			if capacitorESR(c) > 0 {
				continue
			}
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
		if p.Kind == "battery" || p.Kind == "current-source" {
			continue
		}
		if !join(net(p, "positive"), net(p, "negative")) {
			return fail("unsupported", "ideal-constraint-loop", "A loop of ideal voltage sources has indeterminate or inconsistent constraints; remove the redundant source or add series resistance.", p.ID)
		}
	}
	for _, p := range caps {
		if capacitorESR(p) > 0 {
			continue
		}
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
	// Prefer a voltage input when current and voltage sources coexist.
	if out.Source.Kind == "current-source" {
		for _, v := range sources {
			if IsVoltageSource(v.Kind) {
				out.Source = v
				break
			}
		}
	}
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
	} else if len(sequential) > 0 {
		pin := "q"
		if sequential[0].Kind == "shift-register" {
			pin = "q0"
		}
		if IsCounterKind(sequential[0].Kind) {
			pin = "q0"
		}
		out.Output = net(sequential[0], pin)
	} else if len(timers) > 0 {
		out.Output = net(timers[0], "out")
	} else if len(logic) > 0 {
		out.Output = net(logic[0], "out")
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
	if err := validateExtendedConstraints(out); err != nil {
		return fail("unsupported", "ideal-source-capacitor", err.Error(), "")
	}
	return out, "", nil
}
