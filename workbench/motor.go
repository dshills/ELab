package workbench

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const motorSpeedScale = 10000.

func motorArmNode(id string) string    { return "$" + id + "_arm" }
func motorEMFNode(id string) string    { return "$" + id + "_emf" }
func motorSpeedNode(id string) string  { return "$" + id + "_speed" }
func motorInductorID(id string) string { return "$" + id + "_winding" }
func motorEMFID(id string) string      { return "$" + id + "_back_emf" }
func motorInertiaID(id string) string  { return "$" + id + "_inertia" }
func motorDefault() MotorConfig {
	return MotorConfig{Resistance: 8, Inductance: .01, Constant: .02, Inertia: 1e-6, Friction: 1e-5}
}
func validateMotor(m MotorConfig) error {
	for _, v := range []struct{ value, min, max float64 }{{m.Resistance, .1, 100}, {m.Inductance, 1e-5, 1}, {m.Constant, .001, 1}, {m.Inertia, 1e-7, .001}, {m.Friction, 1e-7, .01}, {m.Load, 0, .01}} {
		if !finite(v.value) || v.value < v.min || v.value > v.max {
			return fmt.Errorf("motor parameters exceed their declared electrical/mechanical domains")
		}
	}
	return nil
}

type MotorParams struct {
	ID         string      `json:"id"`
	Parameters MotorConfig `json:"parameters"`
}
type MotorLoadParams struct {
	ID   string  `json:"id"`
	Load float64 `json:"load_n_m_s_per_rad"`
}

func (s *Service) configureMotorLocked(v MotorParams) error {
	if err := validateMotor(v.Parameters); err != nil {
		return err
	}
	for i, p := range s.graph.Components {
		if p.ID == v.ID && p.Kind == "dc-motor" {
			p.Power = &PowerParameters{Motor: &v.Parameters}
			s.graph.Components[i] = p
			return nil
		}
	}
	return fmt.Errorf("choose a DC motor")
}
func (s *Service) setMotorLoadLocked(v MotorLoadParams) error {
	p, ok := s.graph.Component(v.ID)
	if !ok || p.Kind != "dc-motor" {
		return fmt.Errorf("choose a DC motor")
	}
	if !finite(v.Load) || v.Load < 0 || v.Load > .01 {
		return fmt.Errorf("motor load must be 0–0.01 N·m·s/rad")
	}
	if p.Power.Motor.Load == v.Load {
		if s.activeInputWorkLocked() && s.inputJob.work.MotorLoad != nil && s.inputJob.work.MotorLoad.ID == v.ID {
			s.cancelInputWorkLocked("superseded")
		}
		return nil
	}
	g := s.graph.Clone()
	applyMotorLoad(&g, v)
	if s.transient != nil && !s.transient.Stale && s.mixedState != nil && IsNetworkAnalysis(s.mixedState.net.c.Analysis) {
		return s.startInputWorkLocked("", nil, g, fmt.Sprintf("motor-load:%s:%.17g", v.ID, v.Load))
	}
	s.graph = g
	return nil
}
func applyMotorLoad(g *Graph, v MotorLoadParams) {
	for i, p := range g.Components {
		if p.ID == v.ID {
			p.Power = clonePower(p.Power)
			p.Power.Motor.Load = v.Load
			g.Components[i] = p
		}
	}
}
func setMotorWork(w *InputWork, event string) {
	p := strings.Split(event, ":")
	if len(p) != 3 || p[0] != "motor-load" {
		return
	}
	v, _ := strconv.ParseFloat(p[2], 64)
	w.Command = "motor.load.set"
	w.MotorLoad = &MotorLoadParams{p[1], v}
}
func lowerMotor(c *Compiled, p Component) []Component {
	m := p.Power.Motor
	positive, negative := c.TerminalNet[p.ID+".positive"], c.TerminalNet[p.ID+".negative"]
	nodes := []string{motorArmNode(p.ID), motorEMFNode(p.ID)}
	if !m.Locked {
		nodes = append(nodes, motorSpeedNode(p.ID))
	}
	for _, id := range nodes {
		c.Nets = append(c.Nets, Net{ID: id})
	}
	parts := []Component{}
	part := func(id, kind string, value float64, one, two string) Component {
		terminals := []string{"1", "2"}
		if kind == "dc-source" {
			terminals = []string{"positive", "negative"}
		}
		q := Component{ID: id, Kind: kind, Terminals: terminals, Value: value}
		c.TerminalNet[id+"."+terminals[0]] = one
		c.TerminalNet[id+"."+terminals[1]] = two
		return q
	}
	parts = append(parts, part("$"+p.ID+"_arm_r", "resistor", m.Resistance, positive, motorArmNode(p.ID)))
	l := part(motorInductorID(p.ID), "inductor", m.Inductance, motorArmNode(p.ID), motorEMFNode(p.ID))
	zero := 0.
	l.InitialCurrent = &zero
	parts = append(parts, l)
	parts = append(parts, part(motorEMFID(p.ID), "dc-source", 0, motorEMFNode(p.ID), negative))

	if !m.Locked {
		inertia := part(motorInertiaID(p.ID), "capacitor", m.Inertia*motorSpeedScale*motorSpeedScale, motorSpeedNode(p.ID), c.Ground)
		inertia.InitialVoltage = &zero
		parts = append(parts, inertia, part("$"+p.ID+"_drag", "resistor", 1/((m.Friction+m.Load)*motorSpeedScale*motorSpeedScale), motorSpeedNode(p.ID), c.Ground))
	}

	return parts
}
func (n *mixedNetwork) motorInductorIndex(id string) int {
	for k, p := range n.inductors {
		if p.ID == motorInductorID(id) {
			return n.inductorIndex(k)
		}
	}
	return -1
}
func (n *mixedNetwork) stampMotorCoupling(x []float64, add func(int, int, float64), f []float64) {
	for _, p := range n.authored.Components {
		if p.Kind != "dc-motor" {
			continue
		}
		m := p.Power.Motor
		speed := n.motorSpeedIndex(p)
		current := n.motorInductorIndex(p.ID)
		coupling := m.Constant * motorSpeedScale
		if speed >= 0 {
			f[speed] -= coupling * x[current]
			add(speed, current, -coupling)
		}
		for k, source := range n.sources {
			if source.ID == motorEMFID(p.ID) {
				row := len(n.nodes) + k
				f[row] -= coupling * voltage(x, speed)
				add(row, speed, -coupling)
			}
		}
	}
}

