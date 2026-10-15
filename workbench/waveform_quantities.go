package workbench

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Waveform quantity text is an input convenience only. Graphs, saved documents
// and result schemas continue to contain numeric SI values.
var waveformQuantityPattern = regexp.MustCompile(`^([+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+))(?:[eE]([+-]?[0-9]+))?[ \t]*(V|mV|uV|µV|μV|Hz|kHz|mHz|deg|°)?$`)

type waveformQuantityField struct{ name, unit string }

var waveformQuantityFields = map[string][]waveformQuantityField{
	"source.sine.set":  {{"amplitude_v", "V"}, {"offset_v", "V"}, {"frequency_hz", "Hz"}, {"phase_degrees", "deg"}},
	"source.pulse.set": {{"high_v", "V"}, {"low_v", "V"}},
}

func parseWaveformQuantity(text, unit string) (float64, error) {
	if len(text) == 0 || len(text) > 128 {
		return 0, fmt.Errorf("quantity requires 1–128 bytes")
	}
	m := waveformQuantityPattern.FindStringSubmatch(strings.TrimSpace(text))
	if m == nil {
		return 0, fmt.Errorf("use decimal/scientific text with a supported %s unit", unit)
	}
	suffix := m[3]
	exponent := int64(0)
	if m[2] != "" {
		v, err := strconv.ParseInt(m[2], 10, 32)
		if err != nil {
			return 0, fmt.Errorf("quantity exponent is outside the numeric range")
		}
		exponent = v
	}
	valid := suffix == "" || suffix == unit
	switch unit {
	case "V":
		if suffix == "mV" {
			exponent -= 3
			valid = true
		} else if suffix == "uV" || suffix == "µV" || suffix == "μV" {
			exponent -= 6
			valid = true
		}
	case "Hz":
		if suffix == "kHz" {
			exponent += 3
			valid = true
		} else if suffix == "mHz" {
			exponent -= 3
			valid = true
		}
	case "deg":
		valid = valid || suffix == "°"
	}
	if !valid {
		return 0, fmt.Errorf("quantity unit does not match %s", unit)
	}
	// Fold the unit into one decimal conversion, so equivalent unit forms have
	// the same value and retry intent without a parse-then-scale rounding step.
	v, err := strconv.ParseFloat(m[1]+"e"+strconv.FormatInt(exponent, 10), 64)
	if err != nil || !finite(v) || v == 0 && strings.ContainsAny(m[1], "123456789") {
		return 0, fmt.Errorf("quantity is outside the finite representable range")
	}
	return v, nil
}

func normalizeWaveformQuantities(command string, object map[string]json.RawMessage) error {
	fields := waveformQuantityFields[command]
	if len(fields) == 0 {
		return nil
	}
	raw, exists := object["waveform"]
	if !exists {
		return nil // Domain validation owns required/type/extra-field errors.
	}
	var waveform map[string]json.RawMessage
	if err := DecodeStrict(raw, &waveform); err != nil {
		return err
	}
	if waveform == nil {
		return nil
	}
	for _, field := range fields {
		name, unit := field.name, field.unit
		raw := waveform[name]
		if len(raw) == 0 || raw[0] != '"' {
			continue
		}
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return err
		}
		value, err := parseWaveformQuantity(text, unit)
		if err != nil {
			return fmt.Errorf("waveform.%s: %w", name, err)
		}
		waveform[name], _ = json.Marshal(value)
	}
	object["waveform"], _ = json.Marshal(waveform)
	return nil
}

func waveformQuantitySchemaID(unit string) string {
	return "urn:elab:waveform-quantity-" + unit + ":1"
}

func waveformQuantityDefinition(unit string) map[string]any {
	schema := map[string]any{"$id": waveformQuantitySchemaID(unit), "anyOf": []any{map[string]any{"type": "number"}, map[string]any{"type": "string", "minLength": 1, "maxLength": 128}}, "x-unit": unit, "x-max-utf8-bytes": 128, "description": "Finite numeric value or decimal/scientific quantity text; bare text uses " + unit + ". Model/coupled bounds apply after conversion. No hexadecimal, NaN, Inf or nonzero underflow."}
	switch unit {
	case "V":
		schema["x-text-units"] = []string{"V", "mV", "uV", "µV", "μV"}
	case "Hz":
		schema["x-text-units"] = []string{"Hz", "kHz", "mHz"}
	case "deg":
		schema["x-text-units"] = []string{"deg", "°"}
	}
	return schema
}

func waveformQuantitySchema(command string, schema map[string]any) {
	for _, field := range waveformQuantityFields[command] {
		if p := inputProperty(schema, "waveform."+field.name); p != nil {
			delete(p, "type")
			p["$ref"] = waveformQuantitySchemaID(field.unit)
		}
	}
}
