package main

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func awaitNativeRecoveryRead(t *testing.T, s *screen, kind string) *workbench.RecoveryReadResult {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var result *workbench.RecoveryReadResult
		s.app.Do(func() {
			if err := ui.Validate(s.view()); err != nil {
				t.Fatal(err)
			}
			cache := s.recoveryCache(kind)
			if cache.err != "" {
				t.Fatal(cache.err)
			}
			reads := s.service.Snapshot().Recovery.Reads
			if len(reads) > 0 && cache.id == reads[len(reads)-1].ID {
				result = cache.result
			}
		})
		if result != nil {
			return result
		}
		s.app.Drain()
		time.Sleep(time.Millisecond)
	}
	t.Fatal("native recovery result did not load", s.service.Snapshot().Recovery.Reads)
	return nil
}

func TestNativeRecoveryInspectionAndRowsUseSharedCapturedResults(t *testing.T) {
	directory := t.TempDir()
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	source := workbench.New()
	defer source.Close()
	agentAction(t, source, "example.rc", workbench.ReplaceParams{})
	agentAction(t, source, "simulation.pause", struct{}{})
	duration := .03
	agentAction(t, source, "simulation.verify", workbench.VerifyParams{Duration: &duration})
	completedVerification(t, source, source.Snapshot().LastOperation)
	agentAction(t, source, "recovery.configure", workbench.RecoveryConfigureParams{Directory: directory, Interval: 1})
	agentAction(t, source, "recovery.checkpoint", struct{}{})
	source.Close()
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned recovery read test", Width: 1140, Height: 860, View: s.view}); err != nil {
		t.Fatal(err)
	}
	agentAction(t, s.service, "example.reset", workbench.ReplaceParams{})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	agentAction(t, s.service, "recovery.configure", workbench.RecoveryConfigureParams{Directory: directory, Interval: 1})
	choice := s.service.Snapshot().Recovery.Candidates[0]
	uiAction(t, s, "recovery-choice", &ui.ChangeEvent{SelectedID: choice.ID})
	before := s.service.Snapshot()
	uiAction(t, s, "recovery-inspect-copy", nil)
	result := awaitNativeRecoveryRead(t, s, "inspection")
	if result.Inspection.Candidate.ID != choice.ID || result.Inspection.Candidate.SHA256 != choice.SHA256 || len(result.Inspection.Document.Recordings) != 1 {
		t.Fatal("native read lost captured identity/recordings")
	}
	part, _ := result.Inspection.Document.Graph.Component("R1")
	if part.Value != 10000 {
		t.Fatal("native inspection substituted live LED resistor")
	}
	uiAction(t, s, "recovery-recording-read", &ui.ChangeEvent{SelectedID: "0"})
	page := awaitNativeRecoveryRead(t, s, "samples")
	if page.Samples.Offset != 0 || len(page.Samples.Rows) != 20 || page.Samples.ID != choice.ID {
		t.Fatal("native sample result lost paging/source")
	}
	uiAction(t, s, "recovery-samples-next", nil)
	page = awaitNativeRecoveryRead(t, s, "samples")
	if page.Samples.Offset != 20 {
		t.Fatal("native next page starts at wrong row")
	}
	after := s.service.Snapshot()
	if before.Document != after.Document || before.RunID != after.RunID || before.Recovery.Selected != after.Recovery.Selected || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("read controls change live schematic/history/selection")
	}
	work := after.Recovery.Reads[len(after.Recovery.Reads)-1]
	if work.Actor != "user" {
		t.Fatal("native action bypassed shared command actor")
	}
	owned, err := readRecoveryResult(context.Background(), s.service, work)
	if err != nil || !reflect.DeepEqual(owned, page) {
		t.Fatal("native cache differs from public chunked result", err)
	}
	// Queue decoding and leave the candidate before its owner callback can run.
	// The immutable cache must finish so returning to this copy stays usable.
	s.app.Do(func() {
		s.recoverySamples = recoveryReadCache{}
		s.loadRecoveryRead(&work)
		agentAction(t, s.service, "recovery.select", workbench.IDParams{})
	})
	deadline := time.Now().Add(5 * time.Second)
	loaded := false
	for time.Now().Before(deadline) && !loaded {
		s.app.Drain()
		s.app.Do(func() { loaded = !s.recoverySamples.loading && s.recoverySamples.result != nil })
		time.Sleep(time.Millisecond)
	}
	if !loaded || s.service.Snapshot().Recovery.Selected != "" {
		t.Fatal("selection switch strands loading cache or changes selection")
	}
	uiAction(t, s, "recovery-choice", &ui.ChangeEvent{SelectedID: choice.ID})
	if again := awaitNativeRecoveryRead(t, s, "samples"); !reflect.DeepEqual(again, owned) {
		t.Fatal("returning to selected copy lost cached result")
	}
	page.Samples.Rows[0].Values[0] = 999
	again, err := readRecoveryResult(context.Background(), s.service, work)
	if err != nil || again.Samples.Rows[0].Values[0] == 999 {
		t.Fatal("native result aliases service retained data", err)
	}
	// PostWait admits a callback; it does not await execution. Exercise the
	// packaged smoke helper with a separate worker and actual owner draining.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- nativeRecoveryReadSmoke(ctx, s.app, s) }()
	for {
		s.app.Drain()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			return
		case <-ctx.Done():
			t.Fatal("recovery smoke did not await its owner callback")
		case <-time.After(time.Millisecond):
		}
	}
}
