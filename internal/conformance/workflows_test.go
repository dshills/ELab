package conformance

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestThreeReferenceWorkflowsAndComparisonRejectsMissingEvidence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	results, err := Run(ctx, InProcess, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = Compare(results, results); err != nil {
		t.Fatal(err)
	}
	// Require evidence for each workflow, its command sequence, semantic project
	// and measured values. A green report must not tolerate missing coverage.
	damaged := append([]Result{}, results...)
	damaged[0].Measurements = map[string]float64{}
	if Compare(results, damaged) == nil {
		t.Fatal("missing numerical evidence accepted")
	}
	damaged = append([]Result{}, results...)
	damaged[0].Commands = []string{"project.inspect"}
	if Compare(results, damaged) == nil {
		t.Fatal("missing workflow actions accepted")
	}
	damaged = append([]Result{}, results...)
	damaged[0].Project.Graph = results[0].Project.Graph.Clone()
	damaged[0].Project.Graph.Components[0].Value++
	if Compare(results, damaged) == nil {
		t.Fatal("changed semantic project accepted")
	}
}

func TestComparisonPreservesParameterAndRecordedRowEvidence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	results, err := Run(ctx, InProcess, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(results)
	if err != nil {
		t.Fatal(err)
	}
	var changed []Result
	if err = json.Unmarshal(data, &changed); err != nil {
		t.Fatal(err)
	}
	changed[0].Calls[3].Params = json.RawMessage(`{"kind":"resistor","value":"3k"}`)
	if Compare(results, changed) == nil {
		t.Fatal("changed request parameters accepted")
	}
	if err = json.Unmarshal(data, &changed); err != nil {
		t.Fatal(err)
	}
	changed[1].Project.Recordings.Entries[0].Data.Rows[0].Values[0]++
	if Compare(results, changed) == nil {
		t.Fatal("changed recorded sample accepted")
	}
	if err = json.Unmarshal(data, &changed); err != nil {
		t.Fatal(err)
	}
	for key := range changed[0].Measurements {
		changed[0].Measurements[key] = math.NaN()
		break
	}
	if Compare(changed, changed) == nil {
		t.Fatal("nonfinite self comparison accepted")
	}
	if Compare(nil, nil) == nil {
		t.Fatal("empty release comparison accepted")
	}
}
