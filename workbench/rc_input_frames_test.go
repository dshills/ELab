package workbench

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestRCManualInputFramesPreserveBothSides(t *testing.T) {
	for _, path := range []string{"step", "resume", "running"} {
		for orientation := 0; orientation < 4; orientation++ {
			t.Run(fmt.Sprintf("%s/pins-%d", path, orientation), func(t *testing.T) {
				graph := RCExampleGraph()
				signs := map[string]float64{"R1": 1, "C1": 1}
				for i, id := range []string{"R1", "C1"} {
					if orientation&(1<<i) == 0 {
						continue
					}
					signs[id] = -1
					swap := func(ref string) string {
						if ref == id+".1" {
							return id + ".2"
						}
						if ref == id+".2" {
							return id + ".1"
						}
						return ref
					}
					for j := range graph.Wires {
						graph.Wires[j].From = swap(graph.Wires[j].From)
						graph.Wires[j].To = swap(graph.Wires[j].To)
					}
				}
				s := newGraphService(SolveLED, graph)
				defer s.Close()
				call(t, s, "simulation.pause", struct{}{}, "pause")
				call(t, s, "simulation.restart", struct{}{}, "restart")
				ids := []string{"v:V1", "v:R1", "v:C1", "i:V1", "i:R1", "i:C1"}
				call(t, s, "scope.probes.set", ProbeParams{IDs: ids}, "probes")
				call(t, s, "simulation.step", StepParams{Duration: .01}, "charge-10ms")
				for _, mode := range []string{"discharge", "charge", "auto"} {
					// Keep the source high beyond its falling pulse corner, so
					// returning to auto also exercises a real falling edge.
					if mode == "auto" {
						call(t, s, "simulation.step", StepParams{Duration: .05}, "past-corner")
					}
					before := s.Snapshot()
					if path == "running" {
						call(t, s, "simulation.resume", struct{}{}, mode+"-run")
					}
					call(t, s, "simulation.input.set", InputParams{Mode: mode}, mode+"-input")
					if path != "running" {
						queued := s.Snapshot()
						if queued.Transient.PendingMode != mode || !reflect.DeepEqual(before.Transient.Samples, queued.Transient.Samples) {
							t.Fatal("paused input changed accepted history before consumption")
						}
						if path == "step" {
							call(t, s, "simulation.step", StepParams{Duration: .001}, mode+"-step")
						} else {
							call(t, s, "simulation.resume", struct{}{}, mode+"-resume")
						}
					}
					call(t, s, "simulation.pause", struct{}{}, mode+"-pause")
					after := s.Snapshot()
					if after.RunID != before.RunID || after.ElectricalRevision != before.ElectricalRevision || after.Document != before.Document {
						t.Fatal("input actuation changed run, circuit or document identity")
					}
					// Scope holds copied numeric values, independently of the
					// private frames shared by accepted history and snapshots.
					for i, trace := range before.Scope.Traces {
						if !reflect.DeepEqual(trace.Points, after.Scope.Traces[i].Points[:len(trace.Points)]) {
							t.Fatal("new input mutated a previously accepted probe frame")
						}
					}
					at := after.Transient.Events[len(after.Transient.Events)-1].Time
					pair := []TraceSample{}
					for _, sample := range after.Transient.Samples {
						if sample.Time == at {
							pair = append(pair, sample)
						}
					}
					// Consecutive manual actions can share one timestamp when
					// no live clock tick occurs between them. Check the latest
					// event's two sides, retaining all earlier edges as well.
					if len(pair) < 2 {
						t.Fatal("input edge lacks retained left/right samples")
					}
					pair = pair[len(pair)-2:]
					if pair[0].Voltage != pair[1].Voltage || pair[0].Energy != pair[1].Energy || pair[0].Input == pair[1].Input {
						t.Fatalf("input edge lacks two distinct algebraic sides with continuous storage: %+v", pair)
					}
					assertRCProbeLaws(t, after, signs)
					call(t, s, "scope.cursor.set", CursorParams{Cursor: "a", Time: at}, mode+"-a")
					call(t, s, "scope.cursor.set", CursorParams{Cursor: "b", Time: at}, mode+"-b")
					readings := s.Snapshot().Scope.Readings
					if len(readings) != len(ids) {
						t.Fatal("cursor readings omitted selected probes")
					}
					for i, reading := range readings {
						want := rcProbeExpectations(pair[1], signs)[i]
						if math.Abs(reading.A-want) > 1e-12 || reading.A != reading.B || reading.Delta != 0 {
							t.Fatalf("cursor did not use the right-hand edge value: %+v, want %g", reading, want)
						}
					}
				}
				state := s.Snapshot()
				file := filepath.Join(t.TempDir(), "edges.csv")
				call(t, s, "export.scope.csv", ExportParams{Path: file}, "csv")
				data, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
				if err != nil || len(rows) != len(state.Transient.Samples)+1 {
					t.Fatal("CSV omitted accepted edge rows", err)
				}
				for j, sample := range state.Transient.Samples {
					for i, want := range rcProbeExpectations(sample, signs) {
						value, err := strconv.ParseFloat(rows[j+1][7+i], 64)
						if err != nil || math.Abs(value-want) > 1e-12 {
							t.Fatalf("CSV row %d probe %s: got %g, want %g (%v)", j, ids[i], value, want, err)
						}
					}
				}
			})
		}
	}
}

func rcProbeExpectations(p TraceSample, signs map[string]float64) []float64 {
	current := (p.Input - p.Voltage) / 10000
	return []float64{p.Input, signs["R1"] * (p.Input - p.Voltage), signs["C1"] * p.Voltage, current, signs["R1"] * current, signs["C1"] * current}
}

func assertRCProbeLaws(t *testing.T, state Snapshot, signs map[string]float64) {
	t.Helper()
	for i, trace := range state.Scope.Traces {
		minValue, maxValue := math.Inf(1), math.Inf(-1)
		if len(trace.Points) != len(state.Transient.Samples) {
			t.Fatal("scope omitted accepted samples")
		}
		for j, sample := range state.Transient.Samples {
			want := rcProbeExpectations(sample, signs)[i]
			if trace.Points[j].Time != sample.Time || math.Abs(trace.Points[j].Value-want) > 1e-12 {
				t.Fatalf("scope probe %s at %g: got %g, want %g", trace.Probe.ID, sample.Time, trace.Points[j].Value, want)
			}
			minValue, maxValue = math.Min(minValue, want), math.Max(maxValue, want)
		}
		stats := state.Scope.Statistics[i]
		if math.Abs(stats.Minimum-minValue) > 1e-12 || math.Abs(stats.Maximum-maxValue) > 1e-12 {
			t.Fatalf("scope extrema missed an input edge: %+v", stats)
		}
	}
}
