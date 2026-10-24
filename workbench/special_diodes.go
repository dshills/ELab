package workbench

const (
	SchottkyModelID = "elab.generic-schottky-diode.v1"
	ZenerModelID    = "elab.generic-zener-diode.v1"
)

// ZenerModel describes a forward junction and a continuous piecewise-linear
// reverse knee. The per-component value is the knee voltage, not a rating or a
// test-current specification. DynamicResistance includes reverse series loss.
type ZenerModel struct {
	ID                string     `json:"id"`
	Forward           DiodeModel `json:"forward"`
	DynamicResistance float64    `json:"dynamic_resistance_ohm"`
	Limitations       string     `json:"limitations"`
}

func SchottkyDeviceModel() DiodeModel {
	return DiodeModel{SchottkyModelID, 2e-7, 1.05, 2, 27, "Generic Schottky junction with illustrative lower forward drop, 0.2 µA reverse saturation leakage and 2 Ω series resistance at 27°C. No reverse breakdown, capacitance, charge storage, thermal dynamics, ratings or physical part identity."}
}
func ZenerDeviceModel() ZenerModel {
	return ZenerModel{ZenerModelID, SiliconDiodeModel(), 10, "Generic Zener: silicon forward junction plus a continuous linear reverse knee at the component's editable voltage, with 10 Ω reverse dynamic resistance. Value is the zero-breakdown-current knee, not a specified test-current voltage. No soft knee, temperature coefficient, noise, capacitance, charge storage, heating, power rating or physical part identity."}
}
func IsDiodeKind(kind string) bool {
	return kind == "diode" || kind == "schottky" || kind == "zener"
}

// Current is positive anode to cathode; conductance is dI/dV. Both signs
// reverse in Zener breakdown, so absorbed power remains positive. At the knee
// use the blocking-side derivative; the continuous current is unambiguous.
func deviceDiodeCurrent(part Component, v float64) (i, g float64) {
	if part.Kind == "schottky" {
		return junctionDiodeCurrent(SchottkyDeviceModel(), v)
	}
	i, g = siliconDiodeCurrent(v)
	if part.Kind == "zener" && v < -part.Value {
		r := ZenerDeviceModel().DynamicResistance
		i += (v + part.Value) / r
		g += 1 / r
	}
	return i, g
}
