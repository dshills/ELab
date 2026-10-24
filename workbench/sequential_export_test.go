package workbench

import (
	"bytes"
	"context"
	"encoding/xml"
	"image/png"
	"io"
	"testing"

	"github.com/dshills/elab/internal/schematicpdf"
)

func TestSequentialSymbolsExportAtEveryRotation(t *testing.T) {
	for _, kind := range sequentialKinds {
		p, _ := MakeComponent("U1", kind, Point{}, "")
		for _, rotation := range []int{0, 90, 180, 270} {
			p.Rotation = rotation
			decoder := xml.NewDecoder(bytes.NewReader(schematicSVG(Graph{Components: []Component{p}}, 1, 1)))
			for {
				_, err := decoder.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(kind, rotation, err)
				}
			}
		}
	}
}
func TestSequentialControlClosedSwitchRasterContinuity(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("native renderer")
	}
	svg := []byte(`<svg viewBox="0 0 100 150"><g fill="none" stroke="#000000" stroke-width="3" transform="translate(50 75) rotate(90)"><path d="M-55 0 H-25 M25 0 H55 M-25 0 L25 0"/></g></svg>`)
	data, err := schematicpdf.Render(context.Background(), svg, schematicpdf.Page{Width: 100, Height: 150, RasterDPI: 72})
	if err != nil {
		t.Fatal(err)
	}
	im, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	for y := 25; y <= 125; y += 10 {
		r, _, _, _ := im.At(50, y).RGBA()
		if r > 30000 {
			t.Fatal("closed reset/enable switch lost its contact stroke", y, r)
		}
	}
}
