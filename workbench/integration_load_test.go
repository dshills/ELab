package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"runtime"
	"testing"
	"time"
)

func TestIntegrationSampleIgnoresSchedulerPause(t *testing.T) {
	if !integrationThreadClock {
		t.Skip("thread CPU clock is not available on this platform")
	}
	for _, fixture := range []struct {
		name  string
		graph Graph
		step  float64
	}{
		{"555-tone", Piezo555ExampleGraph(), .00001},
		{"flasher", FlasherExampleGraph(), .01},
		{"RLC", RLCExampleGraph(), .001},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			c, _, diagnostic := fixture.graph.Compile()
			if diagnostic != nil {
				t.Fatal(diagnostic)
			}
			kernel, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
			if err != nil {
				t.Fatal(err)
			}
			reference, trace, err := advanceMixedContext(t.Context(), kernel.mixed, kernel.trace, fixture.step)
			if err != nil {
				t.Fatal(err)
			}
			before, _ := json.Marshal(kernel.mixed.result)
			budget := newIntegrationBudgetContext(t.Context())
			reads := 0
			budget.readCPU = func() (time.Duration, error) {
				reads++
				if reads == 2 {
					time.Sleep(2 * MaxIntegrationCPUTime)
				}
				return integrationCPUTime()
			}
			result, samples, err := advanceMixedBudget(kernel.mixed, kernel.trace, fixture.step, budget)
			if err != nil {
				t.Fatal("waiting for CPU rejected a valid sample", err)
			}
			if time.Since(budget.started) <= MaxIntegrationCPUTime || budget.cpuUsed <= 0 {
				t.Fatal("fixture did not cross the old wall limit with measured CPU work")
			}
			expected, _ := json.Marshal(struct {
				Result *MixedResult
				Trace  *Transient
			}{reference.result, trace})
			actual, _ := json.Marshal(struct {
				Result *MixedResult
				Trace  *Transient
			}{result.result, samples})
			after, _ := json.Marshal(kernel.mixed.result)
			if string(expected) != string(actual) || string(before) != string(after) {
				t.Fatal("scheduler pause changed exact accepted samples or source state")
			}
			if budget.depth != 0 || budget.pinned {
				t.Fatal("sample retained its OS thread after returning")
			}
		})
	}
}

func TestIntegrationCPUExhaustionWallBackstopAndClockFallback(t *testing.T) {
	for _, reason := range []string{"cpu", "wall", "clock-fallback", "clock-read-fallback"} {
		t.Run(reason, func(t *testing.T) {
			budget := newIntegrationBudget()
			switch reason {
			case "cpu":
				budget.elapsed = func() time.Duration { return MaxIntegrationCPUTime + time.Nanosecond }
			case "wall":
				budget.started = time.Now().Add(-MaxIntegrationWallTime - time.Millisecond)
				budget.elapsed = func() time.Duration { return 0 }
			case "clock-read-fallback":
				budget.started = time.Now().Add(-2 * MaxIntegrationCPUTime)
				reads := 0
				budget.readCPU = func() (time.Duration, error) {
					reads++
					if reads == 1 {
						return integrationCPUTime()
					}
					return 0, errors.New("clock read failed")
				}
			case "clock-fallback":
				budget.started = time.Now().Add(-2 * MaxIntegrationCPUTime)
				budget.readCPU = func() (time.Duration, error) { return 0, errors.New("clock unavailable") }
			}
			budget.enter()
			defer budget.leave()
			err := budget.checkTime()
			var limit *integrationBudgetError
			if !errors.As(err, &limit) || limit.code != "integration-time-limit" {
				t.Fatal("time safeguard disabled", reason, err)
			}
			basis := reason
			if basis == "clock-fallback" || basis == "clock-read-fallback" {
				basis = "wall-fallback"
			}
			if limit.basis != basis {
				t.Fatal("wrong timing reason", limit)
			}
		})
	}
}

func TestIntegrationCancellationBetweenNewtonIterations(t *testing.T) {
	state, trace := capacitorDecayFixture(t)
	before, _ := json.Marshal(state.result)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	budget := newIntegrationBudgetContext(ctx)
	checks := 0
	budget.elapsed = func() time.Duration {
		checks++
		if checks == 3 {
			cancel()
		}
		return 0
	}
	result, err := state.advanceBudget(trace, .001, budget)
	after, _ := json.Marshal(state.result)
	if result != nil || !errors.Is(err, context.Canceled) || string(before) != string(after) || budget.depth != 0 || budget.pinned {
		t.Fatal("canceled Newton work published or retained thread state", err)
	}
}

