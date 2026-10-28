package workbench

import (
	"fmt"
	"math"
	"strings"
)

const PowerModelID = "elab.power-and-isolation.v1"

// PowerModel identifies generic educational laws, not a fitted manufacturer part.
type PowerModel struct {
	ID          string `json:"id"`
	Battery     string `json:"battery"`
	Motor       string `json:"motor"`
	Optocoupler string `json:"optocoupler"`
	Transformer string `json:"transformer"`
}

func PowerDeviceModel() PowerModel {
	return PowerModel{PowerModelID,
		"Linear charge reservoir: OCV=Vfull*q/Q; Q=3600*capacity_Ah; ESR voltage sag. Equivalent stored energy=Q*Vfull*SOC^2/2. No chemistry, temperature, ageing, self-discharge, protection or physical ratings. Positive delivered current discharges; externally forced charging is an ideal reversible reservoir, not a charger model. Startup SOC is 0..1; live SOC outside that range is reported, not hidden.",
		"Generic brushed DC motor: armature RL, reciprocal Ke=Kt electromechanical coupling, rotor inertia, viscous friction and load. No brushes, commutation ripple, thermal model, gearbox or hardware ratings.",
		"Generic LED/phototransistor optocoupler: separate conductive input/output ports, CTR and smooth saturation. No isolation-voltage rating, capacitance, propagation delay, temperature or ageing model.",
		"Linear mutually coupled windings with series resistance and dot polarity. No core saturation, hysteresis, losses beyond winding resistance or isolation rating. Coupling below one gives a positive-definite inductance matrix."}
}

type BatteryConfig struct {
	CapacityAh float64 `json:"capacity_ah"`
	Resistance float64 `json:"internal_resistance_ohm"`
	Charge     float64 `json:"initial_charge_fraction"`
}
type MotorConfig struct {
	Locked     bool    `json:"locked_rotor"`
	Resistance float64 `json:"armature_resistance_ohm"`
	Inductance float64 `json:"armature_inductance_h"`
	Constant   float64 `json:"ke_v_s_per_rad"`
	Inertia    float64 `json:"inertia_kg_m2"`
	Friction   float64 `json:"friction_n_m_s_per_rad"`
	Load       float64 `json:"load_n_m_s_per_rad"`
}
type OptocouplerConfig struct {
	CTR float64 `json:"current_transfer_ratio"`
}
type TransformerConfig struct {
	Inductance          float64 `json:"primary_inductance_h"`
	Ratio               float64 `json:"secondary_primary_turns_ratio"`
	Coupling            float64 `json:"coupling"`
	PrimaryResistance   float64 `json:"primary_resistance_ohm"`
	SecondaryResistance float64 `json:"secondary_resistance_ohm"`
	Reversed            bool    `json:"reversed_secondary"`
}
type PowerParameters struct {
	Battery     *BatteryConfig     `json:"battery,omitempty"`
	Motor       *MotorConfig       `json:"motor,omitempty"`
	Optocoupler *OptocouplerConfig `json:"optocoupler,omitempty"`
	Transformer *TransformerConfig `json:"transformer,omitempty"`
}

