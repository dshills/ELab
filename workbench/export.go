package workbench

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ExportParams struct {
	ExpectedGeneration *uint64 `json:"expected_document_generation,omitempty"`
	Path               string  `json:"path"`
	Overwrite          bool    `json:"overwrite,omitempty"`
}

type Artifact struct {
	Path               string `json:"path"`
	Format             string `json:"format"`
	RunID              uint64 `json:"run_id"`
	ElectricalRevision uint64 `json:"electrical_revision"`
	Bytes              int    `json:"bytes"`
}

func number(v float64) string { return strconv.FormatFloat(v, 'g', 17, 64) }
func (s *Service) exportLocked(command string, p ExportParams) error {
	if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
		return err
	}
	path, err := documentPath(p.Path)
	if err != nil {
		return err
	}
	if path == s.documentPath {
		return fmt.Errorf("export must use a different path from the schematic")
	}
	var data []byte
	format := "svg"
	run := s.run
	if command == "export.scope.csv" {
		format = "csv"
		compiled, _, _ := s.graph.Compile()
		scope := s.scopeLocked(compiled)
		t := s.transient
		if t == nil || t.Stale || !scope.WindowValid || len(scope.Traces) == 0 || len(scope.Traces[0].Points) == 0 {
			return &actionError{"unavailable", "CSV needs selected traces in a fresh retained scope interval"}
		}
		var b bytes.Buffer
		writer := csv.NewWriter(&b)
		// Identity accompanies every row so extracted rows remain attributable.
		header := []string{"run_id", "electrical_revision", "engine", "window_start_s", "window_end_s", "sample_step_s", "time_s"}
		for _, trace := range scope.Traces {
			header = append(header, trace.Probe.ID+" ["+trace.Probe.Unit+"]")
		}
		if err := writer.Write(header); err != nil {
			return fileError(err)
		}
		for _, sample := range t.Samples {
			if !scopeTimeInside(sample.Time, scope.Window.Start, scope.Window.End) {
				continue
			}
			row := []string{strconv.FormatUint(t.RunID, 10), strconv.FormatUint(t.ElectricalRevision, 10), t.Engine, number(scope.Window.Start), number(scope.Window.End), number(t.Settings.StepSeconds), number(sample.Time)}
			for _, trace := range scope.Traces {
				value, ok := frameValue(sample, trace.Probe)
				text := ""
				if ok {
					text = number(value)
				}
				row = append(row, text)
			}
			if err := writer.Write(row); err != nil {
				return fileError(err)
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return fileError(err)
		}
		data = b.Bytes()
		run = t.RunID
	} else {
		data = schematicSVG(s.graph, s.run, s.electrical)
	}
	if len(data) > 8*1024*1024 {
		return fmt.Errorf("export exceeds 8 MiB")
	}
	if err := publishExport(path, data, p.Overwrite); err != nil {
		return err
	}
	s.lastExport = &Artifact{Path: path, Format: format, RunID: run, ElectricalRevision: s.electrical, Bytes: len(data)}
	return nil
}
func publishExport(path string, data []byte, overwrite bool) error {
	replace := false
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fileError(fmt.Errorf("export target must be a regular file"))
		}
		if !overwrite {
			return &actionError{"file-exists", "Export exists; choose another path or explicitly overwrite"}
		}
		replace = true
	} else if !os.IsNotExist(err) {
		return fileError(err)
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".elab-export-*")
	if err != nil {
		return fileError(err)
	}
	temp := f.Name()
	defer func() { _ = os.Remove(temp) }()
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fileError(err)
	}
	if replace {
		err = os.Rename(temp, path)
	} else {
		err = os.Link(temp, path)
	}
	if err != nil {
		return fileError(err)
	}
	if dir, err := os.Open(filepath.Dir(path)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}
