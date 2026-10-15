package workbench

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func heldPausedStepInitialization(t *testing.T) (*Service, func()) {
	t.Helper()
	s := pausedNPNPulse(t)
	entered, release := blockFirstInitialization(t, s, false)
	promptLiveCommand(t, s, "simulation.restart", struct{}{})
	awaitLiveSampleBlock(t, entered)
	t.Cleanup(func() { release(); s.Close() })
	return s, release
}

func TestStepUnitInitializationWaitParity(t *testing.T) {
	for _, raw := range []string{`{"duration_s":0.001}`, `{"duration_s":"1 ms"}`, `{"duration_s":"1000 us"}`} {
		t.Run(raw, func(t *testing.T) {
			s, _ := heldPausedStepInitialization(t)
			before := s.Snapshot()
			rev := before.CommandRevision
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
			defer cancel()
			response := s.Dispatch(ctx, Request{Version: 1, ID: "waiting", Command: "simulation.step", Params: json.RawMessage(raw), Key: "waiting", ExpectedCommandRevision: &rev}, "test")
			if response.Error == nil || response.Error.Code != "canceled" || ctx.Err() == nil {
				t.Fatal("equivalent duration did not wait for initializer", response.Error)
			}
			after := s.Snapshot()
			if after.CommandRevision != before.CommandRevision || after.StepWork != nil || !reflect.DeepEqual(before.Transient, after.Transient) {
				t.Fatal("pre-admission wait changed physics or admitted a step")
			}
		})
	}
}