func clonePower(in *PowerParameters) *PowerParameters {
	if in == nil {
		return nil
	}
	out := *in
	if in.Battery != nil {
		v := *in.Battery
		out.Battery = &v
	}
	if in.Motor != nil {
		v := *in.Motor
		out.Motor = &v
	}
	if in.Optocoupler != nil {
		v := *in.Optocoupler
		out.Optocoupler = &v
	}
	if in.Transformer != nil {
		v := *in.Transformer
		out.Transformer = &v
	}
	return &out
}
func IsPowerKind(k string) bool {
	return k == "battery" || k == "dc-motor" || k == "optocoupler" || k == "transformer"
}
func powerModelForGraph(g Graph) *PowerModel {
	for _, p := range g.Components {
		if IsPowerKind(p.Kind) {
			m := PowerDeviceModel()
			return &m
		}
	}
	return nil
}
func validatePowerModel(g Graph, m *PowerModel) error {
	if m != nil && *m != PowerDeviceModel() {
		return fmt.Errorf("unsupported power device model")
	}
	if powerModelForGraph(g) != nil && m == nil {
		return fmt.Errorf("power devices require power_model parameters")
	}
	return nil
}
func defaultPower(p *Component) {
	if p.Kind == "transformer" {
		c := transformerDefault()
		p.Power = &PowerParameters{Transformer: &c}
	}
	if p.Kind == "optocoupler" {
		p.Power = &PowerParameters{Optocoupler: &OptocouplerConfig{1}}
	}
	if p.Kind == "dc-motor" {
		m := motorDefault()
		p.Power = &PowerParameters{Motor: &m}
	}
	if p.Kind == "battery" {
		p.Power = &PowerParameters{Battery: &BatteryConfig{.1, 1, 1}}
	}
}
func validatePower(p Component) error {
	if p.Power == nil {
		if IsPowerKind(p.Kind) {
			return fmt.Errorf("power device requires explicit parameters")
		}
		return nil
	}
	if p.Kind == "transformer" && p.Power.Transformer != nil && p.Power.Battery == nil && p.Power.Motor == nil && p.Power.Optocoupler == nil {
		return validateTransformer(*p.Power.Transformer)
	}
	if p.Kind == "optocoupler" && p.Power.Optocoupler != nil && p.Power.Battery == nil && p.Power.Motor == nil && p.Power.Transformer == nil {
		return validateOptocoupler(*p.Power.Optocoupler)
	}
	if p.Kind == "dc-motor" && p.Power.Motor != nil && p.Power.Battery == nil && p.Power.Optocoupler == nil && p.Power.Transformer == nil {
		return validateMotor(*p.Power.Motor)
	}
	if p.Kind != "battery" || p.Power.Battery == nil || p.Power.Motor != nil || p.Power.Optocoupler != nil || p.Power.Transformer != nil {
		return fmt.Errorf("power parameters do not match component kind")
	}
	b := p.Power.Battery
	if !finite(b.CapacityAh) || b.CapacityAh < 1e-6 || b.CapacityAh > 100 || !finite(b.Resistance) || b.Resistance < .01 || b.Resistance > 1000 || !finite(b.Charge) || b.Charge < 0 || b.Charge > 1 {
		return fmt.Errorf("battery capacity must be 1 µAh–100 Ah, internal resistance 0.01–1000 Ω and startup charge 0–1")
	}
	return nil
}

type BatteryParams struct {
	ID         string  `json:"id"`
	CapacityAh float64 `json:"capacity_ah"`
	Resistance float64 `json:"internal_resistance_ohm"`
	Charge     float64 `json:"initial_charge_fraction"`
}

func (s *Service) setBatteryLocked(v BatteryParams) error {
	for i, p := range s.graph.Components {
		if p.ID == v.ID && p.Kind == "battery" {
			p.Power = &PowerParameters{Battery: &BatteryConfig{v.CapacityAh, v.Resistance, v.Charge}}
			if err := validatePower(p); err != nil {
				return err
			}
			s.graph.Components[i] = p
			return nil
		}
	}
	return fmt.Errorf("choose a battery")
}
func privatePart(id string) bool           { return strings.HasPrefix(id, "$") }
func batteryStorageID(id string) string    { return "$" + id + "_charge" }
func batteryResistanceID(id string) string { return "$" + id + "_esr" }
func batteryNode(id string) string         { return "$" + id + "_ocv" }

