package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestComponentValueRefusalExplainsDomainAndPreservesState(t *testing.T) {
	for _, kind := range []string{"resistor", "potentiometer", "capacitor", "inductor", "dc-source", "pulse-source", "sine-source", "finite-pulse-source"} {
		t.Run(kind, func(t *testing.T) {
			part, err := MakeComponent("X1", kind, Point{}, "")
			if err != nil {
				t.Fatal(err)
			}
			s := newGraphService(SolveLED, Graph{Components: []Component{part}})
			defer s.Close()
			settled(t, s)
			call(t, s, "simulation.pause", struct{}{}, "pause")
			before := s.Snapshot()
			domain, ok := ValueDomainForKind(kind)
			if !ok {
				t.Fatal("numeric kind has no domain")
			}
			for i, bad := range []string{"nonsense", fmt.Sprintf("%.17g", domain.Maximum*2), "1e-999"} {
				raw, _ := json.Marshal(ValueParams{ID: "X1", Value: bad})
				revision := before.CommandRevision
				response := s.Dispatch(context.Background(), Request{Version: 1, ID: fmt.Sprint(i), Key: fmt.Sprint("bad-", i), Command: "component.set", Params: raw, ExpectedCommandRevision: &revision}, "agent.test")
				if response.Error == nil || response.Help == nil || response.Error.Code != "invalid-params" {
					t.Fatal("missing refusal", response)
				}
				h := response.Help
				if !strings.Contains(h.Summary, "X1") || !strings.Contains(h.Summary, strings.ReplaceAll(domain.Parameter, "_", " ")) || !strings.Contains(h.Action, domain.Range) || !strings.Contains(h.Action, domain.Syntax) || !strings.Contains(h.Action, FormatValue(part.Value, part.Unit)) {
					t.Fatalf("value refusal hides its actual domain/current value: %+v", h)
				}
				if h.Details != response.Error.Message || !reflect.DeepEqual(h.Objects, []string{"X1"}) || !reflect.DeepEqual(before, s.Snapshot()) {
					t.Fatal("guidance altered state, target or original error", h)
				}
			}
		})
	}
}

func TestComponentValueRefusalUsesCurrentWaveformConstraints(t *testing.T) {
	for _, kind := range []string{"sine-source", "finite-pulse-source"} {
		t.Run(kind, func(t *testing.T) {
			part, err := MakeComponent("V1", kind, Point{}, "")
			if err != nil {
				t.Fatal(err)
			}
			bad, need := "9 V", "8 V"
			if kind == "sine-source" {
				part.Sine.Offset = 4
			} else {
				part.Pulse.LowVoltage = 3
				bad = "2 V"
				need = "3 V"
			}
			s := newGraphService(SolveLED, Graph{Components: []Component{part}})
			defer s.Close()
			settled(t, s)
			call(t, s, "simulation.pause", struct{}{}, "pause")
			before := s.Snapshot()
			raw, _ := json.Marshal(ValueParams{ID: "V1", Value: bad})
			rev := before.CommandRevision
			response := s.Dispatch(context.Background(), Request{Version: 1, ID: "coupled", Key: "coupled", Command: "component.set", Params: raw, ExpectedCommandRevision: &rev}, "agent.test")
			if response.Error == nil || response.Help == nil || !strings.Contains(response.Help.Action, need) || !reflect.DeepEqual(before, s.Snapshot()) {
				t.Fatal("guidance misses the current coupled limit", response.Help)
			}
		})
	}
}

func TestComponentValueGuidanceRetainsItsRequestContext(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	raw, _ := json.Marshal(ValueParams{ID: "R1", Value: "0"})
	rev := s.Snapshot().CommandRevision
	request := Request{Version: 1, ID: "refusal", Key: "refusal", Command: "component.set", Params: raw, ExpectedCommandRevision: &rev}
	first := s.Dispatch(context.Background(), request, "agent.test")
	if first.Help == nil || !strings.Contains(first.Help.Action, "1 kΩ") {
		t.Fatal("missing prior value", first.Help)
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "other-actor")
	before := s.Snapshot()
	retry := s.Dispatch(context.Background(), request, "agent.test")
	if retry.Error == nil || retry.Error.Code != "conflict" || retry.Help != nil || retry.Duplicate || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("stale rejected request acquired current-value guidance", retry)
	}
	if !strings.Contains(first.Help.Action, "1 kΩ") || strings.Contains(first.Help.Action, "2 kΩ") {
		t.Fatal("original refusal changed after another actor's edit", first.Help)
	}
	freshRevision := before.CommandRevision
	fresh := s.Dispatch(context.Background(), Request{Version: 1, ID: "fresh", Key: "fresh", Command: "component.set", Params: raw, ExpectedCommandRevision: &freshRevision}, "agent.test")
	if fresh.Error == nil || fresh.Help == nil || !strings.Contains(fresh.Help.Action, "2 kΩ") || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("fresh refusal does not explain the current accepted value", fresh.Help)
	}
}

func TestMalformedOrUnknownValueTargetDoesNotInventDomainGuidance(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	for i, raw := range []string{`{"id":"R1","value":"0","extra":true}`, `{"id":"R1","id":"missing","value":"0"}`, `{"id":"missing","value":"0"}`, `{"id":"R1"}`, `{"id":"R1","value":0}`} {
		before := s.Snapshot()
		rev := before.CommandRevision
		response := s.Dispatch(context.Background(), Request{Version: 1, ID: fmt.Sprint(i), Key: fmt.Sprint(i), Command: "component.set", Params: json.RawMessage(raw), ExpectedCommandRevision: &rev}, "agent.test")
		if response.Error == nil || response.Help == nil || response.Help.Summary != "That request could not be applied." || strings.Contains(response.Help.Action, "10–100000") || !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("malformed request acquired a fabricated numeric explanation", response)
		}
	}
}
