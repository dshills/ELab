package main

import (
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/ui"
)

func awaitSpectrumDock(t *testing.T, s *workbench.Service) *workbench.LiveSpectrumState {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		v := s.Snapshot().Spectrum.Live
		if v != nil && v.Result != nil {
			return v
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("live FFT not ready", s.Snapshot().Spectrum.Live)
	return nil
}
func prepareSpectrumDock(t *testing.T, s *workbench.Service) {
	t.Helper()
	agentAction(t, s, "example.load", workbench.ExampleLoadParams{ID: "rc"})
	agentAction(t, s, "simulation.pause", struct{}{})
	agentAction(t, s, "simulation.restart", struct{}{})
	if state, err := s.Wait(t.Context()); err != nil || state.Transient == nil || state.Transient.Stale {
		t.Fatal("RC fixture did not initialize", err, state.Status)
	}
	agentAction(t, s, "scope.probes.set", workbench.ProbeParams{IDs: []string{"v:C1", "i:R1"}})
	agentAction(t, s, "simulation.step", workbench.StepParams{Duration: .1})
}
func TestSpectrumDockTabsLiveAndRecordedShareCommands(t *testing.T) {
	u, cleanup := testScreen(t)
	defer cleanup()
	a := workbench.New()
	defer a.Close()
	for _, s := range []*workbench.Service{u.service, a} {
		prepareSpectrumDock(t, s)
	}
	before := u.service.Snapshot().Transient
	uiAction(t, u, "scope-view-spectrum", nil)
	agentAction(t, a, "scope.view.set", workbench.ScopeViewParams{View: "spectrum"})
	user, agent := awaitSpectrumDock(t, u.service), awaitSpectrumDock(t, a)
	if !reflect.DeepEqual(user.Result.Bins, agent.Result.Bins) || !reflect.DeepEqual(before, u.service.Snapshot().Transient) {
		t.Fatal("tab/live command parity or physics changed")
	}
	uiAction(t, u, "spectrum-dock-probe", &ui.ChangeEvent{SelectedID: "i:R1"})
	cfg := a.Snapshot().Spectrum.Live.Settings
	cfg.ProbeID = "i:R1"
	agentAction(t, a, "spectrum.live.set", cfg)
	user, agent = awaitSpectrumDock(t, u.service), awaitSpectrumDock(t, a)
	if !reflect.DeepEqual(user.Result.Bins, agent.Result.Bins) || user.Result.Probe.Unit != "A" {
		t.Fatal("shared current probe")
	}
	uiAction(t, u, "spectrum-live-hold", nil)
	cfg.Hold = true
	agentAction(t, a, "spectrum.live.set", cfg)
	for _, s := range []*workbench.Service{u.service, a} {
		agentAction(t, s, "simulation.step", workbench.StepParams{Duration: .01})
	}
	if got := u.service.Snapshot().Spectrum.Live; got.Status != "held" || !reflect.DeepEqual(got.Result, user.Result) {
		t.Fatal("hold parity")
	}
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		agentAction(t, u.service, "appearance.theme.set", map[string]string{"theme": theme})
		if err := ui.Validate(u.view()); err != nil {
			t.Fatal(err)
		}
		mesh, err := u.spectrumMesh(canvas.Viewport{Width: 800, Height: 160, Zoom: 1})
		if err != nil || mesh == nil {
			t.Fatal("live FFT mesh", err)
		}
	}
	duration, step := .05, .001
	agentAction(t, u.service, "simulation.verify", workbench.VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:C1"}})
	deadline := time.Now().Add(3 * time.Second)
	for u.service.Snapshot().Verification == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if u.service.Snapshot().Verification == nil {
		t.Fatal("recording not completed")
	}
	uiAction(t, u, "spectrum-source-recorded", nil)
	uiAction(t, u, "spectrum-dock-analyze", nil)
	snap := u.service.Snapshot()
	if snap.Spectrum.Result == nil || snap.Spectrum.Live.Settings.Enabled {
		t.Fatal("existing recorded analyser not reachable")
	}
	if err := ui.Validate(u.view()); err != nil {
		t.Fatal(err)
	}
	uiAction(t, u, "scope-view-scope", nil)
	if _, ok := findNode(u.view(), "rc-scope"); !ok {
		t.Fatal("scope tab lost live scope")
	}
}

func TestSpectrumTabRetiresScopeCursorInput(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	prepareSpectrumDock(t, s.service)
	if _, err := s.scopeMesh(canvas.Viewport{Width: 800, Height: 150, Zoom: 1}); err != nil {
		t.Fatal(err)
	}
	frame := s.scope.frame.Load()
	if frame == nil {
		t.Fatal("scope frame unavailable")
	}
	uiAction(t, s, "scope-view-spectrum", nil)
	before := s.service.Snapshot()
	// A retained custom-canvas callback cannot move hidden scope cursors or
	// create a conflict notice after the user switches the shared view command.
	s.scope.applyCursorFrame(frame, (frame.left+frame.right)/2, before.CommandRevision)
	after := s.service.Snapshot()
	if before.CommandRevision != after.CommandRevision || after.Scope.A != nil || s.errorMessage != "" {
		t.Fatal("hidden scope accepted cursor input")
	}
}
