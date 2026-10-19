//go:build offscreen

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedButtonOwnerShutdownLongErrorsKeepAcknowledgementVisible(t *testing.T) {
	for _, fixture := range []struct {
		theme         string
		width, height int
	}{{"light", 1040, 720}, {"dark", 1040, 720}, {"light", 1280, 800}} {
		t.Run(fmt.Sprintf("%s-%dx%d", fixture.theme, fixture.width, fixture.height), func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			agentAction(t, s.service, "example.reset", workbench.ReplaceParams{})
			agentAction(t, s.service, "simulation.pause", struct{}{})
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": fixture.theme})
			directory := t.TempDir()
			manual := filepath.Join(directory, "manual.elab")
			agentAction(t, s.service, "project.save", workbench.SaveParams{Path: manual})
			original, err := os.ReadFile(manual)
			if err != nil {
				t.Fatal(err)
			}
			appearance := filepath.Join(directory, "appearance.json")
			data, err := json.Marshal(map[string]any{"format": "elab.appearance", "version": 1, "theme": "dark", strings.Repeat("bad-field-", 350): true})
			if err != nil || len(data) > workbench.MaxAppearanceBytes {
				t.Fatal("appearance fixture must reach parsing within its real byte limit", err)
			}
			if err := os.WriteFile(appearance, data, 0600); err != nil {
				t.Fatal(err)
			}
			agentAction(t, s.service, "appearance.configure", workbench.AppearanceConfigureParams{Path: appearance})
			agentAction(t, s.service, "appearance.wait", struct{}{})
			recovery := filepath.Join(directory, strings.Repeat("r", 200), strings.Repeat("s", 200), strings.Repeat("t", 200))
			agentAction(t, s.service, "recovery.configure", workbench.RecoveryConfigureParams{Directory: recovery, Interval: 1})
			entries, err := os.ReadDir(recovery)
			if err != nil || len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), "session-") {
				t.Fatal("owned recovery fixture must contain exactly one lease", err)
			}
			lease := filepath.Join(recovery, entries[0].Name())
			if err := os.Remove(lease); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(lease, 0700); err != nil {
				t.Fatal(err)
			}
			protected := filepath.Join(lease, "retain")
			if err := os.WriteFile(protected, []byte("owned fault fixture"), 0600); err != nil {
				t.Fatal(err)
			}
			agentAction(t, s.service, "app.close.start", workbench.ReplaceParams{})
			deadline := time.Now().Add(3 * time.Second)
			for s.service.Snapshot().Shutdown.Status != "failed" {
				if time.Now().After(deadline) {
					t.Fatal("shutdown did not report the actual cleanup failure")
				}
				time.Sleep(time.Millisecond)
			}
			before := s.service.Snapshot()
			if before.ReadyToExit() || len(before.Shutdown.AppearanceError) < 3500 || len(before.Shutdown.RecoveryError) < 600 {
				t.Fatal("fault fixture omitted the long real errors")
			}
			theme := desktopTheme()
			// Manual windows require an explicit managed font and background.
			// Preserve the app's kind styles and sizes with a transparent default.
			theme.Default.Background = ui.Some(color.NRGBA{})
			theme.Default.Typography.Value.Family = "Noto Sans"
			m := newMountedControlWindowSize(t, s, s.view, float32(fixture.width), float32(fixture.height), theme)
			im, err := m.manual.Capture(m.ctx, "main")
			if err != nil {
				t.Fatal(err)
			}
			button := mountedControlElement(t, m, "shutdown-acknowledge")
			t.Logf("scene=%v button=%v visible=%v errors=%d/%d", im.Bounds(), button.Bounds, button.Visible, len(before.Shutdown.RecoveryError), len(before.Shutdown.AppearanceError))
			if artifacts := os.Getenv("ELAB_SHUTDOWN_LAYOUT_ARTIFACTS"); artifacts != "" {
				path := filepath.Join(artifacts, fmt.Sprintf("%s-%dx%d.png", fixture.theme, fixture.width, fixture.height))
				file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
				if err != nil {
					t.Fatal(err)
				}
				if err := png.Encode(file, im); err != nil {
					_ = file.Close()
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if !button.Visible || button.Bounds.Dy() < 32 || !button.Bounds.In(image.Rect(0, 0, fixture.width, fixture.height)) {
				t.Fatal("shutdown acknowledgement is clipped below a readable button height")
			}
			root := s.view()
			for id, message := range map[string]string{"shutdown-recovery-error": before.Shutdown.RecoveryError, "shutdown-appearance-error": before.Shutdown.AppearanceError} {
				if node, ok := findNode(root, id); !ok || node.Text != message {
					t.Fatal("shutdown omitted or shortened the actual error", id)
				}
			}
			if fixture.width == 1040 {
				scroll := mountedControlElement(t, m, "shutdown-error-scroll")
				errorBefore := mountedControlElement(t, m, "shutdown-appearance-error")
				m.input(cogent.ManualInput{Type: events.Scroll, Position: scroll.Bounds.Min.Add(scroll.Bounds.Size().Div(2)), Delta: math32.Vec2(0, 1000)})
				errorAfter := mountedControlElement(t, m, "shutdown-appearance-error")
				t.Logf("error scroll: %v -> %v; footer=%v", errorBefore.Bounds, errorAfter.Bounds, mountedControlElement(t, m, "shutdown-acknowledge").Bounds)
				if errorAfter.Bounds == errorBefore.Bounds || mountedControlElement(t, m, "shutdown-acknowledge").Bounds != button.Bounds {
					t.Fatal("long errors do not scroll independently of the close action")
				}
			}
			clickMountedControl(t, m, "shutdown-acknowledge")
			after := s.service.Snapshot()
			if !after.ReadyToExit() || after.Shutdown.Status != "failed" || after.Shutdown.RecoveryError != before.Shutdown.RecoveryError || after.Shutdown.AppearanceError != before.Shutdown.AppearanceError || after.CommandRevision != before.CommandRevision+1 || after.LastActor != "user" || after.Document != before.Document || !reflect.DeepEqual(after.Components, before.Components) || !reflect.DeepEqual(after.Wires, before.Wires) || !reflect.DeepEqual(after.Result, before.Result) {
				t.Fatal("one shared acknowledgement hid the failure or changed the circuit")
			}
			for path, want := range map[string][]byte{manual: original, appearance: data, protected: []byte("owned fault fixture")} {
				if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, want) {
					t.Fatal("acknowledgement changed protected file", path, err)
				}
			}
		})
	}
}

