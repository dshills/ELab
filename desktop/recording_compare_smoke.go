package main

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeRecordingComparisonSmoke(ctx context.Context, a *app.Application, s *screen) error {
	before := s.service.Snapshot()
	if before.Recording == nil {
		return fmt.Errorf("comparison smoke needs a saved capture")
	}
	reference := before.Recording.ID
	result := make(chan error, 1)
	if err := a.PostWait(ctx, func() { result <- smokeUI(s, "verify-start", nil) }); err != nil {
		return err
	}
	if err := <-result; err != nil {
		return err
	}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		state := s.service.Snapshot()
		ready := false
		for _, op := range state.Operations {
			if op.ID == state.LastOperation {
				if op.Status == "failed" || op.Status == "canceled" {
					return fmt.Errorf("comparison smoke verification failed: %+v", op)
				}
				ready = op.Status == "completed"
			}
		}
		if ready {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
	if err := a.PostWait(ctx, func() {
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{{"recording-show", nil}, {"recorded-compare-reference", &ui.ChangeEvent{SelectedID: reference}}, {"recorded-compare-time", &ui.ChangeEvent{Text: "10 ms"}}, {"recorded-compare-statistics", &ui.ChangeEvent{Checked: true}}} {
			if err := smokeUI(s, action.id, action.event); err != nil {
				result <- err
				return
			}
		}
		state := s.service.Snapshot()
		comparison := state.Recording.Comparison
		if comparison == nil || !comparison.Available || math.Abs(comparison.A.Time-.01) > 1e-12 || math.Abs(comparison.B.Time-.01) > 1e-12 || !comparison.Settings.ShowStatistics || state.RunID != before.RunID || state.Transient.Sample != before.Transient.Sample {
			result <- fmt.Errorf("native comparison time/physics identity")
			return
		}
		root := s.view()
		if _, ok := findNode(root, "recorded-compare-times"); !ok {
			result <- fmt.Errorf("comparison time labels absent")
			return
		}
		if err := smokeUI(s, "recorded-compare-reference", &ui.ChangeEvent{SelectedID: "__none"}); err != nil {
			result <- err
			return
		}
		if response := s.invoke("recording.keep.set", workbench.RecordingKeepParams{ID: state.Recording.ID, Keep: false}); response.Error != nil {
			result <- fmt.Errorf("exclude owned smoke capture: %s", response.Error.Message)
			return
		}
		if response := s.invoke("operation.select", workbench.IDParams{ID: reference}); response.Error != nil {
			result <- fmt.Errorf("restore owned capture selection: %s", response.Error.Message)
			return
		}
		result <- nil
	}); err != nil {
		return err
	}
	return <-result
}
