package workbench

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const TimeInputSchemaID = "urn:elab:time:1"

var timeTextPattern = regexp.MustCompile(`^([+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+))(?:[eE]([+-]?[0-9]+))?[ \t]*(s|ms|us|µs|μs|ns|ps)?$`)

// ParseTimeSeconds parses a finite decimal/scientific quantity in seconds.
// Bare text is seconds; explicit s/ms/us/µs/μs/ns/ps are case-sensitive.
// Domain commands apply their own sign, interval and sampling bounds afterward.
func ParseTimeSeconds(text string) (float64, error) {
	if len(text) > 128 {
		return 0, fmt.Errorf("time text must be at most 128 bytes")
	}
	match := timeTextPattern.FindStringSubmatch(strings.TrimSpace(text))
	if match == nil {
		return 0, fmt.Errorf("use a time such as 10 ms, 250 us, or 0.01 s")
	}
	exponent := int64(0)
	if match[2] != "" {
		value, err := strconv.ParseInt(match[2], 10, 32)
		if err != nil {
			return 0, fmt.Errorf("time exponent is outside the supported numeric range")
		}
		exponent = value
	}
	switch match[3] {
	case "ms":
		exponent -= 3
	case "us", "µs", "μs":
		exponent -= 6
	case "ns":
		exponent -= 9
	case "ps":
		exponent -= 12
	}
	// Compose one decimal exponent before conversion; avoid double rounding from
	// parsing then scaling so equivalent unit forms have the same retry intent.
	value, err := strconv.ParseFloat(match[1]+"e"+strconv.FormatInt(exponent, 10), 64)
	if err != nil || !finite(value) || value == 0 && strings.ContainsAny(match[1], "123456789") {
		return 0, fmt.Errorf("time is outside the finite representable range")
	}
	return value, nil
}

// Only public command inputs accept unit strings. Persisted documents, recorded
// rows and output schemas retain strict numeric SI fields.
var timeCommandFields = map[string][]string{
	"simulation.transient.set": {"step_s", "high_s", "low_s"},
	"simulation.step":          {"duration_s"}, "simulation.step.start": {"duration_s"},
	"verification.settings.set": {"duration_s", "sample_step_s"},
	"simulation.verify":         {"duration_s", "sample_step_s"},
	"scope.cursor.set":          {"time_s"}, "scope.window.set": {"start_s", "end_s"},
	"recording.seek": {"time_s"}, "recording.cursor.set": {"time_s"},
	"recording.timing": {"start_s", "end_s"}, "recording.timing.set": {"start_s", "end_s"},
	"recording.compare": {"time_s"}, "recording.compare.set": {"time_s"},
	"recording.window.set":     {"start_s", "end_s"},
	"source.pulse.set":         {"waveform.high_s", "waveform.low_s", "waveform.rise_s", "waveform.fall_s"},
	"source.pulse.timing.set":  {"period_s"},
	"recovery.configure":       {"interval_s"},
	"recovery.configure.start": {"interval_s"},
}

func normalizeTimeInput(command string, raw json.RawMessage) (json.RawMessage, error) {
	fields := timeCommandFields[command]
	if len(fields) == 0 && command != "edit.batch" {
		return raw, nil
	}
	var object map[string]json.RawMessage
	if err := DecodeStrict(raw, &object); err != nil {
		return nil, err
	}
	if object == nil {
		return nil, fmt.Errorf("params must be a JSON object")
	}
	for _, path := range fields {
		parts := strings.Split(path, ".")
		target := object
		var nested map[string]json.RawMessage
		if len(parts) == 2 {
			value, exists := object[parts[0]]
			if !exists {
				continue
			}
			if err := DecodeStrict(value, &nested); err != nil {
				return nil, err
			}
			if nested == nil {
				continue
			}
			target = nested
		}
		name := parts[len(parts)-1]
		value, exists := target[name]
		if !exists {
			continue
		}
		var text string
		if len(value) > 0 && value[0] == '"' {
			if err := json.Unmarshal(value, &text); err != nil {
				return nil, err
			}
			seconds, err := ParseTimeSeconds(text)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			target[name], _ = json.Marshal(seconds)
			if nested != nil {
				object[parts[0]], _ = json.Marshal(nested)
			}
		}
	}
	if command == "source.pulse.timing.set" {
		if err := normalizePulseTimingQuantities(object); err != nil {
			return nil, err
		}
	}
	if command == "edit.batch" {
		var edits []BatchEdit
		if err := DecodeStrict(object["edits"], &edits); err != nil {
			return nil, err
		}
		if len(edits) > 100 {
			return nil, fmt.Errorf("batch admits at most 100 edits")
		}
		for i := range edits {
			// Nested batches are unsupported; let the existing domain validator refuse
			// them without recursively processing their parameter trees.
			if edits[i].Command == "edit.batch" {
				continue
			}
			normalized, err := normalizeTimeInput(edits[i].Command, edits[i].Params)
			if err != nil {
				return nil, err
			}
			edits[i].Params = normalized
		}
		object["edits"], _ = json.Marshal(edits)
	}
	return json.Marshal(object)
}
func timeInputSchema(command string, schema map[string]any) map[string]any {
	for _, path := range timeCommandFields[command] {
		parts := strings.Split(path, ".")
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			continue
		}
		if len(parts) == 2 {
			nested, ok := properties[parts[0]].(map[string]any)
			if !ok {
				continue
			}
			properties, ok = nested["properties"].(map[string]any)
			if !ok {
				continue
			}
		}
		name := parts[len(parts)-1]
		previous, ok := properties[name].(map[string]any)
		if !ok {
			continue
		}
		ref := map[string]any{"$ref": TimeInputSchemaID}
		if _, optional := previous["anyOf"]; optional {
			properties[name] = map[string]any{"anyOf": []any{ref, map[string]any{"type": "null"}}}
		} else {
			properties[name] = ref
		}
	}
	return schema
}
