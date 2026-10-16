package conformance

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func TestOpAmpPublicMixedWorkflowsAndEvidenceComparison(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	report, err := RunOpAmps(ctx, InProcess, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = CompareOpAmps(report, report); err != nil {
		t.Fatal(err)
	}
	for _, corrupt := range []func(*OpAmpSuiteResult){
		func(r *OpAmpSuiteResult) { r.Model.OpenLoopGain = 1 },
		func(r *OpAmpSuiteResult) { r.Cases = r.Cases[:3] },
		func(r *OpAmpSuiteResult) { r.Checks = r.Checks[1:] },
		func(r *OpAmpSuiteResult) { r.Checks[0].Actual = math.NaN() },
		func(r *OpAmpSuiteResult) { r.Checks[0].Actual += .001 },
		func(r *OpAmpSuiteResult) { r.Cases[0].GraphSHA256 = "changed" },
		func(r *OpAmpSuiteResult) { r.Cases[0].Calls = nil },
		func(r *OpAmpSuiteResult) { r.Cases[0].Rows = r.Cases[0].Rows[:100] },
		func(r *OpAmpSuiteResult) { r.Cases[2].Rows[50].Time = .049 },
		func(r *OpAmpSuiteResult) { r.Cases[2].Rows[20].Values[0] += .1 },
		func(r *OpAmpSuiteResult) { r.Cases[1].Name = "opamp-follower" },
		func(r *OpAmpSuiteResult) { r.Cases[2].Observations[1].Transistors[0].CollectorCurrent += .001 },
		func(r *OpAmpSuiteResult) { r.Cases[3].Observations[1].OpAmps = r.Cases[3].Observations[1].OpAmps[:1] },
	} {
		data, err := json.Marshal(report)
		if err != nil {
			t.Fatal(err)
		}
		var broken OpAmpSuiteResult
		if err = json.Unmarshal(data, &broken); err != nil {
			t.Fatal(err)
		}
		corrupt(&broken)
		if CompareOpAmps(report, broken) == nil || CompareOpAmps(broken, report) == nil {
			t.Fatal("missing or changed amplifier evidence accepted")
		}
	}
}

func TestOpAmpScalarFollowerReferenceMatchesFrozenDecimalPeak(t *testing.T) {
	m := workbench.OpAmpDeviceModel()
	got := opAmpFollowerReference(1, 1./10000+1/m.DifferentialResistance, -1/m.DifferentialResistance)
	if math.Abs(got-.999989811376) > 1e-11 {
		t.Fatal("independent scalar fixture differs from frozen Decimal peak", got)
	}
}
