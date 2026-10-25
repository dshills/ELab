package workbench

import (
	"fmt"
	"math"
	"strings"
)

const ControlModelID = "elab.generic-controls.v1"
const ControlAnalysis = "Controlled devices transient"
const ControlEngineID = "elab.controls-hybrid.v1"

type ControlModel struct {
	ID                string  `json:"id"`
	OnResistance      float64 `json:"on_resistance_ohm"`
	OffResistance     float64 `json:"off_resistance_ohm"`
	BounceInterval    float64 `json:"bounce_interval_s"`
	BounceTransitions int     `json:"bounce_transitions"`
	ClickSeconds      float64 `json:"click_duration_s"`
	InputResistance   float64 `json:"input_resistance_ohm"`
	OutputResistance  float64 `json:"output_resistance_ohm"`
	SupplyResistance  float64 `json:"supply_resistance_ohm"`
	MinimumSupply     float64 `json:"minimum_supply_v"`
	MaximumSupply     float64 `json:"maximum_supply_v"`
	RelayInductance   float64 `json:"relay_inductance_h"`
	RelayResistance   float64 `json:"relay_resistance_ohm"`
	RelayPickup       float64 `json:"relay_pickup_current_a"`
	RelayDropout      float64 `json:"relay_dropout_current_a"`
	Limitations       string  `json:"limitations"`
}

func ControlledDeviceModel() ControlModel {
	return ControlModel{ControlModelID, 1, 1e9, .001, 4, .05, 1e9, 50, 1e6, 2, 12, .1, 100, .03, .01, "Generic educational controls. Normally open momentary buttons; clean or deterministic four-transition 1 ms bounce after press/release; Click holds for 50 ms of simulation time. Powered comparator has explicit differential hysteresis and rail-connected 50 Ω output paths. Relay has a 100 Ω /100 mH unpolarized coil, 30 mA pickup /10 mA dropout and isolated SPDT contacts. No contact arcing/wear, physical mechanical delay, comparator propagation/setup timing, noise, heat, protection, current limit or hardware ratings. External resistance/storage determine circuit behavior; fixed bounce is illustrative, not a measured switch."}
}
func IsButtonKind(kind string) bool { return kind == "pushbutton" || kind == "bouncing-pushbutton" }
func IsControlKind(kind string) bool {
	return IsButtonKind(kind) || kind == "comparator" || kind == "relay"
}
func controlPins(kind string) []string {
	switch kind {
	case "comparator":
		return []string{"noninverting", "inverting", "out", "vcc", "gnd"}
	case "relay":
		return []string{"coil1", "coil2", "common", "no", "nc"}
	}
	return []string{"1", "2"}
}
func controlModelForGraph(g Graph) *ControlModel {
	for _, p := range g.Components {
		if IsControlKind(p.Kind) {
			m := ControlledDeviceModel()
			return &m
		}
	}
	return nil
}
func validateControlModel(g Graph, m *ControlModel) error {
	if m != nil && *m != ControlledDeviceModel() {
		return fmt.Errorf("unsupported controlled-device model")
	}
	if controlModelForGraph(g) != nil && m == nil {
		return fmt.Errorf("controlled devices require control_model parameters")
	}
	return nil
}

type ControlMemory struct {
	High          bool    `json:"high"`
	Powered       bool    `json:"powered"`
	Pressed       bool    `json:"pressed"`
	ContactClosed bool    `json:"contact_closed"`
	Bouncing      bool    `json:"bouncing"`
	StartedAt     float64 `json:"started_at_s"`
	ReleaseAt     float64 `json:"release_at_s"`
}

