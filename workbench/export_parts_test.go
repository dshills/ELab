package workbench

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"
)

type exportedPartLine struct {
	x, y float64
	text string
}

func exportedParts(t *testing.T, data []byte) ([4]float64, []exportedPartLine) {
	t.Helper()
	var bounds [4]float64
	var lines []exportedPartLine
	decoder := xml.NewDecoder(bytes.NewReader(data))
	inParts := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local == "svg" {
			for _, a := range start.Attr {
				if a.Name.Local == "viewBox" {
					if _, err = fmt.Sscanf(a.Value, "%g %g %g %g", &bounds[0], &bounds[1], &bounds[2], &bounds[3]); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
		if start.Name.Local != "text" {
			continue
		}
		var text string
		if err = decoder.DecodeElement(&text, &start); err != nil {
			t.Fatal(err)
		}
		if text == "Components · generic educational models" {
			inParts = true
			continue
		}
		if !inParts {
			continue
		}
		line := exportedPartLine{text: text}
		for _, a := range start.Attr {
			switch a.Name.Local {
			case "x":
				if _, err := fmt.Sscan(a.Value, &line.x); err != nil {
					t.Fatal(err)
				}
			case "y":
				if _, err := fmt.Sscan(a.Value, &line.y); err != nil {
					t.Fatal(err)
				}
			}
		}
		lines = append(lines, line)
	}
	return bounds, lines
}

func TestMotorExportRetainsParametersAndPartsStayWithinColumns(t *testing.T) {
	for _, pwm := range []bool{false, true} {
		for _, locked := range []bool{false, true} {
			t.Run(fmt.Sprintf("pwm=%t/locked=%t", pwm, locked), func(t *testing.T) {
				g := MotorExampleGraph(pwm)
				for i, p := range g.Components {
					if p.Kind == "dc-motor" {
						g.Components[i].Power.Motor = &MotorConfig{Locked: locked, Resistance: 7.3, Inductance: .0021, Constant: .017, Inertia: 3e-6, Friction: 2e-5, Load: 4e-4}
					}
				}
				bounds, lines := exportedParts(t, schematicSVG(g, 1, 1))
				var texts []string
				columns := map[float64]bool{}
				for _, line := range lines {
					texts = append(texts, line.text)
					columns[line.x] = true
				}
				joined := strings.Join(texts, " ")
				for _, want := range []string{"R 7.3 Ω", "L 0.0021 H", "Ke=Kt 0.017 (V·s/rad, N·m/A)", "J 3e-06 kg·m²", "friction 2e-05 N·m·s/rad", "load 0.0004 N·m·s/rad"} {
					if !strings.Contains(joined, want) {
						t.Fatalf("printed motor configuration missing %q", want)
					}
				}
				want := "rotor free"
				if locked {
					want = "rotor locked"
				}
				if !strings.Contains(joined, want) {
					t.Fatal("missing rotor mode", want)
				}
				var xs []float64
				for x := range columns {
					xs = append(xs, x)
				}
				sort.Float64s(xs)
				lastY := map[float64]float64{}
				for _, line := range lines {
					right := bounds[0] + bounds[2] - 15
					for _, x := range xs {
						if x > line.x {
							right = x - 30
							break
						}
					}
					// One em per character is deliberately more conservative than this
					// 12-point sans-serif text's rendered advance, including unit glyphs.
					if line.x+12*float64(utf8.RuneCountInString(line.text)) > right+1e-9 || line.y+4 > bounds[1]+bounds[3] {
						t.Fatal("parts text crosses its column/page", line, right, bounds)
					}
					if previous, ok := lastY[line.x]; ok && line.y-previous < 16 {
						t.Fatal("parts lines overlap", line, previous)
					}
					lastY[line.x] = line.y
				}
			})
		}
	}
}
