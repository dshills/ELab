package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dshills/vellum/app"
)

// Exercise file controls on the app actor, leaving reads/validation and waits
// outside it. The picker adapter selects only a temporary owned fixture.
func nativeProjectOpenSmoke(ctx context.Context, a *app.Application, s *screen) error {
	directory, err := os.MkdirTemp("", "elab-open-smoke-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	path := filepath.Join(directory, "Transient.elab")
	if err := a.PostWait(ctx, func() {
		s.chooseFile = func(context.Context, app.FileOptions) (string, error) { return path, nil }
	}); err != nil {
		return err
	}
	defer func() { _ = a.PostWait(ctx, func() { s.chooseFile = nil }) }()
	for _, control := range []string{"file.save-as", "file.open"} {
		action := make(chan error, 1)
		if err := a.PostWait(ctx, func() { action <- smokeUI(s, control, nil) }); err != nil {
			return err
		}
		if err := <-action; err != nil {
			return err
		}
		for {
			completed := make(chan bool, 1)
			if err := a.PostWait(ctx, func() { completed <- !s.fileBusy }); err != nil {
				return err
			}
			if <-completed {
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Millisecond):
			}
		}
	}
	checked := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		state := s.service.Snapshot()
		if s.errorMessage != "" || state.ProjectWork == nil || state.ProjectWork.Status != "completed" || state.ProjectWork.Actor != "user" || state.Document.Path != path || state.Document.Dirty {
			checked <- fmt.Errorf("native project open failed: %s / %+v", s.errorMessage, state.ProjectWork)
			return
		}
		if _, ok := findNode(s.view(), "project-work-status"); !ok {
			checked <- fmt.Errorf("native project open status missing")
			return
		}
		_, err := s.mesh(*s.drawing.View)
		checked <- err
	}); err != nil {
		return err
	}
	return <-checked
}
