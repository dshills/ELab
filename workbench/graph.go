package workbench

import (
	"fmt"
	"math"
	"reflect"
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
		out.Components[i].Thermal = cloneThermal(out.Components[i].Thermal)
		out.Components[i].Block = cloneBlock(out.Components[i].Block)
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
	compiled, _, diagnostic := g.Compile()
	return g.compiledSummary(compiled, diagnostic)
}

func (g Graph) compiledSummary(compiled Compiled, diagnostic *Diagnostic) Circuit {
	var c Circuit
	for _, p := range g.Components {
		if p.Kind == "dc-source" {
			c.Voltage = p.Value
		}
		if p.Kind == "resistor" {
			c.Resistance = p.Value
		}
	}
	c.Connected = diagnostic == nil
	if diagnostic == nil {
		return compiled.Circuit
	}
	return c
}
func ElectricalEqual(a, b Graph) bool {
	if a.hasBlocks() {
		var err error
		a, err = a.PhysicalGraph()
		if err != nil {
			return false
		}
	}
	if b.hasBlocks() {
		var err error
		b, err = b.PhysicalGraph()
		if err != nil {
			return false
		}
	}
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
		{Kind: "scr", Name: "SCR thyristor", Terminals: thyristorPins("scr"), Model: ThyristorModelID},
		{Kind: "triac", Name: "TRIAC", Terminals: thyristorPins("triac"), Model: ThyristorModelID},
		{Kind: "photodiode", Name: "Photodiode", Terminals: []string{"anode", "cathode"}, Model: PhotodiodeModelID},
		{Kind: "njfet", Name: "N-channel JFET", Terminals: append([]string{}, jfetPins...), Model: JFETModelID},
		{Kind: "pjfet", Name: "P-channel JFET", Terminals: append([]string{}, jfetPins...), Model: JFETModelID},
		{Kind: "current-source", Name: "DC / pulsed current source", Terminals: []string{"positive", "negative"}, Model: CurrentDeviceModelID},
		{Kind: "electronic-load", Name: "CC / CP electronic load", Terminals: []string{"positive", "negative"}, Model: CurrentDeviceModelID},
		{Kind: "shift-register", Name: "8-bit shift / latch register", Terminals: append([]string{}, shiftPins...), Model: ExtendedModelID},
		{Kind: "speaker", Name: "Loudspeaker", Terminals: []string{"positive", "negative"}, Model: ExtendedModelID},
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
	if c.Block != nil {
		p, ok := blockPinLocal(c, name)
		p = RotateLocal(p, c.Rotation)
		return Point{X: p.X + c.Position.X, Y: p.Y + c.Position.Y}, ok
	}
	p := Point{}
	found := true
	switch c.Kind {
	case "speaker", "pwm-controller", "shift-register", "piezo", "active-buzzer":
		p, found = ExtendedTerminalLocal(c.Kind, name)
	case "hbridge":
		p, found = HBridgeTerminalLocal(name)
	case "transformer":
		p, found = TransformerTerminalLocal(name)
	case "optocoupler":
		p, found = OptocouplerTerminalLocal(name)
	case "current-source", "electronic-load", "dc-motor", "battery", "dc-source", "pulse-source", "sine-source", "finite-pulse-source":
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
	case "photodiode", "led", "diode", "zener", "schottky":
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
	case "scr", "triac":
		p, found = ThyristorTerminalLocal(c.Kind, name)
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
	case "njfet", "pjfet", "nmos", "pmos":
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
	if c, pin, ok := g.Terminal(ref); ok && c.Block != nil && c.Block.Expanded {
		for _, p := range c.Block.Ports {
			if p.Name == pin {
				member, terminal, exists := g.Terminal(p.Terminal)
				if !exists || member.Block != nil || member.Kind == "circuit-block" {
					return Point{}, false
				}
				return TerminalPosition(member, terminal)
			}
		}
	}
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
