package workbench

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
)

const GridSize = 10.

type Graph struct {
	Components []Component `json:"components"`
	Wires      []Wire      `json:"wires"`
}

func EmptyGraph() Graph   { return Graph{Components: []Component{}, Wires: []Wire{}} }
func ExampleGraph() Graph { c := DefaultCircuit(); return Graph{Components(c), Wires(c)} }
func (g Graph) Clone() Graph {
	out := Graph{append([]Component{}, g.Components...), append([]Wire{}, g.Wires...)}
	for i := range out.Components {
		out.Components[i].Device = cloneDeviceParameters(out.Components[i].Device)
		out.Components[i].Power = clonePower(out.Components[i].Power)
		out.Components[i].Terminals = append([]string{}, out.Components[i].Terminals...)
		if w := out.Components[i].Sine; w != nil {
			v := *w
			out.Components[i].Sine = &v
		}
		if w := out.Components[i].Pulse; w != nil {
			v := *w
			out.Components[i].Pulse = &v
		}
		if v := out.Components[i].TemperatureC; v != nil {
			c := *v
			out.Components[i].TemperatureC = &c
		}
		if v := out.Components[i].IlluminanceLux; v != nil {
			c := *v
			out.Components[i].IlluminanceLux = &c
		}
		if v := out.Components[i].WiperPosition; v != nil {
			copy := *v
			out.Components[i].WiperPosition = &copy
		}
		if state := out.Components[i].SwitchClosed; state != nil {
			copy := *state
			out.Components[i].SwitchClosed = &copy
		}
		if v := out.Components[i].InitialVoltage; v != nil {
			copy := *v
			out.Components[i].InitialVoltage = &copy
		}
		if v := out.Components[i].InitialCurrent; v != nil {
			copy := *v
			out.Components[i].InitialCurrent = &copy
		}
	}
	return out
}
func (g Graph) Component(id string) (Component, bool) {
	for _, c := range g.Components {
		if c.ID == id {
			return c, true
		}
	}
	return Component{}, false
}
func (g Graph) HasObject(id string) bool {
	if _, ok := g.Component(id); ok {
		return true
	}
	for _, w := range g.Wires {
		if w.ID == id {
			return true
		}
	}
	return false
}
func (g Graph) Terminal(ref string) (Component, string, bool) {
	id, name, ok := strings.Cut(ref, ".")
	if !ok {
		return Component{}, "", false
	}
	c, ok := g.Component(id)
	if !ok {
		return Component{}, "", false
	}
	for _, t := range c.Terminals {
		if name == t {
			return c, name, true
		}
	}
	return Component{}, "", false
}
func (g Graph) Summary() Circuit {
	var c Circuit
	for _, p := range g.Components {
		if p.Kind == "dc-source" {
			c.Voltage = p.Value
		}
		if p.Kind == "resistor" {
			c.Resistance = p.Value
		}
	}
	compiled, _, diagnostic := g.Compile()
	c.Connected = diagnostic == nil
	if diagnostic == nil {
		return compiled.Circuit
	}
	return c
}
func ElectricalEqual(a, b Graph) bool {
	a = a.Clone()
	b = b.Clone()
	for i := range a.Components {
		a.Components[i].Position = Point{}
		a.Components[i].Rotation = 0
	}
	for i := range b.Components {
		b.Components[i].Position = Point{}
		b.Components[i].Rotation = 0
	}
	return reflect.DeepEqual(a, b)
}
func snapPoint(p Point) (Point, error) {
	if !finite(p.X) || !finite(p.Y) || math.Abs(p.X) > 10000 || math.Abs(p.Y) > 10000 {
		return Point{}, fmt.Errorf("position must be finite and within ±10000 units")
	}
	return Point{math.Round(p.X/GridSize) * GridSize, math.Round(p.Y/GridSize) * GridSize}, nil
}
func ValidID(id string) bool {
	if len(id) < 1 || len(id) > 64 {
		return false
	}
	for i, c := range id {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || i > 0 && (c >= '0' && c <= '9' || c == '_' || c == '-')) {
			return false
		}
	}
	return true
}

type Part struct {
	Category     string                `json:"category"`
	Common       bool                  `json:"common"`
	ValueType    string                `json:"value_type"`
	ValueDomain  *ComponentValueDomain `json:"value_domain,omitempty"`
	Aliases      []string              `json:"aliases"`
	Limitations  string                `json:"limitations"`
	Kind         string                `json:"kind"`
	Name         string                `json:"name"`
	Terminals    []string              `json:"terminals"`
	DefaultValue string                `json:"default_value,omitempty"`
	Unit         string                `json:"unit,omitempty"`
	Model        string                `json:"model"`
}

