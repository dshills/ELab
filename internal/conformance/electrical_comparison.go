package conformance

import (
	"fmt"
	"math"
)

// CompareElectrical compares deterministic numerical evidence, never wall times.
func CompareElectrical(reference, actual ElectricalResult) error {
	if reference.ReferenceSHA256 == "" || reference.ReferenceSHA256 != actual.ReferenceSHA256 || reference.ReferenceMethod != actual.ReferenceMethod || len(reference.Checks) != electricalCheckCount || len(actual.Checks) != electricalCheckCount || len(reference.Refinement) != 3 || len(actual.Refinement) != 3 {
		return fmt.Errorf("electrical reference/coverage differs")
	}
	close := func(a, b float64) bool {
		return !math.IsNaN(a) && !math.IsNaN(b) && !math.IsInf(a, 0) && !math.IsInf(b, 0) && math.Abs(a-b) <= 1e-10*math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
	}
	for i, a := range reference.Checks {
		b := actual.Checks[i]
		if !a.Passed || !b.Passed || a.Name != b.Name || a.Reference != b.Reference || a.Tolerance != b.Tolerance || !close(a.Actual, b.Actual) {
			return fmt.Errorf("electrical check %d differs", i)
		}
	}
	for i, a := range reference.Refinement {
		b := actual.Refinement[i]
		if a.GraphSHA256 == "" || a.GraphSHA256 != b.GraphSHA256 || a.Step != b.Step || a.Samples != b.Samples || !close(a.Period, b.Period) || len(a.RisingBrackets) != len(b.RisingBrackets) || len(a.Minimum) != 4 || len(b.Minimum) != 4 || len(a.Maximum) != 4 || len(b.Maximum) != 4 {
			return fmt.Errorf("flasher refinement coverage differs")
		}
		for j, edge := range a.RisingBrackets {
			if !close(edge[0], b.RisingBrackets[j][0]) || !close(edge[1], b.RisingBrackets[j][1]) {
				return fmt.Errorf("flasher edge bracket differs")
			}
		}
		for id, minimum := range a.Minimum {
			other, ok := b.Minimum[id]
			if !ok || !close(minimum, other) {
				return fmt.Errorf("flasher minimum %s differs", id)
			}
		}
		for id, maximum := range a.Maximum {
			other, ok := b.Maximum[id]
			if !ok || !close(maximum, other) {
				return fmt.Errorf("flasher maximum %s differs", id)
			}
		}
	}
	return nil
}
