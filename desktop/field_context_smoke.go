package main

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeFieldContextSmoke(ctx context.Context, a *app.Application, s *screen) error {
	done := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		for _, request := range []struct {
			command string
			params  any
		}{
			{"project.clear", workbench.ReplaceParams{DiscardChanges: true}},
			{"component.place", workbench.PlaceParams{ID: "VFIELD", Kind: "sine-source"}},
		} {
			if r := s.invoke(request.command, request.params); r.Error != nil {
				done <- fmt.Errorf("field setup: %s", r.Error.Message)
				return
			}
		}
		captured, ok := findNode(s.view(), "sine-frequency")
		if !ok || captured.Change == nil {
			done <- fmt.Errorf("native field callback missing")
			return
		}
		waveform := workbench.DefaultSineWaveform()
		waveform.Amplitude = 1
		raw, _ := json.Marshal(workbench.SineParams{ID: "VFIELD", Waveform: waveform})
		revision := s.service.Snapshot().CommandRevision
		r := s.service.Dispatch(ctx, workbench.Request{Version: 1, ID: "field-agent-edit", Key: "field-agent-edit", Command: "source.sine.set", Params: raw, ExpectedCommandRevision: &revision}, "agent.smoke")
		if r.Error != nil {
			done <- fmt.Errorf("field agent edit: %s", r.Error.Message)
			return
		}
		before := s.service.Snapshot()
		captured.Change(ui.ChangeEvent{Text: "20"})
		if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) || s.fieldContext != nil {
			done <- fmt.Errorf("native stale field changed newer agent values")
			return
		}
		if err := smokeUI(s, "sine-frequency", &ui.ChangeEvent{Text: "20"}); err != nil {
			done <- err
			return
		}
		state := s.service.Snapshot()
		if state.Components[0].Sine.Amplitude != 1 || state.Components[0].Sine.Frequency != 20 {
			done <- fmt.Errorf("native fresh field lost newer waveform")
			return
		}
		done <- nil
	}); err != nil {
		return err
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
