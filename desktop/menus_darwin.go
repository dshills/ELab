//go:build darwin && cgo

package main

/*
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>
#include <stdint.h>
void elabShowChoices(uint64_t token, const char *json, double x, double y);
*/
import "C"

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"unsafe"

	"cogentcore.org/core/core"
	"github.com/dshills/vellum/ui"
)

// The menu only transports a selected option. The original Vellum callback
// still performs the shared command and its file/history guards.
var popupCallbacks = struct {
	sync.Mutex
	pending map[uint64]func(int)
}{pending: make(map[uint64]func(int))}
var popupToken atomic.Uint64

func (s *screen) platformSelectors(node *ui.Node, revision uint64) {
	if node.ToolbarOverflow {
		// Keep the framework's More menu and its hidden-action filter intact.
		// Converting that required MenuButton to an AppKit proxy Button breaks
		// the overflow contract and exposes actions already visible in the bar.
		// The existing File/Examples/choice menus retain their AppKit path.
		return
	}
	if node.Kind == ui.Selector || node.Kind == ui.MenuButton {
		*node = s.nativeChoiceButton(*node, node.Kind == ui.Selector, revision)
	}
	for i := range node.Children {
		s.platformSelectors(&node.Children[i], revision)
	}
}

func (s *screen) nativeChoiceButton(node ui.Node, selection bool, revision uint64) ui.Node {
	choices := menuChoices{Kind: node.Kind, Options: node.Options, Change: node.Change}
	if selection {
		for _, option := range node.Options {
			if option.ID == node.SelectedID {
				node.Text = option.Label
			}
		}
	}
	checked := ""
	for _, option := range node.Options {
		if option.Checked {
			checked = option.Label
		}
	}
	node.Kind = ui.Button
	node.Options = nil
	node.Custom = choices
	node.Text += " ▾"
	if node.Accessibility.Help != "" {
		node.Accessibility.Help += " "
	}
	if selection || checked != "" {
		current := checked
		if selection {
			current = node.Text
		}
		node.Accessibility.Help += "Current choice: " + current + ". "
	}
	node.Accessibility.Help += "Opens a native menu. Use arrow keys to choose and Escape to dismiss."
	captured := node
	node.Activate = func() { s.showChoiceMenu(captured, choices, selection, revision) }
	return node
}

func (s *screen) showChoiceMenu(node ui.Node, choices menuChoices, selection bool, revision uint64) {
	if s.service.Snapshot().CommandRevision != revision || node.Disabled || choices.Change == nil || s.drawing == nil || s.drawing.widget == nil || s.drawing.widget.Scene == nil {
		return
	}
	window, windowErr := s.windowReference()
	if windowErr != nil {
		return
	}
	scene := s.drawing.widget.Scene
	var x, y float64
	found := false
	scene.WidgetWalkDown(func(_ core.Widget, w *core.WidgetBase) bool {
		if w.Name == node.ID {
			x = float64(w.Geom.Pos.Total.X) + float64(scene.SceneGeom.Pos.X)
			y = float64(w.Geom.Pos.Total.Y+w.Geom.Size.Actual.Total.Y) + float64(scene.SceneGeom.Pos.Y)
			found = true
		}
		return true
	})
	if !found {
		return
	}
	type item struct {
		Label     string `json:"label"`
		Disabled  bool   `json:"disabled"`
		Checked   bool   `json:"checked"`
		Separator bool   `json:"separator"`
	}
	items := make([]item, len(choices.Options))
	for i, option := range choices.Options {
		items[i] = item{Label: option.Label, Disabled: option.Disabled, Checked: option.Checked || selection && option.ID == node.SelectedID, Separator: option.Separator}
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return
	}
	s.menuSerial++
	serial := s.menuSerial
	token := popupToken.Add(1)
	popupCallbacks.Lock()
	if len(popupCallbacks.pending) >= 8 {
		popupCallbacks.Unlock()
		return
	}
	popupCallbacks.pending[token] = func(index int) {
		if index < 0 || index >= len(choices.Options) {
			return
		}
		_ = s.app.PostWindow(window, func() {
			s.acceptMenuChoice(node, choices, index, serial, revision)
		})
	}
	popupCallbacks.Unlock()
	text := C.CString(string(payload))
	C.elabShowChoices(C.uint64_t(token), text, C.double(x), C.double(y))
	C.free(unsafe.Pointer(text))
}

//export elabMenuSelected
func elabMenuSelected(token C.uint64_t, index C.int) {
	popupCallbacks.Lock()
	callback := popupCallbacks.pending[uint64(token)]
	delete(popupCallbacks.pending, uint64(token))
	popupCallbacks.Unlock()
	if callback != nil {
		callback(int(index))
	}
}
