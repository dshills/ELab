//go:build offscreen

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"io/fs"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/text/fonts"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/execution"
	"github.com/dshills/vellum/ui"
)

// Use the pinned driver's real retained widgets/event router without an OS
// window or a GPU. No SDK source or descriptor callback is replaced by the test.
type mountedControlWindow struct {
	t      *testing.T
	ctx    context.Context
	screen *screen
	manual *cogent.Manual
	clock  *execution.ManualClock
	window cogent.ManualWindowSnapshot
}

type mountedWiper struct {
	*mountedControlWindow
	from, to image.Point
}

func newMountedWiper(t *testing.T, s *screen) *mountedWiper {
	t.Helper()
	id := ""
	for _, p := range s.service.Snapshot().Components {
		if p.Kind == "potentiometer" {
			id = p.ID
		}
	}
	if id == "" {
		t.Fatal("pot fixture missing")
	}
	view := func() ui.Node {
		state := s.service.Snapshot()
		var slider ui.Node
		for _, p := range state.Components {
			if p.ID == id {
				for _, node := range s.potentiometerControls(state, p) {
					if node.ID == "pot-wiper" {
						slider = node
					}
				}
			}
		}
		slider.MinWidth, slider.MinHeight = 360, 40
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: "pot-wiper", Sequence: 1}, Children: []ui.Node{
			slider,
			{ID: "focus-target", Kind: ui.Button, Text: "Continue", Activate: func() {}},
		}}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	}
	m := &mountedWiper{mountedControlWindow: newMountedControlWindow(t, s, view)}
	var bounds image.Rectangle
	for _, e := range m.window.Elements {
		if e.ID == "pot-wiper" {
			bounds = e.Bounds
		}
	}
	if bounds.Empty() {
		t.Fatal("slider has no measured geometry")
	}
	m.from = bounds.Min.Add(bounds.Size().Div(2))
	m.to = m.from.Add(image.Pt(bounds.Dx()/4, 0))
	return m
}

func newMountedControlWindow(t *testing.T, s *screen, view func() ui.Node) *mountedControlWindow {
	t.Helper()
	theme := ui.Theme{Default: ui.Style{Foreground: ui.Some(color.NRGBA{R: 24, G: 24, B: 24, A: 255}), Background: ui.Some(color.NRGBA{R: 248, G: 248, B: 248, A: 255}), Typography: ui.Some(ui.Typography{Family: "Noto Sans", Size: 16})}}
	return newMountedControlWindowSize(t, s, view, 520, 180, theme)
}

func newMountedControlWindowSize(t *testing.T, s *screen, view func() ui.Node, width, height float32, theme ui.Theme, configure ...func(*cogent.ManualConfig)) *mountedControlWindow {
	t.Helper()
	// The offscreen system selects LinuxStandard at package initialization.
	// ELab's macOS fixtures must exercise actual Command modifier mappings.
	previousMap, previousMapName := keymap.ActiveMap, keymap.ActiveMapName
	t.Cleanup(func() { keymap.SetActiveMap(previousMap, previousMapName) })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	if _, err := s.service.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.app.Open(app.Window{ID: "main", Title: "Mounted controls", Width: width, Height: height, View: view}); err != nil {
		t.Fatal(err)
	}
	clock, err := execution.NewManualClock(time.Unix(1730000000, 0).UTC(), 256, 256)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(clock.Close)
	executor, err := execution.NewManual(8)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(executor.Close)
	config := cogent.ManualConfig{Clock: clock, Executor: executor, LogicalDPI: 160, OwnerBatch: 7, EventCapacity: 128, MaxWindows: 1}
	config.Theme = theme
	for _, apply := range configure {
		apply(&config)
	}
	if err := fs.WalkDir(fonts.Default, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".ttf") && !strings.HasSuffix(path, ".otf") {
			return nil
		}
		data, err := fs.ReadFile(fonts.Default, path)
		if err != nil {
			return err
		}
		h := sha256.Sum256(data)
		config.Fonts = append(config.Fonts, cogent.ManualFont{Name: path, Data: data, SHA256: hex.EncodeToString(h[:])})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	manual, err := cogent.NewManual(ctx, s.app, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), time.Second)
		defer closeCancel()
		if err := manual.Close(closeCtx); err != nil {
			t.Error(err)
		}
	})
	m := &mountedControlWindow{t: t, ctx: ctx, screen: s, manual: manual, clock: clock}
	m.settle()
	m.window = m.snapshot()
	keymap.SetActiveMapName("MacStandard")
	return m
}

