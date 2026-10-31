package testutil

import (
	"context"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"
)

// RunAtProductionSpeed runs a deadline-sensitive test at ordinary binary speed
// when the parent is race-instrumented. A true return means the child passed.
// The same unmodified test body runs in a fresh ordinary test process. Worker,
// ownership and controlled-clock tests elsewhere still run under -race.
func RunAtProductionSpeed(t *testing.T) bool {
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
	binary := filepath.Join(t.TempDir(), "production.test")
	args := []string{"test", "-c", "-race=false", "-o", binary}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "-tags" {
				args = append(args, "-tags", setting.Value)
			}
		}
	}
	build := exec.CommandContext(ctx, "go", append(args, ".")...)
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
	t.Log("original assertions passed in the production-speed child")
	return true
}
