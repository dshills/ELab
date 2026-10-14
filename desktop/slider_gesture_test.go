package main

import (
	"image"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"github.com/dshills/vellum/ui"
)

// Exercise the pinned native control's actual gesture events. Vellum forwards
// Change (commit), not Input (preview), to the shared application callback.
// This is control-level evidence; invalidation/focus remains a separate gate.
func TestNativeSliderGestureCommitsOneSharedHistoryEntry(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := smokeConstructPotentiometer(s); err != nil {
		t.Fatal(err)
	}
	node, ok := findNode(s.view(), "pot-wiper")
	if !ok {
		t.Fatal("wiper slider missing")
	}
	slider := core.NewSlider(core.NewBody("gesture-control"))
	slider.Min = 0
	slider.Max = 100
	slider.Step = 1
	slider.EnforceStep = false
	slider.Styler(func(s *styles.Style) { s.Direction = styles.Row })
	slider.Update()
	slider.Geom.Size.Actual.Content.X = 300
	slider.Geom.Size.Actual.Content.Y = 20
	slider.SetValue(50)
	previews, commits := 0, 0
	slider.On(events.Input, func(events.Event) { previews++ })
	slider.OnChange(func(events.Event) { commits++; node.Change(ui.ChangeEvent{Number: slider.Value}) })
	before := s.service.Snapshot()
	start := image.Pt(150, 10)
	begin := events.NewMouse(events.SlideStart, events.Left, start, 0)
	begin.SetLocalOff(image.Point{})
	slider.HandleEvent(begin)
	for _, x := range []int{170, 185, 205, 220} {
		e := events.NewMouseDrag(events.Left, image.Pt(x, 10), start, start, 0)
		e.Typ = events.SlideMove
		e.SetLocalOff(image.Point{})
		slider.HandleEvent(e)
	}
	if commits != 0 || previews == 0 || s.service.Snapshot().CommandRevision != before.CommandRevision {
		t.Fatalf("drag previews=%d commits=%d command revisions=%d/%d value=%g", previews, commits, before.CommandRevision, s.service.Snapshot().CommandRevision, slider.Value)
	}
	end := events.NewMouseDrag(events.Left, image.Pt(220, 10), start, start, 0)
	end.Typ = events.SlideStop
	end.SetLocalOff(image.Point{})
	slider.HandleEvent(end)
	after := s.service.Snapshot()
	if commits != 1 || after.CommandRevision != before.CommandRevision+1 || after.UndoEntry == nil || after.UndoEntry.CommandRevision != after.CommandRevision {
		t.Fatal("native drag did not commit exactly one shared history entry")
	}
	uiAction(t, s, "edit.undo", nil)
	for _, p := range s.service.Snapshot().Components {
		if p.Kind == "potentiometer" && *p.WiperPosition != .5 {
			t.Fatal("one undo did not restore complete slider drag")
		}
	}
}
