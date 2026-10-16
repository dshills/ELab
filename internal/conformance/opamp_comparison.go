package conformance

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"reflect"

	"github.com/dshills/elab/workbench"
)

// CompareOpAmps requires the complete named fixture/trace/observation coverage.
// Only run/electrical revision provenance is normalized when capturing results.
func CompareOpAmps(reference, actual OpAmpSuiteResult) error {
	for _, report := range []OpAmpSuiteResult{reference, actual} {
		if report.Model != workbench.OpAmpDeviceModel() || len(report.Cases) != 4 || len(report.Checks) != opAmpReferenceCount {
			return fmt.Errorf("op-amp model/case/reference coverage differs")
		}
		recomputed := ElectricalResult{}
		for i, entry := range report.Cases {
			observations, rows := 3, 101
			if i >= 2 {
				observations, rows = 4, 105
			}
			data, err := json.Marshal(entry.Graph)
			if err != nil || entry.GraphSHA256 != fmt.Sprintf("%x", sha256.Sum256(data)) || entry.Name != opAmpCaseNames[i] || len(entry.Rows) != rows || len(entry.Observations) != observations || !reflect.DeepEqual(entry.Probes, opAmpCaseProbes(entry.Name)) || len(entry.Calls) == 0 {
				return fmt.Errorf("op-amp fixture %d coverage/identity differs", i)
			}
			if i < 2 {
				want := .999989811376
				if i == 1 {
					want = -.999969336181
				}
				if len(entry.Observations[1].OpAmps) != 1 {
					return fmt.Errorf("op-amp peak observation missing")
				}
				if err := recomputed.check(entry.Name+"-peak", entry.Observations[1].OpAmps[0].OutputVoltage, want, 1e-7); err != nil {
					return err
				}
			}
			if err := checkOpAmpRows(&recomputed, entry); err != nil {
				return err
			}
		}
		if !reflect.DeepEqual(recomputed.Checks, report.Checks) {
			return fmt.Errorf("op-amp checks differ from captured row/peak evidence")
		}
		for _, check := range report.Checks {
			if !check.Passed || !finiteConformance(check.Actual) || !finiteConformance(check.Reference) || !finiteConformance(check.Tolerance) || check.Tolerance < 0 || math.Abs(check.Actual-check.Reference) > check.Tolerance {
				return fmt.Errorf("op-amp numerical check failed")
			}
		}
	}
	left, right := reference, actual
	left.Cases, right.Cases = append([]OpAmpCase(nil), reference.Cases...), append([]OpAmpCase(nil), actual.Cases...)
	for i := range left.Cases {
		left.Cases[i].Calls = collapseCalls(left.Cases[i].Calls)
		right.Cases[i].Calls = collapseCalls(right.Cases[i].Calls)
	}
	left.Limitations, right.Limitations = nil, nil
	a, err := json.Marshal(left)
	if err != nil {
		return err
	}
	b, err := json.Marshal(right)
	if err != nil {
		return err
	}
	if string(a) != string(b) {
		return fmt.Errorf("op-amp numerical/graph/command observations differ")
	}
	return nil
}
