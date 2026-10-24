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
	PageIndex          *int           `json:"page_index,omitempty"`
	DPI                int            `json:"dpi,omitempty"`
	Page               *SchematicPage `json:"page,omitempty"`
	DocumentGeneration uint64         `json:"document_generation,omitempty"`
	SHA256             string         `json:"sha256,omitempty"`
	OperationID        string         `json:"operation_id,omitempty"`
	Path               string         `json:"path"`
	Format             string         `json:"format"`
	RunID              uint64         `json:"run_id"`
	ElectricalRevision uint64         `json:"electrical_revision"`
	Bytes              int            `json:"bytes"`
}

func cloneArtifact(in *Artifact) *Artifact {
	if in == nil {
		return nil
	}
	out := *in
	if in.Page != nil {
		page := cloneSchematicPage(*in.Page)
		out.Page = &page
	}
	if in.PageIndex != nil {
		v := *in.PageIndex
		out.PageIndex = &v
	}
	return &out
}

func number(v float64) string { return strconv.FormatFloat(v, 'g', 17, 64) }
func encodeScopeCSV(t *Transient, scope Scope) ([]byte, error) {
	var b bytes.Buffer
	writer := csv.NewWriter(&b)
	// Identity accompanies every row so extracted rows remain attributable.
	header := []string{"run_id", "electrical_revision", "engine", "window_start_s", "window_end_s", "sample_step_s", "time_s"}
	for _, trace := range scope.Traces {
		header = append(header, trace.Probe.ID+" ["+trace.Probe.Unit+"]")
	}
	if err := writer.Write(header); err != nil {
		return nil, fileError(err)
	}
	for _, sample := range t.Samples {
		if !scopeWindowTimeInside(sample.Time, scope.Window.Start, scope.Window.End) {
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
			return nil, fileError(err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fileError(err)
	}
	return b.Bytes(), nil
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
func schematicSVG(g Graph, run, revision uint64, readings ...PinnedReading) []byte {
	return schematicSVGWithDisplay(g, run, revision, CircuitDisplay{Preferences: DefaultDisplayPreferences()}, readings...)
}
func schematicSVGWithDisplay(g Graph, run, revision uint64, display CircuitDisplay, readings ...PinnedReading) []byte {
	settings := DefaultTransientSettings()
	return schematicSVGWithAnnotations(g, run, revision, display, configuredSourceAnnotations(g, settings), settings, readings...)
}
func schematicSVGWithAnnotations(g Graph, run, revision uint64, display CircuitDisplay, annotations []SourceAnnotation, settings TransientSettings, readings ...PinnedReading) []byte {
	annotated := map[string]bool{}
	for _, a := range annotations {
		annotated[a.ID] = true
	}
	minX, minY, maxX, maxY := -120., -100., 120., 100.
	for _, p := range g.Components {
		minX = math.Min(minX, p.Position.X-90)
		maxX = math.Max(maxX, p.Position.X+140)
		if p.Kind == "net-label" {
			maxX = math.Max(maxX, p.Position.X+50+float64(len(p.Label)+len(p.ID)+3)*7.5)
		}
		if !annotated[p.ID] {
			x, y, text := schematicPartLabel(p)
			minX, maxX = math.Min(minX, x-10), math.Max(maxX, x+schematicLabelWidth(text)+10)
			minY, maxY = math.Min(minY, y-25), math.Max(maxY, y+10)
		}
		if IsDisplayKind(p.Kind) {
			minX = math.Min(minX, p.Position.X-140)
			minY = math.Min(minY, p.Position.Y-250)
			maxY = math.Max(maxY, p.Position.Y+250)
		}
		if p.Kind == "relay" {
			minX = math.Min(minX, p.Position.X-140)
			maxY = math.Max(maxY, p.Position.Y+140)
		}
		minY = math.Min(minY, p.Position.Y-80)
		maxY = math.Max(maxY, p.Position.Y+95)
	}
	for _, a := range annotations {
		min, max := SourceAnnotationBounds(a)
		minX, minY = math.Min(minX, min.X-10), math.Min(minY, min.Y-10)
		maxX, maxY = math.Max(maxX, max.X+10), math.Max(maxY, max.Y+10)
	}
	connected := map[string]bool{}
	for _, wire := range g.Wires {
		if wire.Connected {
			connected[wire.From] = true
			connected[wire.To] = true
		}
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
	for _, reading := range readings {
		min, max := PinnedReadingBounds(reading)
		minX, minY = math.Min(minX, min.X-25), math.Min(minY, min.Y-25)
		maxX, maxY = math.Max(maxX, max.X+25), math.Max(maxY, max.Y+25)
	}
	diagramBottom := maxY
	maxY += 55 + float64((len(g.Components)+1)/2)*24
	minY -= 50
	width, height := maxX-minX, maxY-minY
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="%g" viewBox="%g %g %g %g"><title>ELab schematic</title>`, 1200*height/width, minX, minY, width, height)
	sources := []Component{}
	for _, p := range g.Components {
		if annotated[p.ID] {
			sources = append(sources, p)
		}
	}
	metadataFields := map[string]any{"format": "elab.schematic.svg.v1", "run_id": run, "electrical_revision": revision, "led_model": Model(), "npn_model": TransistorModel(), "pnp_model": PNPDeviceModel(), "switch_model": SwitchDeviceModel(), "nmos_model": NMOSDeviceModel(), "pmos_model": PMOSDeviceModel(), "diode_model": SiliconDiodeModel(), "zener_model": ZenerDeviceModel(), "schottky_model": SchottkyDeviceModel(), "potentiometer_model": LinearPotentiometerModel(), "waveform_model": SourceWaveformModel(), "source_annotations": annotations, "source_components": sources, "source_settings": settings, "display": display, "pinned_readings": readings, "note": "Generic educational models; schematic rendering, not physical build certification"}
	if model := timer555ModelForGraph(g); model != nil {
		metadataFields["timer555_model"] = model
	}
	if model := logicModelForGraph(g); model != nil {
		metadataFields["logic_model"] = model
	}
	if model := controlModelForGraph(g); model != nil {
		metadataFields["control_model"] = model
	}
	if model := regulatorModelForGraph(g); model != nil {
		metadataFields["regulator_model"] = model
	}
	if model := displayModelForGraph(g); model != nil {
		metadataFields["display_model"] = model
	}
	if model := sequentialModelForGraph(g); model != nil {
		metadataFields["sequential_model"] = model
	}
	if model := opAmpModelForGraph(g); model != nil {
		metadataFields["opamp_model"] = model
	}
	metadata, _ := json.Marshal(metadataFields)
	fmt.Fprintf(&b, "<metadata>%s</metadata>", html.EscapeString(string(metadata)))
	fmt.Fprintf(&b, `<rect x="%g" y="%g" width="%g" height="%g" fill="#ffffff"/><g fill="none" stroke="#426574" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">`, minX, minY, width, height)
	wireDisplays := map[string]WireDisplay{}
	for _, wire := range display.Wires {
		wireDisplays[wire.ID] = wire
	}
	for _, w := range g.Wires {
		if !w.Connected {
			continue
		}
		color := "#387d93"
		if reading, ok := wireDisplays[w.ID]; ok {
			color = reading.Color
		}
		fmt.Fprintf(&b, `<polyline id="%s" stroke="%s" points="`, html.EscapeString(w.ID), color)
		for _, p := range paths[w.ID] {
			fmt.Fprintf(&b, "%g,%g ", p.X, p.Y)
		}
		fmt.Fprintf(&b, `"><title>%s → %s</title></polyline>`, html.EscapeString(w.From), html.EscapeString(w.To))
	}
	// Label backgrounds hide passing wires, but must stay below symbols/pins.
	// Conservative text bounds can reach a nearby component even when the
	// actual label does not; drawing masks last erased parts of its symbol.
	for _, p := range g.Components {
		if annotated[p.ID] {
			continue
		}
		x, y, text := schematicPartLabel(p)
		fmt.Fprintf(&b, `<rect x="%g" y="%g" width="%g" height="19" rx="3" fill="#ffffff" stroke="none"/>`, x-3, y-15, schematicLabelWidth(text))
	}
	for _, a := range annotations {
		min, max := SourceAnnotationBounds(a)
		fmt.Fprintf(&b, `<rect x="%g" y="%g" width="%g" height="%g" rx="3" fill="#ffffff" stroke="none"/>`, min.X, min.Y, max.X-min.X, max.Y-min.Y)
	}
	for _, p := range g.Components {
		fmt.Fprintf(&b, `<g id="%s" transform="translate(%g %g) rotate(%d)"><title>%s · %s</title>`, html.EscapeString(p.ID), p.Position.X, p.Position.Y, p.Rotation, html.EscapeString(p.ID), html.EscapeString(p.Name))
		switch p.Kind {
		case "linear-regulator", "ldo-regulator":
			for _, path := range RegulatorSymbolPaths() {
				b.WriteString(`<polyline points="`)
				for _, point := range path {
					fmt.Fprintf(&b, "%g,%g ", point.X, point.Y)
				}
				b.WriteString(`"/>`)
			}
		case "relay", "comparator", "pushbutton", "bouncing-pushbutton":
			for _, path := range ControlSymbolPaths(p.Kind, ControlMemory{}) {
				b.WriteString(`<polyline points="`)
				for _, point := range path {
					fmt.Fprintf(&b, "%g,%g ", point.X, point.Y)
				}
				b.WriteString(`"/>`)
			}
		case "switch":
			endY := -22.
			if p.SwitchClosed != nil && *p.SwitchClosed {
				endY = 0
			}
			fmt.Fprintf(&b, `<path d="M-55 0 H-25 M25 0 H55 M-25 0 L25 %g"/>`, endY)
		case "potentiometer":
			b.WriteString(`<path d="M-55 0 H-34 L-28 -10 L-17 10 L-6 -10 L6 10 L17 -10 L28 10 L34 0 H55"/>`)
			contact := 28 * (1 - 2*(*p.WiperPosition))
			fmt.Fprintf(&b, `<path d="M0 -45 L%g -12 L%g -20 M%g -12 L%g -20"/>`, contact, contact-5, contact, contact+5)
		case "resistor":
			b.WriteString(`<path d="M-55 0 H-34 L-28 -10 L-17 10 L-6 -10 L6 10 L17 -10 L28 10 L34 0 H55"/>`)
		case "inductor":
			b.WriteString(`<polyline points="`)
			for _, point := range InductorPolyline() {
				fmt.Fprintf(&b, "%g,%g ", point.X, point.Y)
			}
			b.WriteString(`"/>`)
		case "capacitor":
			b.WriteString(`<path d="M0 -32 V-7 M-23 -7 H23 M-23 7 H23 M0 7 V32"/>`)
		case "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor":
			for _, path := range LogicSymbolPaths(p.Kind) {
				b.WriteString(`<polyline points="`)
				for _, point := range path {
					fmt.Fprintf(&b, "%g,%g ", point.X, point.Y)
				}
				b.WriteString(`"/>`)
			}
		case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter":
			for _, path := range SequentialSymbolPaths(p.Kind) {
				b.WriteString(`<polyline points="`)
				for _, point := range path {
					fmt.Fprintf(&b, "%g,%g ", point.X, point.Y)
				}
				b.WriteString(`"/>`)
			}
		case "bcd-decoder", "seven-segment":
			for _, path := range DisplaySymbolPaths(p.Kind) {
				b.WriteString(`<polyline points="`)
				for _, point := range path {
					fmt.Fprintf(&b, "%g,%g ", point.X, point.Y)
				}
				b.WriteString(`"/>`)
			}
			if p.Kind == "seven-segment" {
				for _, path := range DisplaySegmentPaths() {
					b.WriteString(`<polyline stroke-width="7" points="`)
					for _, point := range path {
						fmt.Fprintf(&b, "%g,%g ", point.X, point.Y)
					}
					b.WriteString(`"/>`)
				}
			}

		case "timer555":
			for _, path := range Timer555SymbolPaths() {
				b.WriteString(`<polyline points="`)
				for _, point := range path {
					fmt.Fprintf(&b, "%g,%g ", point.X, point.Y)
				}
				b.WriteString(`"/>`)
			}
		case "opamp":
			for _, path := range OpAmpSymbolPaths() {
				b.WriteString(`<polyline points="`)
				for _, point := range path {
					fmt.Fprintf(&b, "%g,%g ", point.X, point.Y)
				}
				b.WriteString(`"/>`)
			}
		case "diode", "zener", "schottky":
			b.WriteString(`<path d="M0 -32 V-12 M-12 -12 H12 L0 10 Z M0 11 V32"/>`)
			switch p.Kind {
			case "zener":
				b.WriteString(`<path d="M-12 5 V11 H12 V17"/>`)
			case "schottky":
				b.WriteString(`<path d="M-7 5 H-12 V11 H12 V17 H7"/>`)
			default:
				b.WriteString(`<path d="M-12 11 H12"/>`)
			}
		case "led":
			b.WriteString(`<path d="M0 -32 V-12 M-12 -12 H12 L0 10 Z M-12 11 H12 M0 11 V32 M18 -5 L28 -15 M22 7 L32 -3 M23 -15 H28 V-10 M27 -3 H32 V2"/>`)
		case "nmos", "pmos":
			b.WriteString(`<path d="M-40 0 H-16 M-16 -20 V20 M-7 -20 V-12 M-7 -4 V4 M-7 12 V20 M-7 -16 H25 V-35 M-7 16 H25 V35 M-7 0 H25 V16 "/>`)
			if p.Kind == "pmos" {
				b.WriteString(`<path d="M-2 -5 L7 0 L-2 5"/>`)
			} else {
				b.WriteString(`<path d="M4 -5 L-5 0 L4 5"/>`)
			}
		case "npn", "pnp":
			b.WriteString(`<circle r="25"/><path d="M-40 0 H-10 M-10 -14 V14 M-10 -10 L25 -35 M-10 10 L25 35"/>`)
			if p.Kind == "pnp" {
				b.WriteString(`<path d="M12 -18 L0 -17 L5 -28"/>`)
			} else {
				b.WriteString(`<path d="M15 31 L25 35 L21 25"/>`)
			}
		case "dc-source", "pulse-source", "sine-source", "finite-pulse-source":
			b.WriteString(`<circle r="22"/><path d="M0 -34 V-22 M0 22 V34 M-6 -9 H6 M0 -15 V-3 M-6 10 H6"/>`)
			if p.Kind == "sine-source" {
				b.WriteString(`<path d="M-15 0 Q-7 -16 0 0 Q7 16 15 0"/>`)
			} else if p.Kind == "finite-pulse-source" {
				b.WriteString(`<path d="M-15 5 L-10 -5 H-3 L3 5 H10 L15 -5"/>`)
			} else if p.Kind == "pulse-source" {
				b.WriteString(`<path d="M-15 5 V-3 H-5 V5 H5 V-3 H15"/>`)
			}
		case "junction":
			fill := "none"
			if connected[p.ID+".node"] {
				fill = "#426574"
			}
			fmt.Fprintf(&b, `<circle r="4" fill="%s"/>`, fill)
		case "net-label":
			b.WriteString(`<path d="M0 0 H10 V-7 L18 0 L10 7 V0"/>`)
		case "ground":
			b.WriteString(`<path d="M0 -20 V0 M-20 0 H20 M-12 6 H12 M-5 12 H5"/>`)
		}
		b.WriteString(`</g>`)
		for _, pin := range p.Terminals {
			at, _ := TerminalPosition(p, pin)
			fill := "#ffffff"
			if p.Kind == "junction" && connected[p.ID+".node"] {
				fill = "#426574"
			}
			fmt.Fprintf(&b, `<circle cx="%g" cy="%g" r="3" fill="%s"><title>%s.%s</title></circle>`, at.X, at.Y, fill, html.EscapeString(p.ID), html.EscapeString(pin))
		}
	}
	b.WriteString(`</g><g fill="#1e2c3a" font-family="system-ui,sans-serif" font-size="14">`)
	fmt.Fprintf(&b, `<text x="%g" y="%g" font-size="19">ELab · schematic · electrical revision %d</text>`, minX+15, minY+25, revision)

	for _, p := range g.Components {
		if IsLogicKind(p.Kind) {
			for _, label := range LogicSymbolLabels(p) {
				fmt.Fprintf(&b, `<text x="%g" y="%g" font-size="10" transform="rotate(%d %g %g)">%s</text>`, label.At.X, label.At.Y, p.Rotation, label.At.X, label.At.Y, html.EscapeString(label.Text))
			}
		}
		if IsSequentialKind(p.Kind) {
			for _, label := range SequentialSymbolLabels(p) {
				fmt.Fprintf(&b, `<text x="%g" y="%g" font-size="10" transform="rotate(%d %g %g)">%s</text>`, label.At.X, label.At.Y, p.Rotation, label.At.X, label.At.Y, html.EscapeString(label.Text))
			}
		}
		if IsControlKind(p.Kind) || IsRegulatorKind(p.Kind) {
			for _, label := range append(ControlSymbolLabels(p), regulatorLabels(p)...) {
				fmt.Fprintf(&b, `<text x="%g" y="%g" font-size="10">%s</text>`, label.At.X, label.At.Y, html.EscapeString(label.Text))
			}
		}
		if IsDisplayKind(p.Kind) {
			for _, label := range DisplaySymbolLabels(p) {
				fmt.Fprintf(&b, `<text x="%g" y="%g" font-size="10" transform="rotate(%d %g %g)">%s</text>`, label.At.X, label.At.Y, p.Rotation, label.At.X, label.At.Y, html.EscapeString(label.Text))
			}
		}
		if p.Kind == "timer555" {
			for _, label := range Timer555SymbolLabels(p) {
				fmt.Fprintf(&b, `<text x="%g" y="%g" font-size="10" transform="rotate(%d %g %g)">%s</text>`, label.At.X, label.At.Y, p.Rotation, label.At.X, label.At.Y, html.EscapeString(label.Text))
			}
		}
		if p.Kind == "opamp" {
			for _, label := range OpAmpSupplyLabels(p) {
				fmt.Fprintf(&b, `<text x="%g" y="%g" font-size="12">%s</text>`, label.At.X, label.At.Y, html.EscapeString(label.Text))
			}
		}
		if annotated[p.ID] {
			continue
		}
		x, y, text := schematicPartLabel(p)
		fmt.Fprintf(&b, `<text x="%g" y="%g">%s</text>`, x, y, html.EscapeString(text))
	}
	for _, a := range annotations {
		_, max := SourceAnnotationBounds(a)
		for i, line := range a.Lines {
			fmt.Fprintf(&b, `<text x="%g" y="%g" text-anchor="end">%s</text>`, max.X, a.Position.Y+float64(i)*SourceAnnotationLineHeight, html.EscapeString(line))
		}
	}
	fmt.Fprintf(&b, `<text x="%g" y="%g" font-size="15">Components · generic educational models</text>`, minX+15, diagramBottom+19)
	for i, p := range g.Components {
		x := minX + 15 + float64(i%2)*width/2
		y := diagramBottom + 44 + float64(i/2)*24
		text := p.ID + " · " + p.Name
		if p.Kind == "net-label" {
			text += " · " + p.Label
		}
		if p.Unit != "" {
			text += " · " + displayPartValue(p)
		}
		if p.InitialVoltage != nil {
			text += fmt.Sprintf(" · initial %.4g V", *p.InitialVoltage)
		}
		if p.InitialCurrent != nil {
			text += fmt.Sprintf(" · initial %.4g A", *p.InitialCurrent)
		}
		fmt.Fprintf(&b, `<text x="%g" y="%g" font-size="12">%s</text>`, x, y, html.EscapeString(text))
	}
	b.WriteString("</g>")
	for _, marker := range MakeDisplayFrame(g, display, 0).Markers {
		at, dx, dy := marker.At, marker.DX, marker.DY
		fmt.Fprintf(&b, `<polyline data-current-wire="%s" fill="none" stroke="#76439e" stroke-width="2" points="%g,%g %g,%g %g,%g"><title>Illustrative %s direction; signed readings use conventional current</title></polyline>`, html.EscapeString(marker.WireID), at.X-7*dx-5*dy, at.Y-7*dy+5*dx, at.X+4*dx, at.Y+4*dy, at.X-7*dx+5*dy, at.Y-7*dy-5*dx, display.Preferences.Direction)
	}
	for _, reading := range readings {
		min, max := PinnedReadingBounds(reading)
		color := "#276e7e"
		if reading.State != "live" {
			color = "#97602b"
		}
		fmt.Fprintf(&b, `<g id="pin-%s"><title>%s; %s → %s; %s</title><rect x="%g" y="%g" width="%g" height="%g" fill="#f3fafb" stroke="%s"/><text x="%g" y="%g" font-family="monospace" font-size="14" fill="%s">%s</text><text x="%g" y="%g" font-family="monospace" font-size="14" fill="%s">%s</text></g>`, html.EscapeString(reading.Definition.ID), html.EscapeString(reading.Probe.Label), html.EscapeString(reading.Probe.Positive), html.EscapeString(reading.Probe.Reference), html.EscapeString(reading.Reason), min.X, min.Y, max.X-min.X, max.Y-min.Y, color, min.X+12, min.Y+23, color, html.EscapeString(reading.Display), min.X+12, min.Y+43, color, html.EscapeString(reading.ReferenceDisplay))
	}
	b.WriteString("</svg>\n")
	return []byte(b.String())
}

func displayPartValue(p Component) string {
	if p.Kind == "inductor" {
		return InductorValueLabel(p.Value)
	}
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

// A conservative one-em advance keeps arbitrary valid IDs inside the exported
// bounds without depending on a particular workstation's system font metrics.
func schematicLabelWidth(text string) float64 { return float64(len([]rune(text)))*14 + 8 }
func schematicPartLabel(p Component) (float64, float64, string) {
	x, y := p.Position.X+31, p.Position.Y-3
	text := p.ID
	if p.Kind == "net-label" {
		text = p.Label + " · " + p.ID
	}
	if p.Unit != "" {
		text += " · " + displayPartValue(p)
	}
	switch p.Kind {
	case "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor":
		text += " · " + strings.ToUpper(strings.TrimPrefix(p.Kind, "logic-"))
		at := LogicLabelPosition(p)
		x, y = at.X, at.Y
	case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter":
		text += " · " + SequentialShortName(p.Kind)
		at := SequentialLabelPosition(p)
		x, y = at.X, at.Y
	case "bcd-decoder", "seven-segment":
		text += " · " + p.Name
		x, y = p.Position.X+130, p.Position.Y-245
	case "timer555":
		text += " · 555"
		at := Timer555LabelPosition(p)
		x, y = at.X, at.Y
	case "linear-regulator", "ldo-regulator":
		return p.Position.X + 125, p.Position.Y - 65, p.ID + " · " + FormatValue(p.Value, "V")
	case "relay":
		x, y = p.Position.X+125, p.Position.Y-110
	case "comparator":
		text += " hysteresis"
		x, y = p.Position.X-80, p.Position.Y-65
	case "opamp":
		at := OpAmpLabelPosition(p)
		x, y = at.X, at.Y
	case "inductor":
		at := InductorLabelPosition(p)
		x, y = at.X, at.Y
	case "led":
		at := LEDLabelPosition(p)
		x, y = at.X, at.Y
	case "nmos", "pmos", "npn", "pnp":
		if p.Rotation == 180 {
			x = p.Position.X - schematicLabelWidth(text) - 31
		}
	case "resistor":
		if p.Rotation == 90 || p.Rotation == 270 {
			x, y = p.Position.X+17, p.Position.Y-2
		} else {
			x, y = p.Position.X-30, p.Position.Y+23
		}
	case "capacitor":
		at := CapacitorLabelPosition(p)
		x, y = at.X, at.Y
	case "dc-source", "pulse-source", "sine-source", "finite-pulse-source":
		x, y = p.Position.X-schematicLabelWidth(text)-32, p.Position.Y-2
	case "junction":
		x, y = p.Position.X+8, p.Position.Y-8
	case "net-label":
		x, y = p.Position.X+22, p.Position.Y+4
	case "ground":
		at := GroundLabelPosition(p)
		x, y = at.X, at.Y
	}
	return x, y, text
}