func TestMountedButtonOwnerKeepsPressContextAcrossAgentSwitchEdit(t *testing.T) {
	for _, agentEdit := range []bool{false, true} {
		name := "unchanged-repaint"
		if agentEdit {
			name = "agent-repaint"
		}
		t.Run(name, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: "S1", Kind: "switch"})
			m := mountedInspectorControl(t, s, "switch-actuate")
			button := mountedControlElement(t, m, "switch-actuate")
			at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
			before := s.service.Snapshot()
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("switch activated before release")
			}
			if agentEdit {
				agentAction(t, s.service, "switch.set", workbench.SwitchParams{ID: "S1", Closed: true})
			}
			current := s.service.Snapshot()
			m.frame()
			fresh := mountedControlElement(t, m, "switch-actuate")
			if fresh.Incarnation != button.Incarnation {
				t.Fatal("fixture must retain the same logical owner")
			}
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
			after := s.service.Snapshot()
			if agentEdit {
				if !reflect.DeepEqual(current, after) {
					t.Fatalf("held Close borrowed repainted Open action: closed %v→%v; revision %d→%d", *current.Components[0].SwitchClosed, *after.Components[0].SwitchClosed, current.CommandRevision, after.CommandRevision)
				}
				if !strings.Contains(s.errorMessage, "expected command revision") {
					t.Fatal("old release did not reach the shared conflict guard", s.errorMessage)
				}
				// A fresh press must still use the newly rendered Open action.
				m.frame()
				for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
					m.input(cogent.ManualInput{Type: kind, Button: events.Left, Position: at})
				}
				accepted := s.service.Snapshot()
				if *accepted.Components[0].SwitchClosed || accepted.CommandRevision != current.CommandRevision+1 {
					t.Fatal("fresh press did not open the current switch once", s.errorMessage)
				}
			} else if !*after.Components[0].SwitchClosed || after.CommandRevision != before.CommandRevision+1 {
				t.Fatal("ordinary press did not close the switch once", s.errorMessage)
			}
		})
	}
}

func TestMountedButtonOwnerKeepsPressContextAcrossAgentHistoryTip(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.reset", nil)
	m := newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		root := ui.Node{ID: "root", Kind: ui.Column, Children: []ui.Node{s.button("edit.undo", "Undo")}}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
	button := mountedControlElement(t, m, "edit.undo")
	at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
	before := s.service.Snapshot()
	m.frame()
	if fresh := mountedControlElement(t, m, "edit.undo"); fresh.Incarnation != button.Incarnation {
		t.Fatal("fixture replaced the retained history button")
	}
	m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("held Undo borrowed the new agent history tip/permission")
	}
	if !strings.Contains(s.errorMessage, "history tip changed") {
		t.Fatal("old Undo did not reach the shared history guard", s.errorMessage)
	}
	m.frame()
	for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
		m.input(cogent.ManualInput{Type: kind, Button: events.Left, Position: at})
	}
	after, err := s.service.Wait(m.ctx)
	if err != nil || after.CommandRevision != before.CommandRevision+1 || after.RedoEntry == nil || after.RedoEntry.Actor != "agent" {
		t.Fatal("fresh labeled Undo failed", err, s.errorMessage)
	}
	for _, p := range after.Components {
		if p.ID == "R1" && p.Value != 1000 {
			t.Fatal("fresh Undo did not restore resistor value", p.Value)
		}
	}
}

