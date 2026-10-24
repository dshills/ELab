package workbench

import "fmt"

const SequentialModelID = "elab.generic-sequential-cmos.v1"
const SequentialAnalysis = "Sequential logic transient"
const SequentialEngineID = "elab.sequential-hybrid.v1"

// SequentialModel is a functional model of single positive-edge devices. It
// does not copy a packaged IC's pinout, propagation delay or setup/hold limits.
type SequentialModel struct {
	ID               string  `json:"id"`
	InputResistance  float64 `json:"input_resistance_ohm"`
	OutputResistance float64 `json:"output_resistance_ohm"`
	OffResistance    float64 `json:"off_resistance_ohm"`
	SupplyResistance float64 `json:"supply_resistance_ohm"`
	LowFraction      float64 `json:"low_supply_fraction"`
	HighFraction     float64 `json:"high_supply_fraction"`
	MinimumSupply    float64 `json:"minimum_supply_v"`
	MaximumSupply    float64 `json:"maximum_supply_v"`
	Limitations      string  `json:"limitations"`
}

func SequentialDeviceModel() SequentialModel {
	return SequentialModel{SequentialModelID, 1e9, 50, 1e9, 1e6, .3, .7, 2, 12, "Generic positive-edge D/JK/T flip-flops and enabled synchronous binary/decade counters. Active-high asynchronous reset has priority. Clock/reset use 30/70% local-supply hysteresis; sampled data/enable must be valid low/high. Startup/restart clears state and establishes the present clock without inventing an edge. Power loss clears state; above 12 V refuses simulation. Explicit 50 Ω output paths, 1 GΩ input/off paths and 1 MΩ supply load. No package pinout, preset, setup/hold timing, propagation delay, metastability, intrinsic charge, input protection, output current limit, noise, heating or physical ratings. Counter bits change together; no ripple delay or interchangeable clock/enable behavior."}
}
func IsSequentialKind(kind string) bool {
	switch kind {
	case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter":
		return true
	}
	return false
}
func IsCounterKind(kind string) bool { return kind == "binary-counter" || kind == "decade-counter" }

var sequentialPinOrder = [...]string{"d", "j", "k", "t", "clk", "reset", "enable", "q", "nq", "q0", "q1", "q2", "q3", "vcc", "gnd"}

func sequentialPins(kind string) []string {
	switch kind {
	case "d-flipflop":
		return []string{"d", "clk", "reset", "q", "nq", "vcc", "gnd"}
	case "jk-flipflop":
		return []string{"j", "k", "clk", "reset", "q", "nq", "vcc", "gnd"}
	case "t-flipflop":
		return []string{"t", "clk", "reset", "q", "nq", "vcc", "gnd"}
	case "binary-counter", "decade-counter":
		return []string{"clk", "reset", "enable", "q0", "q1", "q2", "q3", "vcc", "gnd"}
	}
	return nil
}
func sequentialOutput(pin string) bool {
	switch pin {
	case "q", "nq", "q0", "q1", "q2", "q3":
		return true
	}
	return false
}
func sequentialModulus(kind string) int {
	if kind == "binary-counter" {
		return 16
	}
	if kind == "decade-counter" {
		return 10
	}
	return 2
}

// Memories are values in immutable compiled plans. Trial solves never change
// these fields, including clock arming on accepted falling edges.
type SequentialMemory struct {
	Value      int  `json:"value"`
	Powered    bool `json:"powered"`
	ClockKnown bool `json:"clock_known"`
	ClockHigh  bool `json:"clock_high"`
	ResetHigh  bool `json:"reset_high"`
}

