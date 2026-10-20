package workbench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPinnedPreferencesAcceptEveryLibraryProbe(t *testing.T) {
	for _, part := range Library() {
		t.Run(part.Kind, func(t *testing.T) {
			component, err := MakeComponent("PART", part.Kind, Point{}, "")
			if err != nil {
				t.Fatal(err)
			}
			graph := EmptyGraph()
			graph.Components = append(graph.Components, component)
			for _, probe := range probeInventory(graph) {
				preferences := PinPreferences{Items: []PinSpec{{ID: probe.ID, Position: Point{X: 100, Y: 200}}}}
				raw, err := json.Marshal(preferences)
				if err != nil {
					t.Fatal(err)
				}
				for _, current := range []Graph{graph, EmptyGraph()} {
					decoded, err := validatePinPreferences(raw, current)
					if err != nil || !reflect.DeepEqual(decoded, &preferences) {
						t.Errorf("published probe %s must survive source presence/removal: %v", probe.ID, err)
					}
				}
			}
		})
	}
}

func assertRestoredPinReadings(t *testing.T, expected, actual []PinnedReading) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatal("pin count changed on reopen")
	}
	for i, reading := range actual {
		previous := expected[i]
		if reading.Definition != previous.Definition || reading.Probe != previous.Probe || reading.State != "live" || reading.Value == nil || previous.Value == nil || reading.Time != 0 || !(math.Abs(*reading.Value-*previous.Value) <= 1e-12) {
			t.Fatalf("reopen changed definition, descriptor or fresh reading: %+v, want %+v", reading, previous)
		}
	}
}

func TestPinnedAdvancedProbeSaveReopenAndMissingSource(t *testing.T) {
	for _, example := range []struct {
		name, owner string
		graph       Graph
	}{
		{"nmos", "M1", NMOSExampleGraph()},
		{"pmos", "M1", PMOSExampleGraph()},
		{"opamp-follower", "U1", OpAmpFollowerExampleGraph()},
		{"opamp-inverting", "U1", OpAmpInvertingExampleGraph()},
	} {
		probes := []Probe{}
		for _, probe := range probeInventory(example.graph) {
			owner, _, _ := example.graph.Terminal(probe.Positive)
			if owner.ID == example.owner {
				probes = append(probes, probe)
			}
		}
		for start := 0; start < len(probes); start += MaxPinnedReadings {
			t.Run(fmt.Sprintf("%s/probes-%d", example.name, start), func(t *testing.T) {
				s := newGraphService(SolveLED, example.graph)
				defer s.Close()
				call(t, s, "simulation.pause", struct{}{}, "pause")
				call(t, s, "simulation.restart", struct{}{}, "restart")
				before := s.Snapshot()
				for i, probe := range probes[start:min(start+MaxPinnedReadings, len(probes))] {
					call(t, s, "measurement.pin.add", PinSpec{ID: probe.ID, Position: Point{X: float64(i) * 20, Y: 300}}, probe.ID)
				}
				pinned := s.Snapshot()
				if before.RunID != pinned.RunID || before.ElectricalRevision != pinned.ElectricalRevision || !reflect.DeepEqual(before.Mixed, pinned.Mixed) || !reflect.DeepEqual(before.Transient, pinned.Transient) || !reflect.DeepEqual(before.Components, pinned.Components) || !reflect.DeepEqual(before.Wires, pinned.Wires) {
					t.Fatal("pinning changed circuit or accepted physics")
				}
				directory := t.TempDir()
				path := filepath.Join(directory, "pinned.elab")
				call(t, s, "project.save", SaveParams{Path: path}, "save")
				original, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := decodeDocument(original); err != nil {
					t.Fatalf("successful save must be readable: %v", err)
				}
				fresh := New()
				defer fresh.Close()
				call(t, fresh, "project.load", LoadParams{Path: path}, "load")
				loaded := fresh.Snapshot()
				if loaded.Document.Dirty || !reflect.DeepEqual(loaded.Components, pinned.Components) || !reflect.DeepEqual(loaded.Wires, pinned.Wires) {
					t.Fatal("reopen lost circuit or saved preferences")
				}
				assertRestoredPinReadings(t, pinned.PinnedReadings, loaded.PinnedReadings)
				call(t, s, "object.delete", IDParams{ID: example.owner}, "delete-owner")
				missingPath := filepath.Join(directory, "missing-source.elab")
				call(t, s, "project.save", SaveParams{Path: missingPath}, "save-missing")
				missing := New()
				defer missing.Close()
				call(t, missing, "project.load", LoadParams{Path: missingPath}, "load-missing")
				state := missing.Snapshot()
				if state.Document.Dirty || len(state.PinnedReadings) != len(pinned.PinnedReadings) {
					t.Fatal("missing-source preferences lost on reopen")
				}
				for i, pin := range state.PinnedReadings {
					if pin.Definition != pinned.PinnedReadings[i].Definition || pin.State != "unavailable" || pin.Value != nil {
						t.Fatal("missing source invented a value or lost its pin")
					}
				}
				call(t, s, "edit.undo", HistoryParams{}, "restore-owner")
				call(t, s, "simulation.restart", struct{}{}, "restart-restored")
				assertRestoredPinReadings(t, pinned.PinnedReadings, s.Snapshot().PinnedReadings)
				unchanged, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(unchanged, original) {
					t.Fatal("later save/delete/undo altered the original manual file")
				}
			})
		}
	}
}

