package main

import (
	"math"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
)

// Both captures use one voltage axis and one current axis. Independent scaling
// would make different amplitudes look equal when overlaid.
func scopeRanges(groups ...[]workbench.ScopeTrace) (vmin, vmax, imin, imax float64) {
	vmax, imax = 1, 1
	for _, traces := range groups {
		for _, trace := range traces {
			for _, p := range trace.Points {
				if trace.Probe.Unit == "A" {
					imin, imax = math.Min(imin, p.Value*1000), math.Max(imax, p.Value*1000)
				} else {
					vmin, vmax = math.Min(vmin, p.Value), math.Max(vmax, p.Value)
				}
			}
		}
	}
	return math.Floor(vmin), math.Ceil(vmax), math.Floor(imin), math.Ceil(imax)
}

// Dash lengths are screen-space and continuous through sample vertices. The
// adaptive length bounds dash geometry even on a highly oscillatory reference.
// This is presentation tessellation; it creates no sample values or cursors.
func dashedScopeLine(points []canvas.Point, stroke func(...canvas.Point)) {
	total := 0.
	for i := 1; i < len(points); i++ {
		total += math.Hypot(points[i].X-points[i-1].X, points[i].Y-points[i-1].Y)
	}
	dash := math.Max(6, total/1024)
	period, phase := dash*1.6, 0.
	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		dx, dy := b.X-a.X, b.Y-a.Y
		length := math.Hypot(dx, dy)
		if length < 1e-9 {
			continue
		}
		at := 0.
		for at < length {
			on := phase < dash
			room := period - phase
			if on {
				room = dash - phase
			}
			if room < 1e-9 {
				if on {
					phase = dash
				} else {
					phase = 0
				}
				continue
			}
			advance := math.Min(length-at, room)
			if on {
				stroke(canvas.Point{X: a.X + dx*at/length, Y: a.Y + dy*at/length}, canvas.Point{X: a.X + dx*(at+advance)/length, Y: a.Y + dy*(at+advance)/length})
			}
			at += advance
			if advance == room {
				if on {
					phase = dash
				} else {
					phase = 0
				}
			} else {
				phase += advance
			}
		}
	}
}
