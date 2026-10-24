package workbench

func partAliases(kind string) []string {
	switch kind {
	case "dc-source":
		return []string{"battery", "power supply", "voltage supply"}
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
	case "diode":
		return []string{"rectifier", "silicon diode", "junction diode"}
	case "switch":
		return []string{"toggle", "on/off", "SPST"}
	}
	return []string{}
}
func partLimitations(kind string) string {
	switch kind {
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
	case "opamp":
		return OpAmpDeviceModel().Limitations
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
