package main

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeIntegrationLimitSmoke(ctx context.Context, a *app.Application, s *screen) error {
	done := make(chan error, 1)
	var before workbench.Snapshot
	if err := a.PostWait(ctx, func() {
		if err := smokeConstructWaveform(s, "finite-pulse-source"); err != nil {
			done <- err
			return
		}
		if err := smokeUI(s, "simulation.pause", nil); err != nil {
			done <- err
			return
		}
		source := s.service.Snapshot().Selected
		cfg := workbench.DefaultTransientSettings()
		cfg.StepSeconds = .2
		for _, request := range []struct {
			command string
			params  any
		}{
			{"object.delete", workbench.IDParams{ID: "C1"}},
			{"wire.connect", workbench.ConnectParams{From: "R1.2", To: "GND.reference"}},
			{"source.pulse.set", workbench.PulseParams{ID: source, Waveform: workbench.PulseWaveform{HighVoltage: 5, HighSeconds: .0005, LowSeconds: .0005, RiseSeconds: .0001, FallSeconds: .0001}}},
			{"simulation.transient.set", cfg},
		} {
			if r := s.invoke(request.command, request.params); r.Error != nil {
				done <- fmt.Errorf("integration limit setup: %s", r.Error.Message)
				return
			}
		}
		if err := smokeUI(s, "simulation.restart", nil); err != nil {
			done <- err
			return
		}
		before = s.service.Snapshot()
		node, ok := findAction(s.view(), "transient-step")
		if !ok || node.Disabled {
			done <- fmt.Errorf("coarse native step unavailable")
			return
		}
		node.Activate()
		done <- nil
	}); err != nil {
		return err
	}
	if err := <-done; err != nil {
		return err
	}
	after, err := s.service.Wait(ctx)
	if err != nil {
		return err
	}
	if after.StepWork == nil || after.StepWork.Error == nil || after.StepWork.Error.Code != "integration-attempt-limit" || after.StepWork.Status != "failed" || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Mixed, after.Mixed) || before.Document != after.Document {
		return fmt.Errorf("native budget rejection lost reason or changed physics")
	}
	if err := a.PostWait(ctx, func() {
		root := s.view()
		summary, ok := findNode(root, "diagnostic-0-summary")
		if !ok || !strings.Contains(summary.Text, "calculation attempts") {
			done <- fmt.Errorf("native budget guidance missing")
			return
		}
		if err := smokeUI(s, "diagnostic-details", &ui.ChangeEvent{Checked: true}); err != nil {
			done <- err
			return
		}
		if _, err := s.mesh(*s.drawing.View); err != nil {
			done <- err
			return
		}
		cfg := workbench.DefaultTransientSettings()
		cfg.StepSeconds = .001
		if r := s.invoke("simulation.transient.set", cfg); r.Error != nil {
			done <- fmt.Errorf("native budget recovery settings: %s", r.Error.Message)
			return
		}
		for _, id := range []string{"simulation.restart", "transient-step"} {
			if err := smokeUI(s, id, nil); err != nil {
				done <- err
				return
			}
		}
		state := s.service.Snapshot()
		if state.StepWork.Status != "completed" || state.Transient.Sample.Time != .001 || state.Transient.Stale {
			done <- fmt.Errorf("native smaller-step recovery failed")
			return
		}
		for _, help := range state.DiagnosticPresentation.Items {
			if help.Code == "integration-attempt-limit" {
				done <- fmt.Errorf("native recovery retains old budget guidance")
				return
			}
		}
		done <- nil
	}); err != nil {
		return err
	}
	return <-done
}