func (m *mountedControlWindow) snapshot() cogent.ManualWindowSnapshot {
	m.t.Helper()
	snapshot, err := m.manual.Snapshot(m.ctx)
	if err != nil {
		m.t.Fatal(err)
	}
	if len(snapshot.Windows) != 1 {
		m.t.Fatal("manual window missing")
	}
	return snapshot.Windows[0]
}

func (m *mountedControlWindow) settle() {
	m.t.Helper()
	for range 128 {
		snapshot, err := m.manual.Snapshot(m.ctx)
		if err != nil {
			m.t.Fatal(err)
		}
		ready := snapshot.Application.Ordinary+snapshot.Application.ReadyCompletions > 0
		layout := false
		for _, w := range snapshot.Windows {
			ready = ready || w.Events > 0 || w.ReadyPaint
			layout = layout || w.LayoutPending
		}
		if ready {
			if _, err := m.manual.Step(m.ctx, 7); err != nil {
				m.t.Fatal(err)
			}
			continue
		}
		if layout {
			if _, err := m.manual.Frame(m.ctx, "main"); err != nil {
				m.t.Fatal(err)
			}
			continue
		}
		return
	}
	m.t.Fatal("manual settle bound exhausted")
}

func (m *mountedControlWindow) frame() {
	m.t.Helper()
	if err := m.screen.app.Invalidate("main"); err != nil {
		m.t.Fatal(err)
	}
	if _, err := m.manual.Frame(m.ctx, "main"); err != nil {
		m.t.Fatal(err)
	}
	m.settle()
}

func (m *mountedControlWindow) input(input cogent.ManualInput) {
	m.t.Helper()
	input.Window, input.WindowIncarnation = "main", m.window.Incarnation
	if err := m.manual.Input(m.ctx, input); err != nil {
		m.t.Fatal(err)
	}
	m.settle()
}

func (m *mountedWiper) beginDrag() {
	m.t.Helper()
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: m.from})
	if _, err := m.clock.Advance(core.TimingSettings.SlideStartTime + time.Millisecond); err != nil {
		m.t.Fatal(err)
	}
	m.settle()
	for _, at := range []image.Point{m.from.Add(m.to.Sub(m.from).Div(2)), m.to} {
		m.input(cogent.ManualInput{Type: events.MouseMove, Position: at})
	}
}

func (m *mountedWiper) release() {
	m.t.Helper()
	m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: m.to})
}

func TestMountedSliderDoesNotBorrowAgentContextDuringHeldDrag(t *testing.T) {
	for _, name := range []string{"uninterrupted", "unchanged-repaint", "agent-repaint", "agent-document-replacement"} {
		t.Run(name, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			if err := smokeConstructPotentiometer(s); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "held-wiper.elab")
			if name == "agent-document-replacement" {
				agentAction(t, s.service, "project.save", workbench.SaveParams{Path: path})
			}
			m := newMountedWiper(t, s)
			before := s.service.Snapshot()
			m.beginDrag()
			if s.service.Snapshot().CommandRevision != before.CommandRevision {
				t.Fatal("drag committed before release")
			}
			intervening := name == "agent-repaint" || name == "agent-document-replacement"
			if intervening {
				if name == "agent-document-replacement" {
					agentAction(t, s.service, "project.load", workbench.LoadParams{Path: path, DiscardChanges: true})
				} else {
					id := before.Mixed.Potentiometers[0].ID
					agentAction(t, s.service, "potentiometer.wiper.set", workbench.WiperParams{ID: id, Position: .2})
				}
				next := s.service.Snapshot()
				if name == "agent-document-replacement" && next.Document.Generation == before.Document.Generation {
					t.Fatal("fixture did not replace the document")
				}
				before = next
			}
			if name != "uninterrupted" {
				m.frame() // rebind native callbacks while the mouse is held
			}
			m.release()
			after := s.service.Snapshot()
			if intervening {
				if !reflect.DeepEqual(before, after) {
					t.Fatal("held drag overwrote agent after repaint")
				}
			} else if after.CommandRevision != before.CommandRevision+1 || after.Mixed.Potentiometers[0].WiperPosition <= .5 || after.UndoEntry.Actor != "user" {
				t.Fatal("uninterrupted control drag did not commit once", after.Mixed.Potentiometers)
			}
		})
	}
}

