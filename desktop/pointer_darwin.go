//go:build darwin

package main

/*
#cgo LDFLAGS: -framework AppKit
void elabInstallPointerMonitor(void);
void elabRemovePointerMonitor(void);
*/
import "C"

import (
	"sync"
	"sync/atomic"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/canvas"
)

type pointerFrame struct {
	x, y        float64
	view        canvas.Viewport
	ready       bool
	pane, token string
}
type pointerBridge struct {
	enabled      atomic.Bool
	owner        *app.Application
	window       app.WindowRef
	route        func(string, int, workbench.Point, workbench.Point)
	mu           sync.Mutex
	frame        pointerFrame
	active       bool
	hovered      bool
	lastX, lastY float64
}

var activePointer atomic.Pointer[pointerBridge]

func newPointerBridge(owner *app.Application, window app.WindowRef, route func(string, int, workbench.Point, workbench.Point)) *pointerBridge {
	b := &pointerBridge{owner: owner, window: window, route: route}
	b.enabled.Store(true)
	activePointer.Store(b)
	C.elabInstallPointerMonitor()
	return b
}
func (b *pointerBridge) setEnabled(enabled bool) {
	if b != nil {
		b.enabled.Store(enabled)
	}
}
func (b *pointerBridge) invalidateFrame() {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.frame.ready = false
	b.active = false
	b.mu.Unlock()
}
func (b *pointerBridge) publish(x, y float64, view canvas.Viewport, pane ...string) {
	if b == nil {
		return
	}
	b.mu.Lock()
	kind, token := "schematic", ""
	if len(pane) > 0 {
		kind = pane[0]
	}
	if len(pane) > 1 {
		token = pane[1]
	}
	b.frame = pointerFrame{x: x, y: y, view: view, ready: true, pane: kind, token: token}
	b.mu.Unlock()
}
func (b *pointerBridge) close() {
	if b == nil {
		return
	}
	activePointer.CompareAndSwap(b, nil)
	C.elabRemovePointerMonitor()
}

//export goELabPointer
func goELabPointer(phase C.int, x, y, scroll C.double) C.int {
	b := activePointer.Load()
	if b == nil || b.window.Context().Err() != nil {
		return 0
	}
	px, py := float64(x), float64(y)
	b.mu.Lock()
	frame := b.frame
	if !frame.ready || !b.enabled.Load() {
		b.active = false
		b.mu.Unlock()
		return 0
	}
	inside := px >= frame.x && py >= frame.y && px < frame.x+frame.view.Width && py < frame.y+frame.view.Height
	if phase == 4 {
		if frame.pane != "schematic" {
			b.mu.Unlock()
			return 0
		}
		if !inside {
			if !b.hovered {
				b.mu.Unlock()
				return 0
			}
			phase = 5
		}
		b.hovered = inside
	} else if phase == 3 {
		if !inside {
			b.mu.Unlock()
			return 0
		}
	} else if phase == 0 {
		if !inside {
			b.mu.Unlock()
			return 0
		}
		b.active = true
		b.lastX, b.lastY = px, py
	} else if !b.active {
		b.mu.Unlock()
		return 0
	}
	delta := workbench.Point{X: (px - b.lastX) / frame.view.Zoom, Y: (py - b.lastY) / frame.view.Zoom}
	if phase <= 2 {
		b.lastX, b.lastY = px, py
	} else if phase == 3 {
		delta = workbench.Point{Y: float64(scroll)}
	}
	if phase == 2 {
		b.active = false
	}
	point := frame.view.World(canvas.Point{X: px - frame.x, Y: py - frame.y})
	b.mu.Unlock()
	err := b.owner.PostWindow(b.window, func() { b.deliver(frame, int(phase), workbench.Point{X: point.X, Y: point.Y}, delta) })
	if err != nil {
		b.mu.Lock()
		b.active = false
		b.mu.Unlock()
		return 0
	}
	if phase >= 4 {
		return 0 // Passive hover must not consume AppKit/GLFW movement.
	}
	return 1
}

// Runs on the owner with the packet's original prepared frame. Geometry alone
// cannot distinguish two documents/tools rendered at the same window size.
func (b *pointerBridge) deliver(frame pointerFrame, phase int, point, delta workbench.Point) {
	if !b.enabled.Load() {
		return
	}
	b.mu.Lock()
	current := b.frame
	b.mu.Unlock()
	if (frame.pane == "schematic" || frame.pane == "sweep") && (phase == 1 || phase == 2) {
		// Continuations retain the canvas gesture's accepted command/document
		// guard. Its own selection, pan or sweep cursor can repaint before release.
		// A different actor or replacement is rejected by that guard.
		current.token, frame.token = "", ""
		if frame.pane == "schematic" {
			current.view.Center = frame.view.Center
		}
	}
	if current != frame {
		return
	}
	b.route(frame.pane, phase, point, delta)
}
