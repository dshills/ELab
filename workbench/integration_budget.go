package workbench

import (
	"fmt"
	"time"
)

type integrationBudget struct {
	started  time.Time
	attempts int
}

func newIntegrationBudget() *integrationBudget { return &integrationBudget{started: time.Now()} }
func (b *integrationBudget) take() error {
	b.attempts++
	if b.attempts > 512 || time.Since(b.started) > 100*time.Millisecond {
		return fmt.Errorf("mixed integration exceeded its work budget; reduce sample step")
	}
	return nil
}
