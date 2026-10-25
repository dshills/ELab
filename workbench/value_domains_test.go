package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestValueInputUnderflowMustRefuse(t *testing.T) {
	for _, test := range []struct{ text, unit string }{{"1e-999 V", "V"}, {"1e-320 pF", "F"}, {"1e308M ohm", "Ω"}} {
		if value, err := ParseValue(test.text, test.unit); err == nil {
			t.Errorf("unrepresentable value %q silently accepted as %g", test.text, value)
		}
	}
}

func TestValueEngineeringUnitsAndLegacyRounding(t *testing.T) {
	for _, test := range []struct {
		text, unit string
		want       float64
	}{{"500 mV", "V", .5}, {"250 uV", "V", .00025}, {"250 µV", "V", .00025}, {"250 μV", "V", .00025}, {"-500mV", "V", -.5}, {"2.2k ohm", "Ω", 2200}, {"0.1M Ω", "Ω", 100000}, {"100nF", "F", 1e-7}, {"0e-999 V", "V", 0}, {"-0x0p-999 V", "V", 0}, {"1e-320 V", "V", 1e-320}, {"0.5", "", .5}} {
		value, err := ParseValue(test.text, test.unit)
		if err != nil || math.Abs(value-test.want) > math.Abs(test.want)*1e-15 {
			t.Fatal(test, value, err)
		}
	}
	// Frozen IEEE-754 results from the previous parse-then-scale path. Combining
	// exponents before parsing would change some valid stored component values.
	for _, test := range []struct {
		text, unit string
		bits       uint64
	}{{"2.2uF", "F", 0x3ec27476ca61b882}, {"3.3nF", "F", 0x3e2c58c6d8a67ba9}, {"0.1mF", "F", 0x3f1a36e2eb1c432d}, {"2.2kΩ", "Ω", 0x40a1300000000000}, {"99.9k", "Ω", 0x40f863c000000000}} {
		value, err := ParseValue(test.text, test.unit)
		if err != nil || math.Float64bits(value) != test.bits {
			t.Fatal("legacy value changed", test, value, err)
		}
	}
	for _, text := range []string{"NaN V", "Inf V", "500 mA", "1e-999 V", "1e-320uV", "0x1p-2000 V", strings.Repeat("0", 129)} {
		if _, err := ParseValue(text, "V"); err == nil {
			t.Fatal("invalid voltage accepted", text)
		}
	}
}
func TestPublishedComponentDomainsMatchRuntimeBounds(t *testing.T) {
	for _, part := range Library() {
		if part.ValueDomain == nil {
			continue
		}
		d := *part.ValueDomain
		component, err := MakeComponent("X", part.Kind, Point{}, "")
		if err != nil {
			t.Fatal(err)
		}
		if part.Kind == "finite-pulse-source" {
			component.Pulse.LowVoltage = -12
		}
		for _, value := range []float64{d.Minimum, d.Maximum} {
			next, err := SetValue(component, fmt.Sprintf("%.17g", value))
			if err != nil || next.Value != value {
				t.Fatal("published boundary refused", part.Kind, value, err)
			}
		}
		for _, value := range []float64{math.Nextafter(d.Minimum, math.Inf(-1)), math.Nextafter(d.Maximum, math.Inf(1))} {
			if _, err := SetValue(component, fmt.Sprintf("%.17g", value)); err == nil {
				t.Fatal("value outside published boundary accepted", part.Kind, value)
			}
		}
		again, ok := ValueDomainForKind(part.Kind)
		if !ok || again != d || part.ValueType != "number" {
			t.Fatal("library/runtime domain drift", part)
		}
	}
	parts := Library()
	for _, part := range parts {
		if part.ValueDomain == nil {
			continue
		}
		expected := *part.ValueDomain
		part.ValueDomain.Maximum = 999
		for _, fresh := range Library() {
			if fresh.Kind == part.Kind && *fresh.ValueDomain != expected {
				t.Fatal("library metadata is borrowed mutable state", part.Kind)
			}
		}
	}

}
func TestValueCouplingAndLabelContracts(t *testing.T) {
	sine, err := MakeComponent("V1", "sine-source", Point{}, "")
	if err != nil {
		t.Fatal(err)
	}
	sine.Sine.Offset = 3
	valid, err := SetValue(sine, "9000 mV")
	if err != nil || valid.Value != 9 || valid.Sine.Offset != 3 {
		t.Fatal("valid amplitude/offset relation refused", err)
	}
	if failed, err := SetValue(sine, "9001 mV"); err == nil || !reflect.DeepEqual(failed, sine) {
		t.Fatal("sine coupling relaxed or error changed input")
	}
	pulse, err := MakeComponent("V2", "finite-pulse-source", Point{}, "")
	if err != nil {
		t.Fatal(err)
	}
	pulse.Pulse.LowVoltage = -2
	valid, err = SetValue(pulse, "-1000 mV")
	if err != nil || valid.Pulse.HighVoltage != -1 || valid.Pulse.LowVoltage != -2 {
		t.Fatal("signed pulse high relation", err)
	}
	if _, err := SetValue(pulse, "-3000 mV"); err == nil {
		t.Fatal("pulse high below low accepted")
	}
	caps, err := detailedCapabilities("component.set")
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	_ = json.Unmarshal(caps[0].Input, &schema)
	domains := schema["x-kind-value-domains"].([]any)
	var pattern string
	for _, raw := range domains {
		d := raw.(map[string]any)
		if d["kind"] == "net-label" {
			pattern = d["label"].(map[string]any)["pattern"].(string)
		}
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil || pattern == "" {
		t.Fatal("label pattern unavailable", err)
	}
	for i := 0; i < 128; i++ {
		text := string(rune(i))
		if compiled.MatchString(text) != validNetLabel(text) {
			t.Fatal("label pattern differs from runtime", i)
		}
	}
	for _, text := range []string{"", strings.Repeat("A", 32), strings.Repeat("A", 33), "VCC/+", "GND", "A B", "µ"} {
		if compiled.MatchString(text) != validNetLabel(text) {
			t.Fatal("label domain drift", text)
		}
	}
	for _, kind := range []string{"led", "npn", "switch", "ground", "junction", "diode"} {
		c, err := MakeComponent("X", kind, Point{}, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := SetValue(c, "1"); err == nil {
			t.Fatal("fixed model acquired editable value", kind)
		}
	}
}
func TestValueCommandsRejectUnrepresentableInputsWithoutMutation(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	for _, test := range []struct {
		command string
		p       any
	}{{"component.set", ValueParams{ID: "V1", Value: "1e-999 V"}}, {"capacitor.initial.set", ValueParams{ID: "C1", Value: "1e-999 V"}}, {"component.place", PlaceParams{ID: "V2", Kind: "dc-source", Position: Point{X: 400, Y: 0}, Value: "1e-999 V"}}, {"component.set", ValueParams{ID: "C1", Value: "1e-320pF"}}} {
		before := s.Snapshot()
		refuse(t, s, test.command, test.p, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("invalid exact value changed state", test)
		}
	}
	call(t, s, "component.set", ValueParams{ID: "V1", Value: "500 mV"}, "millivolts")
	call(t, s, "capacitor.initial.set", ValueParams{ID: "C1", Value: "-250 mV"}, "initial")
	graph := Graph{Components: s.Snapshot().Components}
	source, _ := graph.Component("V1")
	capacitor, _ := graph.Component("C1")
	if source.Value != .5 || capacitor.InitialVoltage == nil || *capacitor.InitialVoltage != -.25 {
		t.Fatal("engineering units not applied in shared commands")
	}
	call(t, s, "capacitor.initial.set", ValueParams{ID: "C1", Value: "default"}, "default")
	graph.Components = s.Snapshot().Components
	capacitor, _ = graph.Component("C1")
	if capacitor.InitialVoltage != nil {
		t.Fatal("default initial override did not clear")
	}
}
func TestValueDetailedContractsAndPlacementBounds(t *testing.T) {
	for _, command := range []string{"component.place", "component.set", "capacitor.initial.set"} {
		caps, err := detailedCapabilities(command)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		_ = json.Unmarshal(caps[0].Input, &schema)
		if schema["x-command-contract"] == nil || inputProperty(schema, "value")["x-max-utf8-bytes"] != float64(128) {
			t.Fatal("value domain not discoverable", command)
		}
		if command == "component.place" {
			if inputProperty(schema, "position.x")["minimum"] != float64(-10000) || inputProperty(schema, "position.y")["maximum"] != float64(10000) || schema["x-position-snap-grid"] != GridSize {
				t.Fatal("placement bounds absent")
			}
		}
	}
	s := New()
	defer s.Close()
	call(t, s, "component.place", PlaceParams{ID: "V1", Kind: "dc-source", Position: Point{X: 9999, Y: -9999}, Value: "500 mV"}, "edge")
	c := s.Snapshot().Components[0]
	if c.Position != (Point{X: 10000, Y: -10000}) {
		t.Fatal("documented grid snap differs")
	}
	before := s.Snapshot()
	refuse(t, s, "component.place", PlaceParams{ID: "V2", Kind: "dc-source", Position: Point{X: 10001}}, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("out-of-bounds placement changed state")
	}
}