func libraryParts() []Part {
	return []Part{
		{Kind: "shift-register", Name: "8-bit shift / latch register", Terminals: append([]string{}, shiftPins...), Model: ExtendedModelID},
		{Kind: "piezo", Name: "Piezo sounder", Terminals: []string{"positive", "negative"}, Model: ExtendedModelID},
		{Kind: "active-buzzer", Name: "Active buzzer", Terminals: []string{"positive", "negative"}, Model: ExtendedModelID},
		{Kind: "pwm-controller", Name: "PWM controller", Terminals: append([]string{}, pwmPins...), Model: ExtendedModelID},
		{Kind: "hbridge", Name: "H-bridge motor driver", Terminals: append([]string{}, hbridgePins[:]...), Model: HBridgeModelID},
		{Kind: "transformer", Name: "Transformer / coupled inductors", Terminals: []string{"primary1", "primary2", "secondary1", "secondary2"}, Model: PowerModelID},
		{Kind: "optocoupler", Name: "LED / phototransistor optocoupler", Terminals: []string{"anode", "cathode", "collector", "emitter"}, Model: PowerModelID},
		{Kind: "dc-motor", Name: "Brushed DC motor", Terminals: []string{"positive", "negative"}, Model: PowerModelID},
		{Kind: "battery", Name: "Battery", Terminals: []string{"positive", "negative"}, DefaultValue: "9 V", Unit: "V", Model: PowerModelID},
		{Kind: "dc-source", Name: "DC source", Terminals: []string{"positive", "negative"}, DefaultValue: "5 V", Unit: "V", Model: "ideal-dc.v1"},
		{Kind: "thermistor", Name: "NTC thermistor", Terminals: []string{"1", "2"}, DefaultValue: "10k Ω", Unit: "Ω", Model: SensorModelID},
		{Kind: "ldr", Name: "Light-dependent resistor", Terminals: []string{"1", "2"}, DefaultValue: "10k Ω", Unit: "Ω", Model: SensorModelID},
		{Kind: "resistor", Name: "Resistor", Terminals: []string{"1", "2"}, DefaultValue: "1k", Unit: "Ω", Model: "ideal-resistor.v1"},
		{Kind: "led", Name: "Red LED", Terminals: []string{"anode", "cathode"}, DefaultValue: "", Unit: "", Model: ModelID},
		{Kind: "capacitor", Name: "Capacitor", Terminals: []string{"1", "2"}, DefaultValue: "1uF", Unit: "F", Model: "ideal-capacitor.v1"},
		{Kind: "inductor", Name: "Inductor", Terminals: []string{"1", "2"}, DefaultValue: "10 mH", Unit: "H", Model: InductorModelID},
		{Kind: "sine-source", Name: "Sine source", Terminals: []string{"positive", "negative"}, DefaultValue: "2.5 V", Unit: "V", Model: SineSourceModelID},
		{Kind: "finite-pulse-source", Name: "Finite pulse source", Terminals: []string{"positive", "negative"}, DefaultValue: "5 V", Unit: "V", Model: FinitePulseModelID},
		{Kind: "pulse-source", Name: "Pulse source", Terminals: []string{"positive", "negative"}, DefaultValue: "5 V", Unit: "V", Model: "ideal-square.v1"},
		{Kind: "pmos", Name: "P-channel MOSFET", Terminals: []string{"gate", "drain", "source"}, Model: PMOSModelID},
		{Kind: "nmos", Name: "N-channel MOSFET", Terminals: []string{"gate", "drain", "source"}, Model: NMOSModelID},
		{Kind: "npn", Name: "NPN transistor", Terminals: []string{"base", "collector", "emitter"}, DefaultValue: "", Unit: "", Model: NPNModelID},
		{Kind: "pnp", Name: "PNP transistor", Terminals: []string{"base", "collector", "emitter"}, Model: PNPModelID},
		{Kind: "bcd-decoder", Name: "BCD decoder", Terminals: displayPins("bcd-decoder"), Model: DisplayModelID},
		{Kind: "seven-segment", Name: "Seven-segment display", Terminals: displayPins("seven-segment"), Model: DisplayModelID},
		{Kind: "d-flipflop", Name: "D flip-flop", Terminals: sequentialPins("d-flipflop"), Model: SequentialModelID},
		{Kind: "jk-flipflop", Name: "JK flip-flop", Terminals: sequentialPins("jk-flipflop"), Model: SequentialModelID},
		{Kind: "t-flipflop", Name: "T flip-flop", Terminals: sequentialPins("t-flipflop"), Model: SequentialModelID},
		{Kind: "binary-counter", Name: "4-bit binary counter", Terminals: sequentialPins("binary-counter"), Model: SequentialModelID},
		{Kind: "decade-counter", Name: "Decade counter", Terminals: sequentialPins("decade-counter"), Model: SequentialModelID},
		{Kind: "logic-not", Name: "NOT gate", Terminals: logicPins("logic-not"), Model: LogicModelID},
		{Kind: "logic-and", Name: "AND gate", Terminals: logicPins("logic-and"), Model: LogicModelID},
		{Kind: "logic-or", Name: "OR gate", Terminals: logicPins("logic-or"), Model: LogicModelID},
		{Kind: "logic-nand", Name: "NAND gate", Terminals: logicPins("logic-nand"), Model: LogicModelID},
		{Kind: "logic-nor", Name: "NOR gate", Terminals: logicPins("logic-nor"), Model: LogicModelID},
		{Kind: "logic-xor", Name: "XOR gate", Terminals: logicPins("logic-xor"), Model: LogicModelID},
		{Kind: "logic-xnor", Name: "XNOR gate", Terminals: logicPins("logic-xnor"), Model: LogicModelID},
		{Kind: "relay", Name: "SPDT relay", Terminals: controlPins("relay"), DefaultValue: "100mH", Unit: "H", Model: ControlModelID},
		{Kind: "linear-regulator", Name: "Linear voltage regulator", Terminals: append([]string{}, regulatorPins...), DefaultValue: "5 V", Unit: "V", Model: RegulatorModelID},
		{Kind: "ldo-regulator", Name: "LDO voltage regulator", Terminals: append([]string{}, regulatorPins...), DefaultValue: "3.3 V", Unit: "V", Model: RegulatorModelID},
		{Kind: "comparator", Name: "Comparator with hysteresis", Terminals: controlPins("comparator"), DefaultValue: "20 mV", Unit: "V", Model: ControlModelID},
		{Kind: "opamp", Name: "Powered op-amp", Terminals: []string{"noninverting", "inverting", "out", "vplus", "vminus"}, Model: OpAmpModelID},
		{Kind: "ground", Name: "Ground", Terminals: []string{"reference"}, DefaultValue: "", Unit: "", Model: "ground.v1"},
		{Kind: "junction", Name: "Junction", Terminals: []string{"node"}, DefaultValue: "", Unit: "", Model: "junction.v1"},
		{Kind: "net-label", Name: "Net label", Terminals: []string{"node"}, DefaultValue: "", Unit: "", Model: "net-label.v1"},
		{Kind: "potentiometer", Name: "Potentiometer", Terminals: []string{"1", "2", "wiper"}, DefaultValue: "10k Ω", Unit: "Ω", Model: PotentiometerModelID},
		{Kind: "timer555", Name: "555 timer", Terminals: append([]string{}, timer555Pins[:]...), Model: Timer555ModelID},
		{Kind: "zener", Name: "Zener diode", Terminals: []string{"anode", "cathode"}, DefaultValue: "5.1", Unit: "V", Model: ZenerModelID},
		{Kind: "schottky", Name: "Schottky diode", Terminals: []string{"anode", "cathode"}, Model: SchottkyModelID},
		{Kind: "diode", Name: "Silicon diode", Terminals: []string{"anode", "cathode"}, DefaultValue: "", Unit: "", Model: DiodeModelID},
		{Kind: "pushbutton", Name: "Momentary pushbutton", Terminals: controlPins("pushbutton"), Model: ControlModelID},
		{Kind: "bouncing-pushbutton", Name: "Bouncing pushbutton", Terminals: controlPins("bouncing-pushbutton"), Model: ControlModelID},
		{Kind: "switch", Name: "SPST switch", Terminals: []string{"1", "2"}, DefaultValue: "", Unit: "", Model: SwitchModelID},
	}
}