func TestIntegrationThreadClockMeasuresCPU(t *testing.T) {
	if !integrationThreadClock {
		t.Skip("thread CPU clock unavailable")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	start, err := integrationCPUTime()
	if err != nil {
		t.Fatal(err)
	}
	wall := time.Now()
	time.Sleep(2 * MaxIntegrationCPUTime)
	afterWait, err := integrationCPUTime()
	if err != nil {
		t.Fatal(err)
	}
	if afterWait-start >= MaxIntegrationCPUTime || time.Since(wall) < 2*MaxIntegrationCPUTime {
		t.Fatal("thread clock charged time spent asleep")
	}
	value := 1.
	for i := 0; i < 200000; i++ {
		value = math.Sqrt(value + 1.2345)
	}
	runtime.KeepAlive(value)
	afterWork, err := integrationCPUTime()
	if err != nil || afterWork <= afterWait {
		t.Fatal("thread clock did not count numerical work", err)
	}
}

func TestIntegrationCPUFailureReleasesThreadAndPreservesSample(t *testing.T) {
	if !integrationThreadClock {
		t.Skip("thread CPU clock unavailable")
	}
	state, trace := capacitorDecayFixture(t)
	before, _ := json.Marshal(state.result)
	budget := newIntegrationBudget()
	reads := 0
	budget.readCPU = func() (time.Duration, error) {
		value, err := integrationCPUTime()
		reads++
		if reads > 1 {
			value += MaxIntegrationCPUTime + time.Millisecond
		}
		return value, err
	}
	result, err := state.advanceBudget(trace, .001, budget)
	after, _ := json.Marshal(state.result)
	var limit *integrationBudgetError
	if result != nil || !errors.As(err, &limit) || limit.basis != "cpu" || string(before) != string(after) || budget.pinned || budget.depth != 0 {
		t.Fatal("CPU refusal published state or retained its pinned worker", err)
	}
}

// Opt-in process-local load qualification: CPU burners are confined to the
// benchmark's one Go scheduler CPU and joined before returning.
func Benchmark555ToneIntegrationLoad(b *testing.B) {
	if runtime.GOMAXPROCS(0) != 1 {
		b.Fatal("use -cpu=1 to confine synthetic CPU pressure")
	}
	c, _, diagnostic := Piezo555ExampleGraph().Compile()
	if diagnostic != nil {
		b.Fatal(diagnostic)
	}
	var reference string
	for _, burners := range []int{0, 8} {
		b.Run(map[int]string{0: "idle", 8: "single_cpu_pressure"}[burners], func(b *testing.B) {
			for trial := 0; trial < b.N; trial++ {
				cfg := DefaultTransientSettings()
				cfg.StepSeconds = .00001
				kernel, err := newSimulationKernel(context.Background(), kernelPlan{compiled: c, settings: cfg, mode: "auto"})
				if err != nil {
					b.Fatal(err)
				}
				stop := startAuditPressure(burners)
				digest, elapsed, err := func() (string, time.Duration, error) {
					defer stop()
					hash := sha256.New()
					start := time.Now()
					for sample := 1; sample <= 10000; sample++ {
						if err := kernel.advance(context.Background(), float64(sample)*cfg.StepSeconds); err != nil {
							return "", time.Since(start), err
						}
						if kernel.mixed.result.KCLResidual > 1e-9 || math.Abs(kernel.mixed.result.PowerResidual) > 1e-8 {
							return "", time.Since(start), fmt.Errorf("accepted sample violates conservation")
						}
						if err := json.NewEncoder(hash).Encode(kernel.trace.Sample); err != nil {
							return "", time.Since(start), err
						}
					}
					return hex.EncodeToString(hash.Sum(nil)), time.Since(start), nil
				}()
				if err != nil {
					b.Fatal(err)
				}
				if burners == 0 {
					reference = digest
				} else if digest != reference {
					b.Fatal("CPU pressure changed exact 555 samples", digest, reference)
				}
				b.Logf("burners=%d completed=10000 samples simulated=0.1s wall=%s sample_sha256=%s", burners, elapsed, digest)
			}
		})
	}
}