func TestPinnedAdvancedProbeGrammarStillRejectsUnknownAndWrongOwner(t *testing.T) {
	graph := NMOSExampleGraph()
	for _, id := range []string{"v:M1.no-such-pin", "v:M1.noninverting", "v:M1.input-difference", "v:REMOVED.unknown", "i:REMOVED.gate-source", "v:.gate", "v:M1.gate.source", "p:M1.gate", "v:M1."} {
		raw, err := json.Marshal(PinPreferences{Items: []PinSpec{{ID: id, Position: Point{}}}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := validatePinPreferences(raw, graph); err == nil {
			t.Errorf("invalid/mismatched pin %s accepted", id)
		}
	}
}

func TestPinnedAdvancedProbeRecoveryPreservesManualSave(t *testing.T) {
	for _, example := range []struct {
		name, probe string
		graph       Graph
	}{
		{"nmos", "v:M1.gate-source", NMOSExampleGraph()},
		{"opamp", "v:U1.input-difference", OpAmpFollowerExampleGraph()},
	} {
		t.Run(example.name, func(t *testing.T) {
			s := newGraphService(SolveLED, example.graph)
			defer s.Close()
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			manual := filepath.Join(t.TempDir(), "manual.elab")
			call(t, s, "project.save", SaveParams{Path: manual}, "manual-save")
			original, err := os.ReadFile(manual)
			if err != nil {
				t.Fatal(err)
			}
			directory := t.TempDir()
			configureRecovery(t, s, directory, false)
			call(t, s, "measurement.pin.add", PinSpec{ID: example.probe, Position: Point{X: 500, Y: 300}}, "pin")
			expected := s.Snapshot().PinnedReadings
			call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
			choice := recoveryChoice(t, s)
			s.Close()
			fresh := New()
			defer fresh.Close()
			configureRecovery(t, fresh, directory, false)
			call(t, fresh, "recovery.restore", choice, "restore")
			state := fresh.Snapshot()
			if !state.Document.Dirty || state.Document.Path != "" || state.Document.RecoveredFrom != manual {
				t.Fatal("recovery did not create an unsaved recovered copy")
			}
			assertRestoredPinReadings(t, expected, state.PinnedReadings)
			call(t, fresh, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "recovered.elab")}, "save-copy")
			unchanged, err := os.ReadFile(manual)
			if err != nil || !bytes.Equal(unchanged, original) {
				t.Fatal("recovery or saving its copy altered the last manual save")
			}
		})
	}
}
