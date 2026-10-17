package workbench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRecordedMOSFETReadingsMustMatchUnprobedTerminalState(t *testing.T) {
	for _, kind := range []string{"nmos", "pmos"} {
		t.Run(kind, func(t *testing.T) {
			graph := nmosDCGraph()
			if kind == "pmos" {
				graph = pmosDCGraph(0)
			}
			source := newGraphService(SolveLED, graph)
			defer source.Close()
			call(t, source, "simulation.pause", struct{}{}, "pause")
			initial := settled(t, source)
			if initial.Mixed == nil || len(initial.Mixed.MOSFETs) != 1 || initial.Mixed.MOSFETs[0].State != "triode" {
				t.Fatal("invalid conducting fixture", initial.Diagnostics)
			}
			// Deliberately omit all MOSFET probes: final device metadata must still be
			// validated, rather than trusting whichever readings happen to be plotted.
			path, _ := exportedRecording(t, source, 0, []string{"v:VIN.positive"})
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = decodeRecording(original); err != nil {
				t.Fatal("valid capture refused", err)
			}
			projectPath := filepath.Join(t.TempDir(), "saved.elab")
			call(t, source, "project.save", SaveParams{Path: projectPath}, "save")
			projectBytes, err := os.ReadFile(projectPath)
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{"gate_current_a", "drain_current_a", "source_current_a", "vgs_v", "vds_v", "power_w", "state", "missing-ground", "foreign-node", "missing-terminal-node"} {
				t.Run(field, func(t *testing.T) {
					var document VerificationDocument
					if err := json.Unmarshal(original, &document); err != nil {
						t.Fatal(err)
					}
					result := document.Metadata.Mixed
					switch field {
					case "gate_current_a":
						result.MOSFETs[0].GateCurrent = .001
					case "drain_current_a":
						result.MOSFETs[0].DrainCurrent += .001
					case "source_current_a":
						result.MOSFETs[0].SourceCurrent += .001
					case "vgs_v":
						result.MOSFETs[0].VGS += .25
					case "vds_v":
						result.MOSFETs[0].VDS += .25
					case "power_w":
						result.MOSFETs[0].Power += .001
					case "state":
						result.MOSFETs[0].State = "cutoff"
					case "missing-ground":
						c, _, _ := graph.Compile()
						delete(result.Nodes, c.Ground)
					case "foreign-node":
						result.Nodes["foreign-node"] = 0
					case "missing-terminal-node":
						c, _, _ := graph.Compile()
						delete(result.Nodes, c.TerminalNet["M1.gate"])
					}
					changed, err := json.Marshal(document)
					if err != nil {
						t.Fatal(err)
					}
					bad := filepath.Join(t.TempDir(), "changed.json")
					if err = os.WriteFile(bad, changed, 0600); err != nil {
						t.Fatal(err)
					}
					target := pausedRC(t)
					defer target.Close()
					refuse(t, target, "verification.import", RecordingImportParams{Path: bad}, "invalid-recording")
					var project Document
					if err := json.Unmarshal(projectBytes, &project); err != nil {
						t.Fatal(err)
					}
					project.Recordings.Entries[0].Data = document
					encoded, err := json.Marshal(project)
					if err != nil {
						t.Fatal(err)
					}
					badProject := filepath.Join(t.TempDir(), "changed.elab")
					if err = os.WriteFile(badProject, encoded, 0600); err != nil {
						t.Fatal(err)
					}
					refuse(t, target, "project.load", LoadParams{Path: badProject, DiscardChanges: true}, "invalid-document")
					if field == "gate_current_a" {
						before := target.Snapshot()
						call(t, target, "verification.import.start", RecordingImportParams{Path: bad}, "async-invalid")
						after := awaitProjectStatus(t, target, "failed")
						if after.ProjectWork.Error == nil || after.ProjectWork.Error.Code != "invalid-recording" || after.ProjectWork.ResultOperationID != "" {
							t.Fatal("invalid asynchronous import lost its failure or attached a result")
						}
						after.ProjectWork = before.ProjectWork
						after.Revision, after.CommandRevision, after.LastActor = before.Revision, before.CommandRevision, before.LastActor
						if !reflect.DeepEqual(before, after) {
							t.Fatal("failed asynchronous import changed project, physics, history or recordings")
						}
					}
				})
			}
		})
	}
}

