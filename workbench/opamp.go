package workbench

import "math"

const OpAmpModelID = "elab.generic-opamp.v1"

// OpAmpModel describes the fixed educational five-terminal macromodel. Its
// numerical constants are ELab choices, not a particular physical device.
type OpAmpModel struct {
	ID                     string  `json:"id"`
	OpenLoopGain           float64 `json:"open_loop_gain_v_per_v"`
	DifferentialResistance float64 `json:"differential_resistance_ohm"`
	OutputResistance       float64 `json:"output_resistance_ohm"`
	ActiveCurrentLimit     float64 `json:"active_output_current_limit_a"`
	QuiescentCurrent       float64 `json:"quiescent_current_a"`
	SupplySoftVoltage      float64 `json:"supply_soft_voltage_v"`
	OutputBleedResistance  float64 `json:"output_bleed_resistance_ohm"`
	Limitations            string  `json:"limitations"`
}

// OpAmpDeviceModel returns an owned description of the fixed educational model.
func OpAmpDeviceModel() OpAmpModel {
	return OpAmpModel{
		ID: OpAmpModelID, OpenLoopGain: 1e5, DifferentialResistance: 1e7,
		OutputResistance: 50, ActiveCurrentLimit: .02, QuiescentCurrent: .001,
		SupplySoftVoltage: .1, OutputBleedResistance: 1e9,
		Limitations: "Generic five-pin educational macromodel with differential input resistance, midpoint-centered soft voltage limiting, finite output resistance, bounded active output current, supply-dependent enable and explicit supply currents. A declared 1 Gohm output-to-V- bleed remains with power off or reversed; differential input resistance also remains. No internal bandwidth, slew rate, offset/bias mismatch, common-mode restrictions, protection diodes, noise, thermal behavior, ratings or named physical device. External storage supplies circuit timing; this model cannot qualify physical amplifier speed or stability.",
	}
}

// These indices are internal stamp coordinates, not persisted terminal numbers.
const (
	opAmpNoninverting = iota
	opAmpInverting
	opAmpOutput
	opAmpPositive
	opAmpNegative
	opAmpTerminalCount
)

type opAmpVoltages [opAmpTerminalCount]float64

type opAmpEvaluation struct {
	// All five currents are positive into the device. Supply and output
	// derivatives sum to zero, just as the terminal currents do.
	current      [opAmpTerminalCount]float64
	jacobian     [opAmpTerminalCount][opAmpTerminalCount]float64
	target       float64
	enable       float64
	activeOutput float64
	bleedOutput  float64
}

// opAmpTarget is defined only for positive supply span. The derivative of tanh
// uses exp(-2*abs(z)), retaining a useful derivative after tanh rounds to +/-1.
func opAmpTarget(v opAmpVoltages, gain float64) (float64, [opAmpTerminalCount]float64) {
	span := v[opAmpPositive] - v[opAmpNegative]
	z := 2 * gain * (v[opAmpNoninverting] - v[opAmpInverting]) / span
	t := math.Tanh(z)
	e := math.Exp(-2 * math.Abs(z))
	sech2 := 4 * e / ((1 + e) * (1 + e))
	zsech2 := 0.
	if !math.IsInf(z, 0) {
		zsech2 = z * sech2
	}
	d := [opAmpTerminalCount]float64{}
	d[opAmpNoninverting] = gain * sech2
	d[opAmpInverting] = -d[opAmpNoninverting]
	d[opAmpPositive] = .5 * (1 + t - zsech2)
	d[opAmpNegative] = 1 - d[opAmpPositive]
	midpoint := .5 * (v[opAmpPositive] + v[opAmpNegative])
	return midpoint + .5*span*t, d
}

// opAmpCurrents evaluates the memoryless current law and its analytic stamp.
// Callers supply finite voltages within the network's existing node bounds.
func opAmpCurrents(v opAmpVoltages) opAmpEvaluation {
	m := OpAmpDeviceModel()
	var result opAmpEvaluation
	var dEnable, dActive [opAmpTerminalCount]float64
	span := v[opAmpPositive] - v[opAmpNegative]
	result.target = v[opAmpNegative]
	if span > 0 {
		soft2 := m.SupplySoftVoltage * m.SupplySoftVoltage
		denominator := span*span + soft2
		result.enable = span * span / denominator
		dEnable[opAmpPositive] = 2 * span * soft2 / (denominator * denominator)
		dEnable[opAmpNegative] = -dEnable[opAmpPositive]
		target, derivative := opAmpTarget(v, m.OpenLoopGain)
		result.target = target
		u := (v[opAmpOutput] - target) / (m.OutputResistance * m.ActiveCurrentLimit)
		h := math.Hypot(1, u)
		limited := m.ActiveCurrentLimit * u / h
		result.activeOutput = result.enable * limited
		conductance := 1 / (m.OutputResistance * h * h * h)
		for k := range dActive {
			output := 0.
			if k == opAmpOutput {
				output = 1
			}
			dActive[k] = dEnable[k]*limited + result.enable*conductance*(output-derivative[k])
		}
	}
	gInput := 1 / m.DifferentialResistance
	result.current[opAmpNoninverting] = (v[opAmpNoninverting] - v[opAmpInverting]) * gInput
	result.current[opAmpInverting] = -result.current[opAmpNoninverting]
	result.jacobian[opAmpNoninverting][opAmpNoninverting] = gInput
	result.jacobian[opAmpNoninverting][opAmpInverting] = -gInput
	result.jacobian[opAmpInverting][opAmpNoninverting] = -gInput
	result.jacobian[opAmpInverting][opAmpInverting] = gInput

	gBleed := 1 / m.OutputBleedResistance
	result.bleedOutput = (v[opAmpOutput] - v[opAmpNegative]) * gBleed
	result.current[opAmpOutput] = result.bleedOutput + result.activeOutput
	result.jacobian[opAmpOutput] = dActive
	result.jacobian[opAmpOutput][opAmpOutput] += gBleed
	result.jacobian[opAmpOutput][opAmpNegative] -= gBleed

	// The active output current is drawn from V+ when sourcing, and returned
	// to V- when sinking. Bleed returns to V-. IQ flows directly V+ -> V-.
	result.current[opAmpPositive] = m.QuiescentCurrent*result.enable + math.Max(-result.activeOutput, 0)
	share := 0.
	if result.activeOutput < 0 {
		share = 1
	} else if result.activeOutput == 0 {
		// Central generalized derivative at the source/sink handover.
		share = .5
	}
	for k := range dActive {
		result.jacobian[opAmpPositive][k] = m.QuiescentCurrent*dEnable[k] - share*dActive[k]
		result.jacobian[opAmpNegative][k] = -result.jacobian[opAmpOutput][k] - result.jacobian[opAmpPositive][k]
	}
	result.current[opAmpNegative] = -result.current[opAmpOutput] - result.current[opAmpPositive]
	return result
}
