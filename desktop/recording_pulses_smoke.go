package main

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeRecordingPulsesSmoke(ctx context.Context, a *app.Application, s *screen) error {
	done := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if r := s.invoke("example.npn-pulse", workbench.ReplaceParams{DiscardChanges: true}); r.Error != nil {
			done <- fmt.Errorf("pulse fixture: %s", r.Error.Message)
			return
		}
		done <- smokeUI(s, "simulation.pause", nil)
	}); err != nil {
		return err
	}
	if err := <-done; err != nil {
		return err
	}
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	if err := a.PostWait(ctx, func() {
		duration := .25
		r := s.invoke("simulation.verify", workbench.VerifyParams{Duration: &duration, InputMode: "auto", ProbeIDs: []string{"v:VIN.positive"}})
		if r.Error != nil {
			done <- fmt.Errorf("pulse capture: %s", r.Error.Message)
			return
		}
		done <- nil
	}); err != nil {
		return err
	}
	if err := <-done; err != nil {
		return err
	}
	for {
		state := s.service.Snapshot()
		ready := false
		for _, op := range state.Operations {
			if op.ID == state.LastOperation {
				if op.Status == "failed" || op.Status == "canceled" {
					return fmt.Errorf("pulse capture %s", op.Status)
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
		case <-time.After(10 * time.Millisecond):
		}
	}
	before := s.service.Snapshot()
	if err := a.PostWait(ctx, func() {
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{{"recording-show", nil}, {"recorded-timing-probe", &ui.ChangeEvent{SelectedID: "v:VIN.positive"}}, {"recorded-timing-level", &ui.ChangeEvent{Text: "2.5 V"}}, {"recorded-timing-direction", &ui.ChangeEvent{SelectedID: "falling"}}, {"recorded-timing-second-level", &ui.ChangeEvent{Text: "4.5 V"}}} {
			if err := smokeUI(s, action.id, action.event); err != nil {
				done <- err
				return
			}
		}
		state := s.service.Snapshot()
		c := state.Recording.Timing.Cycle
		if c == nil || c.Duty == nil || math.Abs(c.Duty.Minimum-.5) > 1e-12 || math.Abs(c.High.Minimum-.05) > 1e-12 || math.Abs(c.Low.Minimum-.05) > 1e-12 {
			done <- fmt.Errorf("native pulse timing differs from ideal source")
			return
		}
		tr := state.Recording.Timing.Transitions
		if tr == nil || tr.Rise == nil || tr.Fall == nil || tr.Rise.Minimum != 0 || tr.Rise.Maximum != 0 || tr.Fall.Minimum != 0 || tr.Fall.Maximum != 0 {
			done <- fmt.Errorf("native ideal-source transition timing lost duplicate rows")
			return
		}
		root := s.view()
		for _, id := range []string{"recorded-pulse-high", "recorded-pulse-low", "recorded-pulse-period", "recorded-pulse-duty", "recorded-transition-levels", "recorded-transition-rise", "recorded-transition-fall"} {
			if _, ok := findNode(root, id); !ok {
				done <- fmt.Errorf("native pulse readout absent: %s", id)
				return
			}
		}
		if before.Document != state.Document || before.RunID != state.RunID || !reflect.DeepEqual(before.NPN, state.NPN) || !reflect.DeepEqual(before.UndoEntry, state.UndoEntry) {
			done <- fmt.Errorf("pulse timing changed live circuit/history")
			return
		}
		done <- nil
	}); err != nil {
		return err
	}
	return <-done
}
