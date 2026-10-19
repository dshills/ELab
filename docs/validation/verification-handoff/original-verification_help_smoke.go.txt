package main

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeVerificationFailureSmoke(ctx context.Context, a *app.Application, s *screen) error {
	owner := func(action func() error) error {
		done := make(chan error, 1)
		if err := a.PostWait(ctx, func() { done <- action() }); err != nil {
			return err
		}
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err := owner(func() error {
		for _, request := range []struct {
			id string
			p  any
		}{{"example.rectifier", workbench.ReplaceParams{DiscardChanges: true}}, {"simulation.pause", struct{}{}}} {
			if r := s.invoke(request.id, request.p); r.Error != nil {
				return fmt.Errorf("verification failure setup %s: %s", request.id, r.Error.Message)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	var before workbench.Snapshot
	if err := owner(func() error {
		text := strconv.FormatFloat(math.Nextafter(.001, math.Inf(1)), 'g', -1, 64) + " s"
		if err := smokeUI(s, "verify-duration", &ui.ChangeEvent{Text: text}); err != nil {
			return err
		}
		before = s.service.Snapshot()
		return smokeUI(s, "verify-start", nil)
	}); err != nil {
		return err
	}
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		state := s.service.Snapshot()
		for _, operation := range state.Operations {
			if operation.ID != state.LastOperation || operation.Status == "queued" || operation.Status == "running" {
				continue
			}
			if operation.Status != "failed" || operation.FailureHelp == nil || operation.Error == nil || operation.Error.Code != "solve-failed" || !reflect.DeepEqual(before.Transient, state.Transient) || before.Document != state.Document || state.Verification != nil {
				return fmt.Errorf("native captured failure lost guidance or changed live state")
			}
			return owner(func() error {
				summary, ok := findNode(s.view(), "verify-failure-summary")
				if !ok || summary.Text != operation.FailureHelp.Summary {
					return fmt.Errorf("native verification failure summary differs from shared operation")
				}
				if err := smokeUI(s, "verify-failure-details", &ui.ChangeEvent{Checked: true}); err != nil {
					return err
				}
				if !s.service.Snapshot().DiagnosticPresentation.DetailsVisible {
					return fmt.Errorf("native verification disclosure did not use shared state")
				}
				return smokeUI(s, "verify-failure-details", &ui.ChangeEvent{Checked: false})
			})
		}
	}
}
