package workbench

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"os"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

// This opt-in benchmark reproduces the doubled-capacitance period fixture in
// an owned Go test process, optionally sharing its single worker CPU with eight
// burners. It is not part of ordinary tests, and uses production solver limits.
// Run -cpu=1 -benchtime=1x; an optional new report path preserves the evidence.
func BenchmarkFlasherIntegrationBudgetAudit(b *testing.B) {
	if runtime.GOMAXPROCS(0) != 1 {
		b.Fatal("audit requires -cpu=1 to confine pressure to one owned process CPU")
	}
	type result struct {
		SamplesSHA256  string    `json:"samples_sha256"`
		Burners        int       `json:"cpu_burners"`
		Outcome        string    `json:"outcome"`
		Code           string    `json:"code,omitempty"`
		Attempts       int       `json:"rejected_sample_attempts,omitempty"`
		ElapsedMS      float64   `json:"rejected_sample_elapsed_ms,omitempty"`
		AcceptedTime   float64   `json:"last_accepted_time_s"`
		RequestedTime  float64   `json:"rejected_sample_end_s,omitempty"`
		RisingEdges    []float64 `json:"rising_edges_s"`
		Period         float64   `json:"last_period_s,omitempty"`
		RunWallMS      float64   `json:"run_wall_ms"`
		MaxKCLResidual float64   `json:"max_kcl_residual_a"`
		MaxPowerError  float64   `json:"max_power_residual_w"`
		StatePreserved bool      `json:"failed_sample_state_preserved,omitempty"`
	}
	results := []result{}
	g := FlasherExampleGraph()
	for i := range g.Components {
		if g.Components[i].Kind == "capacitor" {
			g.Components[i].Value *= 2
		}
	}
	data, _ := json.Marshal(g)
	digest := sha256.Sum256(data)
	for _, burners := range []int{0, 8} {
		ok := b.Run(map[int]string{0: "idle", 8: "single_cpu_pressure"}[burners], func(b *testing.B) {
			for trial := 0; trial < b.N; trial++ {
				c, _, diagnostic := g.Compile()
				if diagnostic != nil {
					b.Fatal(diagnostic)
				}
				n := newMixedNetwork(c)
				cfg := DefaultTransientSettings()
				cfg.StepSeconds = .01
				trace := &Transient{Settings: cfg, Mode: "auto"}
				state, err := n.solve(nil, []float64{.1, 0}, 0, trace, 0)
				if err != nil {
					b.Fatal(err)
				}
				stop := startAuditPressure(burners)
				started := time.Now()
				r := result{Burners: burners, Outcome: "completed", RisingEdges: []float64{}}
				sampleHash := sha256.New()
				high := state.result.LEDs[0].Current > .001
				for sample := 1; sample <= 600; sample++ {
					end := float64(sample) * .01
					beforeX := append([]float64{}, state.x...)
					beforeCap := append([]float64{}, state.capV...)
					beforeResult, encodeErr := json.Marshal(state.result)
					if encodeErr != nil {
						stop()
						b.Fatal(encodeErr)
					}
					next, err := state.advance(trace, end)
					if err != nil {
						var limit *integrationBudgetError
						if !errors.As(err, &limit) {
							stop()
							b.Fatal("non-budget flasher failure", err)
						}
						r.Outcome, r.Code = "bounded-failure", limit.code
						r.Attempts, r.ElapsedMS = limit.attempts, float64(limit.elapsed)/float64(time.Millisecond)
						r.RequestedTime = end
						afterResult, encodeErr := json.Marshal(state.result)
						// cloneMixed normalizes omitted nil slices to empty ones;
						// compare canonical wire data plus exact solver memory.
						r.StatePreserved = encodeErr == nil && reflect.DeepEqual(beforeX, state.x) && reflect.DeepEqual(beforeCap, state.capV) && bytes.Equal(beforeResult, afterResult)
						if !r.StatePreserved {
							stop()
							b.Fatal("failed integration changed original state", r)
						}
						break
					}
					state = next
					if err := json.NewEncoder(sampleHash).Encode(state.result); err != nil {
						stop()
						b.Fatal(err)
					}
					r.AcceptedTime = state.result.Time
					r.MaxKCLResidual = math.Max(r.MaxKCLResidual, state.result.KCLResidual)
					r.MaxPowerError = math.Max(r.MaxPowerError, math.Abs(state.result.PowerResidual))
					nowHigh := state.result.LEDs[0].Current > .001
					if nowHigh && !high && state.result.Time > .8 {
						r.RisingEdges = append(r.RisingEdges, state.result.Time)
					}
					high = nowHigh
				}
				r.SamplesSHA256 = hex.EncodeToString(sampleHash.Sum(nil))
				r.RunWallMS = float64(time.Since(started)) / float64(time.Millisecond)
				stop() // Join every owned burner, including after a numerical limit.
				if r.Outcome != "completed" && os.Getenv("ELAB_INTEGRATION_AUDIT_REQUIRE_COMPLETION") == "1" {
					b.Fatal("load regression did not complete", r)
				}
				if r.Outcome == "completed" {
					if len(r.RisingEdges) < 4 {
						b.Fatal("completed fixture has too few settled periods", r)
					}
					r.Period = r.RisingEdges[len(r.RisingEdges)-1] - r.RisingEdges[len(r.RisingEdges)-2]
					if math.Abs(r.Period-.97) > .03 {
						b.Fatal("completed flasher period outside prior fixture tolerance", r)
					}
				}
				if r.MaxKCLResidual > 1e-9 || r.MaxPowerError > 1e-8 {
					b.Fatal("accepted sample violates prior conservation bounds", r)
				}
				results = append(results, r)
				b.Logf("burners=%d outcome=%s code=%s accepted=%.9g s attempts=%d elapsed=%.3f ms period=%.3f s", r.Burners, r.Outcome, r.Code, r.AcceptedTime, r.Attempts, r.ElapsedMS, r.Period)
			}
		})
		if !ok {
			b.Fatal("integration audit fixture failed; no passing report published")
		}
	}
	if len(results) == 2 && results[0].Outcome == "completed" && results[1].Outcome == "completed" && results[0].SamplesSHA256 != results[1].SamplesSHA256 {
		b.Fatal("CPU pressure changed exact accepted samples")
	}
	if path := os.Getenv("ELAB_INTEGRATION_AUDIT_REPORT"); path != "" {
		report := map[string]any{"passed": true, "go_version": runtime.Version(), "platform": runtime.GOOS + "/" + runtime.GOARCH, "gomaxprocs": runtime.GOMAXPROCS(0), "graph_sha256": hex.EncodeToString(digest[:]), "maximum_attempts": MaxIntegrationAttempts, "maximum_wall_ms": MaxIntegrationWallTime / time.Millisecond, "maximum_cpu_ms": MaxIntegrationCPUTime / time.Millisecond, "time_basis": integrationClockBasis, "sample_step_s": .01, "requested_duration_s": 6, "results": results, "limits": []string{"owned one-CPU test process; synthetic scheduler pressure is not participant/visible latency evidence", "CPU/wall/cancellation checks are cooperative between adaptive/Newton stages; a running matrix solve is not preempted", "a reproduced time-limit failure does not establish the cause of the previous unclassified race-run failure"}}
		data, _ := json.MarshalIndent(report, "", "  ")
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			b.Fatal(err)
		}
		_, writeErr := file.Write(append(data, '\n'))
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			b.Fatal(writeErr, closeErr)
		}
	}
}

func startAuditPressure(count int) func() {
	stop, started := make(chan struct{}), make(chan struct{}, count)
	var group sync.WaitGroup
	for i := 0; i < count; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			started <- struct{}{}
			value := 1.
			for {
				select {
				case <-stop:
					runtime.KeepAlive(value)
					return
				default:
					for j := 0; j < 2048; j++ {
						value = math.Sqrt(value + 1.2345)
					}
				}
			}
		}()
	}
	for i := 0; i < count; i++ {
		<-started
	}
	return func() { close(stop); group.Wait() }
}
