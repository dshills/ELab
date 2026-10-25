package workbench

func partAliases(kind string) []string {
	switch kind {
	case "hbridge":
		return []string{"H bridge", "motor driver", "forward", "reverse", "brake", "coast", "PWM", "bidirectional"}
	case "thermistor":
		return []string{"temperature", "sensor", "NTC", "thermistor", "heat"}
	case "ldr":
		return []string{"light", "sensor", "photocell", "photoresistor", "LDR", "lux"}
	case "linear-regulator", "ldo-regulator":
		return []string{"regulator", "supply", "power", "dropout", "LDO", "linear"}
	case "relay":
		return []string{"relay", "coil", "SPDT", "contacts", "flyback", "electromagnet"}
	case "comparator":
		return []string{"comparator", "threshold", "hysteresis", "Schmitt", "analog to digital", "detector"}
	case "pushbutton", "bouncing-pushbutton":
		return []string{"button", "momentary", "normally open", "bounce", "debounce", "press", "release"}
	case "bcd-decoder", "seven-segment":
		return []string{"display", "BCD", "digit", "decoder", "seven segment", "7 segment", "common cathode"}
	case "d-flipflop", "jk-flipflop", "t-flipflop":
		return []string{"sequential", "digital", "CMOS", "clock", "flip-flop", "register", "divider"}
	case "binary-counter":
		return []string{"sequential", "digital", "CMOS", "clock", "counter", "4-bit", "binary", "divider"}
	case "decade-counter":
		return []string{"sequential", "digital", "CMOS", "clock", "counter", "4-bit", "BCD", "divide by ten"}
	case "logic-not":
		return []string{"logic", "digital", "CMOS", "boolean", "inverter", "inverting gate"}
	case "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor":
		return []string{"logic", "digital", "CMOS", "boolean", "gate"}
	case "transformer":
		return []string{"coupled inductors", "mutual inductance", "turns ratio", "winding", "dot polarity", "power supply"}
	case "optocoupler":
		return []string{"optoisolator", "phototransistor", "optical isolation", "CTR", "isolated switch"}
	case "dc-motor":
		return []string{"motor", "DC motor", "PWM", "RPM", "brushed", "fan", "back EMF"}
	case "battery":
		return []string{"cell", "battery pack", "charge", "internal resistance", "voltage sag"}
	case "dc-source":
		return []string{"power supply", "voltage supply", "ideal voltage source"}
	case "resistor":
		return []string{"resistance", "current limiter"}
	case "capacitor":
		return []string{"cap", "charge storage"}
	case "inductor":
		return []string{"coil", "choke", "magnetic energy"}
	case "sine-source":
		return []string{"sine wave", "AC source", "signal generator"}
	case "finite-pulse-source":
		return []string{"square wave", "pulse generator", "PWM", "clock"}
	case "pulse-source":
		return []string{"ideal square wave", "clock"}
	case "led":
		return []string{"LED", "light emitting diode", "indicator"}
	case "pmos":
		return []string{"MOSFET", "PMOS", "P-channel", "FET", "field effect transistor", "high-side switch"}
	case "nmos":
		return []string{"MOSFET", "NMOS", "N-channel", "FET", "field effect transistor", "low-side switch"}
	case "npn":
		return []string{"bipolar transistor", "BJT", "transistor", "emitter follower"}
	case "pnp":
		return []string{"bipolar transistor", "BJT", "transistor", "high-side switch", "complementary"}
	case "timer555":
		return []string{"555", "CMOS timer", "TLC555", "NE555", "oscillator", "one-shot", "monostable", "astable"}
	case "opamp":
		return []string{"op-amp", "operational amplifier", "buffer", "voltage follower", "amplifier"}
	case "ground":
		return []string{"ground", "voltage reference"}
	case "junction":
		return []string{"branch", "join", "connection dot"}
	case "net-label":
		return []string{"label", "net name", "named connection"}
	case "potentiometer":
		return []string{"pot", "variable resistor", "rheostat", "volume knob"}
	case "zener":
		return []string{"breakdown diode", "voltage clamp", "shunt regulator"}
	case "schottky":
		return []string{"barrier diode", "low voltage rectifier"}
	case "diode":
		return []string{"rectifier", "silicon diode", "junction diode"}
	case "switch":
		return []string{"toggle", "on/off", "SPST"}
	}
	return []string{}
}
func partLimitations(kind string) string {
	switch kind {
	case "hbridge":
		return HBridgeDeviceModel().Limitations
	case "relay":
		return ControlledDeviceModel().Limitations
	case "comparator":
		return ControlledDeviceModel().Limitations
	case "pushbutton", "bouncing-pushbutton":
		return ControlledDeviceModel().Limitations
	case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter":
		return SequentialDeviceModel().Limitations
	case "bcd-decoder", "seven-segment":
		return DigitalDisplayModel().Limitations
	case "linear-regulator", "ldo-regulator":
		return VoltageRegulatorModel().Limitations
	case "thermistor", "ldr":
		return ResistiveSensorModel().Limitations
	case "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor":
		return LogicDeviceModel().Limitations
	case "transformer":
		return PowerDeviceModel().Transformer
	case "optocoupler":
		return PowerDeviceModel().Optocoupler
	case "dc-motor":
		return PowerDeviceModel().Motor
	case "battery":
		return PowerDeviceModel().Battery
	case "dc-source":
		return "Ideal DC voltage source; positive minus negative sets its voltage, and either terminal may be above/below ground in a referenced network. No output resistance, current limit, battery discharge, noise, heating, ratings or physical part identity."
	case "resistor":
		return "Ideal constant resistance; no tolerance, heating, parasitics, ratings or physical part identity. Current is positive from pin 1 to pin 2."
	case "capacitor":
		return "Ideal capacitor with configurable initial voltage; no ESR, leakage, polarity, heating, ratings or physical part identity. Voltage and current use pin 1 relative to pin 2."
	case "inductor":
		return "Ideal constant inductance with configurable signed initial current; voltage pin 1 minus pin 2, current into pin 1/out of pin 2, energy L I² / 2. No winding resistance, saturation, mutual coupling, parasitic capacitance, heating, ratings or physical part identity. Add external resistance and a return path for switching current."
	case "ground":
		return "Common zero-volt reference; this symbol establishes the simulation reference, not protective earth or a physical grounding design."
	case "pulse-source":
		return "Ideal square source with instantaneous edges and shared experiment phase timing; no output resistance, current limit, noise, ratings or physical part identity."
	case "sine-source", "finite-pulse-source":
		return SourceWaveformModel().Limitations
	case "led":
		return Model().Limitations
	case "pmos":
		return PMOSDeviceModel().Limitations
	case "nmos":
		return NMOSDeviceModel().Limitations
	case "npn":
		return TransistorModel().Limitations
	case "pnp":
		return PNPDeviceModel().Limitations
	case "timer555":
		return Timer555DeviceModel().Limitations
	case "opamp":
		return OpAmpDeviceModel().Limitations
	case "zener":
		return ZenerDeviceModel().Limitations
	case "schottky":
		return SchottkyDeviceModel().Limitations
	case "diode":
		return SiliconDiodeModel().Limitations
	case "potentiometer":
		return LinearPotentiometerModel().Limitations
	case "switch":
		return SwitchDeviceModel().Limitations
	case "junction":
		return "Explicit electrical join; geometry/crossings alone do not connect wires. This helper has no physical device model."
	case "net-label":
		return "Exact case-sensitive names join attached nets. A name alone never establishes ground. This helper has no physical device model."
	}
	return "Unsupported component kind."
}
