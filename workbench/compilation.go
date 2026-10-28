package workbench

import "sort"

type Compiled struct {
	MixedGraph                       Graph
	Circuit                          Circuit
	Source, Resistor, LED, Capacitor Component
	Transistor, Input, BaseResistor  Component
	Base, Collector, InputNet        string
	Analysis, Output                 string
	TerminalNet                      map[string]string
	Nets                             []Net
	Ground, Supply, Anode, Cathode   string
}

// Compile derives nets from explicit terminal connections and matching named labels. Coordinates,
// line crossings, drawing order, IDs, and symbol rotation never create a net.
func (g Graph) Compile() (Compiled, string, *Diagnostic) {
	if g.hasBlocks() {
		physical, err := g.PhysicalGraph()
		if err != nil {
			return Compiled{}, "error", &Diagnostic{"invalid-block", err.Error(), ""}
		}
		c, status, d := physical.Compile()
		if d == nil {
			for alias, ref := range g.blockAliases() {
				c.TerminalNet[alias] = c.TerminalNet[ref]
			}
		}
		return c, status, d
	}
	for _, p := range g.Components {
		if err := validateThermal(p); err != nil {
			return Compiled{}, "error", &Diagnostic{"invalid-thermal", err.Error(), p.ID}
		}
	}
	out := Compiled{TerminalNet: map[string]string{}, Nets: []Net{}}
	parent := map[string]string{}
	var root func(string) string
	root = func(s string) string {
		if parent[s] != s {
			parent[s] = root(parent[s])
		}
		return parent[s]
	}
	union := func(a, b string) {
		a, b = root(a), root(b)
		if a > b {
			a, b = b, a
		}
		parent[b] = a
	}
	hasInductor := false
	hasMOS := false
	hasOpAmp := false
	hasSwitch := false
	hasDiode := false
	hasPot := false
	hasWaveforms := false
	sources, resistors, leds, grounds := []Component{}, []Component{}, []Component{}, []Component{}
	capacitors := []Component{}
	transistors := []Component{}
	for _, c := range g.Components {
		if IsWaveformSource(c.Kind) {
			hasWaveforms = true
		}
		for _, t := range c.Terminals {
			ref := c.ID + "." + t
			parent[ref] = ref
		}
		switch c.Kind {
		case "battery", "dc-source", "pulse-source", "sine-source", "finite-pulse-source":
			if c.Kind == "battery" {
				hasSwitch = true
			}
			sources = append(sources, c)
		case "scr", "triac", "photodiode", "speaker", "shift-register", "piezo", "active-buzzer", "pwm-controller", "hbridge", "transformer", "optocoupler", "dc-motor", "linear-regulator", "ldo-regulator", "relay", "comparator", "pushbutton", "bouncing-pushbutton", "bcd-decoder", "seven-segment", "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter", "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor", "timer555", "switch":
			hasSwitch = true
		case "diode", "zener", "schottky":
			hasDiode = true
		case "thermistor", "ldr":
			hasSwitch = true
		case "potentiometer":
			hasPot = true
		case "resistor":
			resistors = append(resistors, c)
		case "nmos", "pmos":
			hasMOS = true
		case "opamp":
			hasOpAmp = true
		case "npn", "pnp":
			transistors = append(transistors, c)
		case "capacitor":
			capacitors = append(capacitors, c)
		case "inductor":
			hasInductor = true
		case "led":
			leds = append(leds, c)
		case "ground":
			grounds = append(grounds, c)
		}
	}
	for _, w := range g.Wires {
		if _, ok := parent[w.From]; !ok {
			return out, "error", &Diagnostic{"invalid-terminal", "A wire references a missing terminal.", w.ID}
		}
		if _, ok := parent[w.To]; !ok {
			return out, "error", &Diagnostic{"invalid-terminal", "A wire references a missing terminal.", w.ID}
		}
		if w.Connected {
			union(w.From, w.To)
		}
	}
	// Ground symbols share an explicit common reference, as in ordinary schematics.
	for i := 1; i < len(grounds); i++ {
		union(grounds[0].ID+".reference", grounds[i].ID+".reference")
	}
	labels := map[string]string{}
	for _, part := range g.Components {
		if part.Kind == "net-label" {
			if !validNetLabel(part.Label) {
				return out, "error", &Diagnostic{"invalid-net-label", "Choose a valid net label.", part.ID}
			}
			ref := part.ID + ".node"
			if first, ok := labels[part.Label]; ok {
				union(first, ref)
			} else {
				labels[part.Label] = ref
			}
		}
	}
	groups := map[string][]string{}
	for ref := range parent {
		r := root(ref)
		groups[r] = append(groups[r], ref)
	}
	keys := []string{}
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		sort.Strings(groups[key])
		members := physicalTerminals(g, groups[key])
		first := groups[key][0]
		if len(members) > 0 {
			first = members[0]
		}
		id := "net:" + first
		names := []string{}
		seenNames := map[string]bool{}
		for _, ref := range groups[key] {
			part, _, _ := g.Terminal(ref)
			if part.Kind == "net-label" && !seenNames[part.Label] {
				names = append(names, part.Label)
				seenNames[part.Label] = true
			}
		}
		sort.Strings(names)
		out.Nets = append(out.Nets, Net{ID: id, Terminals: groups[key], Labels: names})
		for _, t := range groups[key] {
			out.TerminalNet[t] = id
		}
	}
	sort.Slice(out.Nets, func(i, j int) bool { return out.Nets[i].ID < out.Nets[j].ID })
	for _, part := range g.Components {
		if IsConnectivityPrimitive(part.Kind) && len(physicalTerminals(g, groups[root(part.ID+".node")])) < 2 {
			return out, "incomplete", &Diagnostic{"unconnected-primitive", "Connect this junction/label net to at least two circuit terminals.", part.ID + ".node"}
		}
	}
	incomplete := func(code, message, id string) (Compiled, string, *Diagnostic) {
		return out, "incomplete", &Diagnostic{code, message, id}
	}
	networkTopology := len(transistors) == 0 && (len(sources) > 1 || len(resistors) > 1 || len(leds) > 1 || len(capacitors) > 1 || len(capacitors) > 0 && len(leds) > 0 || len(capacitors) == 0 && len(leds) == 0 && len(resistors) > 0)
	floatingDC := false
	bipolarNetwork := len(transistors) > 0 && (len(transistors) != 1 || transistors[0].Kind == "pnp" || len(resistors) != 2 || len(leds) > 1)
	if len(grounds) > 0 {
		for _, q := range transistors {
			bipolarNetwork = bipolarNetwork || hasTransistorProfile(q) || out.TerminalNet[q.ID+".emitter"] != out.TerminalNet[grounds[0].ID+".reference"] || out.TerminalNet[q.ID+".base"] == out.TerminalNet[q.ID+".collector"]
		}
		for _, source := range sources {
			if source.Kind == "dc-source" && out.TerminalNet[source.ID+".negative"] != out.TerminalNet[grounds[0].ID+".reference"] {
				floatingDC = true
			}
		}
	}
	// Optional storage losses require the general network equations, not the ideal RC shortcut.
	if hasThermals(g) || hasJFETs(g) || hasCurrentDevices(g) || hasPassiveLosses(g) || hasOpAmp || hasInductor || hasMOS || hasWaveforms || hasSwitch || hasDiode || hasPot || networkTopology || floatingDC || bipolarNetwork || len(transistors) > 0 && len(capacitors) > 0 {
		return compileNetwork(out, g, groups, root)
	}
	if len(transistors) > 0 {
		compiled, status, diagnostic := compileNPN(out, g, sources, resistors, leds, transistors, capacitors, grounds, groups, root)
		// Retain the established grounded-emitter engine where it applies.
		// Followers, emitter resistors and multiple BJTs use the full network.
		if status == "unsupported" && diagnostic != nil && diagnostic.Code == "unsupported-topology" {
			return compileNetwork(out, g, groups, root)
		}
		return compiled, status, diagnostic
	}
	if len(capacitors) > 0 {
		compiled, status, diagnostic := compileRC(out, g, sources, resistors, capacitors, leds, grounds, groups, root)
		// Keep the exact one-pole low-pass path. Other RC arrangements, such as
		// a series-capacitor high-pass, still belong to the supported network
		// family; let its compiler validate topology and ideal constraints.
		if status == "unsupported" && diagnostic != nil && diagnostic.Code == "unsupported-topology" {
			return compileNetwork(out, g, groups, root)
		}
		return compiled, status, diagnostic
	}
	if len(sources) == 0 || len(resistors) == 0 || len(leds) == 0 {
		return incomplete("missing-parts", "Place a DC source, resistor, red LED, and ground to build the LED circuit.", "")
	}
	if len(grounds) == 0 {
		return incomplete("missing-ground", "Place ground and connect it to the source's negative terminal.", sources[0].ID)
	}
	if len(sources) != 1 || len(resistors) != 1 || len(leds) != 1 {
		return out, "unsupported", &Diagnostic{"unsupported-topology", "This live solver supports one source, one resistor, and one LED. Your drawing is retained.", ""}
	}
	if sources[0].Kind != "dc-source" {
		return out, "unsupported", &Diagnostic{"unsupported-topology", "Pulse sources currently support the RC transient circuit. Use a DC source for the LED circuit.", sources[0].ID}
	}
	out.Analysis = "DC operating point"
	out.Source, out.Resistor, out.LED = sources[0], resistors[0], leds[0]
	out.Circuit = Circuit{out.Source.Value, out.Resistor.Value, true}
	for _, c := range []Component{out.Source, out.Resistor, out.LED} {
		for _, name := range c.Terminals {
			ref := c.ID + "." + name
			if len(physicalTerminals(g, groups[root(ref)])) < 2 {
				return incomplete("open-terminal", "Connect terminal "+ref+" to complete the circuit.", ref)
			}
		}
	}
	net := func(c Component, t string) string { return out.TerminalNet[c.ID+"."+t] }
	out.Ground = net(grounds[0], "reference")
	out.Supply = net(out.Source, "positive")
	out.Anode = net(out.LED, "anode")
	out.Cathode = net(out.LED, "cathode")
	if net(out.Source, "negative") != out.Ground {
		return incomplete("reference-disconnected", "Connect the source's negative terminal to ground.", out.Source.ID+".negative")
	}
	r1, r2 := net(out.Resistor, "1"), net(out.Resistor, "2")
	for _, part := range []Component{out.Source, out.Resistor, out.LED} {
		if net(part, part.Terminals[0]) == net(part, part.Terminals[1]) {
			return out, "error", &Diagnostic{"shorted-component", "A component's terminals are shorted together. Remove the unintended wire.", part.ID}
		}
	}
	resistorBetween := func(a, b string) bool { return r1 == a && r2 == b || r1 == b && r2 == a }
	if resistorBetween(out.Supply, out.Anode) && out.Cathode == out.Ground || out.Anode == out.Supply && resistorBetween(out.Cathode, out.Ground) {
		return out, "", nil
	}
	return out, "unsupported", &Diagnostic{"unsupported-topology", "Connect a single forward LED and resistor in series from source positive to ground. Reversed LEDs and other topologies are not simulated yet.", out.LED.ID}
}
func (c Compiled) Annotate(r Result) Result {
	r.SourceID, r.ResistorID, r.LEDID = c.Source.ID, c.Resistor.ID, c.LED.ID
	r.Nodes = map[string]float64{c.Ground: 0, c.Supply: c.Circuit.Voltage, "ground": 0, "supply": c.Circuit.Voltage}
	if c.Anode == c.Supply {
		r.Nodes[c.Cathode] = c.Circuit.Voltage - r.LEDVoltage
	} else {
		r.Nodes[c.Anode] = r.LEDVoltage
	}
	r.Nodes["led-anode"] = r.Nodes[c.Anode]
	return r
}

