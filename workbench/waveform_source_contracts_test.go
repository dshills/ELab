package workbench

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWaveformSourceContractsKeepDefaultDiscoveryAndQuantityBounds(t *testing.T) {
	before, _ := json.Marshal(Catalog())
	registry, _ := json.Marshal(SchemaRegistry())
	for _, id := range []string{"source.sine.set", "source.pulse.set"} {
		details, err := detailedCapabilities(id)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(details[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		contract := schema["x-command-contract"].(map[string]any)
		if contract["scope"] != "per-source-waveform" || len(contract["preconditions"].([]any)) == 0 || len(contract["errors"].([]any)) == 0 || schema["x-normalized-si-relations"] == nil {
			t.Fatal("missing stateful waveform contract or existing coupled bounds", id)
		}
		details[0].Input[0] = '!'
		fresh, err := detailedCapabilities(id)
		if err != nil || !json.Valid(fresh[0].Input) {
			t.Fatal("caller mutation corrupted later waveform discovery", err)
		}
	}
	after, _ := json.Marshal(Catalog())
	output, _ := json.Marshal(SchemaRegistry())
	if !bytes.Equal(before, after) || !bytes.Equal(registry, output) {
		t.Fatal("detailed source contracts changed ordinary catalog or numeric output schemas")
	}
}

func TestWaveformReplacementRequiresEveryFieldAndRefusesAtomically(t *testing.T) {
	for _, kind := range []string{"sine-source", "finite-pulse-source"} {
		t.Run(kind, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "component.place", PlaceParams{ID: "Wave", Kind: kind}, "source")
			command := "source.sine.set"
			wave := map[string]any{"amplitude_v": 0, "offset_v": -12, "frequency_hz": .1, "phase_degrees": -360}
			if kind == "finite-pulse-source" {
				command = "source.pulse.set"
				// Constant levels and no plateaus remain valid definitions.
				wave = map[string]any{"high_v": -12, "low_v": -12, "high_s": .0005, "low_s": .0005, "rise_s": .0005, "fall_s": .0005}
			}
			p := map[string]any{"id": "Wave", "waveform": wave}
			call(t, s, command, p, "constant")
			before := s.Snapshot()
			for field := range wave {
				for _, missing := range []bool{true, false} {
					bad := make(map[string]any, len(wave))
					for name, value := range wave {
						bad[name] = value
					}
					if missing {
						delete(bad, field)
					} else {
						bad[field] = nil
					}
					refuse(t, s, command, map[string]any{"id": "Wave", "waveform": bad}, "invalid-params")
					if !reflect.DeepEqual(before, s.Snapshot()) {
						t.Fatal("incomplete replacement altered configured source", field, missing)
					}
				}
			}
			for _, target := range []string{"missing", ""} {
				refuse(t, s, command, map[string]any{"id": target, "waveform": wave}, "invalid-params")
				if !reflect.DeepEqual(before, s.Snapshot()) {
					t.Fatal("missing source refusal changed state")
				}
			}
			call(t, s, command, p, "same")
			after := s.Snapshot()
			if !reflect.DeepEqual(before.Components, after.Components) || before.Document != after.Document || before.RunID != after.RunID || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
				t.Fatal("reapplying complete constant source changed physics/history")
			}
		})
	}
}

func TestWaveformPausedReplacementPreservesAcceptedSampleAndCapturedRows(t *testing.T) {
	for _, kind := range []string{"sine-source", "finite-pulse-source"} {
		t.Run(kind, func(t *testing.T) {
			s := newGraphService(SolveLED, waveformRCGraph(t, kind))
			defer s.Close()
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			call(t, s, "simulation.step", StepParams{Duration: .005}, "advance")
			duration := .005
			_, record := verify(t, s, VerifyParams{Duration: &duration}, "capture")
			path := filepath.Join(t.TempDir(), "before.json")
			call(t, s, "verification.export", VerificationExportParams{ID: record.Operation.ID, Path: path, Format: "json"}, "export-before")
			before := s.Snapshot()
			command := "source.sine.set"
			var configured any = SineParams{ID: "V1", Waveform: SineWaveform{Amplitude: 1, Offset: 2, Frequency: 20, Phase: 30}}
			if kind == "finite-pulse-source" {
				command = "source.pulse.set"
				configured = PulseParams{ID: "V1", Waveform: PulseWaveform{HighVoltage: 3, LowVoltage: -1, HighSeconds: .02, LowSeconds: .03, RiseSeconds: .002, FallSeconds: .003}}
			}
			batch := BatchParams{Name: "Replace waveform", Edits: []BatchEdit{edit(command, configured)}}
			call(t, s, "edit.batch", batch, "replace")
			after := s.Snapshot()
			if !after.Paused || !after.Transient.Stale || !after.Mixed.Stale || after.Transient.Sample.Time != .005 || !reflect.DeepEqual(before.Transient.Sample, after.Transient.Sample) || !reflect.DeepEqual(before.Mixed.Capacitors, after.Mixed.Capacitors) || after.TransientSettings != before.TransientSettings || !reflect.DeepEqual(before.Wires, after.Wires) {
				t.Fatal("paused waveform edit replaced accepted charge/time or changed sampling/topology")
			}
			for i := range before.Components {
				if before.Components[i].ID != "V1" && !reflect.DeepEqual(before.Components[i], after.Components[i]) {
					t.Fatal("waveform replacement altered another part")
				}
			}
			var stagedChange any = SineParams{ID: "V1", Waveform: DefaultSineWaveform()}
			if kind == "finite-pulse-source" {
				stagedChange = PulseParams{ID: "V1", Waveform: DefaultPulseWaveform()}
			}
			batch.Edits = []BatchEdit{edit(command, stagedChange), edit(command, map[string]any{"id": "missing", "waveform": nil})}
			refuse(t, s, "edit.batch", batch, "invalid-params")
			if !reflect.DeepEqual(after, s.Snapshot()) {
				t.Fatal("failed batch partially changed source, history or physics")
			}
			call(t, s, "edit.undo", HistoryParams{}, "undo")
			if !reflect.DeepEqual(before.Components, s.Snapshot().Components) {
				t.Fatal("one undo did not restore waveform definition")
			}
			second := filepath.Join(t.TempDir(), "after.json")
			call(t, s, "verification.export", VerificationExportParams{ID: record.Operation.ID, Path: second, Format: "json"}, "export-after")
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			retained, err := os.ReadFile(second)
			if err != nil || !bytes.Equal(original, retained) {
				t.Fatal("source edit/undo changed immutable capture bytes", err)
			}
		})
	}
}
