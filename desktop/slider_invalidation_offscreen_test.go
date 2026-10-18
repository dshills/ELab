//go:build offscreen

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"io/fs"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/fonts"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/execution"
	"github.com/dshills/vellum/ui"
)

// Use the pinned driver's real retained widgets/event router without an OS
// window or a GPU. No SDK source or descriptor callback is replaced by the test.
func TestMountedSliderDoesNotBorrowAgentContextDuringHeldDrag(t *testing.T) {
	for _, name := range []string{"uninterrupted", "unchanged-repaint", "agent-repaint", "agent-document-replacement"} {
		intervening := name == "agent-repaint" || name == "agent-document-replacement"
		t.Run(name, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			if err := smokeConstructPotentiometer(s); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if _, err := s.service.Wait(ctx); err != nil {
				t.Fatal(err)
			}
			id := ""
			for _, p := range s.service.Snapshot().Components {
				if p.Kind == "potentiometer" {
					id = p.ID
				}
			}
			if id == "" {
				t.Fatal("pot fixture missing")
			}
			path := filepath.Join(t.TempDir(), "held-wiper.elab")
			if name == "agent-document-replacement" {
				agentAction(t, s.service, "project.save", workbench.SaveParams{Path: path})
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
				root := ui.Node{ID: "root", Kind: ui.Column, Children: []ui.Node{slider}}
				s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
				return root
			}
			if err := s.app.Open(app.Window{ID: "main", Title: "Held wiper", Width: 520, Height: 160, View: view}); err != nil {
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
			config.Theme = ui.Theme{Default: ui.Style{Foreground: ui.Some(color.NRGBA{R: 24, G: 24, B: 24, A: 255}), Background: ui.Some(color.NRGBA{R: 248, G: 248, B: 248, A: 255}), Typography: ui.Some(ui.Typography{Family: "Noto Sans", Size: 16})}}
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
				if err := manual.Close(context.Background()); err != nil {
					t.Error(err)
				}
			})
			settle := func() {
				t.Helper()
				for range 128 {
					snapshot, err := manual.Snapshot(ctx)
					if err != nil {
						t.Fatal(err)
					}
					ready := snapshot.Application.Ordinary+snapshot.Application.ReadyCompletions > 0
					layout := false
					for _, w := range snapshot.Windows {
						ready = ready || w.Events > 0 || w.ReadyPaint
						layout = layout || w.LayoutPending
					}
					if ready {
						if _, err := manual.Step(ctx, 7); err != nil {
							t.Fatal(err)
						}
						continue
					}
					if layout {
						if _, err := manual.Frame(ctx, "main"); err != nil {
							t.Fatal(err)
						}
						continue
					}
					return
				}
				t.Fatal("manual settle bound exhausted")
			}
			settle()
			snapshot, err := manual.Snapshot(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(snapshot.Windows) != 1 {
				t.Fatal("manual window missing")
			}
			window := snapshot.Windows[0]
			var bounds image.Rectangle
			for _, e := range window.Elements {
				if e.ID == "pot-wiper" {
					bounds = e.Bounds
				}
			}
			if bounds.Empty() {
				t.Fatal("slider has no measured geometry")
			}
			from := bounds.Min.Add(bounds.Size().Div(2))
			to := from.Add(image.Pt(bounds.Dx()/4, 0))
			input := func(kind events.Types, at image.Point) {
				t.Helper()
				if err := manual.Input(ctx, cogent.ManualInput{Window: "main", WindowIncarnation: window.Incarnation, Type: kind, Button: events.Left, Position: at}); err != nil {
					t.Fatal(err)
				}
				settle()
			}
			before := s.service.Snapshot()
			input(events.MouseDown, from)
			if _, err := clock.Advance(core.TimingSettings.SlideStartTime + time.Millisecond); err != nil {
				t.Fatal(err)
			}
			settle()
			input(events.MouseMove, from.Add(to.Sub(from).Div(2)))
			input(events.MouseMove, to)
			if s.service.Snapshot().CommandRevision != before.CommandRevision {
				t.Fatal("drag committed before release")
			}
			if intervening {
				if name == "agent-document-replacement" {
					agentAction(t, s.service, "project.load", workbench.LoadParams{Path: path, DiscardChanges: true})
				} else {
					agentAction(t, s.service, "potentiometer.wiper.set", workbench.WiperParams{ID: id, Position: .2})
				}
				next := s.service.Snapshot()
				if name == "agent-document-replacement" && next.Document.Generation == before.Document.Generation {
					t.Fatal("fixture did not replace the document")
				}
				before = next
			}
			if name != "uninterrupted" {
				if err := s.app.Invalidate("main"); err != nil {
					t.Fatal(err)
				}
				if _, err := manual.Frame(ctx, "main"); err != nil {
					t.Fatal(err)
				}
				settle() // owner view and native callback are rebound while mouse is held
			}
			input(events.MouseUp, to)
			after := s.service.Snapshot()
			if intervening {
				if !reflect.DeepEqual(before, after) {
					t.Fatalf("held drag overwrote agent after repaint: before %+v after %+v", before.Mixed.Potentiometers, after.Mixed.Potentiometers)
				}
			} else if after.CommandRevision != before.CommandRevision+1 || after.Mixed.Potentiometers[0].WiperPosition <= .5 || after.UndoEntry.Actor != "user" {
				t.Fatal("uninterrupted control drag did not commit once", after.Mixed.Potentiometers)
			}
		})
	}
}