func TestMountedSliderKeyboardFocusChangePreservesContext(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "replacement"}[replace], func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			if err := smokeConstructPotentiometer(s); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "held-focus.elab")
			agentAction(t, s.service, "project.save", workbench.SaveParams{Path: path})
			m := newMountedWiper(t, s)
			before := s.service.Snapshot()
			m.beginDrag()
			if m.snapshot().FocusedID != "pot-wiper" {
				t.Fatal("fixture did not focus held slider")
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeTab})
			if m.snapshot().FocusedID != "focus-target" {
				t.Fatal("Tab did not move keyboard focus away from held slider")
			}
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("keyboard focus change committed an unfinished drag")
			}
			if replace {
				agentAction(t, s.service, "project.load", workbench.LoadParams{Path: path, DiscardChanges: true})
				before = s.service.Snapshot()
				m.frame()
			}
			m.release()
			after := s.service.Snapshot()
			if replace {
				if !reflect.DeepEqual(before, after) {
					t.Fatal("keyboard focus loss let an old drag mutate a replacement document")
				}
			} else if after.CommandRevision != before.CommandRevision+1 || after.Mixed.Potentiometers[0].WiperPosition <= .5 || after.UndoEntry.Actor != "user" {
				t.Fatal("normal held drag lost or duplicated its commit after Tab")
			}
		})
	}
}

