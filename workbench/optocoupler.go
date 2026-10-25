package workbench

import (
	"fmt"
	"math"
)

var optoPins = []string{"anode", "cathode", "collector", "emitter"}

const optoSaturationVoltage = .1
const optoDarkConductance = 1e-9

type OptocouplerParams struct {
	ID  string  `json:"id"`
	CTR float64 `json:"current_transfer_ratio"`
}

func validateOptocoupler(c OptocouplerConfig) error {
	if !finite(c.CTR) || c.CTR < .01 || c.CTR > 5 {
		return fmt.Errorf("optocoupler CTR must be 0.01–5 A/A")
	}
	return nil
}
func (s *Service) configureOptocouplerLocked(v OptocouplerParams) error {
	if err := validateOptocoupler(OptocouplerConfig{v.CTR}); err != nil {
		return err
	}
	for i, p := range s.graph.Components {
		if p.ID == v.ID && p.Kind == "optocoupler" {
			p.Power = &PowerParameters{Optocoupler: &OptocouplerConfig{v.CTR}}
			s.graph.Components[i] = p
			return nil
		}
	}
	return fmt.Errorf("choose an optocoupler")
}

// Each port balances its own terminal currents. Optical coupling adds a control
// derivative, never a conductive connection across the isolation barrier.
func optoCurrents(vf, vce, ctr float64) (input, output, inputSlope, transferSlope, outputSlope float64) {
	input, inputSlope = (&mixedNetwork{hasOptocoupler: true}).ledCurrent(vf)
	light := math.Max(input, 0) * ctr
	output = vce * optoDarkConductance
	outputSlope = optoDarkConductance
	if vce >= 0 {
		h := math.Hypot(vce, optoSaturationVoltage)
		factor := vce / h
		output += light * factor
		outputSlope += light * optoSaturationVoltage * optoSaturationVoltage / (h * h * h)
		if input > 0 {
			transferSlope = ctr * inputSlope * factor
		}
	}
	return
}
func (n *mixedNetwork) stampOptocoupler(p Component, x []float64, add func(int, int, float64), f []float64) {
	pins := [4]int{}
	for i, pin := range optoPins {
		pins[i] = n.pin(p, pin)
	}
	vf := voltage(x, pins[0]) - voltage(x, pins[1])
	vce := voltage(x, pins[2]) - voltage(x, pins[3])
	input, output, gi, gm, goff := optoCurrents(vf, vce, p.Power.Optocoupler.CTR)
	for _, port := range []struct {
		a, b int
		i, g float64
	}{{pins[0], pins[1], input, gi}, {pins[2], pins[3], output, goff}} {
		if port.a >= 0 {
			f[port.a] += port.i
		}
		if port.b >= 0 {
			f[port.b] -= port.i
		}
		add(port.a, port.a, port.g)
		add(port.a, port.b, -port.g)
		add(port.b, port.a, -port.g)
		add(port.b, port.b, port.g)
	}
	add(pins[2], pins[0], gm)
	add(pins[2], pins[1], -gm)
	add(pins[3], pins[0], -gm)
	add(pins[3], pins[1], gm)
}

type OptocouplerResult struct {
	ID               string  `json:"id"`
	Model            string  `json:"model"`
	State            string  `json:"state"`
	InputVoltage     float64 `json:"led_voltage_v"`
	InputCurrent     float64 `json:"led_current_a"`
	OutputVoltage    float64 `json:"collector_emitter_voltage_v"`
	OutputCurrent    float64 `json:"collector_current_a"`
	AvailableCurrent float64 `json:"available_photocurrent_a"`
	EffectiveCTR     float64 `json:"effective_current_transfer_ratio"`
	InputPower       float64 `json:"input_power_w"`
	OutputPower      float64 `json:"output_power_w"`
	Power            float64 `json:"power_w"`
}

