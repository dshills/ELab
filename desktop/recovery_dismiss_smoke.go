package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeRecoveryDismissSmoke(ctx context.Context, a *app.Application, s *screen) error {
	action := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if !s.service.Snapshot().Paused {
			if err := smokeUI(s, "simulation.pause", nil); err != nil {
				action <- err
				return
			}
		}
		action <- smokeUI(s, "recovery-checkpoint", nil)
	}); err != nil {
		return err
	}
	if err := <-action; err != nil {
		return err
	}
	state, err := s.service.Wait(ctx)
	if err != nil {
		return err
	}
	if state.ProjectWork == nil || state.ProjectWork.Status != "completed" || state.ProjectWork.RecoveryID == "" {
		return fmt.Errorf("native dismiss fixture has no committed checkpoint")
	}
	id := state.ProjectWork.RecoveryID
	if err := a.PostWait(ctx, func() { action <- smokeUI(s, "recovery-choice", &ui.ChangeEvent{SelectedID: id}) }); err != nil {
		return err
	}
	if err := <-action; err != nil {
		return err
	}
	if err := nativeRecoveryReadSmoke(ctx, a, s); err != nil {
		return err
	}
	before := s.service.Snapshot()
	if err := a.PostWait(ctx, func() { action <- smokeUI(s, "recovery-dismiss", nil) }); err != nil {
		return err
	}
	if err := <-action; err != nil {
		return err
	}
	after, err := s.service.Wait(ctx)
	if err != nil {
		return err
	}
	work := after.ProjectWork
	if work == nil || work.Command != "recovery.dismiss" || work.Status != "completed" || work.Actor != "user" || work.RecoveryID != id || work.RecoveryDismissed == nil || !*work.RecoveryDismissed || work.SourceSHA256 == "" {
		return fmt.Errorf("native dismissal lost exact result/actor")
	}
	if !reflect.DeepEqual(before.Components, after.Components) || before.Document != after.Document || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Mixed, after.Mixed) || !reflect.DeepEqual(before.NPN, after.NPN) || !reflect.DeepEqual(before.Result, after.Result) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		return fmt.Errorf("native dismissal changes live circuit or history")
	}
	if _, err := os.Stat(filepath.Join(after.Recovery.Directory, id+".elab-recovery")); !os.IsNotExist(err) {
		return fmt.Errorf("native dismissal did not remove confirmed snapshot: %v", err)
	}
	return nil
}
