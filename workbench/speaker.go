package workbench

import (
	"fmt"
	"math"
)

const SpeakerModelID = "elab.loudspeaker-impedance.v1"
const speakerDescription = SpeakerModelID + ": Educational passive electrical equivalent: series coil R/L and parallel motional R/L/C; C=1/((2*pi*fs)^2*Lm). Constant parameters, zero-energy startup. No calibrated SPL, cone displacement, enclosure, radiation, frequency-dependent losses, heating, excursion or manufacturer identity. Audio preview is explicit normalized terminal voltage, not acoustic prediction."

type SpeakerConfig struct {
	Version            string  `json:"version"`
	Resistance         float64 `json:"coil_resistance_ohm"`
	Inductance         float64 `json:"coil_inductance_h"`
	Resonance          float64 `json:"resonance_hz"`
	MotionalResistance float64 `json:"motional_resistance_ohm"`
	MotionalInductance float64 `json:"motional_inductance_h"`
}
type SpeakerParams struct {
	ID         string        `json:"id"`
	Parameters SpeakerConfig `json:"parameters"`
}

func DefaultSpeakerConfig() SpeakerConfig {
	return SpeakerConfig{SpeakerModelID, 6, .0005, 100, 30, .02}
}
func validateSpeaker(v SpeakerConfig) error {
	if v.Version != SpeakerModelID || !finite(v.Resistance) || v.Resistance < 1 || v.Resistance > 100 || !finite(v.Inductance) || v.Inductance < 1e-5 || v.Inductance > .1 || !finite(v.Resonance) || v.Resonance < 20 || v.Resonance > 1000 || !finite(v.MotionalResistance) || v.MotionalResistance < 1 || v.MotionalResistance > 1000 || !finite(v.MotionalInductance) || v.MotionalInductance < .0001 || v.MotionalInductance > 1 {
		return fmt.Errorf("speaker requires supported version, coil R 1–100 ohm/L 10 µH–100 mH, resonance 20–1000 Hz, motional R 1–1000 ohm/L 100 µH–1 H")
	}
	return nil
}

// Requires a validated compiled component; Graph.Compile and recording
// admission reject missing/unsupported parameters before any lowering.
func speakerCapacitance(p Component) float64 {
	c := p.Device.Speaker
	return 1 / (math.Pow(2*math.Pi*c.Resonance, 2) * c.MotionalInductance)
}
func speakerPrivate(id, suffix string) string { return "$" + id + "_speaker_" + suffix }
func hasSpeakers(g Graph) bool {
	for _, p := range g.Components {
		if p.Kind == "speaker" {
			return true
		}
	}
	return false
}
func lowerSpeaker(c *Compiled, p Component) []Component {
	v := p.Device.Speaker
	coil, motion := speakerPrivate(p.ID, "coil"), speakerPrivate(p.ID, "motion")
	c.Nets = append(c.Nets, Net{ID: coil}, Net{ID: motion})
	positive, negative := c.TerminalNet[p.ID+".positive"], c.TerminalNet[p.ID+".negative"]
	parts := []Component{p}
	for _, q := range []struct {
		suffix, kind, a, b string
		value              float64
	}{{"rc", "resistor", positive, coil, v.Resistance}, {"lc", "inductor", coil, motion, v.Inductance}, {"rm", "resistor", motion, negative, v.MotionalResistance}, {"lm", "inductor", motion, negative, v.MotionalInductance}, {"cm", "capacitor", motion, negative, speakerCapacitance(p)}} {
		part := Component{ID: speakerPrivate(p.ID, q.suffix), Kind: q.kind, Value: q.value, Terminals: []string{"1", "2"}}
		zero := 0.
		if q.kind == "capacitor" {
			part.InitialVoltage = &zero
		}
		if q.kind == "inductor" {
			part.InitialCurrent = &zero
		}
		parts = append(parts, part)
		c.TerminalNet[part.ID+".1"], c.TerminalNet[part.ID+".2"] = q.a, q.b
	}
	return parts
}

type SpeakerResult struct {
	ID               string  `json:"id"`
	Model            string  `json:"model"`
	Voltage          float64 `json:"voltage_v"`
	Current          float64 `json:"current_a"`
	Power            float64 `json:"terminal_power_w"`
	CoilHeat         float64 `json:"coil_heat_w"`
	MotionalHeat     float64 `json:"motional_loss_w"`
	Energy           float64 `json:"stored_energy_j"`
	MotionalVoltage  float64 `json:"motional_voltage_v"`
	MotionalCurrent  float64 `json:"motional_inductor_current_a"`
	CapacitorCurrent float64 `json:"motional_capacitor_current_a"`
}

