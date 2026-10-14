package workbench

import (
	"errors"
	"fmt"
	"time"
)

const MaxIntegrationAttempts = 512
const MaxIntegrationWallTime = 100 * time.Millisecond

type integrationBudgetError struct {
	code     string
	attempts int
	elapsed  time.Duration
}

func (e *integrationBudgetError) Error() string {
	reason := "attempt limit"
	if e.code == "integration-time-limit" {
		reason = "elapsed-time limit"
	}
	return fmt.Sprintf("mixed integration exceeded its work budget: %s (%d attempted, maximum %d; %.3f ms elapsed, maximum %d ms); reduce sample step", reason, e.attempts, MaxIntegrationAttempts, float64(e.elapsed)/float64(time.Millisecond), MaxIntegrationWallTime/time.Millisecond)
}

func simulationFailureCode(err error, fallback string) string {
	var limit *integrationBudgetError
	if errors.As(err, &limit) {
		return limit.code
	}
	return fallback
}

type integrationBudget struct {
	started  time.Time
	attempts int
	elapsed  func() time.Duration // Test clock; nil uses the production monotonic clock.
}

func newIntegrationBudget() *integrationBudget { return &integrationBudget{started: time.Now()} }
func (b *integrationBudget) take() error {
	b.attempts++
	elapsed := time.Since(b.started)
	if b.elapsed != nil {
		elapsed = b.elapsed()
	}
	if b.attempts > MaxIntegrationAttempts {
		return &integrationBudgetError{code: "integration-attempt-limit", attempts: b.attempts, elapsed: elapsed}
	}
	if elapsed > MaxIntegrationWallTime {
		return &integrationBudgetError{code: "integration-time-limit", attempts: b.attempts, elapsed: elapsed}
	}
	return nil
}
