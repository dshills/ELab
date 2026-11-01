package workbench

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"
)

const MaxIntegrationAttempts = 512
const MaxIntegrationCPUTime = 100 * time.Millisecond

// Host scheduling delays are not numerical work. Retain a separate finite
// backstop for a sample whose worker cannot make progress; operation contexts
// can impose an earlier deadline or cancel at any numerical checkpoint.
const MaxIntegrationWallTime = 5 * time.Second

type integrationBudgetError struct {
	code     string
	attempts int
	elapsed  time.Duration
	basis    string
}

func (e *integrationBudgetError) Error() string {
	reason := "attempt limit"
	if e.code == "integration-time-limit" {
		reason = "calculation time limit (" + e.basis + ")"
	}
	return fmt.Sprintf("mixed integration exceeded its work budget: %s (%d attempted, maximum %d; %.3f ms measured; maximum %d ms CPU, %d ms wall); reduce sample step", reason, e.attempts, MaxIntegrationAttempts, float64(e.elapsed)/float64(time.Millisecond), MaxIntegrationCPUTime/time.Millisecond, MaxIntegrationWallTime/time.Millisecond)
}
func simulationFailureCode(err error, fallback string) string {
	var limit *integrationBudgetError
	if errors.As(err, &limit) {
		return limit.code
	}
	return fallback
}

type integrationBudget struct {
	started           time.Time
	attempts          int
	newtonIterations  int // Diagnostic counts, not additional adaptive attempts.
	modalAnalyses     int
	roundoffRetries   int
	ctx               context.Context
	solveCtx          context.Context
	depth             int
	pinned            bool
	cpuStart, cpuUsed time.Duration
	elapsed           func() time.Duration          // Deterministic CPU-budget test clock.
	readCPU           func() (time.Duration, error) // Clock failure/scheduling regression seam.
}

func newIntegrationBudget() *integrationBudget {
	return newIntegrationBudgetContext(context.Background())
}
func newIntegrationBudgetContext(ctx context.Context) *integrationBudget {
	return &integrationBudget{started: time.Now(), ctx: ctx}
}
func (b *integrationBudget) cpuNow() (time.Duration, error) {
	if b.readCPU != nil {
		return b.readCPU()
	}
	return integrationCPUTime()
}

// Nested source/hybrid stages share one pinned thread, CPU clock and attempt
// counter. Never hold a service lock here. Every outer caller defers leave,
// including on cancellation, budget rejection and numerical failure.
func (b *integrationBudget) enter() {
	b.depth++
	if b.depth != 1 || b.elapsed != nil || !integrationThreadClock {
		return
	}
	runtime.LockOSThread()
	start, err := b.cpuNow()
	if err != nil {
		runtime.UnlockOSThread()
		return
	} // Fail closed to the old 100 ms wall measurement.
	b.cpuStart, b.pinned = start, true
}
func (b *integrationBudget) leave() {
	b.depth--
	if b.depth != 0 || !b.pinned {
		return
	}
	if end, err := b.cpuNow(); err == nil {
		b.cpuUsed += end - b.cpuStart
	}
	b.pinned = false
	runtime.UnlockOSThread()
}
func (b *integrationBudget) take() error {
	b.attempts++
	if b.attempts > MaxIntegrationAttempts {
		return &integrationBudgetError{code: "integration-attempt-limit", attempts: b.attempts, elapsed: time.Since(b.started), basis: "attempts"}
	}
	return b.checkTime()
}

// Check between adaptive and Newton stages, and before publishing a completed
// sample. CPU exhaustion still rejects atomically; waiting for the host's CPU
// does not become a fictitious convergence failure.
func (b *integrationBudget) checkTime() error {
	if b.started.IsZero() {
		b.started = time.Now()
	}
	if b.ctx != nil {
		if err := b.ctx.Err(); err != nil {
			return err
		}
	}
	wall := time.Since(b.started)
	if wall > MaxIntegrationWallTime {
		return &integrationBudgetError{code: "integration-time-limit", attempts: b.attempts, elapsed: wall, basis: "wall"}
	}
	measured, basis := wall, "wall-fallback"
	if b.elapsed != nil {
		measured, basis = b.elapsed(), "cpu"
	} else if b.pinned {
		if now, err := b.cpuNow(); err == nil {
			measured, basis = b.cpuUsed+now-b.cpuStart, "cpu"
		}
	}
	if measured > MaxIntegrationCPUTime {
		return &integrationBudgetError{code: "integration-time-limit", attempts: b.attempts, elapsed: measured, basis: basis}
	}
	return nil
}

type integrationBudgetContextKey struct{}

func (b *integrationBudget) solverContext() context.Context {
	if b.solveCtx == nil {
		ctx := b.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		b.solveCtx = context.WithValue(ctx, integrationBudgetContextKey{}, b)
	}
	return b.solveCtx
}