// Lower composite components only in a private solver plan. Authored components,
// ports, net names and saved graphs retain their original identities.
func lowerPowerDevices(c Compiled) (Compiled, Graph) {
	original := c.MixedGraph
	if powerModelForGraph(original) == nil && extendedModelForGraph(original) == nil && !hasTransistorProfiles(original) {
		return c, Graph{}
	}
	c.MixedGraph = original.Clone()
	c.TerminalNet = cloneStringMap(c.TerminalNet)
	c.Nets = append([]Net(nil), c.Nets...)
	parts := []Component{}
	for _, p := range original.Components {
		parts = append(parts, lowerThermal(&c, p)...)
		if p.Kind == "photodiode" {
			parts = append(parts, lowerPhotodiode(&c, p)...)
			continue
		}
		if p.Kind == "shift-register" {
			parts = append(parts, lowerShift(&c, p)...)
			continue
		}
		if p.Kind == "speaker" {
			parts = append(parts, lowerSpeaker(&c, p)...)
			continue
		}
		if isSoundKind(p.Kind) {
			parts = append(parts, lowerSound(&c, p)...)
			continue
		}
		if hasTransistorProfile(p) {
			parts = append(parts, lowerTransistorProfile(&c, p)...)
			continue
		}
		if hasDynamics(p) {
			parts = append(parts, lowerOpAmpDynamics(&c, p)...)
			continue
		}
		if p.Kind == "transformer" {
			parts = append(parts, lowerTransformer(&c, p)...)
			continue
		}
		if p.Kind == "dc-motor" {
			parts = append(parts, lowerMotor(&c, p)...)
			continue
		}
		if p.Kind != "battery" {
			parts = append(parts, p)
			continue
		}
		b := p.Power.Battery
		initial := p.Value * b.Charge
		cap := Component{ID: batteryStorageID(p.ID), Kind: "capacitor", Terminals: []string{"1", "2"}, Value: b.CapacityAh * 3600 / p.Value, InitialVoltage: &initial}
		resistor := Component{ID: batteryResistanceID(p.ID), Kind: "resistor", Terminals: []string{"1", "2"}, Value: b.Resistance}
		node := batteryNode(p.ID)
		c.Nets = append(c.Nets, Net{ID: node})
		c.TerminalNet[cap.ID+".1"] = node
		c.TerminalNet[cap.ID+".2"] = c.TerminalNet[p.ID+".negative"]
		c.TerminalNet[resistor.ID+".1"] = node
		c.TerminalNet[resistor.ID+".2"] = c.TerminalNet[p.ID+".positive"]
		if c.Analysis == NetworkDCAnalysis {
			cap.Kind = "dc-source"
			cap.Terminals = []string{"positive", "negative"}
			cap.Value = initial
			c.TerminalNet[cap.ID+".positive"] = node
			c.TerminalNet[cap.ID+".negative"] = c.TerminalNet[p.ID+".negative"]
		}
		parts = append(parts, cap, resistor)
	}
	c.MixedGraph.Components = parts
	return c, original
}
func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

type BatteryResult struct {
	ID                 string  `json:"id"`
	Model              string  `json:"model"`
	State              string  `json:"state"`
	Charge             float64 `json:"charge_fraction"`
	RemainingAh        float64 `json:"remaining_ah"`
	OpenCircuitVoltage float64 `json:"open_circuit_voltage_v"`
	Voltage            float64 `json:"terminal_voltage_v"`
	Current            float64 `json:"delivered_current_a"`
	InternalHeat       float64 `json:"internal_heat_w"`
	Energy             float64 `json:"stored_energy_j"`
}

