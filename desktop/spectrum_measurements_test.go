package main

import (
	"math"
	"reflect"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/ui"
)

func prepareSpectrumMeasurements(t *testing.T, s *screen) {
	t.Helper()
	prepareSpectrumDock(t, s.service)
	agentAction(t, s.service, "scope.view.set", workbench.ScopeViewParams{View: "spectrum"})
	awaitSpectrumDock(t, s.service)
	p := s.service.Snapshot().Spectrum.Live.Settings
	p.Hold = true
	agentAction(t, s.service, "spectrum.live.set", p)
	s.view()
}
func TestSpectrumMeasurementUIAgentParity(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	prepareSpectrumMeasurements(t, s)
	a := workbench.New()
	defer a.Close()
	prepareSpectrumDock(t, a)
	agentAction(t, a, "scope.view.set", workbench.ScopeViewParams{View: "spectrum"})
	live := awaitSpectrumDock(t, a)
	p := live.Settings
	p.Hold = true
	agentAction(t, a, "spectrum.live.set", p)
	before := s.service.Snapshot()
	uiAction(t, s, "spectrum-cursor-a", &ui.ChangeEvent{Text: "100 Hz"})
	agentAction(t, a, "spectrum.cursor.set", map[string]any{"source": "live", "cursor": "a", "frequency_hz": "100 Hz"})
	uiAction(t, s, "spectrum-cursor-select-b", nil)
	agentAction(t, a, "spectrum.cursor.set", workbench.SpectrumCursorParams{Source: "live", Cursor: "b", Select: true})
	uiAction(t, s, "spectrum-cursor-peak", nil)
	agentAction(t, a, "spectrum.cursor.set", workbench.SpectrumCursorParams{Source: "live", Cursor: "b", Peak: 1})
	uiAction(t, s, "spectrum-peaks", nil)
	agentAction(t, a, "spectrum.markers.set", workbench.ScopeVisibilityParams{Visible: false})
	u, v := s.service.Snapshot().Spectrum.Live.Measurements, a.Snapshot().Spectrum.Live.Measurements
	if !reflect.DeepEqual(u, v) || u.Visible || u.A == nil || u.B == nil {
		t.Fatal("user/agent measurement mismatch", u, v)
	}
	if after := s.service.Snapshot(); !reflect.DeepEqual(before.Transient, after.Transient) || before.Document.Dirty != after.Document.Dirty {
		t.Fatal("UI measurements changed physics")
	}
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
		if err := ui.Validate(s.view()); err != nil {
			t.Fatal(err)
		}
		if mesh, err := s.spectrumMesh(canvas.Viewport{Width: 800, Height: 160, Zoom: 1}); err != nil || mesh == nil {
			t.Fatal("cursor mesh", err)
		}
	}
	uiAction(t, s, "spectrum-cursor-clear-a", nil)
	if s.service.Snapshot().Spectrum.Live.Measurements.A != nil {
		t.Fatal("clear UI")
	}
	// The same controls target an independently captured recorded result.
	duration, step := .05, .001
	agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:C1"}})
	deadline := time.Now().Add(3 * time.Second)
	for s.service.Snapshot().Verification == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.service.Snapshot().Verification == nil {
		t.Fatal("recording not ready")
	}
	uiAction(t, s, "spectrum-source-recorded", nil)
	uiAction(t, s, "spectrum-dock-analyze", nil)
	uiAction(t, s, "spectrum-cursor-b", &ui.ChangeEvent{Text: "0 Hz"})
	if m := s.service.Snapshot().Spectrum.Measurements; m.B == nil || !m.B.Valid || m.B.Reading.Frequency != 0 {
		t.Fatal("recorded control target", m)
	}
}
func TestSpectrumCursorPointerOwnWritesAndRetiredFrames(t *testing.T) {
	for _, change := range []string{"none", "probe", "tab", "document", "agent-cursor", "modal"} {
		t.Run(change, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			prepareSpectrumMeasurements(t, s)
			paint := func() *spectrumPlotFrame {
				t.Helper()
				if _, err := s.spectrumMesh(canvas.Viewport{Width: 800, Height: 160, Zoom: 1}); err != nil {
					t.Fatal(err)
				}
				return s.spectrumDrawing.frame.Load()
			}
			f := paint()
			c := s.spectrumDrawing
			c.pointer(f, events.MouseDown, events.Left, f.left)
			if m := s.service.Snapshot().Spectrum.Live.Measurements; m.A == nil || m.A.Reading.Frequency != 0 {
				t.Fatal("pointer down not accepted")
			}
			// A fresh paint contains this gesture's own command revision. SlideStop and
			// quick MouseUp must both commit the release even without any SlideMove.
			f = paint()
			c.pointer(f, events.SlideStop, events.Left, (f.left+f.right)/2)
			if m := s.service.Snapshot().Spectrum.Live.Measurements; m.A == nil || m.A.Requested != f.nyquist/2 {
				t.Fatal("release lost")
			}
			f = paint()
			c.pointer(f, events.MouseDown, events.Left, f.left)
			f = paint()
			before := s.service.Snapshot().CommandRevision
			switch change {
			case "probe":
				p := s.service.Snapshot().Spectrum.Live.Settings
				p.ProbeID = "i:R1"
				agentAction(t, s.service, "spectrum.live.set", p)
			case "tab":
				agentAction(t, s.service, "scope.view.set", workbench.ScopeViewParams{View: "scope"})
			case "document":
				agentAction(t, s.service, "project.clear", map[string]bool{"discard_changes": true})
			case "agent-cursor":
				agentAction(t, s.service, "spectrum.cursor.set", workbench.SpectrumCursorParams{Source: "live", Cursor: "b", Select: true})
			case "modal":
				s.fileBusy = true
			}
			changed := s.service.Snapshot()
			c.pointer(f, events.MouseUp, events.Left, f.right)
			after := s.service.Snapshot()
			if change == "none" {
				if after.CommandRevision != before+1 || after.Spectrum.Live.Measurements.A.Requested != f.nyquist {
					t.Fatal("quick drag release")
				}
			} else if changed.CommandRevision != after.CommandRevision {
				t.Fatal("retired pointer changed service")
			}
			if c.gesture != nil {
				t.Fatal("gesture retained after release")
			}
		})
	}
}
func TestSpectrumCursorPointerBoundsAndAmplitudeScale(t *testing.T) {
	for _, tt := range []struct{ db, low, high float64 }{{-180, -280, -180}, {-17, -100, 0}, {5, -80, 20}, {-300, -100, 0}} {
		lo, hi := spectrumAmplitudeRange(&workbench.SpectrumResult{Bins: []workbench.SpectrumBin{{DB: tt.db}}})
		if lo != tt.low || hi != tt.high {
			t.Fatal("amplitude scale", tt, lo, hi)
		}
	}
	s, cleanup := testScreen(t)
	defer cleanup()
	prepareSpectrumMeasurements(t, s)
	s.spectrumMesh(canvas.Viewport{Width: 800, Height: 160, Zoom: 1})
	c := s.spectrumDrawing
	f := c.frame.Load()
	before := s.service.Snapshot().CommandRevision
	for _, x := range []float64{math.NaN(), math.Inf(1)} {
		c.pointer(f, events.MouseDown, events.Left, x)
	}
	c.pointer(f, events.MouseDown, events.Right, 100)
	if before != s.service.Snapshot().CommandRevision {
		t.Fatal("invalid/right mouse mutated cursors")
	}
	c.pointer(f, events.MouseDown, events.Left, -100)
	if m := s.service.Snapshot().Spectrum.Live.Measurements; m.A.Requested != 0 {
		t.Fatal("left clamp")
	}
	s.spectrumMesh(canvas.Viewport{Width: 800, Height: 160, Zoom: 1})
	c.pointer(c.frame.Load(), events.MouseUp, events.Left, 1e9)
	if m := s.service.Snapshot().Spectrum.Live.Measurements; m.A.Requested != f.nyquist {
		t.Fatal("right clamp")
	}
	s.spectrumMesh(canvas.Viewport{Width: 10, Height: 10, Zoom: 1})
	if c.frame.Load() != nil {
		t.Fatal("invalid preparation retained input context")
	}
}