func schematicSVG(g Graph, run, revision uint64) []byte {
	minX, minY, maxX, maxY := -120., -100., 120., 100.
	for _, p := range g.Components {
		minX = math.Min(minX, p.Position.X-90)
		maxX = math.Max(maxX, p.Position.X+140)
		minY = math.Min(minY, p.Position.Y-80)
		maxY = math.Max(maxY, p.Position.Y+95)
	}
	paths := map[string][]Point{}
	for _, w := range g.Wires {
		path := WirePath(g, w)
		paths[w.ID] = path
		for _, p := range path {
			minX = math.Min(minX, p.X-25)
			maxX = math.Max(maxX, p.X+25)
			minY = math.Min(minY, p.Y-25)
			maxY = math.Max(maxY, p.Y+25)
		}
	}
	diagramBottom := maxY
	maxY += 55 + float64((len(g.Components)+1)/2)*24
	minY -= 50
	width, height := maxX-minX, maxY-minY
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="%g" viewBox="%g %g %g %g"><title>ELab schematic</title>`, 1200*height/width, minX, minY, width, height)
	metadata, _ := json.Marshal(map[string]any{"format": "elab.schematic.svg.v1", "run_id": run, "electrical_revision": revision, "led_model": Model(), "npn_model": TransistorModel(), "note": "Generic educational models; schematic rendering, not physical build certification"})
	fmt.Fprintf(&b, "<metadata>%s</metadata>", html.EscapeString(string(metadata)))
	fmt.Fprintf(&b, `<rect x="%g" y="%g" width="%g" height="%g" fill="#ffffff"/><g fill="none" stroke="#426574" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">`, minX, minY, width, height)
	for _, w := range g.Wires {
		if !w.Connected {
			continue
		}
		fmt.Fprintf(&b, `<polyline id="%s" points="`, html.EscapeString(w.ID))
		for _, p := range paths[w.ID] {
			fmt.Fprintf(&b, "%g,%g ", p.X, p.Y)
		}
		fmt.Fprintf(&b, `"><title>%s → %s</title></polyline>`, html.EscapeString(w.From), html.EscapeString(w.To))
	}
	for _, p := range g.Components {
		fmt.Fprintf(&b, `<g id="%s" transform="translate(%g %g) rotate(%d)"><title>%s · %s</title>`, html.EscapeString(p.ID), p.Position.X, p.Position.Y, p.Rotation, html.EscapeString(p.ID), html.EscapeString(p.Name))
		switch p.Kind {
		case "resistor":
			b.WriteString(`<path d="M-55 0 H-34 L-28 -10 L-17 10 L-6 -10 L6 10 L17 -10 L28 10 L34 0 H55"/>`)
		case "capacitor":
			b.WriteString(`<path d="M0 -32 V-7 M-23 -7 H23 M-23 7 H23 M0 7 V32"/>`)
		case "led":
			b.WriteString(`<path d="M0 -32 V-12 M-12 -12 H12 L0 10 Z M-12 11 H12 M0 11 V32 M18 -5 L28 -15 M22 7 L32 -3 M23 -15 H28 V-10 M27 -3 H32 V2"/>`)
		case "npn":
			b.WriteString(`<circle r="25"/><path d="M-40 0 H-10 M-10 -14 V14 M-10 -10 L25 -35 M-10 10 L25 35 M15 31 L25 35 L21 25"/>`)
		case "dc-source", "pulse-source":
			b.WriteString(`<circle r="22"/><path d="M0 -34 V-22 M0 22 V34 M-6 -9 H6 M0 -15 V-3 M-6 10 H6"/>`)
			if p.Kind == "pulse-source" {
				b.WriteString(`<path d="M-15 5 V-3 H-5 V5 H5 V-3 H15"/>`)
			}
		case "ground":
			b.WriteString(`<path d="M0 -20 V0 M-20 0 H20 M-12 6 H12 M-5 12 H5"/>`)
		}
		b.WriteString(`</g>`)
		for _, pin := range p.Terminals {
			at, _ := TerminalPosition(p, pin)
			fmt.Fprintf(&b, `<circle cx="%g" cy="%g" r="3" fill="#ffffff"><title>%s.%s</title></circle>`, at.X, at.Y, html.EscapeString(p.ID), html.EscapeString(pin))
		}
	}
	b.WriteString(`</g><g fill="#1e2c3a" font-family="system-ui,sans-serif" font-size="14">`)
	fmt.Fprintf(&b, `<text x="%g" y="%g" font-size="19">ELab · schematic · electrical revision %d</text>`, minX+15, minY+25, revision)

	for _, p := range g.Components {
		x, y := p.Position.X+31, p.Position.Y-3
		text := p.ID
		if p.Unit != "" {
			text += " · " + displayPartValue(p)
		}
		switch p.Kind {
		case "resistor":
			if p.Rotation == 90 || p.Rotation == 270 {
				x, y = p.Position.X+17, p.Position.Y-2
			} else {
				x, y = p.Position.X-30, p.Position.Y+23
			}
		case "capacitor":
			if p.Rotation == 90 || p.Rotation == 270 {
				x, y = p.Position.X-35, p.Position.Y-23
			} else {
				x, y = p.Position.X+27, p.Position.Y-3
			}
		case "dc-source", "pulse-source":
			x, y = p.Position.X-88, p.Position.Y-2
		case "ground":
			x, y = p.Position.X+29, p.Position.Y+5
		}
		labelWidth := float64(len([]rune(text)))*7.5 + 8
		fmt.Fprintf(&b, `<rect x="%g" y="%g" width="%g" height="19" rx="3" fill="#ffffff"/><text x="%g" y="%g">%s</text>`, x-3, y-15, labelWidth, x, y, html.EscapeString(text))
	}
	fmt.Fprintf(&b, `<text x="%g" y="%g" font-size="15">Components · generic educational models</text>`, minX+15, diagramBottom+19)
	for i, p := range g.Components {
		x := minX + 15 + float64(i%2)*width/2
		y := diagramBottom + 44 + float64(i/2)*24
		text := p.ID + " · " + p.Name
		if p.Unit != "" {
			text += " · " + displayPartValue(p)
		}
		if p.InitialVoltage != nil {
			text += fmt.Sprintf(" · initial %.4g V", *p.InitialVoltage)
		}
		fmt.Fprintf(&b, `<text x="%g" y="%g" font-size="12">%s</text>`, x, y, html.EscapeString(text))
	}
	b.WriteString("</g></svg>\n")
	return []byte(b.String())
}

func displayPartValue(p Component) string {
	value, suffix := p.Value, p.Unit
	if p.Kind == "resistor" && value >= 1000 {
		value /= 1000
		suffix = "kΩ"
	}
	if p.Kind == "capacitor" {
		switch {
		case value < 1e-9:
			value *= 1e12
			suffix = "pF"
		case value < 1e-6:
			value *= 1e9
			suffix = "nF"
		case value < 1e-3:
			value *= 1e6
			suffix = "µF"
		case value < 1:
			value *= 1e3
			suffix = "mF"
		}
	}
	return fmt.Sprintf("%.4g %s", value, suffix)
}