func (n *mixedNetwork) batteryResult(p Component, x []float64) BatteryResult {
	b := p.Power.Battery
	ocv := voltage(x, n.index[batteryNode(p.ID)]) - voltage(x, n.pin(p, "negative"))
	v := voltage(x, n.pin(p, "positive")) - voltage(x, n.pin(p, "negative"))
	i := (ocv - v) / b.Resistance
	charge := ocv / p.Value
	state := "discharging"
	if math.Abs(i) < 1e-10 {
		state = "idle"
	} else if i < 0 {
		state = "charging"
	}
	if charge < 1e-6 {
		state = "empty"
	}
	if charge > 1+1e-8 || charge < -1e-8 {
		state = "outside nominal charge range"
	}
	return BatteryResult{p.ID, PowerModelID, state, charge, charge * b.CapacityAh, ocv, v, i, i * i * b.Resistance, .5 * b.CapacityAh * 3600 / p.Value * ocv * ocv}
}
func (n *mixedNetwork) publishPower(r *MixedResult, x []float64) {
	for _, p := range n.authored.Components {
		if p.Kind == "transformer" {
			r.Transformers = append(r.Transformers, n.transformerResult(p, x))
		}
		if p.Kind == "dc-motor" {
			r.Motors = append(r.Motors, n.motorResult(p, x))
		}
		if p.Kind == "battery" {
			b := n.batteryResult(p, x)
			r.Batteries = append(r.Batteries, b)
			r.Sources = append(r.Sources, BranchResult{p.ID, b.Current, b.Voltage, b.Current * b.Voltage})
		}
	}
}
func validateRecordedPower(c Compiled, g Graph, r *MixedResult) error {
	ids := []string{}
	for _, b := range r.Batteries {
		ids = append(ids, b.ID)
	}
	if err := recordedObjectIDs(g, "battery", ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, b := range r.Batteries {
		p, _ := g.Component(b.ID)
		e := n.batteryResult(p, x)
		if b.Model != e.Model || b.State != e.State {
			return fmt.Errorf("invalid battery model/state")
		}
		for _, v := range [][2]float64{{b.Charge, e.Charge}, {b.RemainingAh, e.RemainingAh}, {b.OpenCircuitVoltage, e.OpenCircuitVoltage}, {b.Voltage, e.Voltage}, {b.Current, e.Current}, {b.InternalHeat, e.InternalHeat}, {b.Energy, e.Energy}} {
			if !recordingNear(v[0], v[1]) {
				return fmt.Errorf("battery readings disagree with stored charge and terminal voltages")
			}
		}
	}
	return nil
}
func (n *mixedNetwork) restorePowerNodes(r *MixedResult, x []float64) error {
	if err := n.restorePhotodiodes(r, x); err != nil {
		return err
	}
	if err := n.restoreThermals(r, x); err != nil {
		return err
	}
	if err := n.restoreSpeakers(r, x); err != nil {
		return err
	}
	if err := n.restoreJunctions(r, x); err != nil {
		return err
	}
	if err := n.restoreDynamics(r, x); err != nil {
		return err
	}
	if err := n.restoreShift(r, x); err != nil {
		return err
	}
	if err := n.restoreSounds(r, x); err != nil {
		return err
	}
	for _, p := range n.authored.Components {
		if p.Kind == "transformer" {
			if err := n.restoreTransformerNodes(p, r, x); err != nil {
				return err
			}
		}
		if p.Kind == "dc-motor" {
			if err := n.restoreMotorNodes(p, r, x); err != nil {
				return err
			}
		}
		if p.Kind == "battery" {
			found := false
			for _, b := range r.Batteries {
				if b.ID == p.ID {
					if !finite(b.OpenCircuitVoltage) || math.Abs(b.OpenCircuitVoltage) > maxMixedNodeVoltage {
						return fmt.Errorf("invalid stored battery voltage")
					}
					x[n.index[batteryNode(p.ID)]] = voltage(x, n.pin(p, "negative")) + b.OpenCircuitVoltage
					found = true
				}
			}
			if !found {
				return fmt.Errorf("missing battery state")
			}
		}
	}
	return nil
}

func (n *mixedNetwork) usesCollocation() bool {
	if len(n.inductors) > 0 {
		return true
	}
	for _, p := range n.authored.Components {
		if p.Kind == "photodiode" && p.Device != nil && p.Device.Photo != nil && p.Device.Photo.Capacitance > 0 || p.Kind == "piezo" || hasDynamics(p) || hasJunctionStorage(p) || p.Kind == "battery" || p.Kind == "dc-motor" {
			return !n.dc
		}
	}
	return false
}

type powerResourceUsage struct{ parts, caps, inductors, resistors, sources, nodes int }

func powerResources(g Graph) (u powerResourceUsage) {
	for _, p := range g.Components {
		if p.Kind == "photodiode" && p.Device.Photo.Capacitance > 0 {
			u.parts++
			u.caps++
		}
		if hasThermal(p) && p.Thermal.SelfHeating {
			u.parts += 2
			u.caps++
			u.resistors++
			u.nodes++
		}
		if p.Kind == "speaker" {
			u.parts += 5
			u.inductors += 2
			u.caps++
			u.resistors += 2
			u.nodes += 2
		}
		if hasTransistorProfile(p) {
			for _, v := range []float64{p.Device.Bipolar.CBE, p.Device.Bipolar.CBC} {
				if v > 0 {
					u.parts++
					u.caps++
				}
			}
		}
		if hasDynamics(p) {
			u.parts++
			u.caps++
			u.nodes++
		}
		if p.Kind == "shift-register" {
			u.parts += 17
			u.nodes++
		}
		if p.Kind == "piezo" {
			u.parts++
			u.caps++
		}
		switch p.Kind {
		case "transformer":
			u.parts++
			u.inductors += 2
		case "battery":
			u.parts++
			u.caps++
			u.resistors++
			u.nodes++
		case "dc-motor":
			u.parts += 2
			u.inductors++
			u.resistors++
			u.sources++
			u.nodes += 2
			if !p.Power.Motor.Locked {
				u.parts += 2
				u.caps++
				u.resistors++
				u.nodes++
			}
		}
	}
	return
}