func TestStaleStepContextDoesNotWaitForInitialization(t *testing.T) {
	for _, guard := range []string{"revision", "command", "document"} {
		t.Run(guard, func(t *testing.T) {
			s, _ := heldPausedStepInitialization(t)
			before := s.Snapshot()
			current := before.CommandRevision
			staleRevision := before.Revision - 1
			staleCommand := current - 1
			otherDocument := before.Document.Generation + 1
			request := Request{Version: 1, ID: "stale", Command: "simulation.step", Params: json.RawMessage(`{"duration_s":0.001}`), Key: "stale", ExpectedCommandRevision: &current}
			switch guard {
			case "revision":
				request.ExpectedRevision = &staleRevision
			case "command":
				request.ExpectedCommandRevision = &staleCommand
			case "document":
				request.ExpectedDocumentGeneration = &otherDocument
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
			defer cancel()
			response := s.Dispatch(ctx, request, "test")
			if response.Error == nil || response.Error.Code != "conflict" || ctx.Err() != nil {
				t.Fatal("known-stale request waited instead of returning conflict", guard, response.Error)
			}
			if !reflect.DeepEqual(before, s.Snapshot()) {
				t.Fatal("stale request changed held state")
			}
		})
	}
}

func TestMalformedUnitStepSkipsInitializationWait(t *testing.T) {
	for _, raw := range []string{
		`{"duration_s":"bad"}`, `{"duration_s":"1 MS"}`, `{"duration_s":"1 ms","duration_s":0.001}`,
		`{"duration_s":"1 ms","extra":true}`, `{"duration_s":null}`, `{}`, `{"Duration_s":"1 ms"}`,
		`{"duration_s":{"seconds":0.001}}`, `{"duration_s":"` + strings.Repeat("1", 129) + ` ms"}`,
	} {
		t.Run(raw, func(t *testing.T) {
			s, _ := heldPausedStepInitialization(t)
			before := s.Snapshot()
			rev := before.CommandRevision
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
			defer cancel()
			response := s.Dispatch(ctx, Request{Version: 1, ID: "bad", Command: "simulation.step", Params: json.RawMessage(raw), Key: "bad", ExpectedCommandRevision: &rev}, "test")
			if response.Error == nil || response.Error.Code != "invalid-params" || ctx.Err() != nil {
				t.Fatal("malformed request waited for initializer", response.Error)
			}
			if !reflect.DeepEqual(before, s.Snapshot()) {
				t.Fatal("invalid request changed state")
			}
		})
	}
}

func TestUnitStepWaitCompletesOnceAndReplaysAcrossUnits(t *testing.T) {
	s, release := heldPausedStepInitialization(t)
	before := s.Snapshot()
	rev := before.CommandRevision
	request := Request{Version: 1, ID: "unit-step", Command: "simulation.step", Params: json.RawMessage(`{"duration_s":"1 ms"}`), Key: "unit-step", ExpectedCommandRevision: &rev}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result := make(chan Response, 1)
	go func() { result <- s.Dispatch(ctx, request, "test") }()
	select {
	case response := <-result:
		t.Fatal("step returned before held initialization", response.Error)
	case <-time.After(20 * time.Millisecond):
	}
	release()
	var original Response
	select {
	case original = <-result:
	case <-ctx.Done():
		t.Fatal("unit step did not finish after initializer release")
	}
	if original.Error != nil {
		t.Fatal(original.Error)
	}
	final := original.Result.(Snapshot)
	if final.StepWork == nil || final.StepWork.Status != "completed" || final.Transient.Sample.Time != .001 || final.CommandRevision != before.CommandRevision+1 {
		t.Fatal("step failed or admitted more than once", final.StepWork)
	}
	// An unrelated initializer must not delay replay. Cached replies deliberately
	// retain precedence over newer context guards, as in ordinary dispatch.
	entered, releaseAgain := blockFirstInitialization(t, s, false)
	t.Cleanup(releaseAgain)
	promptLiveCommand(t, s, "simulation.restart", struct{}{})
	awaitLiveSampleBlock(t, entered)
	unrelated := s.Snapshot()
	otherDocument := unrelated.Document.Generation + 1
	retry := request
	retry.ID = "equivalent-retry"
	retry.Params = json.RawMessage(`{"duration_s":"1000 us"}`)
	retry.ExpectedDocumentGeneration = &otherDocument
	short, stop := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer stop()
	duplicate := s.Dispatch(short, retry, "test")
	if duplicate.Error != nil || !duplicate.Duplicate || short.Err() != nil || !reflect.DeepEqual(duplicate.Result, original.Result) {
		t.Fatal("equivalent retry waited, recomputed or lost original outcome", duplicate.Error)
	}
	retry.Params = json.RawMessage(`{"duration_s":"2 ms"}`)
	wrong := s.Dispatch(short, retry, "test")
	if wrong.Error == nil || wrong.Error.Code != "key-reused" || short.Err() != nil {
		t.Fatal("reused-key conflict waited", wrong.Error)
	}
	if !reflect.DeepEqual(unrelated, s.Snapshot()) {
		t.Fatal("replay changed unrelated initialization")
	}
}

func TestStepWaitRechecksContextAfterInitialization(t *testing.T) {
	for _, intervening := range []bool{false, true} {
		t.Run(map[bool]string{false: "all-state-revision", true: "intervening-command"}[intervening], func(t *testing.T) {
			s, release := heldPausedStepInitialization(t)
			before := s.Snapshot()
			commandRev := before.CommandRevision
			allRev := before.Revision
			request := Request{Version: 1, ID: "guarded", Command: "simulation.step", Params: json.RawMessage(`{"duration_s":"1 ms"}`), Key: "guarded", ExpectedCommandRevision: &commandRev}
			if !intervening {
				request.ExpectedRevision = &allRev
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			result := make(chan Response, 1)
			go func() { result <- s.Dispatch(ctx, request, "test") }()
			select {
			case response := <-result:
				t.Fatal("returned before held initialization", response.Error)
			case <-time.After(20 * time.Millisecond):
			}
			if intervening {
				promptLiveCommand(t, s, "component.move", MoveParams{ID: before.Components[0].ID, Position: Point{100, 100}})
			}
			release()
			select {
			case response := <-result:
				if response.Error == nil || response.Error.Code != "conflict" || s.Snapshot().StepWork != nil || s.Snapshot().Transient.Sample.Time != 0 {
					t.Fatal("wait bypassed original context guard", response.Error)
				}
			case <-ctx.Done():
				t.Fatal("guarded wait did not finish")
			}
		})
	}
}

func TestCancelUnitWaitBeforeAdmissionLeavesInitializerAndRetryUsable(t *testing.T) {
	s, release := heldPausedStepInitialization(t)
	before := s.Snapshot()
	rev := before.CommandRevision
	request := Request{Version: 1, ID: "cancel-before-admission", Command: "simulation.step", Params: json.RawMessage(`{"duration_s":"1 ms"}`), Key: "cancel-before-admission", ExpectedCommandRevision: &rev}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	response := s.Dispatch(ctx, request, "test")
	if response.Error == nil || response.Error.Code != "canceled" || s.Snapshot().StepWork != nil {
		t.Fatal("canceled observation admitted work", response.Error)
	}
	release()
	readyCtx, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	if _, err := s.Wait(readyCtx); err != nil {
		t.Fatal(err)
	}
	response = s.Dispatch(readyCtx, request, "test")
	if response.Error != nil || response.Duplicate || response.Result.(Snapshot).Transient.Sample.Time != .001 {
		t.Fatal("pre-admission cancel affected initialization or reserved retry key", response.Error)
	}
}