func Library() []Part {
	parts := libraryParts()
	for i := range parts {
		parts[i].Category, parts[i].Common = partGroup(parts[i].Kind)
		parts[i].ValueType = "none"
		if parts[i].Kind == "net-label" {
			parts[i].ValueType = "label"
		}
		if domain, ok := ValueDomainForKind(parts[i].Kind); ok {
			parts[i].ValueType = "number"
			parts[i].ValueDomain = &domain
		}
		parts[i].Aliases = partAliases(parts[i].Kind)
		parts[i].Limitations = partLimitations(parts[i].Kind)
	}
	return parts
}
func MakeComponent(id, kind string, p Point, value string) (Component, error) {
	if !ValidID(id) {
		return Component{}, fmt.Errorf("component ID must be 1–64 ASCII letters/digits/_/- and start with a letter")
	}
	position, err := snapPoint(p)
	if err != nil {
		return Component{}, err
	}
	for _, part := range libraryParts() {
		if part.Kind != kind {
			continue
		}
		c := Component{ID: id, Kind: kind, Name: part.Name, Terminals: append([]string{}, part.Terminals...), Position: position, Unit: part.Unit, Model: part.Model}
		if kind == "sine-source" {
			w := DefaultSineWaveform()
			c.Sine = &w
		}
		if kind == "finite-pulse-source" {
			w := DefaultPulseWaveform()
			c.Pulse = &w
		}
		defaultPower(&c)
		defaultDeviceParameters(&c)
		if IsSensorKind(kind) {
			v := 25.
			if kind == "ldr" {
				v = 100
			}
			setSensorEnvironment(&c, v)
		}
		if kind == "potentiometer" {
			w := .5
			c.WiperPosition = &w
		}
		if kind == "switch" {
			closed := false
			c.SwitchClosed = &closed
		}
		if kind == "net-label" {
			if value == "" {
				value = id
			}
			return SetValue(c, value)
		}
		if c.Unit != "" {
			if value == "" {
				value = part.DefaultValue
			}
			c, err = SetValue(c, value)
			return c, err
		}
		if value != "" {
			return Component{}, fmt.Errorf("%s has no editable value", part.Name)
		}
		return c, nil
	}
	return Component{}, fmt.Errorf("unknown part kind; use parts.list")
}
func SetValue(c Component, text string) (Component, error) {
	if c.Kind == "net-label" {
		if !validNetLabel(text) {
			return c, fmt.Errorf("net label must be 1–32 ASCII letters/digits or _+-./, without spaces")
		}
		c.Label = text
		return c, nil
	}
	if c.Kind != "battery" && c.Kind != "dc-source" && c.Kind != "pulse-source" && c.Kind != "resistor" && c.Kind != "capacitor" && c.Kind != "inductor" && c.Kind != "potentiometer" && c.Kind != "zener" && c.Kind != "comparator" && c.Kind != "relay" && !IsSensorKind(c.Kind) && !IsRegulatorKind(c.Kind) && !IsWaveformSource(c.Kind) {
		return c, fmt.Errorf("this component has no editable value")
	}
	value, err := ParseValue(text, c.Unit)
	if err != nil {
		return c, err
	}
	if err := validateComponentValue(c.Kind, value); err != nil {
		return c, err
	}
	if c.Kind == "sine-source" {
		w := *c.Sine
		w.Amplitude = value
		if err := w.Validate(); err != nil {
			return c, err
		}
		c.Sine = &w
	}
	if c.Kind == "finite-pulse-source" {
		w := *c.Pulse
		w.HighVoltage = value
		if err := w.Validate(); err != nil {
			return c, err
		}
		c.Pulse = &w
	}

	c.Value = value
	return c, nil
}
func RotateLocal(p Point, degrees int) Point {
	switch degrees {
	case 90:
		return Point{-p.Y, p.X}
	case 180:
		return Point{-p.X, -p.Y}
	case 270:
		return Point{p.Y, -p.X}
	}
	return p
}
func TerminalPosition(c Component, name string) (Point, bool) {
	p := Point{}
	found := true
	switch c.Kind {
	case "pwm-controller", "shift-register", "piezo", "active-buzzer":
		p, found = ExtendedTerminalLocal(c.Kind, name)
	case "hbridge":
		p, found = HBridgeTerminalLocal(name)
	case "transformer":
		p, found = TransformerTerminalLocal(name)
	case "optocoupler":
		p, found = OptocouplerTerminalLocal(name)
	case "dc-motor", "battery", "dc-source", "pulse-source", "sine-source", "finite-pulse-source":
		switch name {
		case "positive":
			p = Point{0, -34}
			if c.Kind == "battery" || c.Kind == "dc-motor" {
				p.Y = -55
			}
		case "negative":
			p = Point{0, 34}
			if c.Kind == "battery" || c.Kind == "dc-motor" {
				p.Y = 55
			}
		default:
			found = false
		}
	case "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor":
		p, found = LogicTerminalLocal(c.Kind, name)
	case "bcd-decoder", "seven-segment":
		p, found = DisplayTerminalLocal(c.Kind, name)
	case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter":
		p, found = SequentialTerminalLocal(c.Kind, name)
	case "timer555":
		p, found = Timer555TerminalLocal(name)
	case "capacitor":
		switch name {
		case "1":
			p = Point{0, -32}
		case "2":
			p = Point{0, 32}
		default:
			found = false
		}
	case "potentiometer":
		switch name {
		case "1":
			p = Point{-55, 0}
		case "2":
			p = Point{55, 0}
		case "wiper":
			p = Point{0, -45}
		default:
			found = false
		}
	case "thermistor", "ldr", "pushbutton", "bouncing-pushbutton", "switch", "resistor", "inductor":
		switch name {
		case "1":
			p = Point{-55, 0}
		case "2":
			p = Point{55, 0}
		default:
			found = false
		}
	case "led", "diode", "zener", "schottky":
		switch name {
		case "anode":
			p = Point{0, -32}
		case "cathode":
			p = Point{0, 32}
		default:
			found = false
		}
	case "linear-regulator", "ldo-regulator":
		p, found = RegulatorTerminalLocal(name)
	case "relay", "comparator":
		p, found = ControlTerminalLocal(c.Kind, name)
	case "opamp":
		switch name {
		case "noninverting":
			p = Point{-55, -20}
		case "inverting":
			p = Point{-55, 20}
		case "out":
			p = Point{55, 0}
		case "vplus":
			p = Point{0, -45}
		case "vminus":
			p = Point{0, 45}
		default:
			found = false
		}
	case "nmos", "pmos":
		switch name {
		case "gate":
			p = Point{-40, 0}
		case "drain":
			p = Point{25, -35}
		case "source":
			p = Point{25, 35}
		default:
			found = false
		}
	case "npn", "pnp":
		switch name {
		case "base":
			p = Point{-40, 0}
		case "collector":
			p = Point{25, -35}
			if c.Kind == "pnp" {
				p.Y = 35
			}
		case "emitter":
			p = Point{25, 35}
			if c.Kind == "pnp" {
				p.Y = -35
			}
		default:
			found = false
		}
	case "junction", "net-label":
		found = name == "node"
	case "ground":
		found = name == "reference"
		p = Point{0, -20}
	default:
		found = false
	}
	p = RotateLocal(p, c.Rotation)
	return Point{p.X + c.Position.X, p.Y + c.Position.Y}, found
}
func (g Graph) TerminalPoint(ref string) (Point, bool) {
	c, name, ok := g.Terminal(ref)
	if !ok {
		return Point{}, false
	}
	return TerminalPosition(c, name)
}

