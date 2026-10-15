package main

import (
	"encoding/json"
	"math"
	"strconv"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// Text changes arrive on native commit, preserving partial keyboard/IME input.
func timeField(id, label string, seconds float64, disabled bool, change func(string)) ui.Node {
	return ui.Node{ID: id, Kind: ui.TextField, Text: timeText(seconds), MinWidth: 90, Disabled: disabled, Accessibility: ui.Accessibility{Name: label, Help: "Enter seconds or a unit such as 10 ms or 250 us; bare numbers are seconds."}, Change: func(e ui.ChangeEvent) { change(e.Text) }}
}
func timeText(seconds float64) string {
	scale, unit := 1., "s"
	absolute := math.Abs(seconds)
	if absolute != 0 && absolute < 1 {
		scale, unit = 1000, "ms"
	}
	if absolute != 0 && absolute < .001 {
		scale, unit = 1e6, "us"
	}
	candidate := strconv.FormatFloat(seconds*scale, 'g', -1, 64) + " " + unit
	if parsed, err := workbench.ParseTimeSeconds(candidate); err == nil && parsed == seconds {
		return candidate
	}
	return strconv.FormatFloat(seconds, 'g', -1, 64) + " s"
}

// textFieldParams replaces one captured numeric field with committed quantity
// text; the shared command parser validates units and all coupled fields.
func textFieldParams(base any, field, text string) map[string]any {
	raw, _ := json.Marshal(base)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	out[field] = text
	return out
}
