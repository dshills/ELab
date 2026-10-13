//go:build !darwin

package main

import (
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/canvas"
)

type pointerBridge struct{}

func newPointerBridge(*app.Application, func(int, workbench.Point, workbench.Point)) *pointerBridge {
	return nil
}
func (*pointerBridge) publish(float64, float64, canvas.Viewport) {}
func (*pointerBridge) close()                                    {}

func (*pointerBridge) setEnabled(bool) {}
