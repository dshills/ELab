package conformance

import (
	"context"
	"testing"
	"time"
)

func TestElectricalReferencesAndFlasherRefinementThroughPublicCommands(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	report, err := RunElectrical(ctx, InProcess, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Checks) != electricalCheckCount || len(report.Refinement) != 3 || report.ReferenceSHA256 == "" {
		t.Fatal("electrical suite omitted required evidence")
	}
	for _, check := range report.Checks {
		if !check.Passed {
			t.Fatal("failed reference accepted", check)
		}
	}
	for _, run := range report.Refinement {
		t.Logf("step %g: %d samples, period %.9g, Verify200ms %.3g s", run.Step, run.Samples, run.Period, run.VerifyWall)
	}
	if err = CompareElectrical(report, report); err != nil {
		t.Fatal(err)
	}
	broken := report
	broken.Checks = nil
	if CompareElectrical(report, broken) == nil {
		t.Fatal("missing check evidence accepted")
	}
	broken = report
	broken.Refinement = nil
	if CompareElectrical(report, broken) == nil {
		t.Fatal("missing refinement evidence accepted")
	}
	broken = report
	broken.Checks = append([]ElectricalCheck{}, report.Checks...)
	broken.Checks[0].Actual += .0001
	if CompareElectrical(report, broken) == nil {
		t.Fatal("changed cross-adapter current accepted")
	}
	// A plausible-looking output outside tolerance must fail qualification.
	if report.check("corrupted measurement", 1, 0, .001) == nil {
		t.Fatal("reference validator accepted a changed measurement")
	}
}
