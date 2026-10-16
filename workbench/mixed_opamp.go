package workbench

import "math"

// OpAmpResult reports the powered five-terminal model. OutputCurrent is
// delivered by the output pin; input and supply currents enter their pins.
// OutputVoltage is relative to the circuit's explicit ground reference.
type OpAmpResult struct {
	ID                    string     `json:"id"`
	Model                 OpAmpModel `json:"model"`
	InputDifference       float64    `json:"input_difference_v"`
	OutputVoltage         float64    `json:"output_voltage_v"`
	SupplyVoltage         float64    `json:"supply_voltage_v"`
	NoninvertingCurrent   float64    `json:"noninverting_current_a"`
	InvertingCurrent      float64    `json:"inverting_current_a"`
	OutputCurrent         float64    `json:"output_delivered_current_a"`
	PositiveSupplyCurrent float64    `json:"positive_supply_current_a"`
	NegativeSupplyCurrent float64    `json:"negative_supply_current_a"`
	Power                 float64    `json:"power_w"`
	State                 string     `json:"state"`
}

func (n *mixedNetwork) opAmpPins(p Component) [opAmpTerminalCount]int {
	return [opAmpTerminalCount]int{
		n.pin(p, "noninverting"), n.pin(p, "inverting"), n.pin(p, "out"),
		n.pin(p, "vplus"), n.pin(p, "vminus"),
	}
}

func (n *mixedNetwork) opAmpVoltages(p Component, x []float64) opAmpVoltages {
	var volts opAmpVoltages
	for k, node := range n.opAmpPins(p) {
		volts[k] = voltage(x, node)
	}
	return volts
}

func (n *mixedNetwork) stampOpAmp(p Component, x []float64, add func(int, int, float64), residual []float64) {
	pins := n.opAmpPins(p)
	device := opAmpCurrents(n.opAmpVoltages(p, x))
	// Intentional pin joins (including follower feedback) sum both currents
	// and all derivative contributions into their shared physical KCL row.
	for row, node := range pins {
		if node >= 0 {
			residual[node] += device.current[row]
		}
		for column, other := range pins {
			add(node, other, device.jacobian[row][column])
		}
	}
}

func (n *mixedNetwork) opAmpResult(p Component, x []float64) OpAmpResult {
	v := n.opAmpVoltages(p, x)
	device := opAmpCurrents(v)
	power := 0.
	for pin, current := range device.current {
		// Express terminal power relative to V- to avoid cancellation from
		// arbitrary common voltage translation. Terminal currents sum to zero.
		power += (v[pin] - v[opAmpNegative]) * current
	}
	state := "powered"
	span := v[opAmpPositive] - v[opAmpNegative]
	if span == 0 {
		state = "unpowered"
	} else if span < 0 {
		state = "reversed-supply"
	}
	return OpAmpResult{
		ID: p.ID, Model: OpAmpDeviceModel(), InputDifference: v[0] - v[1],
		OutputVoltage: v[opAmpOutput], SupplyVoltage: span,
		NoninvertingCurrent: device.current[opAmpNoninverting], InvertingCurrent: device.current[opAmpInverting],
		OutputCurrent: -device.current[opAmpOutput], PositiveSupplyCurrent: device.current[opAmpPositive],
		NegativeSupplyCurrent: device.current[opAmpNegative], Power: power, State: state,
	}
}

// opAmpNewtonScale resolves the narrow high-gain transition while permitting
// outward growth into saturation. Outward proposals add at most half the
// existing differential magnitude plus 0.1 normalized input units. An inward
// proposal may approach zero in one step, crossing only that central interval.
// This changes only the proposed iterate: equations, accuracy, voltage and work
// bounds remain.
// Existing circuits take the original path and keep their arithmetic order.
func (n *mixedNetwork) opAmpNewtonScale(x, delta []float64, stages int, scale float64) float64 {
	if len(n.opamps) == 0 {
		return scale
	}
	size := len(x) / stages
	for stage := range stages {
		values := x[stage*size : (stage+1)*size]
		changes := delta[stage*size : (stage+1)*size]
		for _, p := range n.opamps {
			pins := n.opAmpPins(p)
			span := voltage(values, pins[opAmpPositive]) - voltage(values, pins[opAmpNegative])
			if span <= 0 {
				continue
			}
			width := .1 * span / (2 * OpAmpDeviceModel().OpenLoopGain)
			if width == 0 {
				continue // The supply-dependent active stage has underflowed off.
			}
			difference := voltage(values, pins[opAmpNoninverting]) - voltage(values, pins[opAmpInverting])
			change := voltage(changes, pins[opAmpNoninverting]) - voltage(changes, pins[opAmpInverting])
			allowed := width
			if change > 0 {
				allowed += math.Max(0, -difference) + .5*math.Max(0, difference)
			} else {
				allowed += math.Max(0, difference) + .5*math.Max(0, -difference)
			}
			if math.Abs(scale*change) > allowed {
				scale = math.Min(scale, allowed/math.Abs(change))
			}
		}
	}
	return scale
}
