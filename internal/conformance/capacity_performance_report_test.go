package conformance

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"math"
	"os"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestCapacityPerformancePublicReportRejectsCorruption(t *testing.T) {
	file, err := os.Open("testdata/capacity-performance-public-reference.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	var reference PerformanceResult
	if err := json.Unmarshal(raw, &reference); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCapacityPerformance(reference); err != nil {
		t.Fatal("genuine public timing evidence refused", err)
	}
	for _, mutation := range []string{"missing-family", "missing-case", "missing-sample", "missing-graph", "bad-graph-hash", "no-op-edit", "warm-run-reuse", "wrong-live-engine", "wrong-live-settings", "missing-capture", "wrong-captured-revision", "wrong-captured-engine", "wrong-captured-graph", "wrong-model", "wrong-model-map", "missing-probe", "wrong-unit", "missing-edge", "changed-voltage", "changed-current", "foreign-node", "stale-final", "negative-kcl", "nan-final-time", "changed-statistics", "changed-percentile", "negative-timing", "nan-timing", "false-target", "false-aggregate"} {
		t.Run(mutation, func(t *testing.T) {
			changed := reference
			changed.Cases = append([]PerformanceCase(nil), reference.Cases...)
			for i := range changed.Cases {
				changed.Cases[i].Samples = append([]PerformanceSample(nil), reference.Cases[i].Samples...)
			}
			changed.Circuits = map[string]workbench.Graph{}
			for key, graph := range reference.Circuits {
				changed.Circuits[key] = graph.Clone()
			}
			sample := &changed.Cases[4].Samples[0]
			clone, err := json.Marshal(sample)
			if err != nil {
				t.Fatal(err)
			}
			var copied PerformanceSample
			if err := json.Unmarshal(clone, &copied); err != nil {
				t.Fatal(err)
			}
			*sample = copied
			capture := sample.CapacityVerification
			switch mutation {
			case "missing-family":
				changed.Families = reference.Families[:2]
			case "missing-case":
				changed.Cases = changed.Cases[:5]
			case "missing-sample":
				changed.Cases[0].Samples = changed.Cases[0].Samples[:19]
			case "missing-graph":
				delete(changed.Circuits, sample.GraphSHA256)
			case "bad-graph-hash":
				sample.GraphSHA256 = "foreign"
			case "no-op-edit":
				changed.Cases[5].Samples[1] = changed.Cases[5].Samples[0]
				changed.Cases[5].Samples[1].Index = 1
			case "warm-run-reuse":
				changed.Cases[5].Samples[1].RunID = changed.Cases[5].Samples[0].RunID
			case "wrong-live-engine":
				sample.Engine = "foreign"
			case "wrong-live-settings":
				sample.Settings.HighSeconds = .05
			case "missing-capture":
				sample.CapacityVerification = nil
			case "wrong-captured-revision":
				capture.Metadata.ElectricalRevision++
			case "wrong-captured-engine":
				capture.Metadata.Mixed.Engine = "foreign"
			case "wrong-captured-graph":
				capture.Metadata.Graph.Components[0].Value = 6
			case "wrong-model":
				capture.Metadata.NMOSModel.Threshold++
			case "wrong-model-map":
				sample.Models["M1"] = "foreign"
			case "missing-probe":
				capture.Metadata.Probes = capture.Metadata.Probes[:3]
			case "wrong-unit":
				capture.Metadata.Probes[0].Unit = "A"
			case "missing-edge":
				capture.Rows = append(capture.Rows[:100], capture.Rows[101:]...)
			case "changed-voltage":
				capture.Rows[50].Values[0] += .1
			case "changed-current":
				capture.Rows[50].Values[3] += .001
			case "foreign-node":
				for key, value := range capture.Metadata.Mixed.Nodes {
					delete(capture.Metadata.Mixed.Nodes, key)
					capture.Metadata.Mixed.Nodes["foreign"] = value
					break
				}
			case "stale-final":
				capture.Metadata.Mixed.Stale = true
			case "negative-kcl":
				capture.Metadata.Mixed.KCLResidual = -1e-9
			case "nan-final-time":
				capture.Metadata.Mixed.Time = math.NaN()
			case "changed-statistics":
				capture.Metadata.Statistics[0].Mean += .1
			case "changed-percentile":
				changed.Cases[0].Verify.P95++
			case "negative-timing":
				sample.AcknowledgmentMS = -1
			case "nan-timing":
				sample.VerifyMS = math.NaN()
			case "false-target":
				changed.Cases[0].FreshTargetPassed = !changed.Cases[0].FreshTargetPassed
			case "false-aggregate":
				changed.TargetsPassed = !changed.TargetsPassed
			}
			if err := ValidateCapacityPerformance(changed); err == nil {
				t.Fatal("corrupted timing report accepted")
			}
		})
	}
	if err := ValidateCapacityPerformance(reference); err != nil {
		t.Fatal("corruption tests changed the retained original", err)
	}
}
