package workbench

import "encoding/json"

func motorParameterFields() []string {
	return []string{"locked_rotor", "armature_resistance_ohm", "armature_inductance_h", "ke_v_s_per_rad", "inertia_kg_m2", "friction_n_m_s_per_rad", "load_n_m_s_per_rad"}
}
func validatePowerFields(raw json.RawMessage, p Component) error {
	if p.Power == nil {
		return nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	if p.Kind == "transformer" {
		return paramsNested(fields["power"], "transformer", &TransformerConfig{}, transformerParameterFields()...)
	}
	if p.Kind == "optocoupler" {
		return paramsNested(fields["power"], "optocoupler", &OptocouplerConfig{}, "current_transfer_ratio")
	}
	if p.Kind == "battery" {
		return paramsNested(fields["power"], "battery", &BatteryConfig{}, "capacity_ah", "internal_resistance_ohm", "initial_charge_fraction")
	}
	if p.Kind == "dc-motor" {
		return paramsNested(fields["power"], "motor", &MotorConfig{}, motorParameterFields()...)
	}
	return nil
}