type MotorResult struct {
	ID               string  `json:"id"`
	Model            string  `json:"model"`
	State            string  `json:"state"`
	Voltage          float64 `json:"voltage_v"`
	Current          float64 `json:"armature_current_a"`
	Omega            float64 `json:"speed_rad_s"`
	RPM              float64 `json:"rpm"`
	BackEMF          float64 `json:"back_emf_v"`
	Torque           float64 `json:"torque_n_m"`
	StallCurrent     float64 `json:"blocked_rotor_reference_current_a"`
	WindingHeat      float64 `json:"winding_heat_w"`
	DragPower        float64 `json:"drag_power_w"`
	MagneticEnergy   float64 `json:"magnetic_energy_j"`
	RotationalEnergy float64 `json:"rotational_energy_j"`
	Power            float64 `json:"terminal_power_w"`
}

func (n *mixedNetwork) motorResult(p Component, x []float64) MotorResult {
	m := p.Power.Motor
	omega := voltage(x, n.motorSpeedIndex(p)) * motorSpeedScale
	i := x[n.motorInductorIndex(p.ID)]
	v := voltage(x, n.pin(p, "positive")) - voltage(x, n.pin(p, "negative"))
	state := "running"
	if m.Locked {
		state = "locked rotor"
	} else if math.Abs(omega) < 1 {
		state = "starting / near stall"
	}
	if i*omega < 0 {
		state = "regenerating / braking"
	}
	return MotorResult{p.ID, PowerModelID, state, v, i, omega, omega * 60 / (2 * math.Pi), m.Constant * omega, m.Constant * i, v / m.Resistance, m.Resistance * i * i, (m.Friction + m.Load) * omega * omega, .5 * m.Inductance * i * i, .5 * m.Inertia * omega * omega, v * i}
}
func (n *mixedNetwork) restoreMotorNodes(p Component, r *MixedResult, x []float64) error {
	for _, m := range r.Motors {
		if m.ID == p.ID {
			if !finite(m.Omega) || !finite(m.Current) {
				return fmt.Errorf("invalid motor storage state")
			}
			negative := voltage(x, n.pin(p, "negative"))
			positive := voltage(x, n.pin(p, "positive"))
			if !p.Power.Motor.Locked {
				x[n.motorSpeedIndex(p)] = m.Omega / motorSpeedScale
			}
			x[n.index[motorArmNode(p.ID)]] = positive - p.Power.Motor.Resistance*m.Current
			x[n.index[motorEMFNode(p.ID)]] = negative + p.Power.Motor.Constant*m.Omega
			x[n.motorInductorIndex(p.ID)] = m.Current
			return nil
		}
	}
	return fmt.Errorf("missing motor storage state")
}
func validateRecordedMotors(c Compiled, g Graph, r *MixedResult) error {
	ids := []string{}
	for _, m := range r.Motors {
		ids = append(ids, m.ID)
	}
	if err := recordedObjectIDs(g, "dc-motor", ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, m := range r.Motors {
		p, _ := g.Component(m.ID)
		e := n.motorResult(p, x)
		if m.Model != e.Model || m.State != e.State || p.Power.Motor.Locked && m.Omega != 0 {
			return fmt.Errorf("invalid motor model/state")
		}
		for _, v := range [][2]float64{{m.Voltage, e.Voltage}, {m.RPM, e.RPM}, {m.BackEMF, e.BackEMF}, {m.Torque, e.Torque}, {m.StallCurrent, e.StallCurrent}, {m.WindingHeat, e.WindingHeat}, {m.DragPower, e.DragPower}, {m.MagneticEnergy, e.MagneticEnergy}, {m.RotationalEnergy, e.RotationalEnergy}, {m.Power, e.Power}} {
			if !recordingNear(v[0], v[1]) {
				return fmt.Errorf("motor readings disagree with circuit and mechanical state")
			}
		}
	}
	return nil
}

func (n *mixedNetwork) motorSpeedIndex(p Component) int {
	if p.Power.Motor.Locked {
		return -1
	}
	return n.index[motorSpeedNode(p.ID)]
}