func sequentialDecision(kind string, v [15]float64, old SequentialMemory, sample bool) (SequentialMemory, error) {
	m := SequentialDeviceModel()
	span := v[13] - v[14]
	if span > m.MaximumSupply+1e-9 {
		return old, fmt.Errorf("sequential supply exceeds its 12 V educational model range")
	}
	if span < m.MinimumSupply {
		return SequentialMemory{}, nil
	}
	next := old
	if !old.Powered {
		next = SequentialMemory{Powered: true}
	}
	level := func(pin int) string { return logicLevel(v[pin]-v[14], span) }
	reset, clock := level(5), level(4)
	if !old.Powered && reset == "intermediate" {
		return old, fmt.Errorf("reset must be a valid logic level when sequential power is established")
	}
	if reset != "intermediate" {
		next.ResetHigh = reset == "high"
	}
	rising := false
	if clock != "intermediate" {
		high := clock == "high"
		rising = old.Powered && old.ClockKnown && !old.ClockHigh && high
		next.ClockKnown = true
		next.ClockHigh = high
	}
	if next.ResetHigh {
		next.Value = 0
		return next, nil
	}
	if !rising || !sample {
		return next, nil
	}
	read := func(pin int) (bool, error) {
		l := level(pin)
		if l == "intermediate" {
			return false, fmt.Errorf("%s is intermediate at a rising clock edge", sequentialPinOrder[pin])
		}
		return l == "high", nil
	}
	switch kind {
	case "d-flipflop":
		d, err := read(0)
		if err != nil {
			return old, err
		}
		next.Value = 0
		if d {
			next.Value = 1
		}
	case "t-flipflop":
		t, err := read(3)
		if err != nil {
			return old, err
		}
		if t {
			next.Value = 1 - old.Value
		}
	case "jk-flipflop":
		j, err := read(1)
		if err != nil {
			return old, err
		}
		k, err := read(2)
		if err != nil {
			return old, err
		}
		switch {
		case j && k:
			next.Value = 1 - old.Value
		case j:
			next.Value = 1
		case k:
			next.Value = 0
		}
	case "binary-counter", "decade-counter":
		enable, err := read(6)
		if err != nil {
			return old, err
		}
		if enable {
			next.Value = (old.Value + 1) % sequentialModulus(kind)
		}
	}
	return next, nil
}

type sequentialEvaluation struct {
	Current  [15]float64
	Jacobian [15][15]float64
}

func sequentialCurrents(kind string, v [15]float64, memory SequentialMemory) sequentialEvaluation {
	m := SequentialDeviceModel()
	e := sequentialEvaluation{}
	branch := func(a, b int, r float64) {
		g := 1 / r
		i := (v[a] - v[b]) * g
		e.Current[a] += i
		e.Current[b] -= i
		e.Jacobian[a][a] += g
		e.Jacobian[b][b] += g
		e.Jacobian[a][b] -= g
		e.Jacobian[b][a] -= g
	}
	branch(13, 14, m.SupplyResistance)
	for index, pin := range sequentialPinOrder {
		present := false
		for _, p := range sequentialPins(kind) {
			present = present || p == pin
		}
		if !present || pin == "vcc" || pin == "gnd" {
			continue
		}
		if !sequentialOutput(pin) {
			branch(index, 14, m.InputResistance)
			continue
		}
		high := memory.Value == 1
		if pin == "nq" {
			high = !high
		}
		if index >= 9 && index <= 12 {
			high = memory.Value&(1<<(index-9)) != 0
		}
		rail, r := 14, m.OffResistance
		if memory.Powered {
			r = m.OutputResistance
			if high {
				rail = 13
			}
		}
		branch(index, rail, r)
	}
	return e
}
func sequentialModelForGraph(g Graph) *SequentialModel {
	for _, p := range g.Components {
		if IsSequentialKind(p.Kind) {
			m := SequentialDeviceModel()
			return &m
		}
	}
	return nil
}
func validateSequentialModel(g Graph, m *SequentialModel) error {
	if m != nil && *m != SequentialDeviceModel() {
		return fmt.Errorf("unsupported sequential model parameters/version")
	}
	if sequentialModelForGraph(g) != nil && m == nil {
		return fmt.Errorf("sequential schematics require supported sequential_model parameters")
	}
	return nil
}
