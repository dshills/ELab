package conformance

import (
	"fmt"
	"math"

	"github.com/dshills/elab/workbench"
)

// Captured statistics are consistency evidence, separate from the circuit
// references. Rows have already passed complete finite channel/grid checks.
func checkCapacityCaptureStatistics(capture workbench.VerificationDocument) error {
	if len(capture.Metadata.Statistics) != 4 {
		return fmt.Errorf("capacity timing captured statistics missing")
	}
	for channel, actual := range capture.Metadata.Statistics {
		sum, squares := 0., 0.
		low, high := math.Inf(1), math.Inf(-1)
		for _, row := range capture.Rows {
			value := row.Values[channel]
			sum += value
			squares += value * value
			low, high = math.Min(low, value), math.Max(high, value)
		}
		near := func(value, want float64) bool {
			return finiteConformance(value) && math.Abs(value-want) <= 1e-12*math.Max(1, math.Abs(want))
		}
		count := len(capture.Rows)
		if actual.ProbeID != capture.Metadata.Probes[channel].ID || actual.Count != count || !near(actual.Minimum, low) || !near(actual.Maximum, high) || !near(actual.Mean, sum/float64(count)) || !near(actual.RMS, math.Sqrt(squares/float64(count))) {
			return fmt.Errorf("capacity timing captured statistics disagree with rows")
		}
	}
	return nil
}