func (n *mixedNetwork) speakerNodes(p Component) (coil, motion int, ok bool) {
	coil, cok := n.index[speakerPrivate(p.ID, "coil")]
	motion, mok := n.index[speakerPrivate(p.ID, "motion")]
	return coil, motion, cok && mok && coil >= 0 && motion >= 0 && coil < len(n.nodes) && motion < len(n.nodes)
}
func invalidSpeakerResult(id string) SpeakerResult {
	return SpeakerResult{ID: id, Model: SpeakerModelID, Voltage: math.NaN(), Current: math.NaN(), Energy: math.NaN(), CapacitorCurrent: math.NaN()}
}
func (n *mixedNetwork) speakerResult(p Component, x []float64) SpeakerResult {
	if p.Device == nil || p.Device.Speaker == nil {
		return invalidSpeakerResult(p.ID)
	}
	_, motion, ok := n.speakerNodes(p)
	if !ok || motion >= len(x) {
		return invalidSpeakerResult(p.ID)
	}
	v := p.Device.Speaker
	r := SpeakerResult{ID: p.ID, Model: SpeakerModelID, Voltage: voltage(x, n.pin(p, "positive")) - voltage(x, n.pin(p, "negative")), MotionalVoltage: voltage(x, motion) - voltage(x, n.pin(p, "negative"))}
	coilFound, motionFound := false, false
	for k, q := range n.inductors {
		if q.ID != speakerPrivate(p.ID, "lc") && q.ID != speakerPrivate(p.ID, "lm") {
			continue
		}
		index := n.inductorIndex(k)
		if index < 0 || index >= len(x) {
			return invalidSpeakerResult(p.ID)
		}
		if q.ID == speakerPrivate(p.ID, "lc") {
			r.Current = x[index]
			coilFound = true
		} else {
			r.MotionalCurrent = x[index]
			motionFound = true
		}
	}
	if !coilFound || !motionFound {
		return invalidSpeakerResult(p.ID)
	}
	capFound := n.dc // DC deliberately opens storage and has no capacitor-current slot.
	for k, q := range n.caps {
		if q.ID != speakerPrivate(p.ID, "cm") {
			continue
		}
		index := n.capacitorIndex(k)
		if index < 0 || index >= len(x) {
			return invalidSpeakerResult(p.ID)
		}
		r.CapacitorCurrent = x[index]
		capFound = true
	}
	if !capFound {
		return invalidSpeakerResult(p.ID)
	}
	r.Power = r.Voltage * r.Current
	r.CoilHeat = v.Resistance * r.Current * r.Current
	r.MotionalHeat = r.MotionalVoltage * r.MotionalVoltage / v.MotionalResistance
	r.Energy = .5*v.Inductance*r.Current*r.Current + .5*v.MotionalInductance*r.MotionalCurrent*r.MotionalCurrent + .5*speakerCapacitance(p)*r.MotionalVoltage*r.MotionalVoltage
	return r
}

func (n *mixedNetwork) restoreSpeakers(r *MixedResult, x []float64) error {
	for _, p := range n.authored.Components {
		if p.Kind != "speaker" {
			continue
		}
		found := false
		for _, q := range r.Speakers {
			if q.ID != p.ID {
				continue
			}
			found = true
			if !finite(q.Current) || !finite(q.MotionalCurrent) || !finite(q.MotionalVoltage) || !finite(q.CapacitorCurrent) {
				return fmt.Errorf("invalid speaker state")
			}
			coil, motion, ok := n.speakerNodes(p)
			if !ok || coil >= len(x) || motion >= len(x) {
				return fmt.Errorf("missing private speaker nodes")
			}
			negative := voltage(x, n.pin(p, "negative"))
			x[motion] = negative + q.MotionalVoltage
			x[coil] = voltage(x, n.pin(p, "positive")) - p.Device.Speaker.Resistance*q.Current
			for k, c := range n.inductors {
				if c.ID == speakerPrivate(p.ID, "lc") {
					if n.inductorIndex(k) < 0 || n.inductorIndex(k) >= len(x) {
						return fmt.Errorf("missing speaker inductor current slot")
					}
					x[n.inductorIndex(k)] = q.Current
				}
				if c.ID == speakerPrivate(p.ID, "lm") {
					if n.inductorIndex(k) < 0 || n.inductorIndex(k) >= len(x) {
						return fmt.Errorf("missing speaker inductor current slot")
					}
					x[n.inductorIndex(k)] = q.MotionalCurrent
				}
			}
			for k, c := range n.caps {
				if c.ID == speakerPrivate(p.ID, "cm") {
					index := n.capacitorIndex(k)
					if index < 0 || index >= len(x) {
						return fmt.Errorf("missing speaker capacitor current slot")
					}
					x[index] = q.CapacitorCurrent
				}
			}
		}
		if !found {
			return fmt.Errorf("missing speaker state")
		}
	}
	return nil
}
func validateRecordedSpeakers(c Compiled, g Graph, r *MixedResult) error {
	ids := []string{}
	for _, q := range r.Speakers {
		ids = append(ids, q.ID)
	}
	if err := recordedObjectIDs(g, "speaker", ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, q := range r.Speakers {
		p, ok := g.Component(q.ID)
		if !ok || p.Kind != "speaker" || p.Device == nil || p.Device.Speaker == nil {
			return fmt.Errorf("invalid recorded speaker identity")
		}
		e := n.speakerResult(p, x)
		if q.Model != e.Model {
			return fmt.Errorf("unsupported speaker model")
		}
		for _, v := range [][2]float64{{q.Voltage, e.Voltage}, {q.Current, e.Current}, {q.Power, e.Power}, {q.CoilHeat, e.CoilHeat}, {q.MotionalHeat, e.MotionalHeat}, {q.Energy, e.Energy}, {q.MotionalVoltage, e.MotionalVoltage}, {q.MotionalCurrent, e.MotionalCurrent}, {q.CapacitorCurrent, e.CapacitorCurrent}} {
			if !recordingNear(v[0], v[1]) {
				return fmt.Errorf("recorded speaker readings disagree with electrical state")
			}
		}
		if !recordingNear(q.Current, q.MotionalCurrent+q.CapacitorCurrent+q.MotionalVoltage/p.Device.Speaker.MotionalResistance) {
			return fmt.Errorf("speaker motional KCL mismatch")
		}
	}
	return nil
}

func speakerRequiredFields() []string {
	return []string{"version", "coil_resistance_ohm", "coil_inductance_h", "resonance_hz", "motional_resistance_ohm", "motional_inductance_h"}
}
