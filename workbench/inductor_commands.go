package workbench

import (
	"encoding/json"
	"fmt"
	"math"
)

func (s *Service) inductorInitialLocked(p ValueParams) error {
	part, ok := s.graph.Component(p.ID)
	if !ok || part.Kind != "inductor" {
		return fmt.Errorf("choose an inductor to set its initial current")
	}
	var initial *float64
	if p.Value != "default" {
		value, err := ParseValue(p.Value, "A")
		if err != nil {
			return err
		}
		if math.Abs(value) > MaxInductorInitialCurrent {
			return fmt.Errorf("inductor initial current must be within ±1 A")
		}
		if value == 0 {
			value = 0
		}
		initial = &value
	}
	for k := range s.graph.Components {
		if s.graph.Components[k].ID == p.ID {
			s.graph.Components[k].InitialCurrent = initial
			break
		}
	}
	return nil
}

func validateStartupFields(raw json.RawMessage) error {
	if err := validateCapacitorInitialFields(raw); err != nil {
		return err
	}
	var object map[string]json.RawMessage
	_ = json.Unmarshal(raw, &object)
	field, exists := object["inductor_initials"]
	if !exists {
		return nil
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(field, &rows); err != nil {
		return err
	}
	if rows == nil {
		return fmt.Errorf("inductor_initials must be an array")
	}
	for _, row := range rows {
		if err := params(row, &InductorInitial{}, "id", "current_a"); err != nil {
			return err
		}
	}
	return nil
}
