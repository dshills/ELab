package conformance

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"
)

func TestPerformancePercentilesAndRejectMissingTiming(t *testing.T) {
	values := []float64{20, 1, 19, 2, 18, 3, 17, 4, 16, 5, 15, 6, 14, 7, 13, 8, 12, 9, 11, 10}
	result, err := summarizePerformance(values)
	if err != nil || result.Count != 20 || result.P50 != 10 || result.P95 != 19 || result.Maximum != 20 || values[0] != 20 {
		t.Fatal("percentile calculation or input ownership", result, err)
	}
	for _, bad := range [][]float64{nil, {-1}, {math.NaN()}, {math.Inf(1)}} {
		if _, err = summarizePerformance(bad); err == nil {
			t.Fatal("invalid evidence accepted")
		}
	}
}

func TestPerformanceSuiteRetainsCoverageAndCircuitIdentities(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, count := range []int{0, 19, 101} {
		if _, err := RunPerformance(ctx, nil, t.TempDir(), count); err == nil {
			t.Fatal("unsupported repetition count admitted", count)
		}
	}
	result, err := RunPerformance(ctx, InProcess, t.TempDir(), 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Cases) != 10 || len(result.Circuits) != 10 || result.Repetitions != 20 {
		t.Fatal("missing circuit/condition/variant coverage")
	}
	targetsPassed := true
	for _, entry := range result.Cases {
		if len(entry.Samples) != 20 || entry.Acknowledgment.Count != 20 || entry.FreshResult.Count != 20 || entry.Verify.Count != 20 {
			t.Fatal("missing repetitions", entry.Name, entry.Condition)
		}
		if entry.FreshTargetPassed != (entry.FreshResult.P95 <= 500) || entry.VerifyTargetPassed != (entry.Verify.Maximum <= 2000) {
			t.Fatal("target differs from evidence")
		}
		targetsPassed = targetsPassed && entry.FreshTargetPassed && entry.VerifyTargetPassed
		for _, sample := range entry.Samples {
			graph, exists := result.Circuits[sample.GraphSHA256]
			data, err := json.Marshal(graph)
			if !exists || err != nil || sample.GraphSHA256 != fmt.Sprintf("%x", sha256.Sum256(data)) {
				t.Fatal("circuit cannot be recovered from its hash")
			}
		}
	}
	if result.TargetsPassed != targetsPassed {
		t.Fatal("aggregate target result differs")
	}
}
func TestPerformanceMeasurementsUseFreshChangedRuns(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, name := range []string{"led", "rc", "npn", "npn-pulse", "flasher"} {
		t.Run(name, func(t *testing.T) {
			err := electricalSession(ctx, InProcess, t.TempDir(), func(s *session) error {
				if err := preparePerformance(s, name); err != nil {
					return err
				}
				for i := 0; i < 10; i++ {
					sample, err := measurePerformance(s, name, i, "test")
					if err != nil {
						return err
					}
					if sample.FreshResultMS < sample.AcknowledgmentMS || sample.GraphSHA256 == "" || sample.Engine == "" || len(sample.ProbeIDs) != 4 || len(sample.Models) == 0 || sample.RunID == 0 || sample.ElectricalRevision == 0 {
						t.Fatal("measurement identity/timing omitted")
					}
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
