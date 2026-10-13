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
	x, y  float64
	view  canvas.Viewport
	ready bool
}
type pointerBridge struct {
	enabled      atomic.Bool
	owner        *app.Application
	route        func(int, workbench.Point, workbench.Point)
	mu           sync.Mutex
	frame        pointerFrame
	active       bool
	lastX, lastY float64
}

var activePointer atomic.Pointer[pointerBridge]

func newPointerBridge(owner *app.Application, route func(int, workbench.Point, workbench.Point)) *pointerBridge {
	b := &pointerBridge{owner: owner, route: route}
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
func (b *pointerBridge) publish(x, y float64, view canvas.Viewport) {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.frame = pointerFrame{x, y, view, true}
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
	if b == nil || b.owner.Context().Err() != nil {
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
	if phase == 3 {
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
	if phase != 3 {
		b.lastX, b.lastY = px, py
	} else {
		delta = workbench.Point{Y: float64(scroll)}
	}
	if phase == 2 {
		b.active = false
	}
	point := frame.view.World(canvas.Point{X: px - frame.x, Y: py - frame.y})
	b.mu.Unlock()
	err := b.owner.PostWindow("main", func() {
		if !b.enabled.Load() {
			return
		}
		b.route(int(phase), workbench.Point{X: point.X, Y: point.Y}, delta)
	})
	if err != nil {
		b.mu.Lock()
		b.active = false
		b.mu.Unlock()
		return 0
	}
	return 1
}
