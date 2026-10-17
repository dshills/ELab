package workbench

import (
	"fmt"
	"math"
)

const SourceAnnotationLineHeight = 18.

// SourceAnnotation is an owned presentation of configured excitation. Exact
// values remain in Components/TransientSettings; these lines use six digits.
type SourceAnnotation struct {
	ID       string   `json:"id"`
	Position Point    `json:"position"`
	Lines    []string `json:"lines"`
	State    string   `json:"state"` // configured, current or stale
	Mode     string   `json:"input_mode,omitempty"`
	Pending  string   `json:"pending_input_mode,omitempty"`
}

func cloneSourceAnnotations(in []SourceAnnotation) []SourceAnnotation {
	out := append([]SourceAnnotation{}, in...)
	for i := range out {
		out[i].Lines = append([]string{}, out[i].Lines...)
	}
	return out
}

func annotationTime(seconds float64) string {
	scale, unit := 1., "s"
	if seconds < .001 {
		scale, unit = 1e6, "us"
	} else if seconds < 1 {
		scale, unit = 1000, "ms"
	}
	return fmt.Sprintf("%.6g %s", seconds*scale, unit)
}

func configuredSourceAnnotations(g Graph, settings TransientSettings) []SourceAnnotation {
	out := []SourceAnnotation{}
	for _, p := range g.Components {
		var lines []string
		switch {
		case p.Kind == "sine-source" && p.Sine != nil:
			w := p.Sine
			lines = []string{p.ID + " | sine", fmt.Sprintf("%.6g Vpk | %.6g Hz", w.Amplitude, w.Frequency), fmt.Sprintf("offset %.6g V | phase %.6g deg", w.Offset, w.Phase)}
		case p.Kind == "finite-pulse-source" && p.Pulse != nil:
			w := p.Pulse
			lines = []string{p.ID + " | pulse", fmt.Sprintf("low %.6g V | high %.6g V", w.LowVoltage, w.HighVoltage), "high " + annotationTime(w.HighSeconds) + " | low " + annotationTime(w.LowSeconds), "rise " + annotationTime(w.RiseSeconds) + " | fall " + annotationTime(w.FallSeconds)}
		case p.Kind == "pulse-source":
			lines = []string{p.ID + " | pulse (ideal edges)", fmt.Sprintf("low 0 V | high %.6g V", p.Value), "high " + annotationTime(settings.HighSeconds) + " | low " + annotationTime(settings.LowSeconds)}
		default:
			continue
		}
		out = append(out, SourceAnnotation{ID: p.ID, Lines: lines, State: "configured"})
	}
	positionSourceAnnotations(g, out)
	return out
}

func positionSourceAnnotations(g Graph, annotations []SourceAnnotation) {
	for i := range annotations {
		p, _ := g.Component(annotations[i].ID)
		width := 0.
		for _, line := range annotations[i].Lines {
			width = math.Max(width, schematicLabelWidth(line))
		}
		if p.Rotation == 90 || p.Rotation == 270 {
			// Horizontal sources have terminal leads on both sides. Put the
			// entire block above them, including held/pending/stale lines,
			// with its right edge aligned over the right-hand terminal.
			annotations[i].Position = Point{X: p.Position.X + 40 - width, Y: p.Position.Y - 60 - float64(len(annotations[i].Lines)-1)*SourceAnnotationLineHeight}
			continue
		}
		annotations[i].Position = Point{X: p.Position.X - width - 40, Y: p.Position.Y - float64(len(annotations[i].Lines)-1)*SourceAnnotationLineHeight/2 + 5}
	}
}

func SourceAnnotationBounds(a SourceAnnotation) (Point, Point) {
	width := 0.
	for _, line := range a.Lines {
		width = math.Max(width, schematicLabelWidth(line))
	}
	return Point{X: a.Position.X - 3, Y: a.Position.Y - 15}, Point{X: a.Position.X + width, Y: a.Position.Y + float64(len(a.Lines)-1)*SourceAnnotationLineHeight + 4}
}

func (s *Service) sourceAnnotationsLocked() []SourceAnnotation {
	out := configuredSourceAnnotations(s.graph, s.transientSettings)
	if s.transient != nil {
		fresh := !s.transient.Stale && s.transient.RunID == s.run && s.transient.ElectricalRevision == s.electrical
		for i := range out {
			if !fresh {
				out[i].State = "stale"
				out[i].Lines = append(out[i].Lines, "configured | live stale")
				continue
			}
			out[i].State = "current"
			if out[i].ID != s.transient.SourceID {
				continue
			}
			out[i].Mode = s.transient.Mode
			out[i].Pending = s.transient.PendingMode
			if out[i].Mode != "auto" {
				out[i].Lines = append(out[i].Lines, "input held: "+out[i].Mode)
			}
			if out[i].Pending != "" {
				out[i].Lines = append(out[i].Lines, "pending input: "+out[i].Pending)
			}
		}
	}
	positionSourceAnnotations(s.graph, out)
	return out
}
