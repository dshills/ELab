package schematicpdf

import (
	"encoding/xml"
	"fmt"
	"math"
	"strings"
)

const MaxPages = 16
const tileFooterHeight = 18.

type Placement struct {
	Index, Row, Column int
	Scale, X, Y        float64
	Footer             string
}
type Layout struct {
	Rows, Columns int
	Pages         []Placement
}

// Plan covers the complete generated viewBox. Tiles advance by printable size
// minus overlap, in row-major order; all pages keep the requested scale.
func Plan(svg []byte, page Page) (Layout, error) {
	if !validPage(page) {
		return Layout{}, fmt.Errorf("invalid schematic page geometry or raster resolution")
	}
	decoder := xml.NewDecoder(strings.NewReader(string(svg)))
	var bounds []float64
	for {
		token, err := decoder.Token()
		if err != nil {
			return Layout{}, fmt.Errorf("missing generated SVG bounds: %w", err)
		}
		if start, ok := token.(xml.StartElement); ok {
			if start.Name.Local != "svg" {
				return Layout{}, fmt.Errorf("missing generated SVG root")
			}
			for _, a := range start.Attr {
				if a.Name.Local == "viewBox" {
					bounds, err = numbers(a.Value)
					if err != nil {
						return Layout{}, err
					}
				}
			}
			break
		}
	}
	if len(bounds) != 4 || bounds[2] <= 0 || bounds[3] <= 0 {
		return Layout{}, fmt.Errorf("invalid generated schematic bounds")
	}
	width, height := page.Width-2*page.Margin, page.Height-2*page.Margin
	scale := page.Scale
	if !page.Tile {
		if scale == 0 {
			scale = math.Min(width/bounds[2], height/bounds[3])
		}
		if scale <= 0 || bounds[2]*scale > width+1e-7 || bounds[3]*scale > height+1e-7 {
			return Layout{}, fmt.Errorf("schematic exceeds printable page at this scale; use fit, a smaller scale or tiled pages")
		}
		return Layout{Rows: 1, Columns: 1, Pages: []Placement{{Scale: scale, X: (page.Width-bounds[2]*scale)/2 - bounds[0]*scale, Y: (page.Height-bounds[3]*scale)/2 - bounds[1]*scale}}}, nil
	}
	height -= tileFooterHeight
	if scale <= 0 || height <= page.Overlap || width <= page.Overlap {
		return Layout{}, fmt.Errorf("tiled pages require an explicit scale and overlap smaller than printable content")
	}
	count := func(extent, window float64) float64 {
		if extent <= window+1e-7 {
			return 1
		}
		return math.Ceil((extent-window-1e-7)/(window-page.Overlap)) + 1
	}
	cols, rows := count(bounds[2]*scale, width), count(bounds[3]*scale, height)
	if !isFinite(cols) || !isFinite(rows) || cols*rows > MaxPages {
		return Layout{}, fmt.Errorf("schematic needs more than %d tiled pages; reduce scale", MaxPages)
	}
	out := Layout{Rows: int(rows), Columns: int(cols)}
	for row := 0; row < out.Rows; row++ {
		for col := 0; col < out.Columns; col++ {
			index := len(out.Pages)
			out.Pages = append(out.Pages, Placement{Index: index, Row: row, Column: col, Scale: scale, X: page.Margin - bounds[0]*scale - float64(col)*(width-page.Overlap), Y: page.Margin - bounds[1]*scale - float64(row)*(height-page.Overlap), Footer: fmt.Sprintf("ELab | %d/%d | row %d/%d col %d/%d | scale %.6g%% | overlap %.4g mm", index+1, int(rows*cols), row+1, out.Rows, col+1, out.Columns, scale*100, page.Overlap*25.4/72)})
		}
	}
	return out, nil
}
func isFinite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
