package workbench

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func eventRequest(revision uint64, ms int) Request {
	raw, _ := json.Marshal(EventPollParams{AfterRevision: revision, TimeoutMS: &ms})
	return Request{Version: 1, ID: "events", Command: "events.poll", Params: raw}
}
func eventTopicsContain(result EventPollResult, topic string) bool {
	for _, publication := range result.Publications {
		if publication.Topic == topic {
			return true
		}
	}
	return false
}
func awaitEventWaiters(t *testing.T, s *Service, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		s.mu.Lock()
		n := s.eventWaiters
		s.mu.Unlock()
		if n == count {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("event waiters %d, want %d", n, count)
		}
		time.Sleep(time.Millisecond)
	}
}
func TestEventPollIdleTimeoutAndReadOnlyState(t *testing.T) {
	s := New()
	defer s.Close()
	before := s.Snapshot()
	response := s.Dispatch(context.Background(), eventRequest(before.Revision, 0), "observer")
	result := response.Result.(EventPollResult)
	if response.Error != nil || result.Changed || result.TimedOut || result.Snapshot != nil || len(result.Publications) != 0 {
		t.Fatal("idle nonblocking poll invents an event/snapshot")
	}
	response = s.Dispatch(context.Background(), eventRequest(before.Revision, 5), "observer")
	result = response.Result.(EventPollResult)
	if !result.TimedOut || result.Changed || result.Snapshot != nil || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("idle timeout is not successful/read-only/bounded")
	}
	response = s.Dispatch(context.Background(), eventRequest(before.Revision+1, 0), "observer")
	if response.Error == nil || response.Error.Code != "conflict" || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("future process cursor accepted or changes state")
	}
	for _, ms := range []int{-1, MaxEventWaitMS + 1} {
		response = s.Dispatch(context.Background(), eventRequest(before.Revision, ms), "observer")
		if response.Error == nil || response.Error.Code != "invalid-params" {
			t.Fatal("invalid timeout admitted")
		}
	}
}
func TestEventPollWakesForOtherActorAndCoalescesOwnedSnapshot(t *testing.T) {
	s := New()
	defer s.Close()
	before := s.Snapshot()
	done := make(chan Response, 1)
	go func() { done <- s.Dispatch(context.Background(), eventRequest(before.Revision, 1000), "observer") }()
	awaitEventWaiters(t, s, 1)
	call(t, s, "component.place", PlaceParams{Kind: "resistor", Position: Point{}}, "place")
	select {
	case response := <-done:
		if response.Error != nil {
			t.Fatal(response.Error)
		}
		result := response.Result.(EventPollResult)
		if !result.Changed || result.Snapshot == nil || !eventTopicsContain(result, "editor") || !eventTopicsContain(result, "project") || result.Snapshot.LastActor != "test" || result.CommandRevision != result.Snapshot.CommandRevision {
			t.Fatal("event loses attributed shared editor state")
		}
		result.Snapshot.Components[0].Value = 123
		if s.Snapshot().Components[0].Value == 123 {
			t.Fatal("event snapshot aliases committed graph")
		}
	case <-time.After(time.Second):
		t.Fatal("publication did not wake observer")
	}
	call(t, s, "view.pan", map[string]float64{"dx": 10, "dy": 0}, "pan")
	call(t, s, "view.zoom", map[string]float64{"factor": 1.1}, "zoom")
	result := s.Dispatch(context.Background(), eventRequest(before.Revision, 0), "observer").Result.(EventPollResult)
	if !result.Coalesced || !eventTopicsContain(result, "view") || !eventTopicsContain(result, "editor") || result.Revision != result.Snapshot.Revision {
		t.Fatal("reconnect cursor loses coalesced state/publication topics")
	}
	awaitEventWaiters(t, s, 0)
}
func TestEventWaitQuotaCancellationAndClose(t *testing.T) {
	s := New()
	defer s.Close()
	rev := s.Snapshot().Revision
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	responses := make(chan Response, MaxEventWaiters)
	for i := 0; i < MaxEventWaiters; i++ {
		go func() { responses <- s.Dispatch(ctx, eventRequest(rev, 5000), "observer") }()
	}
	awaitEventWaiters(t, s, MaxEventWaiters)
	response := s.Dispatch(context.Background(), eventRequest(rev, 1000), "observer")
	if response.Error == nil || response.Error.Code != "busy" {
		t.Fatal("unbounded event waits admitted")
	}
	// Nonblocking reads remain available at quota.
	if response = s.Dispatch(context.Background(), eventRequest(rev, 0), "observer"); response.Error != nil {
		t.Fatal(response.Error)
	}
	cancel()
	for i := 0; i < MaxEventWaiters; i++ {
		select {
		case response = <-responses:
			if response.Error == nil || response.Error.Code != "canceled" {
				t.Fatal("wait cancellation not reported")
			}
		case <-time.After(time.Second):
			t.Fatal("canceled wait retained quota")
		}
	}
	awaitEventWaiters(t, s, 0)
	done := make(chan Response, 1)
	go func() { done <- s.Dispatch(context.Background(), eventRequest(rev, 1000), "observer") }()
	awaitEventWaiters(t, s, 1)
	s.Close()
	select {
	case response = <-done:
		result := response.Result.(EventPollResult)
		if !result.Closed || result.Snapshot == nil || !result.Snapshot.Closed || !eventTopicsContain(result, "lifecycle") {
			t.Fatal("close does not publish coherent terminal state")
		}
	case <-time.After(time.Second):
		t.Fatal("close stranded observer")
	}
	closed := s.Snapshot()
	result := s.Dispatch(context.Background(), eventRequest(closed.Revision, 5000), "observer").Result.(EventPollResult)
	if !result.Closed || result.Changed || result.Snapshot == nil {
		t.Fatal("closed cursor waits or invents a new change")
	}
}
func TestEventsObserveStepWithoutChangingAcceptedPhysics(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	call(t, s, "simulation.step", StepParams{Duration: .01}, "step")
	after := s.Snapshot()
	result := s.Dispatch(context.Background(), eventRequest(before.Revision, 0), "observer").Result.(EventPollResult)
	if !eventTopicsContain(result, "simulation") || !eventTopicsContain(result, "jobs") || result.Snapshot.StepWork.Status != "completed" || !reflect.DeepEqual(after.Transient, result.Snapshot.Transient) || !reflect.DeepEqual(after, s.Snapshot()) {
		t.Fatal("event fails to expose terminal sample/job or mutates physics")
	}
}

func TestFullEventQuotaStillAllowsMutationAdmission(t *testing.T) {
	s := New()
	defer s.Close()
	before := s.Snapshot()
	done := make(chan Response, MaxEventWaiters)
	for i := 0; i < MaxEventWaiters; i++ {
		go func() { done <- s.Dispatch(context.Background(), eventRequest(before.Revision, 1000), "observer") }()
	}
	awaitEventWaiters(t, s, MaxEventWaiters)
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 10, "dy": 0})
	for i := 0; i < MaxEventWaiters; i++ {
		select {
		case response := <-done:
			if response.Error != nil || !response.Result.(EventPollResult).Changed {
				t.Fatal("mutation cannot wake quota of observers")
			}
		case <-time.After(time.Second):
			t.Fatal("observers blocked mutation/wake")
		}
	}
	awaitEventWaiters(t, s, 0)
}
