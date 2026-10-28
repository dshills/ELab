package workbench

import (
	"context"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// productionNumericalReference preserves the real wall-budget contract for
// timing-sensitive numerical references under a race-instrumented test runner.
// The same unmodified test body runs in a fresh ordinary test process. Worker,
// ownership and controlled-clock tests elsewhere still run under -race.
func productionNumericalReference(t *testing.T) bool {
	t.Helper()
	if !raceInstrumentedTests {
		return false
	}
	// Leave the parent time to report a child failure before the global panic.
	deadline := time.Now().Add(5 * time.Minute)
	if outer, ok := t.Deadline(); ok && outer.Before(deadline.Add(time.Second)) {
		deadline = outer.Add(-time.Second)
	}
	if !deadline.After(time.Now()) {
		t.Fatal("no time remains for the production reference")
	}
	ctx, cancel := context.WithDeadline(t.Context(), deadline)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "workbench-production.test")
	build := exec.CommandContext(ctx, "go", "test", "-c", "-race=false", "-o", binary, ".")
	build.WaitDelay = 250 * time.Millisecond
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build production numerical reference: %v\n%s", err, output)
	}
	remaining := time.Until(deadline) - time.Second
	if remaining <= 0 {
		t.Fatal("no time remains to run the production reference after building it")
	}
	segments := strings.Split(t.Name(), "/")
	for i, name := range segments {
		segments[i] = "^" + regexp.QuoteMeta(name) + "$"
	}
	command := exec.CommandContext(ctx, binary, "-test.run="+strings.Join(segments, "/"), "-test.count=1", "-test.v", "-test.short="+strconv.FormatBool(testing.Short()), "-test.timeout="+remaining.String())
	command.WaitDelay = 250 * time.Millisecond
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("production numerical reference failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "--- PASS: "+t.Name()+" ") {
		t.Fatalf("production child did not prove it ran %s:\n%s", t.Name(), output)
	}
	t.Logf("production-speed numerical reference passed; original assertions and %s/%d-attempt guards unchanged (child without -race)", MaxIntegrationWallTime, MaxIntegrationAttempts)
	return true
}
