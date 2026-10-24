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
	case "zener":
		d = ComponentValueDomain{Parameter: "breakdown_voltage", Unit: "V", Minimum: 1, Maximum: 24, Range: "1–24 V", Relation: "Reverse knee voltage; breakdown adds 10 Ω dynamic resistance."}
	case "linear-regulator", "ldo-regulator":
		d = ComponentValueDomain{Parameter: "output_voltage", Unit: "V", Minimum: 1.2, Maximum: 12, Range: "1.2–12 V", Relation: "Output setpoint; available input headroom and load determine actual voltage."}
	case "comparator":
		d = ComponentValueDomain{Parameter: "hysteresis", Unit: "V", Minimum: .001, Maximum: 1, Range: "1 mV–1 V", Relation: "Full differential hysteresis width; rising at +H/2 and falling at −H/2."}
	case "relay", "inductor":
		d = ComponentValueDomain{Parameter: "inductance", Unit: "H", Minimum: 1e-6, Maximum: 100, Range: "1 µH–100 H"}
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
	case "H":
		return "Bare SI or H; n/u/µ/μ/m prefixes, for example 10 mH or 100uH"
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
