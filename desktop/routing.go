package main

import (
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
)

type routeRect struct{ left, right, top, bottom float64 }

func routeBody(p workbench.Component) routeRect {
	a, b, c, d := workbench.BodyBounds(p)
	return routeRect{a, b, c, d}
}
func routedWirePath(g workbench.Graph, w workbench.Wire) []canvas.Point {
	points := workbench.WirePath(g, w)
	out := make([]canvas.Point, len(points))
	for i, p := range points {
		out[i] = canvas.Point{X: p.X, Y: p.Y}
	}
	return out
}