func (n *mixedNetwork) optoResult(p Component, x []float64) OptocouplerResult {
	vf := voltage(x, n.pin(p, "anode")) - voltage(x, n.pin(p, "cathode"))
	vce := voltage(x, n.pin(p, "collector")) - voltage(x, n.pin(p, "emitter"))
	input, output, _, _, _ := optoCurrents(vf, vce, p.Power.Optocoupler.CTR)
	available := math.Max(input, 0) * p.Power.Optocoupler.CTR
	effective := 0.
	if input > 0 {
		effective = (output - vce*optoDarkConductance) / input
	}
	state := "dark"
	if input > 1e-9 {
		state = "active"
		if vce < .3 {
			state = "saturated / limited output headroom"
		}
	}
	return OptocouplerResult{p.ID, PowerModelID, state, vf, input, vce, output, available, effective, vf * input, vce * output, vf*input + vce*output}
}
func validateRecordedOptocouplers(c Compiled, g Graph, r *MixedResult) error {
	ids := []string{}
	for _, p := range r.Optocouplers {
		ids = append(ids, p.ID)
	}
	if err := recordedObjectIDs(g, "optocoupler", ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, p := range r.Optocouplers {
		part, _ := g.Component(p.ID)
		want := n.optoResult(part, x)
		if p.ID != want.ID || p.Model != want.Model || p.State != want.State {
			return fmt.Errorf("invalid optocoupler identity/model/state")
		}
		for _, v := range [][2]float64{{p.InputVoltage, want.InputVoltage}, {p.InputCurrent, want.InputCurrent}, {p.OutputVoltage, want.OutputVoltage}, {p.OutputCurrent, want.OutputCurrent}, {p.AvailableCurrent, want.AvailableCurrent}, {p.EffectiveCTR, want.EffectiveCTR}, {p.InputPower, want.InputPower}, {p.OutputPower, want.OutputPower}, {p.Power, want.Power}} {
			if !recordingNear(v[0], v[1]) {
				return fmt.Errorf("optocoupler readings disagree with separate port laws")
			}
		}
	}
	return nil
}
func OptocouplerTerminalLocal(pin string) (Point, bool) {
	switch pin {
	case "anode":
		return Point{-70, -30}, true
	case "cathode":
		return Point{-70, 30}, true
	case "collector":
		return Point{70, -30}, true
	case "emitter":
		return Point{70, 30}, true
	}
	return Point{}, false
}
func OptocouplerSymbolPaths() [][]Point {
	return [][]Point{
		{{-48, -52}, {48, -52}, {48, 52}, {-48, 52}, {-48, -52}}, {{0, -43}, {0, -30}}, {{0, -20}, {0, -7}}, {{0, 3}, {0, 16}}, {{0, 26}, {0, 43}},
		{{-70, -30}, {-28, -30}, {-28, -10}}, {{-70, 30}, {-28, 30}, {-28, 10}}, {{-40, -10}, {-16, -10}, {-28, 10}, {-40, -10}}, {{-40, 10}, {-16, 10}},
		{{14, -17}, {14, 17}}, {{14, -9}, {40, -30}, {70, -30}}, {{14, 9}, {40, 30}, {70, 30}}, {{30, 28}, {40, 30}, {36, 20}},
		{{-18, -18}, {3, -28}}, {{-4, -29}, {3, -28}, {0, -21}}, {{-18, 0}, {3, -10}}, {{-4, -11}, {3, -10}, {0, -3}},
	}
}
func OptocouplerExampleGraph() Graph {
	b := newExampleBuilder()
	b.part("VIN", "dc-source", "5", -500, 0, 0)
	b.ground("GNDIN", "VIN.negative", -500, 180)
	b.part("S1", "switch", "", -320, -120, 0)
	b.wire("VIN.positive", "S1.1")
	b.part("RIN", "resistor", "1k", -120, -120, 0)
	b.wire("S1.2", "RIN.1")
	b.part("OC1", "optocoupler", "", 100, 0, 0)
	b.wire("RIN.2", "OC1.anode")
	b.ground("GNDLED", "OC1.cathode", -100, 180)
	b.part("VOUT", "dc-source", "5", 620, 0, 0)
	b.ground("GNDOUT", "VOUT.negative", 620, 180)
	b.part("RL", "resistor", "1k", 420, -260, 90)
	b.wire("VOUT.positive", "RL.1")
	b.part("D1", "led", "", 420, -100, 0)
	b.wire("RL.2", "D1.anode")
	b.wire("D1.cathode", "OC1.collector")
	b.ground("GNDE", "OC1.emitter", 220, 180)
	return b.graph
}
