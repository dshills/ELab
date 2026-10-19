package main

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
)

func nativeScopeContextSmoke(ctx context.Context, a *app.Application, s *screen) error {
	ready := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		for _, action := range []struct {
			id     string
			params any
		}{
			{"example.rc", workbench.ReplaceParams{DiscardChanges: true}},
			{"simulation.pause", struct{}{}},
			{"component.set", workbench.ValueParams{ID: "R1", Value: "12k"}},
			{"simulation.restart", struct{}{}},
			{"scope.cursor.select", workbench.CursorSelectParams{Cursor: "a"}},
			{"simulation.step.start", workbench.StepParams{Duration: .02}},
		} {
			if response := s.invoke(action.id, action.params); response.Error != nil {
				ready <- fmt.Errorf("scope context %s: %s", action.id, response.Error.Message)
				return
			}
		}
		ready <- nil
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	if _, err := smokeScopeHash(ctx, a, s, nil); err != nil {
		return err
	}
	if err := a.PostWait(ctx, func() {
		old := s.scope.frame.Load()
		if old == nil {
			ready <- fmt.Errorf("GPU scope has no prepared input context")
			return
		}
		if response := s.invoke("scope.window.set", workbench.ScopeWindow{Mode: "fixed", Start: .002, End: .018}); response.Error != nil {
			ready <- fmt.Errorf("scope interval: %s", response.Error.Message)
			return
		}
		before := s.service.Snapshot()
		if _, ok := s.scope.applyCursorFrame(old, (old.left+old.right)/2, old.revision); ok || !reflect.DeepEqual(before, s.service.Snapshot()) {
			ready <- fmt.Errorf("stale prepared scope accepted a cursor")
			return
		}
		ready <- nil
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	// A canvas readback may precede a deferred viewport publication. Wait for
	// an actually rebuilt plot with current input context, then apply both
	// gesture events on one owner turn. Never rebase a stale prepared frame.
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for {
		if _, err := smokeScopeHash(bounded, a, s, nil); err != nil {
			return err
		}
		type outcome struct {
			fresh bool
			err   error
		}
		checked := make(chan outcome, 1)
		if err := a.PostWait(bounded, func() {
			frame := s.scope.frame.Load()
			before := s.service.Snapshot()
			if frame == nil || frame.revision != before.CommandRevision || frame.generation != before.Document.Generation {
				checked <- outcome{}
				return
			}
			s.scope.cursorGesture(frame, frame.left+(frame.right-frame.left)*.25, true)
			s.scope.cursorGesture(frame, frame.left+(frame.right-frame.left)*.75, false)
			s.scope.gesture = nil
			after := s.service.Snapshot()
			var failure error
			if after.Scope.A == nil || math.Abs(after.Scope.A.Requested-.014) > 1e-15 || s.errorMessage != "" || before.RunID != after.RunID || before.Transient.Sample != after.Transient.Sample || before.Document != after.Document || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
				failure = fmt.Errorf("fresh prepared scope drag failed or changed live work: frame=%+v command_revision=%d→%d cursor=%+v error=%q run_preserved=%t sample_preserved=%t document_preserved=%t history_preserved=%t", frame, before.CommandRevision, after.CommandRevision, after.Scope.A, s.errorMessage, before.RunID == after.RunID, before.Transient.Sample == after.Transient.Sample, before.Document == after.Document, reflect.DeepEqual(before.UndoEntry, after.UndoEntry))
			}
			checked <- outcome{fresh: true, err: failure}
		}); err != nil {
			return err
		}
		select {
		case result := <-checked:
			if result.fresh {
				return result.err
			}
		case <-bounded.Done():
			return bounded.Err()
		}
		select {
		case <-bounded.Done():
			return fmt.Errorf("scope did not prepare current input context: %w", bounded.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
}
