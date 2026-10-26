package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func explanationQuery(t *testing.T, s *Service, id string) *ComponentExplanation {
	t.Helper()
	raw, _ := json.Marshal(IDParams{ID: id})
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "explain", Command: "component.explain", Params: raw}, "reader")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	return r.Result.(*ComponentExplanation)
}

func TestComponentExplanationCurrentStaleUnavailableAndOwned(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "component.select", IDParams{ID: "R1"}, "select")
	before := s.Snapshot()
	e := explanationQuery(t, s, "R1")
	if e.ReadingState != "live" || !strings.Contains(e.Summary, "Paused") || len(e.Readings) == 0 || e.Limitations == "" || !reflect.DeepEqual(e, before.Explanation) || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("query and visible explanation differ or change accepted state", e)
	}
	copy := cloneSnapshot(before)
	copy.Explanation.Facts[0] = "caller"
	*copy.Explanation.Readings[0].Value = 999
	if reflect.DeepEqual(copy.Explanation, before.Explanation) || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("caller mutated retained explanation")
	}
	call(t, s, "measurement.select", MeasurementSelectParams{Kind: "terminal", ID: "D1.anode"}, "inspect-other-source")
	if !reflect.DeepEqual(s.Snapshot().Explanation, explanationQuery(t, s, "R1")) {
		t.Fatal("component explanation borrowed an explicitly different measurement focus")
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "change")
	stale := explanationQuery(t, s, "D1")
	if stale.ReadingState != "stale" || !strings.Contains(stale.Summary, "Previous") {
		t.Fatal("previous LED reading explained as current", stale)
	}
	blank := New()
	defer blank.Close()
	call(t, blank, "component.place", PlaceParams{ID: "R1", Kind: "resistor"}, "resistor")
	missing := explanationQuery(t, blank, "R1")
	if missing.ReadingState != "unavailable" || !strings.Contains(strings.Join(missing.Facts, " "), "no other terminal") {
		t.Fatal("disconnected component invented a reading", missing)
	}
}

func TestComponentExplanationTargetsAndGenericModelLimits(t *testing.T) {
	s := New()
	defer s.Close()
	for i, kind := range Library() {
		id := fmt.Sprintf("Part%d", i)
		call(t, s, "component.place", PlaceParams{ID: id, Kind: kind.Kind}, "place-"+kind.Kind)
		e := explanationQuery(t, s, id)
		if e.Model != kind.Model || e.Limitations != kind.Limitations {
			t.Fatal("explanation lost published model limits", kind.Kind)
		}
		call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "clear-"+kind.Kind)
	}
	for _, raw := range []string{`{}`, `{"id":"Part","extra":true}`, `{"id":null}`} {
		r := s.Dispatch(context.Background(), Request{Version: 1, ID: "bad", Command: "component.explain", Params: json.RawMessage(raw)}, "reader")
		if r.Error == nil {
			t.Fatal("invalid explanation selector accepted", raw)
		}
	}
}
