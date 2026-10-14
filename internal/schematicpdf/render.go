// Package schematicpdf renders ELab's own generated SVG primitives into PDF or PNG.
// It does not accept arbitrary imported SVG or execute embedded content.
package schematicpdf

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type Page struct {
	Width, Height, Margin float64 // PDF points
	Scale                 float64 // 0 fits the full schematic
	Grayscale             bool
	Metadata              string
	RasterDPI             int // 0 for vector PDF; 72–300 for PNG
}
type painter interface {
	push()
	pop()
	transform(float64, float64, float64)
	style(style)
	path(string) error
	polyline([]float64)
	ellipse(float64, float64, float64)
	rect(float64, float64, float64, float64)
	text(float64, float64, float64, string) error
	finish() ([]byte, error)
	close()
}
type style struct {
	fill, stroke string
	width, font  float64
}

func Render(ctx context.Context, svg []byte, page Page) ([]byte, error) {
	if !validPage(page) {
		return nil, fmt.Errorf("invalid schematic page geometry or raster resolution")
	}
	decoder := xml.NewDecoder(strings.NewReader(string(svg)))
	var out painter
	defer func() {
		if out != nil {
			out.close()
		}
	}()
	current := style{fill: "#000000", stroke: "none", width: 1, font: 14}
	var styles []style
	var text strings.Builder
	var textX, textY float64
	insideText := false
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			a := map[string]string{}
			for _, attr := range t.Attr {
				a[attr.Name.Local] = attr.Value
			}
			if t.Name.Local == "svg" {
				bounds, err := numbers(a["viewBox"])
				if err != nil || len(bounds) != 4 || bounds[2] <= 0 || bounds[3] <= 0 {
					return nil, fmt.Errorf("invalid generated schematic bounds")
				}
				scale := page.Scale
				if scale == 0 {
					scale = math.Min((page.Width-2*page.Margin)/bounds[2], (page.Height-2*page.Margin)/bounds[3])
				}
				if scale <= 0 || bounds[2]*scale > page.Width-2*page.Margin+1e-7 || bounds[3]*scale > page.Height-2*page.Margin+1e-7 {
					return nil, fmt.Errorf("schematic exceeds printable page at this scale; use fit or a smaller scale")
				}
				x := (page.Width-bounds[2]*scale)/2 - bounds[0]*scale
				y := (page.Height-bounds[3]*scale)/2 - bounds[1]*scale
				out, err = newPainter(page, scale, x, y)
				if err != nil {
					return nil, err
				}
			}
			if out == nil {
				return nil, fmt.Errorf("missing generated SVG root")
			}
			out.push()
			styles = append(styles, current)
			if v, ok := a["fill"]; ok {
				current.fill = v
			}
			if v, ok := a["stroke"]; ok {
				current.stroke = v
			}
			if v, ok := a["stroke-width"]; ok {
				current.width, err = strconv.ParseFloat(v, 64)
				if err != nil {
					return nil, err
				}
			}
			if v, ok := a["font-size"]; ok {
				current.font, err = strconv.ParseFloat(v, 64)
				if err != nil {
					return nil, err
				}
			}
			out.style(current)
			get := func(key string) float64 { v, _ := strconv.ParseFloat(a[key], 64); return v }
			switch t.Name.Local {
			case "g":
				if transform := a["transform"]; transform != "" {
					var x, y, rotation float64
					if _, err := fmt.Sscanf(transform, "translate(%g %g) rotate(%g)", &x, &y, &rotation); err != nil {
						return nil, fmt.Errorf("unsupported generated transform: %w", err)
					}
					out.transform(x, y, rotation)
				}
			case "rect":
				out.rect(get("x"), get("y"), get("width"), get("height"))
			case "circle":
				out.ellipse(get("cx"), get("cy"), get("r"))
			case "polyline":
				points, err := numbers(a["points"])
				if err != nil || len(points)%2 != 0 {
					return nil, fmt.Errorf("invalid generated polyline")
				}
				out.polyline(points)
			case "path":
				if err := out.path(a["d"]); err != nil {
					return nil, err
				}
			case "text":
				text.Reset()
				textX, textY = get("x"), get("y")
				insideText = true
			case "svg", "metadata", "title":
			default:
				return nil, fmt.Errorf("unsupported generated SVG element %q", t.Name.Local)
			}
		case xml.CharData:
			if insideText {
				text.Write(t)
			}
		case xml.EndElement:
			if t.Name.Local == "text" {
				if err := out.text(textX, textY, current.font, text.String()); err != nil {
					return nil, err
				}
				insideText = false
			}
			if len(styles) == 0 {
				return nil, fmt.Errorf("unbalanced generated SVG")
			}
			current = styles[len(styles)-1]
			styles = styles[:len(styles)-1]
			out.pop()
		}
	}
	if out == nil || len(styles) != 0 {
		return nil, fmt.Errorf("incomplete generated SVG")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return out.finish()
}

func validPage(page Page) bool {
	for _, v := range []float64{page.Width, page.Height, page.Margin, page.Scale} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	if page.Width <= 0 || page.Height <= 0 || page.Margin < 0 || page.Margin*2 >= math.Min(page.Width, page.Height) || page.Scale < 0 {
		return false
	}
	if page.RasterDPI == 0 {
		return true
	}
	w, h := rasterDimensions(page)
	return page.RasterDPI >= 72 && page.RasterDPI <= 300 && w > 0 && h > 0 && w*h <= 12_000_000
}

func rasterDimensions(page Page) (float64, float64) {
	return math.Ceil(page.Width * float64(page.RasterDPI) / 72), math.Ceil(page.Height * float64(page.RasterDPI) / 72)
}

func numbers(text string) ([]float64, error) {
	var out []float64
	for _, field := range strings.Fields(strings.ReplaceAll(text, ",", " ")) {
		v, err := strconv.ParseFloat(field, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("invalid generated number")
		}
		out = append(out, v)
	}
	return out, nil
}

var pathTokens = regexp.MustCompile(`[MLHVQZ]|[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?`)
