package workbench

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var pulseQuantityPattern = regexp.MustCompile(`^([+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+))(?:[eE]([+-]?[0-9]+))?[ \t]*(Hz|kHz|mHz|%)?$`)

func parsePulseQuantity(text string, frequency bool) (float64, error) {
	if len(text) == 0 || len(text) > 128 {
		return 0, fmt.Errorf("quantity requires 1–128 bytes")
	}
	match := pulseQuantityPattern.FindStringSubmatch(strings.TrimSpace(text))
	if match == nil || frequency && match[3] == "%" || !frequency && match[3] != "" && match[3] != "%" {
		return 0, fmt.Errorf("enter a decimal/scientific frequency in Hz/kHz/mHz, or a duty percentage in %%")
	}
	exponent := int64(0)
	if match[2] != "" {
		value, err := strconv.ParseInt(match[2], 10, 32)
		if err != nil {
			return 0, fmt.Errorf("quantity exponent is outside the numeric range")
		}
		exponent = value
	}
	if match[3] == "kHz" {
		exponent += 3
	} else if match[3] == "mHz" {
		exponent -= 3
	}
	value, err := strconv.ParseFloat(match[1]+"e"+strconv.FormatInt(exponent, 10), 64)
	if err != nil || !finite(value) || value == 0 && strings.ContainsAny(match[1], "123456789") {
		return 0, fmt.Errorf("quantity is outside the finite representable range")
	}
	return value, nil
}

// Canonicalize quantity text before retry fingerprinting, including inside
// edit.batch. This uses one decimal conversion for equivalent unit forms.
func normalizePulseTimingQuantities(object map[string]json.RawMessage) error {
	for _, field := range []string{"frequency_hz", "duty_percent"} {
		raw, exists := object[field]
		if !exists {
			continue
		}
		var text string
		if len(raw) == 0 || raw[0] != '"' || json.Unmarshal(raw, &text) != nil {
			return fmt.Errorf("%s requires quantity text", field)
		}
		value, err := parsePulseQuantity(text, field == "frequency_hz")
		if err != nil {
			return fmt.Errorf("%s: %w", field, err)
		}
		object[field], _ = json.Marshal(strconv.FormatFloat(value, 'g', -1, 64))
	}
	return nil
}
