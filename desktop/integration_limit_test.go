package main

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func TestNativeStepShowsIntegrationAttemptGuidanceAndPreservesPhysics(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := smokeConstructWaveform(s, "finite-pulse-source"); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "simulation.pause", nil)
	source := s.service.Snapshot().Selected
	agentAction(t, s.service, "object.delete", workbench.IDParams{ID: "C1"})
	agentAction(t, s.service, "wire.connect", workbench.ConnectParams{From: "R1.2", To: "GND.reference"})
	agentAction(t, s.service, "source.pulse.set", workbench.PulseParams{ID: source, Waveform: workbench.PulseWaveform{HighVoltage: 5, HighSeconds: .0005, LowSeconds: .0005, RiseSeconds: .0001, FallSeconds: .0001}})
	cfg := workbench.DefaultTransientSettings()
	cfg.StepSeconds = .2
	agentAction(t, s.service, "simulation.transient.set", cfg)
	uiAction(t, s, "simulation.restart", nil)
	before := s.service.Snapshot()
	s.app.Do(func() {
		node, ok := findAction(s.view(), "transient-step")
		if !ok || node.Disabled {
			t.Fatal("coarse native step unavailable")
		}
		node.Activate()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	after, err := s.service.Wait(ctx)
	if err != nil || after.StepWork == nil || after.StepWork.Error == nil || after.StepWork.Error.Code != "integration-attempt-limit" || after.StepWork.Status != "failed" || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Mixed, after.Mixed) || before.Document != after.Document {
		t.Fatal("native attempt limit changed physics or lost reason", err, after.StepWork)
	}
	s.app.Do(func() {
		root := s.view()
		summary, ok := findNode(root, "diagnostic-0-summary")
		if !ok || !strings.Contains(summary.Text, "calculation attempts") {
			t.Fatal("native failure lacks readable attempt reason", summary.Text)
		}
		action, ok := findNode(root, "diagnostic-0-action")
		if !ok || !strings.Contains(action.Text, "sample interval") || !strings.Contains(action.Text, "preserved") {
			t.Fatal("native failure lacks safe next action", action.Text)
		}
	})
	cfg.StepSeconds = .001
	agentAction(t, s.service, "simulation.transient.set", cfg)
	uiAction(t, s, "simulation.restart", nil)
	uiAction(t, s, "transient-step", nil)
	recovered := s.service.Snapshot()
	if recovered.StepWork.Status != "completed" || recovered.Transient.Sample.Time != .001 || recovered.Transient.Stale {
		t.Fatal("smaller native sample interval cannot recover", recovered.StepWork)
	}
	for _, help := range recovered.DiagnosticPresentation.Items {
		if help.Code == "integration-attempt-limit" {
			t.Fatal("successful native retry retains obsolete budget failure")
		}
	}
}
