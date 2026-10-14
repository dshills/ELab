package workbench

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

func TestHobbyistAliasesAndLibraryMetadataIsolation(t *testing.T) {
	for _, test := range []struct{ query, kind string }{{" BATTERY ", "dc-source"}, {"power  supply", "dc-source"}, {"VARIABLE RESISTOR", "potentiometer"}, {"square wave", "finite-pulse-source"}, {"light emitting diode", "led"}, {"BJT", "npn"}, {"AC source", "sine-source"}, {"rectifier", "diode"}} {
		matches := filteredParts(test.query)
		found := false
		for _, p := range matches {
			if p.Kind == test.kind {
				found = true
			}
		}
		if !found {
			t.Fatalf("%q did not find %s", test.query, test.kind)
		}
	}
	parts := Library()
	for _, p := range parts {
		if p.Limitations == "" || len(p.Aliases) == 0 || p.Model == "" || len(p.Terminals) == 0 {
			t.Fatalf("part metadata incomplete: %s", p.Kind)
		}
	}
	parts[0].Aliases[0] = "corrupted"
	if Library()[0].Aliases[0] != "battery" {
		t.Fatal("library alias storage is shared")
	}
	s := NewExample()
	defer s.Close()
	settled(t, s)
	before := s.Snapshot()
	call(t, s, "parts.search.set", SearchParams{Query: "battery"}, "search")
	after := s.Snapshot()
	if after.ElectricalRevision != before.ElectricalRevision || after.RunID != before.RunID || !reflect.DeepEqual(after.Components, before.Components) || after.MatchingParts[0].Kind != "dc-source" {
		t.Fatal("alias search changed physics or did not reach shared state")
	}
}
func TestDuplicateNegativePulsePreservesCompleteParameters(t *testing.T) {
	s := newGraphService(SolveLED, waveformRCGraph(t, "finite-pulse-source"))
	defer s.Close()
	settled(t, s)
	cfg := PulseWaveform{HighVoltage: -1, LowVoltage: -3, HighSeconds: .02, LowSeconds: .03, RiseSeconds: .001, FallSeconds: .002}
	call(t, s, "source.pulse.set", PulseParams{ID: "V1", Waveform: cfg}, "configure")
	call(t, s, "component.rotate", IDParams{ID: "V1"}, "rotate")
	before := s.Snapshot()
	call(t, s, "component.duplicate", DuplicateParams{ID: "V1"}, "duplicate")
	state := s.Snapshot()
	copy, ok := s.graph.Component(state.Selected)
	if !ok || copy.Pulse == nil || *copy.Pulse != cfg || copy.Value != -1 || copy.Rotation != 90 || len(state.Wires) != len(before.Wires) {
		t.Fatal("negative pulse duplication lost source metadata or copied wires")
	}
	*copy.Pulse = DefaultPulseWaveform()
	original, _ := s.graph.Component("V1")
	if *original.Pulse != cfg {
		t.Fatal("duplicate shares source parameter storage")
	}
	call(t, s, "edit.undo", struct{}{}, "undo")
	if !reflect.DeepEqual(s.Snapshot().Components, before.Components) {
		t.Fatal("duplicate undo failed")
	}
}
func TestLiveCursorFrequencyDoesNotOverflowTransport(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{Duration: 1e-320}, "tiny-step")
	call(t, s, "scope.cursor.set", CursorParams{Cursor: "a", Time: 0}, "a")
	call(t, s, "scope.cursor.set", CursorParams{Cursor: "b", Time: 1e-320}, "b")
	state := s.Snapshot()
	if state.Scope.DeltaSeconds == nil || *state.Scope.DeltaSeconds == 0 || state.Scope.Frequency != nil || !state.Scope.A.Valid || !state.Scope.B.Valid {
		t.Fatal("tiny cursor interval was lost or produced an infinite frequency")
	}
	if state.Transient.Sample.Voltage <= 0 {
		t.Fatal("exact RC update lost a representable tiny voltage change")
	}
	if _, err := json.Marshal(state); err != nil {
		t.Fatalf("snapshot cannot cross the transport: %v", err)
	}
	if !math.IsInf(1 / *state.Scope.DeltaSeconds, 1) {
		t.Fatal("fixture did not exercise frequency overflow")
	}
}

func TestRemovedProbeRemainsUnavailableUntilUndoOrExplicitRemoval(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"v:C1", "i:C1", "v:R1"}}, "probes")
	call(t, s, "object.delete", IDParams{ID: "C1"}, "remove-source")
	state := s.Snapshot()
	if len(state.Scope.Unavailable) != 2 || len(state.Scope.Selected) != 1 || state.Scope.Unavailable[0].ID != "v:C1" || state.Scope.Unavailable[0].Unit != "V" {
		t.Fatal("removed probe disappeared without an unavailable marker")
	}
	call(t, s, "edit.undo", struct{}{}, "undo-source")
	if len(s.Snapshot().Scope.Unavailable) != 0 || len(s.Snapshot().Scope.Selected) != 3 {
		t.Fatal("undo did not restore source/probe associations")
	}
	call(t, s, "object.delete", IDParams{ID: "C1"}, "remove-again")
	before := s.Snapshot()
	call(t, s, "scope.probe.remove", IDParams{ID: "v:C1"}, "remove-probe")
	after := s.Snapshot()
	if len(after.Scope.Unavailable) != 1 || after.Scope.Unavailable[0].ID != "i:C1" || !reflect.DeepEqual(after.Components, before.Components) || after.ElectricalRevision != before.ElectricalRevision || after.RunID != before.RunID {
		t.Fatal("single unavailable-probe removal changed physics or discarded another request")
	}
	refuse(t, s, "scope.probe.remove", IDParams{ID: "v:C1"}, "not-found")
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"v:R1"}}, "reassign")
	if len(s.Snapshot().Scope.Unavailable) != 0 {
		t.Fatal("explicit reassignment retained unavailable probes")
	}
}

func TestInterruptedDefaultScopeRetainsStaleTraceAssociations(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	initial := s.Snapshot()
	call(t, s, "object.delete", IDParams{ID: "W2"}, "wire")
	broken := s.Snapshot()
	if !reflect.DeepEqual(broken.Scope.Selected, initial.Scope.Selected) || !broken.Scope.Stale || len(broken.Scope.Unavailable) != 0 {
		t.Fatal("incomplete wire edit discarded the visible default scope")
	}
	call(t, s, "edit.undo", struct{}{}, "undo-wire")
	call(t, s, "object.delete", IDParams{ID: "C1"}, "source")
	if len(s.Snapshot().Scope.Unavailable) != 1 {
		t.Fatal("default plotted source disappeared without unavailable markers")
	}
}