func buttonAt(kind string, m ControlMemory, at float64) ControlMemory {
	model := ControlledDeviceModel()
	if m.ReleaseAt > 0 && controlTimeReached(at, m.ReleaseAt) {
		m.Pressed = false
		m.StartedAt = m.ReleaseAt
		m.ReleaseAt = 0
		m.Bouncing = kind == "bouncing-pushbutton"
	}
	contact := m.Pressed
	if m.Bouncing {
		for k := 1; k <= model.BounceTransitions; k++ {
			if controlTimeReached(at, m.StartedAt+float64(k)*model.BounceInterval) {
				contact = !contact
			}
		}
		if controlTimeReached(at, m.StartedAt+float64(model.BounceTransitions)*model.BounceInterval) {
			m.Bouncing = false
		}
	}
	m.ContactClosed = contact
	return m
}
func nextButtonEvent(kind string, m ControlMemory, at float64) float64 {
	next := math.Inf(1)
	if m.ReleaseAt > 0 && !controlTimeReached(at, m.ReleaseAt) {
		next = m.ReleaseAt
	}
	if m.Bouncing {
		for k := 1; k <= ControlledDeviceModel().BounceTransitions; k++ {
			when := m.StartedAt + float64(k)*ControlledDeviceModel().BounceInterval
			if !controlTimeReached(at, when) {
				next = math.Min(next, when)
			}
		}
	}
	return next
}
func controlDecision(p Component, v [5]float64, m ControlMemory, at, current float64) (ControlMemory, error) {
	if IsButtonKind(p.Kind) {
		return buttonAt(p.Kind, m, at), nil
	}
	model := ControlledDeviceModel()
	if p.Kind == "comparator" {
		span := v[3] - v[4]
		if span > model.MaximumSupply+1e-9 {
			return m, fmt.Errorf("comparator supply exceeds 12 V model range")
		}
		if span < model.MinimumSupply {
			return ControlMemory{}, nil
		}
		next := m
		next.Powered = true
		if v[0] >= v[1]+p.Value/2 {
			next.High = true
		} else if v[0] <= v[1]-p.Value/2 {
			next.High = false
		}
		return next, nil
	}
	if p.Kind == "relay" {
		next := m
		if math.Abs(current) >= model.RelayPickup {
			next.High = true
		} else if math.Abs(current) <= model.RelayDropout {
			next.High = false
		}
		return next, nil
	}
	return m, nil
}
func (n *mixedNetwork) controlMemoryFor(p Component) ControlMemory {
	for k, c := range n.controls {
		if c.ID == p.ID {
			return n.controlMemory[k]
		}
	}
	return ControlMemory{}
}
func (n *mixedNetwork) controlVoltages(p Component, x []float64) (v [5]float64) {
	for k, pin := range p.Terminals {
		v[k] = voltage(x, n.pin(p, pin))
	}
	return
}

type controlEvaluation struct {
	Current  [5]float64
	Jacobian [5][5]float64
}

func controlCurrents(p Component, v [5]float64, m ControlMemory) controlEvaluation {
	e := controlEvaluation{}
	model := ControlledDeviceModel()
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
	if IsButtonKind(p.Kind) {
		r := model.OffResistance
		if m.ContactClosed {
			r = model.OnResistance
		}
		branch(0, 1, r)
	}
	if p.Kind == "comparator" {
		branch(0, 4, model.InputResistance)
		branch(1, 4, model.InputResistance)
		branch(3, 4, model.SupplyResistance)
		rail, r := 4, model.OffResistance
		if m.Powered {
			r = model.OutputResistance
			if m.High {
				rail = 3
			}
		}
		branch(2, rail, r)
	}
	if p.Kind == "relay" {
		on, off := model.OffResistance, model.OnResistance
		if m.High {
			on, off = off, on
		}
		branch(2, 3, on)
		branch(2, 4, off)
	}
	return e
}
func (n *mixedNetwork) stampControl(p Component, x []float64, add func(int, int, float64), f []float64) {
	e := controlCurrents(p, n.controlVoltages(p, x), n.controlMemoryFor(p))
	for k, pin := range p.Terminals {
		row := n.pin(p, pin)
		if row >= 0 {
			f[row] += e.Current[k]
		}
		for j, other := range p.Terminals {
			add(row, n.pin(p, other), e.Jacobian[k][j])
		}
	}
}

