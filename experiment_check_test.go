package main

import (
	"encoding/json"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestSavedExperimentCheckRoundTrip(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	f.wait()
	original := f.state()
	check := workbench.ExperimentCheck{Name: "Capacitor reaches 3 V", Probe: "v:C1", Unit: "V", Rule: "rises", Level: "3 V", Duration: "20 ms"}
	f.command("experiment.checks.set", workbench.ExperimentChecksParams{Checks: []workbench.ExperimentCheck{check}})
	changed := f.state()
	if !changed.Document.Dirty || !changed.Document.PreferencesDirty || changed.Document.FormatVersion != workbench.ExperimentDocumentVersion || original.RunID != changed.RunID {
		t.Fatal(changed.Document)
	}
	saved := f.project("save", "", "Experiment test", false)
	if saved.code != 200 {
		t.Fatal(string(saved.body))
	}
	entries, _, _ := f.projects()
	if len(entries) != 1 {
		t.Fatal(entries)
	}
	f.command("project.clear", workbench.ReplaceParams{DiscardChanges: true})
	if len(f.state().ExperimentChecks) != 0 {
		t.Fatal("new drawing inherited experiment checks")
	}
	reopened := f.project("open", entries[0].ID, "", true)
	if reopened.code != 200 {
		t.Fatal(string(reopened.body))
	}
	f.wait()
	state := f.state()
	if len(state.ExperimentChecks) != 1 || state.ExperimentChecks[0] != check {
		t.Fatal(state.ExperimentChecks)
	}
	run := func(expected bool) {
		t.Helper()
		body, _ := json.Marshal(check)
		res := f.request("POST", "/api/experiment-check?"+f.query(""), body, "")
		var out struct{ Evidence checkEvidence }
		if res.Code != 200 || json.Unmarshal(res.Body.Bytes(), &out) != nil || out.Evidence.Pass != expected {
			t.Fatal(res.Body.String())
		}
		if expected && (out.Evidence.Before == nil || out.Evidence.After == nil ||
			out.Evidence.Before.Values[0] >= 3 || out.Evidence.After.Values[0] < 3) {
			t.Fatal(out.Evidence)
		}
	}
	run(true)
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "100 kΩ"})
	f.wait()
	run(false)
}

func TestExperimentCheckReversalBracketsZeroSamples(t *testing.T) {
	rows := []workbench.VerificationRow{
		{Time: 0, Values: []float64{-1}},
		{Time: 1, Values: []float64{0}},
		{Time: 2, Values: []float64{1}},
	}
	evidence := evaluateCheck(rows, "reverses", 0)
	if !evidence.Pass || evidence.Before != &rows[0] || evidence.After != &rows[2] {
		t.Fatal(evidence)
	}
}