func compileRC(out Compiled, g Graph, sources, resistors, capacitors, leds, grounds []Component, groups map[string][]string, root func(string) string) (Compiled, string, *Diagnostic) {
	out.Analysis = "RC transient"
	fail := func(status, code, msg, id string) (Compiled, string, *Diagnostic) {
		return out, status, &Diagnostic{code, msg, id}
	}
	if len(sources) == 0 || len(resistors) == 0 {
		return fail("incomplete", "missing-parts", "Place one DC or pulse source, resistor, capacitor, and ground.", "")
	}
	if len(grounds) == 0 {
		return fail("incomplete", "missing-ground", "Connect source negative and the capacitor return to ground.", "")
	}
	if len(sources) != 1 || len(resistors) != 1 || len(capacitors) != 1 || len(leds) != 0 {
		return fail("unsupported", "unsupported-topology", "The RC solver supports one source, one resistor, and one capacitor; mixed LED/RC and other networks are not simulated yet.", "")
	}
	out.Source, out.Resistor, out.Capacitor = sources[0], resistors[0], capacitors[0]
	out.Circuit = Circuit{out.Source.Value, out.Resistor.Value, true}
	net := func(c Component, t string) string { return out.TerminalNet[c.ID+"."+t] }
	out.Ground = net(grounds[0], "reference")
	out.Supply = net(out.Source, "positive")
	for _, c := range []Component{out.Source, out.Resistor, out.Capacitor} {
		for _, t := range c.Terminals {
			ref := c.ID + "." + t
			if len(physicalTerminals(g, groups[root(ref)])) < 2 {
				return fail("incomplete", "open-terminal", "Connect terminal "+ref+" to complete the RC circuit.", ref)
			}
		}
	}
	if net(out.Source, "negative") != out.Ground {
		return fail("incomplete", "reference-disconnected", "Connect source negative to ground.", out.Source.ID+".negative")
	}
	r1, r2, c1, c2 := net(out.Resistor, "1"), net(out.Resistor, "2"), net(out.Capacitor, "1"), net(out.Capacitor, "2")
	if out.Supply == out.Ground || r1 == r2 || c1 == c2 {
		return fail("error", "shorted-component", "A component's terminals are shorted. Remove the unintended wire.", "")
	}
	if c1 == out.Ground {
		out.Output = c2
	} else if c2 == out.Ground {
		out.Output = c1
	} else {
		return fail("unsupported", "unsupported-topology", "Connect the capacitor between the RC output and ground.", out.Capacitor.ID)
	}
	if out.Output == out.Supply || !(r1 == out.Supply && r2 == out.Output || r2 == out.Supply && r1 == out.Output) {
		return fail("unsupported", "unsupported-topology", "Connect source positive through the resistor to the capacitor, with capacitor return grounded.", out.Capacitor.ID)
	}
	return out, "", nil
}