type Net struct {
	Labels    []string `json:"labels,omitempty"`
	ID        string   `json:"id"`
	Terminals []string `json:"terminals"`
}
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
		case "shift-register", "piezo", "active-buzzer", "pwm-controller", "hbridge", "transformer", "optocoupler", "dc-motor", "linear-regulator", "ldo-regulator", "relay", "comparator", "pushbutton", "bouncing-pushbutton", "bcd-decoder", "seven-segment", "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter", "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor", "timer555", "switch":
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
			bipolarNetwork = bipolarNetwork || out.TerminalNet[q.ID+".emitter"] != out.TerminalNet[grounds[0].ID+".reference"] || out.TerminalNet[q.ID+".base"] == out.TerminalNet[q.ID+".collector"]
		}
		for _, source := range sources {
			if source.Kind == "dc-source" && out.TerminalNet[source.ID+".negative"] != out.TerminalNet[grounds[0].ID+".reference"] {
				floatingDC = true
			}
		}
	}
	if hasOpAmp || hasInductor || hasMOS || hasWaveforms || hasSwitch || hasDiode || hasPot || networkTopology || floatingDC || bipolarNetwork || len(transistors) > 0 && len(capacitors) > 0 {
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

type DrawingState struct {
	Tool      string `json:"tool"`
	Kind      string `json:"kind"`
	Position  Point  `json:"position"`
	WireStart string `json:"wire_start"`
	From      string `json:"from"`
	To        string `json:"to"`
	Value     string `json:"value"`
}

func DefaultDrawing() DrawingState {
	return DrawingState{Tool: "select", Kind: "dc-source", Position: Point{-200, 0}}
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
