package workbench

import "fmt"

const Timer555ModelID = "elab.generic-cmos-555.v1"
const Timer555Analysis = "555 timer network transient"
const Timer555EngineID = "elab.timer555-hybrid.v1"
const MaxNetworkTimers555 = 2

// Timer555Model is a functional educational CMOS timer, not a fitted TLC555.
// Resistors make pin currents and supply power explicit. The latch switches at
// localized input events; no output delay or internal charge is modeled.
type Timer555Model struct {
	ID                  string  `json:"id"`
	DividerResistance   float64 `json:"divider_segment_ohm"`
	InputResistance     float64 `json:"input_resistance_ohm"`
	OutputResistance    float64 `json:"output_resistance_ohm"`
	DischargeResistance float64 `json:"discharge_on_resistance_ohm"`
	OffResistance       float64 `json:"off_resistance_ohm"`
	SupplyResistance    float64 `json:"supply_resistance_ohm"`
	ResetThreshold      float64 `json:"reset_threshold_v"`
	MinimumSupply       float64 `json:"minimum_supply_v"`
	MaximumSupply       float64 `json:"maximum_supply_v"`
	Limitations         string  `json:"limitations"`
}

func Timer555DeviceModel() Timer555Model {
	return Timer555Model{Timer555ModelID, 100000, 1e9, 50, 10, 1e9, 50000, .7, 2, 12, "Generic CMOS 555 functional model: trigger below half control sets the latch; threshold above control resets it when trigger is not low; active-low reset has priority. Three 100 kΩ divider segments give nominal 1/3 and 2/3 supply thresholds. Fixed 50 Ω output, 10 Ω discharge, 1 GΩ input/off paths and 50 kΩ supply load. Supply below 2 V forces the latch low with output/discharge off; above 12 V refuses simulation. No physical part identity, rated output current, propagation delay, internal charge, noise, heating, tolerance or frequency qualification."}
}

var timer555Pins = [...]string{"gnd", "trigger", "out", "reset", "control", "threshold", "discharge", "vcc"}

type timer555Memory struct{ High, Powered bool }

func timer555Decision(v [8]float64, previous timer555Memory) (timer555Memory, error) {
	m := Timer555DeviceModel()
	span := v[7] - v[0]
	if span > m.MaximumSupply+1e-9 {
		return previous, fmt.Errorf("555 supply exceeds its 12 V educational model range")
	}
	if span < m.MinimumSupply {
		return timer555Memory{}, nil
	}
	next := timer555Memory{High: previous.High, Powered: true}
	switch {
	case v[3]-v[0] < m.ResetThreshold:
		next.High = false
	case v[1]-v[0] < (v[4]-v[0])/2:
		next.High = true
	case v[5] > v[4]:
		next.High = false
	}
	return next, nil
}

// Every current is positive into its pin. Each branch is passive, including
// the output's connection to its selected supply rail.
type timer555Evaluation struct {
	Current  [8]float64
	Jacobian [8][8]float64
}

func timer555Currents(v [8]float64, memory timer555Memory) timer555Evaluation {
	m := Timer555DeviceModel()
	e := timer555Evaluation{}
	branch := func(a, b int, r float64) {
		g := 1 / r
		i := (v[a] - v[b]) * g
		e.Current[a] += i
		e.Current[b] -= i
		e.Jacobian[a][a] += g
		e.Jacobian[a][b] -= g
		e.Jacobian[b][a] -= g
		e.Jacobian[b][b] += g
	}
	branch(7, 4, m.DividerResistance)
	branch(4, 0, 2*m.DividerResistance)
	branch(7, 0, m.SupplyResistance)
	for _, pin := range []int{1, 3, 5} {
		branch(pin, 0, m.InputResistance)
	}
	rail, r := 0, m.OffResistance
	if memory.Powered {
		r = m.OutputResistance
		if memory.High {
			rail = 7
		}
	}
	branch(2, rail, r)
	r = m.OffResistance
	if memory.Powered && !memory.High {
		r = m.DischargeResistance
	}
	branch(6, 0, r)
	return e
}
func timer555ModelForGraph(g Graph) *Timer555Model {
	for _, p := range g.Components {
		if p.Kind == "timer555" {
			m := Timer555DeviceModel()
			return &m
		}
	}
	return nil
}
func validateTimer555Model(g Graph, m *Timer555Model) error {
	if m != nil && *m != Timer555DeviceModel() {
		return fmt.Errorf("unsupported 555 model parameters/version")
	}
	if timer555ModelForGraph(g) != nil && m == nil {
		return fmt.Errorf("555 schematics require supported timer555_model parameters")
	}
	return nil
}
