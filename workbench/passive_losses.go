package workbench

import (
	"fmt"
)

const PassiveLossModelID = "elab.passive-loss.v1"
const passiveLossDescription = PassiveLossModelID + ": Optional lumped capacitor series ESR plus leakage resistance across its external terminals, and standalone inductor winding resistance. Zero disables each loss. Capacitor startup/energy use internal stored voltage; external voltage and total current include ESR/leakage. Constant parameters, no dielectric absorption, saturation, frequency/temperature dependence, polarity, ratings or manufacturer identity. Existing ideal components remain exact when configuration is disabled."

type PassiveConfig struct {
	Version string  `json:"version"`
	ESR     float64 `json:"esr_ohm"`
	Leakage float64 `json:"leakage_resistance_ohm"`
	Winding float64 `json:"winding_resistance_ohm"`
}
type PassiveParams struct {
	ID         string        `json:"id"`
	Enabled    bool          `json:"enabled"`
	Parameters PassiveConfig `json:"parameters"`
}
type PassiveReading struct {
	Model          string  `json:"model"`
	StoredVoltage  float64 `json:"storage_voltage_v"`
	StorageCurrent float64 `json:"storage_current_a"`
	ESRHeat        float64 `json:"esr_heat_w"`
	LeakageCurrent float64 `json:"leakage_current_a"`
	LeakageHeat    float64 `json:"leakage_heat_w"`
	WindingHeat    float64 `json:"winding_heat_w"`
}

func IsPassiveStorageKind(k string) bool { return k == "capacitor" || k == "inductor" }
func hasPassiveLoss(p Component) bool {
	return IsPassiveStorageKind(p.Kind) && p.Device != nil && p.Device.Passive != nil
}
func hasPassiveLosses(g Graph) bool {
	for _, p := range g.Components {
		if hasPassiveLoss(p) {
			return true
		}
	}
	return false
}
func passiveRequiredFields() []string {
	return []string{"version", "esr_ohm", "leakage_resistance_ohm", "winding_resistance_ohm"}
}
func validatePassiveConfig(kind string, p PassiveConfig) error {
	if !IsPassiveStorageKind(kind) || p.Version != PassiveLossModelID {
		return fmt.Errorf("choose a capacitor/inductor and supported passive model version")
	}
	valid := func(v, min, max float64) bool { return finite(v) && (v == 0 || v >= min && v <= max) }
	if !valid(p.ESR, .001, 1e5) || !valid(p.Leakage, 1000, 1e12) || !valid(p.Winding, .001, 1e5) {
		return fmt.Errorf("ESR/winding resistance: 0 off or 0.001–100000 ohm; leakage resistance: 0 off or 1000–1e12 ohm")
	}
	if kind == "capacitor" && p.Winding != 0 || kind == "inductor" && (p.ESR != 0 || p.Leakage != 0) {
		return fmt.Errorf("unused resistance fields must be zero for this component kind")
	}
	return nil
}
func (s *Service) configurePassiveLocked(p PassiveParams) error {
	for i, part := range s.graph.Components {
		if part.ID != p.ID {
			continue
		}
		if err := validatePassiveConfig(part.Kind, p.Parameters); err != nil {
			return err
		}
		part.Device = nil
		if p.Enabled {
			v := p.Parameters
			part.Device = &DeviceParameters{Passive: &v}
		}
		s.graph.Components[i] = part
		return nil
	}
	return fmt.Errorf("choose an existing capacitor or inductor")
}
func capacitorESR(p Component) float64 {
	if hasPassiveLoss(p) {
		return p.Device.Passive.ESR
	}
	return 0
}
func capacitorLeakage(p Component) float64 {
	if hasPassiveLoss(p) && p.Device.Passive.Leakage > 0 {
		return 1 / p.Device.Passive.Leakage
	}
	return 0
}
func capacitorStoredVoltage(p Component, v, chargeCurrent float64) float64 {
	if r := capacitorESR(p); r > 0 {
		return v - r*chargeCurrent
	}
	return v
}
func passiveReading(p Component, v, storageCurrent float64) *PassiveReading {
	if !hasPassiveLoss(p) {
		return nil
	}
	r := &PassiveReading{Model: PassiveLossModelID, StoredVoltage: v, StorageCurrent: storageCurrent}
	if p.Kind == "capacitor" {
		r.StoredVoltage = capacitorStoredVoltage(p, v, storageCurrent)
		r.ESRHeat = capacitorESR(p) * storageCurrent * storageCurrent
		r.LeakageCurrent = capacitorLeakage(p) * v
		r.LeakageHeat = r.LeakageCurrent * v
	} else {
		resistance := p.Device.Passive.Winding
		r.StoredVoltage = v - resistance*storageCurrent
		r.WindingHeat = resistance * storageCurrent * storageCurrent
	}
	return r
}
func clonePassiveReading(p *PassiveReading) *PassiveReading {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
func capacitorResult(p Component, v, charging float64) CapacitorResult {
	stored := capacitorStoredVoltage(p, v, charging)
	current := charging
	if hasPassiveLoss(p) {
		current += capacitorLeakage(p) * v
	}
	return CapacitorResult{ID: p.ID, Voltage: v, Current: current, Energy: .5 * p.Value * stored * stored, Loss: passiveReading(p, v, charging)}
}
func validatePassiveReading(actual, expected *PassiveReading) error {
	if actual == nil && expected == nil {
		return nil
	}
	if actual == nil || expected == nil || actual.Model != expected.Model || !finitePassiveReadings(actual) {
		return fmt.Errorf("recorded passive model is missing or unexpected")
	}
	for _, v := range [][2]float64{{actual.StoredVoltage, expected.StoredVoltage}, {actual.StorageCurrent, expected.StorageCurrent}, {actual.ESRHeat, expected.ESRHeat}, {actual.LeakageCurrent, expected.LeakageCurrent}, {actual.LeakageHeat, expected.LeakageHeat}, {actual.WindingHeat, expected.WindingHeat}} {
		if !finite(v[0]) || !recordingNear(v[0], v[1]) {
			return fmt.Errorf("recorded passive losses disagree with its electrical state")
		}
	}
	return nil
}
func passiveCapEnergy(p Component, terminal, totalCurrent float64) float64 {
	stored := capacitorStoredVoltage(p, terminal, totalCurrent-capacitorLeakage(p)*terminal)
	return .5 * p.Value * stored * stored
}
func finitePassiveReadings(r *PassiveReading) bool {
	return r == nil || finite(r.StoredVoltage) && finite(r.StorageCurrent) && finite(r.ESRHeat) && finite(r.LeakageCurrent) && finite(r.LeakageHeat) && finite(r.WindingHeat) && r.ESRHeat >= 0 && r.LeakageHeat >= 0 && r.WindingHeat >= 0
}
func passiveACAdmittance(p Component, w float64) complex128 {
	y := complex(0, w*p.Value)
	if r := capacitorESR(p); r > 0 {
		return y / (1 + complex(r, 0)*y)
	}
	return y
}
func passiveCapCompanion(p Component, dt float64) float64 {
	if r := capacitorESR(p); r > 0 {
		return p.Value / (dt + r*p.Value)
	}
	return p.Value / dt
}
func hasPassiveDissipation(g Graph) bool {
	for _, p := range g.Components {
		if hasPassiveLoss(p) {
			v := p.Device.Passive
			if v.ESR > 0 || v.Leakage > 0 || v.Winding > 0 {
				return true
			}
		}
	}
	return false
}
