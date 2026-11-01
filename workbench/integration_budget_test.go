package workbench

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestIntegrationBudgetReasonsPreserveWorkBounds(t *testing.T) {
	if MaxIntegrationAttempts != 512 || MaxIntegrationCPUTime != 100*time.Millisecond || MaxIntegrationWallTime != 5*time.Second || Limits().IntegrationAttempts != 512 || Limits().IntegrationWallMS != 5000 || Limits().IntegrationCPUMS != 100 {
		t.Fatal("published CPU, wall and attempt bounds disagree")
	}
	budget := &integrationBudget{elapsed: func() time.Duration { return 0 }}
	for i := 0; i < MaxIntegrationAttempts; i++ {
		if err := budget.take(); err != nil {
			t.Fatal("attempt rejected before original cap", i, err)
		}
	}
	err := budget.take()
	var limit *integrationBudgetError
	if !errors.As(err, &limit) || limit.code != "integration-attempt-limit" || limit.attempts != 513 || !strings.Contains(err.Error(), "maximum 512") {
		t.Fatal("attempt limit missing reason/count", err)
	}
	// Isolate elapsed-time rejection without scheduler timing, sleeps or
	// changing the production CPU clock or quotas.
	budget = &integrationBudget{elapsed: func() time.Duration { return 2 * MaxIntegrationWallTime }}
	err = budget.take()
	if !errors.As(err, &limit) || limit.code != "integration-time-limit" || limit.attempts != 1 || limit.elapsed < MaxIntegrationWallTime {
		t.Fatal("elapsed limit conflated with attempt cap", err)
	}
	if simulationFailureCode(fmt.Errorf("sample context: %w", err), "solve-failed") != "integration-time-limit" || simulationFailureCode(errors.New("other solver error"), "solve-failed") != "solve-failed" {
		t.Fatal("wrapped error classification changed unrelated failures")
	}
}

func TestIntegrationLimitStepFailurePreservesPhysicsAndGuidance(t *testing.T) {
	for _, code := range []string{"integration-attempt-limit", "integration-time-limit"} {
		t.Run(code, func(t *testing.T) {
			s := pausedFlasher(t)
			defer s.Close()
			before := s.Snapshot()
			attempts := 513
			if code == "integration-time-limit" {
				attempts = 10
			}
			s.mu.Lock()
			s.networkAdvance = func(context.Context, *mixedState, *Transient, float64, Graph) (*mixedState, *Transient, error) {
				return nil, nil, fmt.Errorf("sample context: %w", &integrationBudgetError{code: code, attempts: attempts, elapsed: 101 * time.Millisecond})
			}
			s.mu.Unlock()
			refuseStepWork(t, s, "simulation.step", StepParams{Duration: .005}, code)
			after := s.Snapshot()
			if after.StepWork == nil || after.StepWork.Status != "failed" || after.StepWork.Error.Code != code || before.RunID != after.RunID || before.Document != after.Document || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Mixed, after.Mixed) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
				t.Fatal("failed budget step loses classification or changes committed state")
			}
			help := after.DiagnosticPresentation.Items[0]
			if help.Code != code || help.Source != "step" || !strings.Contains(help.Summary, "Manual step") || !strings.Contains(help.Action, "preserved") || !strings.Contains(help.Details, "maximum 512") || len(help.Objects) != 0 {
				t.Fatal("budget guidance missing or invents a causal part", help)
			}
			if code == "integration-time-limit" && !strings.Contains(help.Action, "less busy") {
				t.Fatal("elapsed limit lacks computer-load distinction")
			}
		})
	}
}

func coarseCornerGraph(t *testing.T) Graph {
	t.Helper()
	g := waveformRCGraph(t, "finite-pulse-source")
	g.Components = append(g.Components[:2], g.Components[3:]...)
	g.Wires = []Wire{{ID: "A", From: "V1.positive", To: "R1.1", Connected: true}, {ID: "B", From: "R1.2", To: "GND.reference", Connected: true}, {ID: "C", From: "V1.negative", To: "GND.reference", Connected: true}}
	g.Components[0].Pulse = &PulseWaveform{HighVoltage: 5, HighSeconds: .0005, LowSeconds: .0005, RiseSeconds: .0001, FallSeconds: .0001}
	return g
}

func TestRecordedIntegrationAttemptFailureDoesNotKeepPartialRecord(t *testing.T) {
	s := newGraphService(SolveLED, coarseCornerGraph(t))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	before := s.Snapshot()
	duration, step := .2, .2
	r := call(t, s, "simulation.verify", VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:V1.positive"}}, "verify")
	op := awaitOperation(t, s, r.Result.(Snapshot).LastOperation)
	after := s.Snapshot()
	if op.Operation.Status != "failed" || op.Operation.Error == nil || op.Operation.Error.Code != "integration-attempt-limit" || op.Metadata == nil || op.Operation.KeptInProject || before.Document != after.Document || !reflect.DeepEqual(before.Transient, after.Transient) {
		t.Fatal("recorded budget failure loses reason or publishes partial data", op.Operation)
	}
	raw, _ := json.Marshal(VerificationPageParams{ID: op.Operation.ID, Limit: 1})
	response := s.Dispatch(context.Background(), Request{Version: 1, ID: "samples", Command: "operation.samples", Params: raw}, "test")
	if response.Error == nil || response.Error.Code != "not-ready" {
		t.Fatal("failed recording exposes partial result rows", response.Error)
	}
}

func TestLiveIntegrationTimeFailureRetainsLastAcceptedSample(t *testing.T) {
	s := pausedFlasher(t)
	defer s.Close()
	before := s.Snapshot()
	s.mu.Lock()
	s.networkAdvance = func(context.Context, *mixedState, *Transient, float64, Graph) (*mixedState, *Transient, error) {
		return nil, nil, fmt.Errorf("sample context: %w", &integrationBudgetError{code: "integration-time-limit", attempts: 10, elapsed: 101 * time.Millisecond})
	}
	s.mu.Unlock()
	call(t, s, "simulation.resume", struct{}{}, "resume")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for s.Snapshot().Status != "error" {
		select {
		case <-ctx.Done():
			t.Fatal("live sample failure not published", ctx.Err())
		case <-time.After(time.Millisecond):
		}
	}
	after := s.Snapshot()
	if after.Status != "error" || len(after.Diagnostics) != 1 || after.Diagnostics[0].Code != "integration-time-limit" || !after.Transient.Stale || after.Transient.Sample != before.Transient.Sample || after.RunID != before.RunID || after.Document != before.Document {
		t.Fatal("live budget failure advances or mislabels state", after.Diagnostics)
	}
	if help := after.DiagnosticPresentation.Items[0]; help.Code != "integration-time-limit" || !strings.Contains(help.Summary, "time limit") || len(help.Objects) != 0 {
		t.Fatal("live budget guidance missing or guesses location", help)
	}
}
