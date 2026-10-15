package main

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

// The caller has selected an owned checkpoint with an embedded recording.
func nativeRecoveryReadSmoke(ctx context.Context, a *app.Application, s *screen) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	before := s.service.Snapshot()
	action := make(chan error, 1)
	if err := a.PostWait(ctx, func() { action <- smokeUI(s, "recovery-inspect-copy", nil) }); err != nil {
		return err
	}
	if err := <-action; err != nil {
		return err
	}
	wait := func(kind string) (*workbench.RecoveryReadResult, error) {
		last := "no owner observation"
		type observation struct {
			result *workbench.RecoveryReadResult
			err    error
			detail string
		}
		for {
			observed := make(chan observation, 1)
			if err := a.PostWait(ctx, func() {
				var o observation
				_ = s.view() // Only schedules result decoding; no disk IO on owner.
				reads := s.service.Snapshot().Recovery.Reads
				work := reads[len(reads)-1]
				if work.Error != nil {
					o.err = fmt.Errorf("recovery read: %s", work.Error.Message)
				}
				cache := s.recoveryCache(kind)
				o.detail = fmt.Sprintf("job=%s/%s/%s selected=%s cache=%s loading=%t result=%t", work.ID, work.Status, work.Input.Kind, s.service.Snapshot().Recovery.Selected, cache.id, cache.loading, cache.result != nil)
				if cache.err != "" {
					o.err = fmt.Errorf("recovery cache: %s", cache.err)
				}
				if cache.id == work.ID {
					o.result = cache.result
				}
				observed <- o
			}); err != nil {
				return nil, fmt.Errorf("recovery %s owner wait (%s): %w", kind, last, err)
			}
			select {
			case o := <-observed:
				last = o.detail
				if o.result != nil || o.err != nil {
					return o.result, o.err
				}
			case <-ctx.Done():
				return nil, fmt.Errorf("recovery %s callback wait (%s): %w", kind, last, ctx.Err())
			}
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("recovery %s result wait (%s): %w", kind, last, ctx.Err())
			case <-time.After(20 * time.Millisecond):
			}
		}
	}
	inspection, err := wait("inspection")
	if err != nil {
		return err
	}
	if inspection.Inspection == nil || inspection.Inspection.Candidate.ID != before.Recovery.Selected || len(inspection.Inspection.Document.Recordings) == 0 {
		return fmt.Errorf("recovery read smoke lost captured source/recordings")
	}
	if err := a.PostWait(ctx, func() { action <- smokeUI(s, "recovery-recording-read", &ui.ChangeEvent{SelectedID: "0"}) }); err != nil {
		return err
	}
	if err := <-action; err != nil {
		return err
	}
	rows, err := wait("samples")
	if err != nil {
		return err
	}
	if rows.Samples == nil || rows.Samples.ID != before.Recovery.Selected || len(rows.Samples.Rows) == 0 {
		return fmt.Errorf("recovery read smoke lost captured rows")
	}
	after := s.service.Snapshot()
	if before.Document != after.Document || before.RunID != after.RunID || before.Recovery.Selected != after.Recovery.Selected || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		return fmt.Errorf("recovery reads changed live circuit/history/selection")
	}
	return nil
}
