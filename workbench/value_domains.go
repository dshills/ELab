package workbench

import (
	"fmt"
	"strings"
)

// ComponentValueDomain describes ordinary numeric value edits. Per-waveform
// coupling still validates against the captured component's other parameters.
type ComponentValueDomain struct {
	Parameter string  `json:"parameter"`
	Unit      string  `json:"unit"`
	Minimum   float64 `json:"minimum"`
	Maximum   float64 `json:"maximum"`
	Range     string  `json:"range_text"`
	Syntax    string  `json:"syntax"`
	Relation  string  `json:"relation,omitempty"`
}

// ValueDomainForKind returns an owned domain for a numeric library component.
// Labels and fixed-model symbols do not have a numeric value domain.
func ValueDomainForKind(kind string) (ComponentValueDomain, bool) {
	d := ComponentValueDomain{}
	switch kind {
	case "dc-source", "pulse-source":
		d = ComponentValueDomain{Parameter: "voltage", Unit: "V", Minimum: 0, Maximum: 12, Range: "0–12 V"}
	case "sine-source":
		d = ComponentValueDomain{Parameter: "amplitude", Unit: "V", Minimum: 0, Maximum: 12, Range: "0–12 V", Relation: "Amplitude plus the absolute offset must be ≤12 V."}
	case "finite-pulse-source":
		d = ComponentValueDomain{Parameter: "high_voltage", Unit: "V", Minimum: -12, Maximum: 12, Range: "−12–12 V", Relation: "High level must be at least the current low level (factory: 0 V)."}
	case "resistor", "potentiometer":
		d = ComponentValueDomain{Parameter: "resistance", Unit: "Ω", Minimum: 10, Maximum: 100000, Range: "10–100000 Ω"}
	case "capacitor":
		d = ComponentValueDomain{Parameter: "capacitance", Unit: "F", Minimum: 1e-12, Maximum: 1, Range: "1 pF–1 F"}
	default:
		return d, false
	}
	d.Syntax = valueSyntax(d.Unit)
	return d, true
}
func valueSyntax(unit string) string {
	switch unit {
	case "V":
		return "Bare SI or V; m/u/µ/μ prefixes, for example 500 mV or 1 V"
	case "Ω":
		return "Bare SI or Ω/ohm; k/K/M prefixes, for example 2.2k or 0.1M Ω"
	case "F":
		return "Bare SI or F; p/n/u/µ/μ/m prefixes, for example 1 µF or 100nF"
	}
	return ""
}
func validateComponentValue(kind string, value float64) error {
	d, ok := ValueDomainForKind(kind)
	if !ok {
		return fmt.Errorf("this component has no numeric value")
	}
	if !finite(value) || value < d.Minimum || value > d.Maximum {
		return fmt.Errorf("%s must be %s", strings.ReplaceAll(d.Parameter, "_", " "), d.Range)
	}
	return nil
}