func TestRecordedMOSFETLegitimateRegionsAndBodyDirectionsRoundTrip(t *testing.T) {
	reverseGraph := func(kind string) Graph {
		graph := EmptyGraph()
		for i, entry := range []struct{ id, kind, value string }{{"V1", "dc-source", "5 V"}, {"R1", "resistor", "1k"}, {"M1", kind, ""}, {"GND", "ground", ""}} {
			p, err := MakeComponent(entry.id, entry.kind, Point{X: float64(i) * 200}, entry.value)
			if err != nil {
				t.Fatal(err)
			}
			graph.Components = append(graph.Components, p)
		}
		source, drain := "M1.source", "M1.drain"
		if kind == "pmos" {
			source, drain = drain, source
		}
		for i, pins := range [][2]string{{"V1.positive", source}, {"M1.gate", "M1.source"}, {drain, "R1.1"}, {"R1.2", "GND.reference"}, {"V1.negative", "GND.reference"}} {
			graph.Wires = append(graph.Wires, Wire{ID: fmt.Sprintf("W%d", i+1), From: pins[0], To: pins[1], Connected: true})
		}
		return graph
	}
	nmosGate := func(value float64) Graph {
		g := nmosDCGraph()
		for i := range g.Components {
			if g.Components[i].ID == "VIN" {
				g.Components[i].Value = value
			}
		}
		return g
	}
	for _, entry := range []struct {
		name, state, probe string
		graph              Graph
	}{
		{"nmos-cutoff", "cutoff", "v:VIN.positive", nmosGate(0)},
		{"nmos-saturation", "saturation", "v:VIN.positive", nmosGate(1.6)},
		{"nmos-triode", "triode", "v:VIN.positive", nmosGate(5)},
		{"pmos-cutoff", "cutoff", "v:VIN.positive", pmosDCGraph(5)},
		{"pmos-saturation", "saturation", "v:VIN.positive", pmosDCGraph(3.4)},
		{"pmos-triode", "triode", "v:VIN.positive", pmosDCGraph(0)},
		{"nmos-reverse", "reverse", "v:V1.positive", reverseGraph("nmos")},
		{"pmos-reverse", "reverse", "v:V1.positive", reverseGraph("pmos")},
		{"complementary", "", "v:VIN.positive", cmosTestGraph(t, 2.5)},
	} {
		t.Run(entry.name, func(t *testing.T) {
			source := newGraphService(SolveLED, entry.graph)
			defer source.Close()
			call(t, source, "simulation.pause", struct{}{}, "pause")
			initial := settled(t, source)
			if initial.Mixed == nil || entry.state != "" && initial.Mixed.MOSFETs[0].State != entry.state {
				t.Fatal("wrong region fixture", initial.Diagnostics, initial.Mixed)
			}
			path, _ := exportedRecording(t, source, 0, []string{entry.probe})
			target := pausedRC(t)
			defer target.Close()
			before := target.Snapshot()
			call(t, target, "verification.import", RecordingImportParams{Path: path}, "import")
			after := target.Snapshot()
			if after.RunID != before.RunID || !reflect.DeepEqual(after.Mixed, before.Mixed) || !reflect.DeepEqual(after.Transient, before.Transient) || !reflect.DeepEqual(after.Components, before.Components) {
				t.Fatal("import changed live physics")
			}
			copied := filepath.Join(t.TempDir(), "copy.json")
			call(t, target, "verification.export", VerificationExportParams{ID: after.Recording.ID, Path: copied, Format: "json"}, "copy")
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			reexported, err := os.ReadFile(copied)
			if err != nil || !bytes.Equal(original, reexported) {
				t.Fatal("legitimate MOSFET capture did not round trip exactly", err)
			}
		})
	}
}

func TestRecordedStorageAndFinalProbeMustDescribeTheSameState(t *testing.T) {
	source := New()
	defer source.Close()
	call(t, source, "example.load", ExampleLoadParams{ID: "rlc"}, "example")
	call(t, source, "simulation.pause", struct{}{}, "pause")
	call(t, source, "simulation.restart", struct{}{}, "restart")
	settled(t, source)
	path, _ := exportedRecording(t, source, .005, []string{"v:C1", "i:L1"})
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"capacitor-voltage", "capacitor-energy", "inductor-current-with-consistent-energy"} {
		t.Run(name, func(t *testing.T) {
			var document VerificationDocument
			if err := json.Unmarshal(original, &document); err != nil {
				t.Fatal(err)
			}
			result := document.Metadata.Mixed
			switch name {
			case "capacitor-voltage":
				result.Capacitors[0].Voltage += .25
			case "capacitor-energy":
				result.Capacitors[0].Energy += .001
			case "inductor-current-with-consistent-energy":
				result.Inductors[0].Current += .001
				part, _ := document.Metadata.Graph.Component(result.Inductors[0].ID)
				result.Inductors[0].Energy = .5 * part.Value * result.Inductors[0].Current * result.Inductors[0].Current
			}
			raw, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			bad := filepath.Join(t.TempDir(), "changed.json")
			if err = os.WriteFile(bad, raw, 0600); err != nil {
				t.Fatal(err)
			}
			target := pausedRC(t)
			defer target.Close()
			refuse(t, target, "verification.import", RecordingImportParams{Path: bad}, "invalid-recording")
		})
	}
}
