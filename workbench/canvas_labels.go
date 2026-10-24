package workbench

import (
	"fmt"
	"strings"
)

// CanvasFontScale matches the retained 7×13 bitmap captions. Below normal
// zoom, enlarge text toward one source pixel per screen pixel. The 2× cap
// still permits an overview of very large drawings without giant captions.
func CanvasFontScale(zoom float64) float64 {
	return max(1.25, min(2.5, 1/zoom))
}

func CanvasTextWidth(text string, scale float64) float64 {
	text = strings.ReplaceAll(text, "Ω", "ohm")
	return float64(len([]rune(text))) * 7 * scale
}

func CanvasTextBounds(at Point, text string, scale float64) (Point, Point) {
	return Point{at.X - scale, at.Y - 12*scale}, Point{at.X + CanvasTextWidth(text, scale) + scale, at.Y + 3*scale}
}

// CanvasAnnotationLayout keeps a source block right aligned and its lines
// separated as text grows. Horizontal sources keep the last baseline above
// their leads; vertical sources keep the block's original vertical center.
type CanvasAnnotationLayout struct {
	Right, FirstBaseline, LineHeight, Scale float64
}

func LayoutCanvasAnnotation(a SourceAnnotation, rotation int, zoom float64) CanvasAnnotationLayout {
	_, hi := SourceAnnotationBounds(a)
	scale := CanvasFontScale(zoom)
	lineHeight := SourceAnnotationLineHeight * scale / 1.25
	extra := float64(len(a.Lines)-1) * (lineHeight - SourceAnnotationLineHeight)
	first := a.Position.Y - extra/2
	if rotation == 90 || rotation == 270 {
		first = a.Position.Y - extra
	}
	return CanvasAnnotationLayout{Right: hi.X, FirstBaseline: first, LineHeight: lineHeight, Scale: scale}
}

func (l CanvasAnnotationLayout) Position(text string, index int) Point {
	return Point{X: l.Right - CanvasTextWidth(text, l.Scale), Y: l.FirstBaseline + float64(index)*l.LineHeight}
}

// CanvasPartLabel provides caption text and the base upright anchor. Interactive
// drawing and Fit use CanvasPartLabelPositionAtZoom when enlarging captions.
func CanvasPartLabel(part Component) (Point, string) {
	title := part.ID
	if IsLogicKind(part.Kind) {
		title += " " + strings.ToUpper(strings.TrimPrefix(part.Kind, "logic-"))
	}
	if IsSequentialKind(part.Kind) {
		title += " " + SequentialShortName(part.Kind)
	}
	if IsDisplayKind(part.Kind) {
		title += " " + part.Name
	}
	if part.Kind == "comparator" {
		title += " Hys"
	}
	if part.Kind == "timer555" {
		title += " 555"
	}
	if part.Kind == "inductor" {
		title += " " + InductorValueLabel(part.Value)
	} else if part.Unit != "" {
		if part.Unit == "F" {
			scale, suffix := 1., "F"
			if part.Value < 1e-9 {
				scale, suffix = 1e12, "pF"
			} else if part.Value < 1e-6 {
				scale, suffix = 1e9, "nF"
			} else if part.Value < 1e-3 {
				scale, suffix = 1e6, "uF"
			} else if part.Value < 1 {
				scale, suffix = 1e3, "mF"
			}
			title += fmt.Sprintf(" %.4g %s", part.Value*scale, suffix)
		} else if part.Unit == "Ω" && part.Value >= 1000 {
			title += fmt.Sprintf(" %.4g kΩ", part.Value/1000)
		} else {
			title += fmt.Sprintf(" %.4g %s", part.Value, part.Unit)
		}
	}
	return CanvasPartLabelPosition(part, title), title
}

// CanvasPartLabelPositionAtZoom keeps a DC caption's right edge clear of the
// source as its text grows. Horizontal sources put text above both leads.
// Other symbols retain their established anchors; export geometry is unchanged.
func CanvasPartLabelPositionAtZoom(part Component, title string, zoom float64) Point {
	if part.Kind != "dc-source" {
		return CanvasPartLabelPosition(part, title)
	}
	width := CanvasTextWidth(title, CanvasFontScale(zoom))
	if part.Rotation == 90 || part.Rotation == 270 {
		return Point{X: part.Position.X + 40 - width, Y: part.Position.Y - 60}
	}
	return Point{X: part.Position.X - 45 - width, Y: part.Position.Y + 3}
}

func CanvasPartLabelPosition(part Component, title string) Point {
	switch part.Kind {
	case "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor":
		return LogicLabelPosition(part)
	case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter":
		return SequentialLabelPosition(part)
	case "bcd-decoder", "seven-segment":
		return Point{part.Position.X + 130, part.Position.Y - 245}
	case "timer555":
		return Timer555LabelPosition(part)
	case "relay":
		return Point{part.Position.X + 125, part.Position.Y - 110}
	case "comparator":
		return Point{part.Position.X - 80, part.Position.Y - 65}
	case "opamp":
		return OpAmpLabelPosition(part)
	case "inductor":
		return InductorLabelPosition(part)
	case "led":
		return LEDLabelPosition(part)
	case "capacitor":
		return CapacitorLabelPosition(part)
	case "ground":
		return GroundLabelPosition(part)
	}
	y := part.Position.Y - 55
	if IsConnectivityPrimitive(part.Kind) {
		y = part.Position.Y - 18
	}
	if part.Rotation == 90 || part.Rotation == 270 {
		y = part.Position.Y - 75
	}
	x := part.Position.X - float64(len(title))*4.4
	if part.Kind == "dc-source" {
		x, y = part.Position.X-45-float64(len(title))*8.8, part.Position.Y+3
	} else if part.Kind == "resistor" && (part.Rotation == 90 || part.Rotation == 270) {
		x, y = part.Position.X+27, part.Position.Y+3
	}
	return Point{X: x, Y: y}
}
