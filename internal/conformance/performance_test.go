package conformance

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
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

func TestInductorPerformanceFixturesUseChangedRunsAndFullCaptures(t *testing.T) {
	factory := ordinaryAgentFactory(t)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	for _, count := range []int{0, 19, 101} {
		if _, err := RunInductorPerformance(ctx, nil, t.TempDir(), count); err == nil {
			t.Fatal("unsupported coil repetition count admitted", count)
		}
	}
	for _, name := range []string{"rl", "rlc", "flyback", "npn-coil"} {
		t.Run(name, func(t *testing.T) {
			err := electricalSession(ctx, factory, t.TempDir(), func(s *session) error {
				if err := preparePerformance(s, name); err != nil {
					return err
				}
				previous := ""
				for i := 0; i < 3; i++ {
					sample, err := measurePerformance(s, name, i, "test")
					if err != nil {
						return err
					}
					if sample.Engine != workbench.InductorEngineID || sample.Models["L1"] != "ideal-inductor.v1" || sample.VerifyDuration != .2 || sample.VerifySampleStep != .001 || !reflect.DeepEqual(sample.ProbeIDs, performanceProbes(name)) || sample.FreshResultMS < sample.AcknowledgmentMS {
						t.Fatal("coil timing omitted engine, capture, probe or timing identity", sample)
					}
					if sample.GraphSHA256 == "" || sample.GraphSHA256 == previous {
						t.Fatal("coil timing repeated a no-op instead of changing the resistor")
					}
					previous = sample.GraphSHA256
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestOpAmpPerformanceUsesChangedRunsAndTwoHundredMillisecondCaptures(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	for _, count := range []int{0, 19, 101} {
		if _, err := RunOpAmpPerformance(ctx, nil, t.TempDir(), count); err == nil {
			t.Fatal("unsupported amplifier repetition count admitted", count)
		}
	}
	for _, name := range opAmpCaseNames {
		t.Run(name, func(t *testing.T) {
			err := electricalSession(ctx, InProcess, t.TempDir(), func(s *session) error {
				if err := preparePerformance(s, name); err != nil {
					return err
				}
				variants := make(map[string]bool)
				previous := ""
				for i := 0; i < 3; i++ {
					sample, err := measurePerformance(s, name, i, "test")
					if err != nil {
						return err
					}
					if sample.Engine == "" || sample.Models["U1"] != workbench.OpAmpDeviceModel().ID || sample.VerifyDuration != .2 || sample.VerifySampleStep != .001 || !reflect.DeepEqual(sample.ProbeIDs, performanceProbes(name)) || sample.FreshResultMS < sample.AcknowledgmentMS {
						t.Fatal("amplifier timing lost model, capture, probes or timing identity", sample)
					}
					if name == "opamp-rc-mosfet" && (sample.Models["U2"] != workbench.OpAmpDeviceModel().ID || sample.Models["M1"] != workbench.NMOSDeviceModel().ID) {
						t.Fatal("two-buffer MOSFET timing lost mixed device identities", sample.Models)
					}
					if sample.GraphSHA256 == "" || sample.GraphSHA256 == previous {
						t.Fatal("amplifier measurement repeated a no-op")
					}
					data, err := json.Marshal(sample.graph)
					if err != nil || sample.GraphSHA256 != fmt.Sprintf("%x", sha256.Sum256(data)) {
						t.Fatal("amplifier graph cannot be recovered from its identity")
					}
					variants[sample.GraphSHA256], previous = true, sample.GraphSHA256
				}
				if len(variants) != 2 {
					t.Fatal("measurements did not alternate two electrical variants")
				}
				current, err := s.state("simulation.wait", nil, false)
				if err != nil {
					return err
				}
				if !performanceFresh(current, name) {
					t.Fatal("valid current amplifier result refused")
				}
				for _, damage := range []string{"stale-mixed", "old-run", "old-revision", "missing-model", "missing-amplifier"} {
					bad := current
					mixed := *current.Mixed
					bad.Mixed = &mixed
					switch damage {
					case "stale-mixed":
						bad.Mixed.Stale = true
					case "old-run":
						bad.Mixed.RunID--
					case "old-revision":
						bad.Mixed.ElectricalRevision--
					case "missing-model":
						bad.OpAmpModel = nil
					case "missing-amplifier":
						bad.Mixed.OpAmps = nil
					}
					if performanceFresh(bad, name) {
						t.Fatal("invalid amplifier evidence considered fresh", damage)
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