func TestMountedSliderRunningChargeAndAgentParity(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	if err := smokeConstructPotRC(s); err != nil {
		t.Fatal(err)
	}
	agentAction(t, s.service, "simulation.input.set", workbench.InputParams{Mode: "high"})
	path := filepath.Join(t.TempDir(), "running-pot.elab")
	agentAction(t, s.service, "project.save", workbench.SaveParams{Path: path})
	agentAction(t, s.service, "simulation.resume", struct{}{})
	m := newMountedWiper(t, s)
	before := s.service.Snapshot()
	m.beginDrag()
	held := waitMountedSimulationTime(t, s, before.Transient.Sample.Time+2*before.TransientSettings.StepSeconds)
	if held.CommandRevision != before.CommandRevision || !reflect.DeepEqual(held.Components, before.Components) || !reflect.DeepEqual(held.UndoEntry, before.UndoEntry) || held.Paused || held.Transient.Stale {
		t.Fatal("drag previews changed source/history or stopped the live clock")
	}
	m.frame() // real solver publications refresh the view without a new command
	m.release()
	after, err := s.service.Wait(m.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after.CommandRevision != before.CommandRevision+1 || after.RunID != before.RunID || after.ElectricalRevision != before.ElectricalRevision || after.Paused || after.Transient.Stale || after.Mixed.Stale || after.UndoEntry == nil || after.UndoEntry.Actor != "user" || after.UndoEntry.CommandRevision != after.CommandRevision {
		t.Fatal("live drag lost its one user commit or restarted/staled the experiment", s.errorMessage)
	}
	// Observe later accepted output as well as the same-time input solve.
	after = waitMountedSimulationTime(t, s, after.Transient.Sample.Time+2*after.TransientSettings.StepSeconds)
	if after.CommandRevision != before.CommandRevision+1 || after.RunID != before.RunID || after.Paused || after.Transient.Stale || after.Mixed.Stale {
		t.Fatal("accepted wiper edit did not continue the existing live run")
	}
	wiper := after.Mixed.Potentiometers[0]
	if wiper.WiperPosition <= .5 || len(after.Transient.Events) != len(before.Transient.Events)+1 {
		t.Fatal("live drag did not produce exactly one accepted wiper event")
	}
	event := after.Transient.Events[len(after.Transient.Events)-1]
	if !strings.HasPrefix(event.Mode, "wiper:"+wiper.ID+":") || event.Time < held.Transient.Sample.Time {
		t.Fatal("live drag event does not identify its accepted time", event)
	}
	left, right, count := 0., 0., 0
	for _, sample := range after.Transient.Samples {
		if sample.Time == event.Time {
			if count == 0 {
				left = sample.CapacitorVoltage
			}
			right = sample.CapacitorVoltage
			count++
		}
	}
	wantEvent := 2.5 * (1 - math.Exp(-event.Time/.0025))
	if count < 2 || math.Abs(left-right) > 1e-10 || math.Abs(right-wantEvent) > .002 {
		t.Fatalf("event loses charge/RC law: frames=%d left=%g right=%g reference=%g", count, left, right, wantEvent)
	}
	upper, lower := 1+9998*(1-wiper.WiperPosition), 1+9998*wiper.WiperPosition
	target, tau := 5*lower/10000, upper*lower/10000*1e-6
	wantAfter := target + (right-target)*math.Exp(-(after.Transient.Sample.Time-event.Time)/tau)
	if math.Abs(after.Mixed.Capacitors[0].Voltage-wantAfter) > .002 {
		t.Fatal("running circuit diverged from independently derived Thevenin RC law")
	}

	// Replay through public agent commands at the observed accepted time.
	agent := workbench.New()
	t.Cleanup(agent.Close)
	agentAction(t, agent, "project.load", workbench.LoadParams{Path: path})
	agentAction(t, agent, "simulation.input.set", workbench.InputParams{Mode: "high"})
	agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: event.Time})
	agentAction(t, agent, "potentiometer.wiper.set", workbench.WiperParams{ID: wiper.ID, Position: wiper.WiperPosition})
	if duration := after.Transient.Sample.Time - event.Time; duration > 0 {
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: duration})
	}
	replayed := agent.Snapshot()
	if !reflect.DeepEqual(after.Components, replayed.Components) || !reflect.DeepEqual(after.Wires, replayed.Wires) || math.Abs(after.Mixed.Capacitors[0].Voltage-replayed.Mixed.Capacitors[0].Voltage) > .002 || math.Abs(after.Transient.Sample.Time-replayed.Transient.Sample.Time) > 1e-12 {
		t.Fatal("native running drag disagrees with public agent replay")
	}
	agentAction(t, s.service, "simulation.pause", struct{}{})
	uiAction(t, s, "edit.undo", nil)
	undone := s.service.Snapshot()
	if !reflect.DeepEqual(undone.Components, before.Components) || !reflect.DeepEqual(undone.Wires, before.Wires) || !reflect.DeepEqual(undone.UndoEntry, before.UndoEntry) || !undone.Paused || !undone.Transient.Stale || !undone.Mixed.Stale {
		t.Fatal("one Undo did not restore the drag source with truthful stale physics")
	}
	t.Logf("held %g→%g s; accepted at %g s; wiper %.8g; event charge %.10g V; one user history/event; agent replay %.10g V", before.Transient.Sample.Time, held.Transient.Sample.Time, event.Time, wiper.WiperPosition, right, replayed.Mixed.Capacitors[0].Voltage)
}

// The service uses its real live clock; the driver's manual clock controls only
// native gesture timing. Waiting for accepted samples proves genuine publication.
func waitMountedSimulationTime(t *testing.T, s *screen, at float64) workbench.Snapshot {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		state := s.service.Snapshot()
		if state.Status == "error" || state.Paused || state.Transient == nil || state.Transient.Stale {
			t.Fatal("live simulation stopped while waiting for accepted output", state.Diagnostics)
		}
		if state.Transient.Sample.Time >= at {
			return state
		}
		if time.Now().After(deadline) {
			t.Fatalf("live clock reached %g s; waiting for %g s", state.Transient.Sample.Time, at)
		}
		time.Sleep(time.Millisecond)
	}
}
