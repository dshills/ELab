//go:build darwin && cgo

package main

/*
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>
void elabShowMenu(const char *group, double x, double y);
*/
import "C"

import (
	"fmt"
	"unsafe"

	"cogentcore.org/core/core"
	"github.com/dshills/vellum/ui"
)

func (s *screen) platformMenu(node ui.Node, group string) ui.Node {
	choices := menuChoices{Options: node.Options, Change: node.Change}
	node.Kind = ui.Button
	node.Options = nil
	node.Custom = choices
	node.Text += " ⌄"
	id := node.ID
	node.Activate = func() { s.showPlatformMenu(id, group) }
	return node
}
func (s *screen) showPlatformMenu(id, group string) {
	if s.drawing == nil || s.drawing.widget == nil || s.drawing.widget.Scene == nil {
		return
	}
	scene := s.drawing.widget.Scene
	var x, y float64
	found := false
	scene.WidgetWalkDown(func(_ core.Widget, w *core.WidgetBase) bool {
		if w.Name == id {
			x = float64(w.Geom.Pos.Total.X) + float64(scene.SceneGeom.Pos.X)
			y = float64(w.Geom.Pos.Total.Y+w.Geom.Size.Actual.Total.Y) + float64(scene.SceneGeom.Pos.Y)
			found = true
		}
		return true
	})
	if !found {
		return
	}
	s.menuSerial++
	serial := s.menuSerial
	go func() {
		services, err := s.app.Services()
		if err == nil {
			err = services.SetMenu(s.app.Context(), applicationMenus)
		}
		_ = s.app.PostWindow("main", func() {
			if s.menuSerial != serial || s.fileBusy || s.pendingCommand != "" {
				return
			}
			if err != nil {
				s.errorMessage = fmt.Sprintf("Menu unavailable: %v", err)
				s.app.Invalidate("main")
				return
			}
			text := C.CString(group)
			C.elabShowMenu(text, C.double(x), C.double(y))
			C.free(unsafe.Pointer(text))
		})
	}()
}