func TestSpectrumMeasurementActionBridgesOnlyOwnFrequencyCommit(t *testing.T) {
	for _, agent := range []bool{false, true} {
		t.Run(map[bool]string{false: "own-focus-loss", true: "agent-interleaving"}[agent], func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			prepareSpectrumMeasurements(t, s)
			var field, clear ui.Node
			s.app.Do(func() {
				view := s.view()
				field, _ = findNode(view, "spectrum-cursor-a")
				clear, _ = findNode(view, "spectrum-cursor-clear-a")
				field.Change(ui.ChangeEvent{Text: "100 Hz"})
			})
			if agent {
				agentAction(t, s.service, "spectrum.cursor.set", workbench.SpectrumCursorParams{Source: "live", Cursor: "b", Select: true})
			}
			before := s.service.Snapshot().CommandRevision
			s.app.Do(clear.Activate)
			after := s.service.Snapshot()
			if agent {
				if after.CommandRevision != before || after.Spectrum.Live.Measurements.A == nil {
					t.Fatal("action borrowed newer agent authority")
				}
			} else if after.CommandRevision != before+1 || after.Spectrum.Live.Measurements.A != nil || s.errorMessage != "" {
				t.Fatal("own focus-loss continuation refused", s.errorMessage)
			}
		})
	}
}
