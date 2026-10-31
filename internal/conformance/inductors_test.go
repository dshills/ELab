package conformance

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/dshills/elab/internal/testutil"
)

func TestInductorPublicWorkflowAndEvidenceComparison(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	report, err := RunInductors(ctx, InProcess, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = CompareInductors(report, report); err != nil {
		t.Fatal(err)
	}
	for _, corrupt := range []func(*InductorResult){
		func(r *InductorResult) { r.Checks = r.Checks[1:] },
		func(r *InductorResult) { r.Calls = nil },
		func(r *InductorResult) { r.Observations[0].Name = "rl" },
		func(r *InductorResult) { r.Checks[0].Actual = math.NaN() },
		func(r *InductorResult) { r.Checks[0].Actual += .01 },
		func(r *InductorResult) { r.Observations[6].Result.Inductors[0].Current += .001 },
	} {
		data, err := json.Marshal(report)
		if err != nil {
			t.Fatal(err)
		}
		var broken InductorResult
		if err = json.Unmarshal(data, &broken); err != nil {
			t.Fatal(err)
		}
		corrupt(&broken)
		if CompareInductors(report, broken) == nil {
			t.Fatal("missing or changed coil evidence accepted")
		}
	}
}
