package workbench

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

var benchmarkMixedMatrix [][]float64
var benchmarkMixedResidual []float64

// Assembly excludes initialization and result serialization. Heap escape is
// deliberate: Newton consumes both returned arrays outside the assembler.
func BenchmarkMixedMatrixAssembly(b *testing.B) {
	c, _, diagnostic := FlasherExampleGraph().Compile()
	if diagnostic != nil {
		b.Fatal(diagnostic)
	}
	n := newMixedNetwork(c)
	trace := &Transient{Settings: DefaultTransientSettings(), Mode: "auto"}
	state, err := n.solve(nil, []float64{.1, 0}, 0, trace, 0)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkMixedMatrix, benchmarkMixedResidual = n.equations(state.x, state.capV, .001, trace, .001)
	}
}

// Compare complete accepted solver memory/results before and after an
// allocation-only change on the same toolchain. This is backend work evidence,
// not UI latency. The numerical fixture keeps the 512-attempt cap but uses the
// existing test clock so a scheduler pause cannot change its output digest.
// Production CPU/wall-time rejection remains covered by integration_budget_test.
func BenchmarkFlasherMatrixWork(b *testing.B) {
	type result struct {
		Scale          float64 `json:"capacitance_scale"`
		Samples        int     `json:"samples"`
		GraphSHA256    string  `json:"graph_sha256"`
		AcceptedSHA256 string  `json:"accepted_memory_and_result_sha256"`
		Attempts       int     `json:"total_attempts"`
		PeakAttempts   int     `json:"peak_sample_attempts"`
		AdvanceMS      float64 `json:"advance_wall_ms"`
	}
	results := []result{}
	for _, scale := range []float64{1, 2} {
		if !b.Run(fmt.Sprint(scale), func(b *testing.B) {
			b.ReportAllocs()
			for trial := 0; trial < b.N; trial++ {
				g := FlasherExampleGraph()
				for i := range g.Components {
					if g.Components[i].Kind == "capacitor" {
						g.Components[i].Value *= scale
					}
				}
				graph, _ := json.Marshal(g)
				graphHash := sha256.Sum256(graph)
				compiled, _, diagnostic := g.Compile()
				if diagnostic != nil {
					b.Fatal(diagnostic)
				}
				trace := &Transient{Settings: DefaultTransientSettings(), Mode: "auto"}
				trace.Settings.StepSeconds = .005 * scale
				state, err := newMixedNetwork(compiled).solve(nil, []float64{.1, 0}, 0, trace, 0)
				if err != nil {
					b.Fatal(err)
				}
				r := result{Scale: scale, Samples: 600, GraphSHA256: hex.EncodeToString(graphHash[:])}
				digest := sha256.New()
				encoder := json.NewEncoder(digest)
				for sample := 1; sample <= r.Samples; sample++ {
					budget := newIntegrationBudget()
					budget.elapsed = func() time.Duration { return 0 }
					start := time.Now()
					next, err := state.advanceBudget(trace, float64(sample)*trace.Settings.StepSeconds, budget)
					r.AdvanceMS += float64(time.Since(start)) / float64(time.Millisecond)
					if err != nil {
						b.Fatalf("scale %g, sample %d: %v", scale, sample, err)
					}
					state = next
					r.Attempts += budget.attempts
					r.PeakAttempts = max(r.PeakAttempts, budget.attempts)
					if err := encoder.Encode(struct {
						Unknowns   []float64
						Capacitors []float64
						Result     *MixedResult
					}{state.x, state.capV, state.result}); err != nil {
						b.Fatal(err)
					}
				}
				r.AcceptedSHA256 = hex.EncodeToString(digest.Sum(nil))
				results = append(results, r)
			}
		}) {
			b.Fatal("allocation fixture failed")
		}
	}
	if path := os.Getenv("ELAB_MIXED_MATRIX_AUDIT_REPORT"); path != "" {
		data, _ := json.MarshalIndent(results, "", "  ")
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
