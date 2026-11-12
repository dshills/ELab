// Package schematicpdf renders ELab's own generated SVG primitives into PDF or PNG.
// It does not accept arbitrary imported SVG or execute embedded content.
package schematicpdf

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

type Page struct {
	Width, Height, Margin float64 // PDF points
	Scale                 float64 // 0 fits the full schematic
	Grayscale             bool
	Metadata              string
	RasterDPI             int // 0 for vector PDF; 72–300 for PNG
	Tile                  bool
	Overlap               float64 // PDF points
	PageIndex             *int    // PNG selection; nil means whole document, which must have one page
	Footer                string  // generated placement label
}
type painter interface {
	nextPage(Page, float64, float64, float64) error
	push()
	pop()
	transform(float64, float64, float64)
	style(style)
	draw(drawing)
	text(float64, float64, float64, string) error
	finish() ([]byte, error)
}
type style struct {
	fill, stroke string
	anchor       string
	width, font  float64
}

func Render(ctx context.Context, svg []byte, page Page) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	plan, err := Plan(svg, page)
	if err != nil {
		return nil, err
	}
	pages := plan.Pages
	if page.RasterDPI != 0 {
		if page.PageIndex == nil && len(pages) > 1 {
			return nil, fmt.Errorf("multi-page PNG requires explicit page_index; export PDF for all pages")
		}
		index := 0
		if page.PageIndex != nil {
			index = *page.PageIndex
		}
		if index < 0 || index >= len(pages) {
			return nil, fmt.Errorf("page_index is outside the captured page count")
		}
		pages = pages[index : index+1]
	} else if page.PageIndex != nil {
		return nil, fmt.Errorf("page_index applies only to PNG output")
	}
	var out painter
	for _, tile := range pages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		page.Footer = tile.Footer
		if out == nil {
			if page.RasterDPI == 0 {
				out, err = newPDF(ctx, page, tile.Scale, tile.X, tile.Y)
			} else {
				out, err = newPNG(ctx, page, tile.Scale, tile.X, tile.Y)
			}
		} else {
			err = out.nextPage(page, tile.Scale, tile.X, tile.Y)
		}
		if err != nil {
			return nil, err
		}
		if err = renderSVG(ctx, svg, out); err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := out.finish()
	if err == nil {
		err = ctx.Err()
	}
	return data, err
}

func renderSVG(ctx context.Context, svg []byte, out painter) error {
	decoder := xml.NewDecoder(strings.NewReader(string(svg)))
	current := style{fill: "#000000", stroke: "none", width: 1, font: 14}
	var styles []style
	var text strings.Builder
	var textX, textY float64
	insideText := false
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch t := token.(type) {
		case xml.StartElement:
			a := map[string]string{}
			for _, attr := range t.Attr {
				a[attr.Name.Local] = attr.Value
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
					return err
				}
			}
			if v, ok := a["font-size"]; ok {
				current.font, err = strconv.ParseFloat(v, 64)
				if err != nil {
					return err
				}
			}
			if v, ok := a["text-anchor"]; ok {
				if v != "start" && v != "end" {
					return fmt.Errorf("unsupported generated text anchor")
				}
				current.anchor = v
			}
			out.style(current)
			get := func(key string) float64 { v, _ := strconv.ParseFloat(a[key], 64); return v }
			switch t.Name.Local {
			case "g":
				if transform := a["transform"]; transform != "" {
					var x, y, rotation float64
					if _, err := fmt.Sscanf(transform, "translate(%g %g) rotate(%g)", &x, &y, &rotation); err != nil {
						return fmt.Errorf("unsupported generated transform: %w", err)
					}
					out.transform(x, y, rotation)
				}
			case "rect":
				out.draw(rectangle(get("x"), get("y"), get("width"), get("height")))
			case "circle":
				out.draw(circle(get("cx"), get("cy"), get("r")))
			case "polyline":
				points, err := numbers(a["points"])
				if err != nil || len(points)%2 != 0 {
					return fmt.Errorf("invalid generated polyline")
				}
				var d drawing
				for i := 0; i < len(points); i += 2 {
					p := point{points[i], points[i+1]}
					if i == 0 {
						d.move(p)
					} else {
						d.line(p)
					}
				}
				out.draw(d)
			case "path":
				d, err := parsePath(a["d"])
				if err != nil {
					return err
				}
				out.draw(d)
			case "text":
				text.Reset()
				textX, textY = get("x"), get("y")
				insideText = true
			case "svg", "metadata", "title":
			default:
				return fmt.Errorf("unsupported generated SVG element %q", t.Name.Local)
			}
		case xml.CharData:
			if insideText {
				text.Write(t)
			}
		case xml.EndElement:
			if t.Name.Local == "text" {
				if err := out.text(textX, textY, current.font, text.String()); err != nil {
					return err
				}
				insideText = false
			}
			if len(styles) == 0 {
				return fmt.Errorf("unbalanced generated SVG")
			}
			current = styles[len(styles)-1]
			styles = styles[:len(styles)-1]
			out.pop()
		}
	}
	if out == nil || len(styles) != 0 {
		return fmt.Errorf("incomplete generated SVG")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func validPage(page Page) bool {
	for _, v := range []float64{page.Width, page.Height, page.Margin, page.Scale, page.Overlap} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	if page.Overlap < 0 || page.Overlap > 20*72/25.4 {
		return false
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

const maxOutput = 8 << 20

type outputBuffer struct {
	bytes.Buffer
	ctx context.Context
}

func (b *outputBuffer) Write(data []byte) (int, error) {
	if err := b.ctx.Err(); err != nil {
		return 0, err
	}
	if len(data) > maxOutput-b.Len() {
		return 0, fmt.Errorf("schematic output exceeds export bounds")
	}
	return b.Buffer.Write(data)
}
