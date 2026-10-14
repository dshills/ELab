package workbench

import (
	"encoding/json"
	"fmt"
	"math"
)

type DisplayFrameParams struct {
	Phase float64 `json:"phase,omitempty"`
}
type CurrentMarker struct {
	WireID string  `json:"wire_id"`
	Kind   string  `json:"kind"`
	At     Point   `json:"at"`
	DX     float64 `json:"dx"`
	DY     float64 `json:"dy"`
}
type DisplayFrame struct {
	Display CircuitDisplay  `json:"display"`
	Phase   float64         `json:"phase"`
	Markers []CurrentMarker `json:"markers"`
}

func (s *Service) displayQueryLocked(raw json.RawMessage) (DisplayFrame, error) {
	var p DisplayFrameParams
	if err := params(raw, &p); err != nil {
		return DisplayFrame{}, err
	}
	if !finite(p.Phase) || p.Phase < 0 || p.Phase > 1 {
		return DisplayFrame{}, fmt.Errorf("illustrative phase must be 0–1")
	}
	return MakeDisplayFrame(s.graph, s.circuitDisplayLocked(), p.Phase), nil
}

// MakeDisplayFrame is the portable renderer input used by native drawing, agent
// queries and static exports. Phase is illustrative wall animation, not physics.
func MakeDisplayFrame(g Graph, display CircuitDisplay, phase float64) DisplayFrame {
	phase = math.Mod(phase, 1)
	out := DisplayFrame{Display: display, Phase: phase, Markers: []CurrentMarker{}}
	if !display.Preferences.CurrentCues {
		return out
	}
	wires := map[string]Wire{}
	for _, w := range g.Wires {
		wires[w.ID] = w
	}
	for _, reading := range display.Wires {
		if reading.State != "live" || reading.Current == nil || math.Abs(*reading.Current) <= 1e-12 {
			continue
		}
		w, ok := wires[reading.ID]
		if !ok {
			continue
		}
		sign := 1.
		if *reading.Current < 0 {
			sign = -1
		}
		if display.Preferences.Direction == "electron" {
			sign = -sign
		}
		path := WirePath(g, w)
		add := func(kind string, fraction float64) {
			at, dx, dy, ok := pathPoint(path, fraction)
			if !ok {
				return
			}
			out.Markers = append(out.Markers, CurrentMarker{WireID: w.ID, Kind: kind, At: at, DX: dx * sign, DY: dy * sign})
		}
		add("arrow", .5)
		if display.Animate {
			for _, offset := range []float64{0, .5} {
				fraction := math.Mod(phase+offset, 1)
				if sign < 0 {
					fraction = 1 - fraction
				}
				add("dot", fraction)
			}
		}
	}
	return out
}
func pathPoint(path []Point, fraction float64) (Point, float64, float64, bool) {
	total := 0.
	for i := 1; i < len(path); i++ {
		total += math.Hypot(path[i].X-path[i-1].X, path[i].Y-path[i-1].Y)
	}
	if total <= 1e-9 {
		return Point{}, 0, 0, false
	}
	target := math.Max(0, math.Min(1, fraction)) * total
	for i := 1; i < len(path); i++ {
		dx, dy := path[i].X-path[i-1].X, path[i].Y-path[i-1].Y
		length := math.Hypot(dx, dy)
		if length <= 1e-9 {
			continue
		}
		if target <= length || i == len(path)-1 {
			return Point{X: path[i-1].X + dx*target/length, Y: path[i-1].Y + dy*target/length}, dx / length, dy / length, true
		}
		target -= length
	}
	return Point{}, 0, 0, false
}