func TestMountedButtonOwnerPauseAcceptsLiveRepaintDuringHeldPress(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.rc", nil)
	m := newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		root := ui.Node{ID: "root", Kind: ui.Column, Children: []ui.Node{s.button("simulation.pause", "Pause")}}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
	button := mountedControlElement(t, m, "simulation.pause")
	at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
	before := s.service.Snapshot()
	deadline := time.Now().Add(3 * time.Second)
	for s.service.Snapshot().Transient.Sample.Time <= before.Transient.Sample.Time {
		if time.Now().After(deadline) {
			t.Fatal("real RC clock did not advance during held Pause")
		}
		time.Sleep(5 * time.Millisecond)
	}
	current := s.service.Snapshot()
	if current.CommandRevision != before.CommandRevision {
		t.Fatal("fixture changed command context instead of publishing samples")
	}
	m.frame()
	if mountedControlElement(t, m, "simulation.pause").Incarnation != button.Incarnation {
		t.Fatal("live sample replaced the held button")
	}
	m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
	after := s.service.Snapshot()
	if s.errorMessage != "" || !after.Paused || after.Transient.Stale || after.RunID != before.RunID || after.CommandRevision != before.CommandRevision+1 || after.Transient.Sample.Time < current.Transient.Sample.Time {
		t.Fatal("held Pause refused live publication or reset accepted time", s.errorMessage)
	}
}

func TestMountedButtonOwnerCanceledPressDoesNotPoisonFreshInput(t *testing.T) {
	for _, keyboard := range []bool{false, true} {
		name := "fresh-pointer"
		if keyboard {
			name = "fresh-keyboard"
		}
		t.Run(name, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: "S1", Kind: "switch"})
			m := mountedInspectorControl(t, s, "switch-actuate")
			button := mountedControlElement(t, m, "switch-actuate")
			at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
			agentAction(t, s.service, "switch.set", workbench.SwitchParams{ID: "S1", Closed: true})
			before := s.service.Snapshot()
			m.frame()
			outside := button.Bounds.Max
			m.input(cogent.ManualInput{Type: events.MouseMove, Position: outside})
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: outside})
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("release outside activated the canceled press")
			}
			if keyboard {
				if m.snapshot().FocusedID != "switch-actuate" {
					t.Fatal("canceled button lost actual keyboard focus")
				}
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			} else {
				for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
					m.input(cogent.ManualInput{Type: kind, Button: events.Left, Position: at})
				}
			}
			after := s.service.Snapshot()
			if *after.Components[0].SwitchClosed || after.CommandRevision != before.CommandRevision+1 {
				t.Fatal("fresh input inherited canceled press context", s.errorMessage)
			}
		})
	}
}

func TestMountedButtonOwnerSwitchPreservesRCPhysicsLikePublicAgent(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	agent := workbench.New()
	t.Cleanup(agent.Close)
	for _, service := range []*workbench.Service{s.service, agent} {
		agentAction(t, service, "example.rc", workbench.ReplaceParams{})
		agentAction(t, service, "simulation.pause", struct{}{})
		agentAction(t, service, "component.place", workbench.PlaceParams{ID: "S1", Kind: "switch", Position: workbench.Point{X: -100, Y: -60}})
		agentAction(t, service, "object.delete", workbench.IDParams{ID: "W1"})
		agentAction(t, service, "wire.connect", workbench.ConnectParams{From: "V1.positive", To: "S1.1"})
		agentAction(t, service, "wire.connect", workbench.ConnectParams{From: "S1.2", To: "R1.1"})
		agentAction(t, service, "switch.set", workbench.SwitchParams{ID: "S1", Closed: true})
		agentAction(t, service, "simulation.restart", struct{}{})
		agentAction(t, service, "simulation.step", workbench.StepParams{Duration: .01})
	}
	m := mountedInspectorControl(t, s, "switch-actuate")
	button := mountedControlElement(t, m, "switch-actuate")
	at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
	before := s.service.Snapshot()
	for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
		m.input(cogent.ManualInput{Type: kind, Button: events.Left, Position: at})
	}
	agentAction(t, agent, "switch.set", workbench.SwitchParams{ID: "S1", Closed: false})
	u, a := s.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(u.Components, a.Components) || !reflect.DeepEqual(u.Mixed, a.Mixed) || !reflect.DeepEqual(u.Transient, a.Transient) || !reflect.DeepEqual(u.SwitchStates, a.SwitchStates) {
		t.Fatal("mounted actuation differs from public agent")
	}
	if u.CommandRevision != before.CommandRevision+1 || u.RunID != before.RunID || !reflect.DeepEqual(u.Mixed, before.Mixed) || !reflect.DeepEqual(u.Transient, before.Transient) || !u.SwitchStates[0].Pending || !*u.SwitchStates[0].AppliedClosed || u.UndoEntry.Actor != "user" {
		t.Fatal("paused actuation changed committed charge/time or hid pending contact state")
	}
	for _, service := range []*workbench.Service{s.service, agent} {
		agentAction(t, service, "simulation.step", workbench.StepParams{Duration: .001})
	}
	u, a = s.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(u.Mixed, a.Mixed) || !reflect.DeepEqual(u.Transient, a.Transient) || u.SwitchStates[0].Pending || *u.SwitchStates[0].AppliedClosed || u.Transient.Events[len(u.Transient.Events)-1].Mode != "switch:S1:open" {
		t.Fatal("mounted switch next-step physics/event differs from public agent")
	}
}
