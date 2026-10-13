package workbench

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"
)

func call(t *testing.T, s *Service, command string, p any, key string) Response {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	rev := s.Snapshot().Revision
	r := s.Dispatch(context.Background(), Request{1, key, command, raw, &rev, key}, "test")
	// Resolve only a known admission conflict with an async result commit.
	if r.Error != nil && r.Error.Code == "conflict" {
		rev = r.Revision
		r = s.Dispatch(context.Background(), Request{1, key, command, raw, &rev, key}, "test")
	}
	if r.Error != nil {
		t.Fatalf("%s: %+v", command, r.Error)
	}
	return r
}
func settled(t *testing.T, s *Service) Snapshot {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	state, err := s.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func TestLEDReferenceAndConservation(t *testing.T) {
	for _, r := range []float64{10, 1000, 2000, 100000} {
		for _, v := range []float64{0, .2, 5, 12} {
			result, err := SolveLED(context.Background(), Circuit{v, r, true})
			if err != nil {
				t.Fatal(err)
			}
			if math.Abs(result.KVLResidual) > 1e-12 || result.Current < 0 || result.Current > v/r {
				t.Fatalf("KVL/bounds failed: %+v", result)
			}
			if math.Abs(result.ResistorPower+result.LEDPower-v*result.Current) > 1e-12 {
				t.Fatal("power conservation failed")
			}
			if v == 5 && r == 1000 && math.Abs(result.Current-0.00291679) > 2e-8 {
				t.Fatalf("research reference mismatch: %g", result.Current)
			}
		}
	}
}
func TestLiveEditsRecoveryAndUndo(t *testing.T) {
	s := NewExample()
	defer s.Close()
	initial := settled(t, s)
	call(t, s, "component.set", map[string]string{"id": "R1", "value": "2kΩ"}, "resistor")
	edited := settled(t, s)
	if edited.Result.Current >= initial.Result.Current || edited.ElectricalRevision != initial.ElectricalRevision+1 {
		t.Fatal("live value edit failed")
	}
	call(t, s, "circuit.connection.set", map[string]bool{"connected": false}, "disconnect")
	broken := s.Snapshot()
	if broken.Status != "incomplete" || !broken.Result.Stale || len(broken.Diagnostics) != 1 {
		t.Fatalf("invalid result looked valid: %+v", broken)
	}
	call(t, s, "edit.undo", struct{}{}, "undo-disconnect")
	restored := settled(t, s)
	if !restored.Circuit.Connected || restored.Result.Stale {
		t.Fatal("undo failed to recover")
	}
	call(t, s, "edit.undo", struct{}{}, "undo-r")
	if settled(t, s).Circuit.Resistance != 1000 {
		t.Fatal("second undo failed")
	}
	call(t, s, "edit.redo", struct{}{}, "redo-r")
	if settled(t, s).Circuit.Resistance != 2000 {
		t.Fatal("redo failed")
	}
}
func TestPauseAndPresentationChanges(t *testing.T) {
	s := NewExample()
	defer s.Close()
	initial := settled(t, s)
	call(t, s, "component.select", map[string]string{"id": "D1"}, "select")
	call(t, s, "view.zoom", map[string]float64{"factor": 2}, "zoom")
	if s.Snapshot().RunID != initial.RunID {
		t.Fatal("presentation edit restarted solver")
	}
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "component.set", map[string]string{"id": "V1", "value": "3 V"}, "voltage")
	paused := s.Snapshot()
	if paused.Status != "paused" || !paused.Result.Stale {
		t.Fatal("paused edit published current result")
	}
	call(t, s, "simulation.resume", struct{}{}, "resume")
	if settled(t, s).Result.Stale {
		t.Fatal("resume did not calculate")
	}
}
func TestConflictRetryAndValidation(t *testing.T) {
	s := NewExample()
	defer s.Close()
	state := settled(t, s)
	rev := state.Revision
	request := Request{1, "first", "component.set", json.RawMessage(`{"id":"R1","value":"2k"}`), &rev, "same"}
	first := s.Dispatch(context.Background(), request, "test")
	if first.Error != nil {
		t.Fatal(first.Error)
	}
	settled(t, s)
	firstState := first.Result.(Snapshot)
	firstState.Components[0].Terminals[0] = "mutated"
	duplicate := s.Dispatch(context.Background(), request, "test")
	if !duplicate.Duplicate || duplicate.Error != nil || s.Snapshot().Circuit.Resistance != 2000 {
		t.Fatal("retry was not deduplicated")
	}
	if duplicate.Result.(Snapshot).Components[0].Terminals[0] != "positive" {
		t.Fatal("caller mutated cached retry result")
	}
	request.Params = json.RawMessage(`{"id":"R1","value":"3k"}`)
	if r := s.Dispatch(context.Background(), request, "test"); r.Error == nil || r.Error.Code != "key-reused" {
		t.Fatal("key reuse accepted")
	}
	request.Key = "new-key"
	if r := s.Dispatch(context.Background(), request, "test"); r.Error == nil || r.Error.Code != "conflict" {
		t.Fatal("stale edit accepted")
	}
	for i, p := range []string{`{"id":"R1","value":"0"}`, `{"id":"R1","value":"NaN"}`, `{"id":"V1","value":"13 V"}`, `{"id":"R1","value":"2k","extra":true}`, `{"connected":null}`, `{}`, `{"id":"R1","id":"V1","value":"3"}`} {
		before := s.Snapshot()
		rev := before.Revision
		command := "component.set"
		if i == 4 {
			command = "circuit.connection.set"
		}
		r := s.Dispatch(context.Background(), Request{1, "invalid", command, json.RawMessage(p), &rev, "invalid"}, "test")
		if r.Error == nil || s.Snapshot().Circuit != before.Circuit || s.Snapshot().Revision != before.Revision {
			t.Fatalf("invalid request changed state: %s %+v", p, r)
		}
	}
}
func TestCanceledOldSolveCannotPublish(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	s := newService(func(ctx context.Context, c Circuit) (Result, error) {
		if c.Resistance == 1000 {
			close(started)
			<-release
		}
		return SolveLED(context.Background(), c)
	})
	defer s.Close()
	<-started
	call(t, s, "component.set", map[string]string{"id": "R1", "value": "2k"}, "new")
	state := settled(t, s)
	close(release)
	time.Sleep(10 * time.Millisecond)
	after := s.Snapshot()
	if after.Result.RunID != state.RunID || after.Result.ElectricalRevision != after.ElectricalRevision || after.Circuit.Resistance != 2000 {
		t.Fatal("obsolete solve was published")
	}
}
func TestSnapshotIsDetached(t *testing.T) {
	s := NewExample()
	defer s.Close()
	a := settled(t, s)
	a.Result.Nodes["supply"] = 100
	a.Components[0].Terminals[0] = "bad"
	b := s.Snapshot()
	if b.Result.Nodes["supply"] != 5 || b.Components[0].Terminals[0] != "positive" {
		t.Fatal("snapshot leaked mutable state")
	}
}
