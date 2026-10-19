package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
)

func waitNativeFile(ctx context.Context, a *app.Application, s *screen) error {
	for {
		completed := make(chan bool, 1)
		if err := a.PostWait(ctx, func() { completed <- !s.fileBusy }); err != nil {
			return err
		}
		if <-completed {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}
func nativeExportSmoke(ctx context.Context, a *app.Application, s *screen) error {
	directory, err := os.MkdirTemp("", "elab-export-smoke-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	defer func() { _ = a.PostWait(ctx, func() { s.chooseFile = nil }) }()
	action := make(chan error, 1)
	if err := a.PostWait(ctx, func() { action <- smokeUI(s, "transient-step", nil) }); err != nil {
		return err
	}
	if err := <-action; err != nil {
		return err
	}
	for _, control := range []string{"export-svg", "export-csv", "verify-export-json", "verify-export-csv"} {
		if control == "verify-export-json" {
			if err := a.PostWait(ctx, func() { action <- smokeUI(s, "verify-start", nil) }); err != nil {
				return err
			}
			if err := <-action; err != nil {
				return err
			}
			id := s.service.Snapshot().LastOperation
			for {
				raw := []byte(fmt.Sprintf(`{"id":%q}`, id))
				response := s.service.Dispatch(ctx, workbench.Request{Version: 1, ID: "smoke-inspect", Command: "operation.inspect", Params: raw}, "smoke")
				if response.Error != nil {
					return fmt.Errorf("native export Verify: %s", response.Error.Message)
				}
				status := response.Result.(workbench.OperationView).Operation.Status
				if status == "completed" {
					break
				}
				if status != "queued" && status != "running" && status != "canceling" {
					return fmt.Errorf("native export Verify: %s", status)
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(10 * time.Millisecond):
				}
			}
		}
		path := filepath.Join(directory, control)
		if err := a.PostWait(ctx, func() {
			s.chooseFile = func(context.Context, app.FileOptions) (string, error) { return path, nil }
			action <- smokeUI(s, control, nil)
		}); err != nil {
			return err
		}
		if err := <-action; err != nil {
			return err
		}
		if err := waitNativeFile(ctx, a, s); err != nil {
			return err
		}
		checked := make(chan error, 1)
		if err := a.PostWait(ctx, func() {
			state := s.service.Snapshot()
			if s.errorMessage != "" || state.ProjectWork == nil || state.ProjectWork.Status != "completed" || state.ProjectWork.Actor != "user" || state.LastExport == nil || state.LastExport.Path != path || state.LastExport.SHA256 == "" {
				checked <- fmt.Errorf("native export failed: %s / %+v", s.errorMessage, state.ProjectWork)
				return
			}
			_, err := s.mesh(*s.drawing.View)
			checked <- err
		}); err != nil {
			return err
		}
		if err := <-checked; err != nil {
			return err
		}
		if data, err := os.ReadFile(path); err != nil || len(data) == 0 {
			return fmt.Errorf("native export file missing: %v", err)
		}
	}
	return nativeImportedExportSmoke(ctx, a, s, filepath.Join(directory, "verify-export-json"))
}

func nativeImportedExportSmoke(ctx context.Context, a *app.Application, s *screen, path string) error {
	before := s.service.Snapshot()
	action := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		s.chooseFile = func(context.Context, app.FileOptions) (string, error) { return path, nil }
		action <- smokeUI(s, "recording-import", nil)
	}); err != nil {
		return err
	}
	if err := <-action; err != nil {
		return err
	}
	if err := waitNativeFile(ctx, a, s); err != nil {
		return err
	}
	checked := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		state := s.service.Snapshot()
		if s.errorMessage != "" || state.ProjectWork == nil || state.ProjectWork.Status != "completed" || state.ProjectWork.Command != "verification.import" || state.ProjectWork.Actor != "user" || state.ProjectWork.SourceSHA256 == "" || state.Recording == nil || !state.Recording.Imported || state.Recording.ID != state.ProjectWork.ResultOperationID || state.Transient.Sample != before.Transient.Sample || state.RunID != before.RunID {
			checked <- fmt.Errorf("native recording import failed: %s / %+v", s.errorMessage, state.ProjectWork)
			return
		}
		_, err := s.mesh(*s.drawing.View)
		checked <- err
	}); err != nil {
		return err
	}
	return <-checked
}