type ControlResult struct {
	ID          string        `json:"id"`
	Kind        string        `json:"kind"`
	Model       ControlModel  `json:"model"`
	Memory      ControlMemory `json:"memory"`
	State       string        `json:"state"`
	Voltage     float64       `json:"voltage_v"`
	Current     float64       `json:"current_a"`
	PinCurrents [5]float64    `json:"pin_currents_a"`
	Power       float64       `json:"power_w"`
	Energy      float64       `json:"energy_j"`
}

func (n *mixedNetwork) controlResult(p Component, x []float64) ControlResult {
	v := n.controlVoltages(p, x)
	m := n.controlMemoryFor(p)
	e := controlCurrents(p, v, m)
	r := ControlResult{ID: p.ID, Kind: p.Kind, Model: ControlledDeviceModel(), Memory: m, PinCurrents: e.Current, Voltage: v[0] - v[1], Current: e.Current[0], State: "released"}
	if IsButtonKind(p.Kind) {
		if m.Pressed {
			r.State = "pressed"
		}
		if m.Bouncing {
			r.State += " · bouncing"
		}
	}
	if p.Kind == "comparator" {
		r.State = "low"
		if m.High {
			r.State = "high"
		}
		if !m.Powered {
			r.State = "unpowered"
		}
		r.Voltage = v[2] - v[4]
		r.Current = -e.Current[2]
	}
	if p.Kind == "relay" {
		r.Current = n.relayCurrent(p, x)
		r.PinCurrents[0] = r.Current
		r.PinCurrents[1] = -r.Current
		r.Energy = .5 * p.Value * r.Current * r.Current
		r.State = "released"
		if m.High {
			r.State = "energized"
		}
	}
	for k, i := range e.Current {
		r.Power += v[k] * i
	}
	if p.Kind == "relay" {
		r.Power += r.Current * r.Current * inductorWindingResistance(p)
	}
	return r
}
func (s *mixedState) controlChanges() ([]ControlMemory, bool, error) {
	next := append([]ControlMemory{}, s.net.controlMemory...)
	changed := false
	for k, p := range s.net.controls {
		m, err := controlDecision(p, s.net.controlVoltages(p, s.x), next[k], s.result.Time, s.net.relayCurrent(p, s.x))
		if err != nil {
			return nil, false, fmt.Errorf("%s: %w", p.ID, err)
		}
		changed = changed || m != next[k]
		next[k] = m
	}
	return next, changed, nil
}
func (n *mixedNetwork) inheritControls(old *mixedNetwork) {
	for i, p := range n.controls {
		for j, q := range old.controls {
			if p.ID == q.ID && p.Kind == q.Kind {
				n.controlMemory[i] = old.controlMemory[j]
			}
		}
	}
}
func (n *mixedNetwork) nextControlEvent(at float64) float64 {
	next := math.Inf(1)
	for k, p := range n.controls {
		if IsButtonKind(p.Kind) {
			next = math.Min(next, nextButtonEvent(p.Kind, n.controlMemory[k], at))
		}
	}
	return next
}

func controlTimeReached(at, event float64) bool { return at >= event || sameSampleTime(at, event) }

func inductorWindingResistance(p Component) float64 {
	if privatePart(p.ID) && p.Power != nil && p.Power.Transformer != nil {
		if strings.HasSuffix(p.ID, "_secondary") {
			return p.Power.Transformer.SecondaryResistance
		}
		return p.Power.Transformer.PrimaryResistance
	}
	if p.Kind == "relay" {
		return ControlledDeviceModel().RelayResistance
	}
	return 0
}
func (n *mixedNetwork) relayCurrent(p Component, x []float64) float64 {
	if p.Kind != "relay" {
		return 0
	}
	for k, q := range n.inductors {
		if q.ID == p.ID {
			index := n.inductorIndex(k)
			if index < len(x) {
				return x[index]
			}
		}
	}
	return 0
}
