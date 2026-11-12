package main

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestCompareChangeMeasuresAnIsolatedRCEdit(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	f.wait()
	original := f.state()
	probe := original.Scope.Selected[0]
	raw, err := json.Marshal(compareChangeParams{Part: "C1", Value: "2.2 uF", Probe: probe, Duration: .04})
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/compare-change?" + f.query("")
	response := f.request("POST", path, raw, "")
	var report struct {
		Before []workbench.VerificationRow
		After  []workbench.VerificationRow
	}
	if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &report) != nil {
		t.Fatal(response.Code, response.Body.String())
	}
	if len(report.Before) != 41 || len(report.After) != 41 {
		t.Fatal("comparison did not use the same time grid")
	}
	for _, item := range []struct {
		rows []workbench.VerificationRow
		tau  float64
	}{{report.Before, .01}, {report.After, .022}} {
		want := 5 * (1 - math.Exp(-.02/item.tau))
		if math.Abs(item.rows[20].Values[0]-want) > 1e-8 {
			t.Fatalf("at 20 ms got %.9g V, want %.9g V", item.rows[20].Values[0], want)
		}
	}
	after := f.state()
	if original.CommandRevision != after.CommandRevision || original.RunID != after.RunID ||
		original.Document != after.Document || len(after.Operations) != len(original.Operations) {
		t.Fatal("comparison changed the live project")
	}
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "22k"})
	if stale := f.request("POST", path, raw, ""); stale.Code != 409 {
		t.Fatal("stale comparison guard accepted", stale.Code)
	}
}
