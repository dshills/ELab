package workbench

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSourceAnnotationsDescribeConfiguredWaveformsAndLegacyTiming(t *testing.T) {
	sine, _ := MakeComponent("Tone", "sine-source", Point{}, "")
	sine.Sine = &SineWaveform{Amplitude: 2.5, Offset: -1, Frequency: 123.456, Phase: -90}
	pulse, _ := MakeComponent("Ramp", "finite-pulse-source", Point{300, 0}, "")
	pulse.Pulse = &PulseWaveform{LowVoltage: -2, HighVoltage: 4, HighSeconds: .002, LowSeconds: .003, RiseSeconds: 1e-6, FallSeconds: .0005}
	legacy, _ := MakeComponent("Square", "pulse-source", Point{600, 0}, "5")
	cfg := DefaultTransientSettings()
	cfg.HighSeconds, cfg.LowSeconds = .123456, 2
	g := Graph{Components: []Component{sine, pulse, legacy}}
	got := configuredSourceAnnotations(g, cfg)
	want := [][]string{{"Tone | sine", "2.5 Vpk | 123.456 Hz", "offset -1 V | phase -90 deg"}, {"Ramp | pulse", "low -2 V | high 4 V", "high 2 ms | low 3 ms", "rise 1 us | fall 500 us"}, {"Square | pulse (ideal edges)", "low 0 V | high 5 V", "high 123.456 ms | low 2 s"}}
	if len(got) != len(want) {
		t.Fatal("incomplete source annotations")
	}
	for i, a := range got {
		if !reflect.DeepEqual(a.Lines, want[i]) || a.State != "configured" {
			t.Fatal("incorrect waveform wording/units", a)
		}
		min, max := SourceAnnotationBounds(a)
		if !finite(min.X) || !finite(min.Y) || max.X > g.Components[i].Position.X-39 || max.Y <= min.Y {
			t.Fatal("annotation overlaps source or has invalid bounds")
		}
	}
}

func TestSourceAnnotationBlocksClearEveryRotatedSourceAndItsTerminalStubs(t *testing.T) {
	for _, kind := range []string{"sine-source", "finite-pulse-source", "pulse-source"} {
		for _, rotation := range []int{0, 90, 180, 270} {
			p, err := MakeComponent(strings.Repeat("V", 32), kind, Point{100, 40}, "")
			if err != nil {
				t.Fatal(err)
			}
			p.Rotation = rotation
			g := Graph{Components: []Component{p}}
			for _, extra := range [][]string{nil, {"configured | live stale"}, {"input held: high", "pending input: low"}} {
				annotations := configuredSourceAnnotations(g, DefaultTransientSettings())
				annotations[0].Lines = append(annotations[0].Lines, extra...)
				positionSourceAnnotations(g, annotations)
				a := annotations[0]
				lo, hi := SourceAnnotationBounds(a)
				if rotation == 90 || rotation == 270 {
					if hi.Y > p.Position.Y-50 || hi.X != p.Position.X+40 {
						t.Fatal("horizontal source block crosses symbol or leads", kind, rotation, a)
					}
				} else if hi.X > p.Position.X-39 {
					t.Fatal("vertical source block crosses symbol or leads", kind, rotation, a)
				}
				// Native bitmap ink and exported conservative text bounds must both
				// fit the same block, including any extra live-state lines.
				for i, line := range a.Lines {
					y := a.Position.Y + float64(i)*SourceAnnotationLineHeight
					if hi.X-float64(len([]rune(line)))*8.75 < lo.X || hi.X-schematicLabelWidth(line) < lo.X || y-15 < lo.Y || y+4 > hi.Y {
						t.Fatal("annotation bounds clip a rendered line", kind, rotation, a)
					}
				}
			}
		}
	}
}

func TestSourceAnnotationsOwnDataAndDistinguishAppliedPendingAndStale(t *testing.T) {
	s := pausedRectifier(t)
	defer s.Close()
	before := s.Snapshot()
	a := before.SourceAnnotations[0]
	if a.ID != "V1" || a.State != "current" || a.Mode != "auto" || a.Pending != "" {
		t.Fatal("fresh sine source annotation", a)
	}
	a.Lines[0] = "caller mutation"
	if s.Snapshot().SourceAnnotations[0].Lines[0] == "caller mutation" {
		t.Fatal("snapshot source text aliases service")
	}
	call(t, s, "simulation.input.set", InputParams{Mode: "high"}, "hold")
	queued := s.Snapshot().SourceAnnotations[0]
	if queued.Mode != "auto" || queued.Pending != "high" {
		t.Fatal("queued hold claims application before a step", queued)
	}
	call(t, s, "simulation.step", StepParams{Duration: .001}, "apply-hold")
	held := s.Snapshot().SourceAnnotations[0]
	if held.Mode != "high" || held.State != "current" || held.Lines[len(held.Lines)-1] != "input held: high" {
		t.Fatal("held input omitted", held)
	}
	call(t, s, "simulation.input.set", InputParams{Mode: "low"}, "queue-low")
	pending := s.Snapshot().SourceAnnotations[0]
	if pending.Mode != "high" || pending.Pending != "low" || pending.Lines[len(pending.Lines)-1] != "pending input: low" {
		t.Fatal("queued mode substituted for applied mode", pending)
	}
	call(t, s, "source.sine.set", SineParams{ID: "V1", Waveform: SineWaveform{Amplitude: 3, Frequency: 25}}, "edit")
	stale := s.Snapshot().SourceAnnotations[0]
	if stale.State != "stale" || stale.Mode != "" || stale.Pending != "" || stale.Lines[len(stale.Lines)-1] != "configured | live stale" || !strings.Contains(stale.Lines[1], "25 Hz") {
		t.Fatal("new configuration claims old held source fresh", stale)
	}
}

func TestSourceAnnotationFitPreservesPhysicsAndFramesWholeBlock(t *testing.T) {
	s := pausedRectifier(t)
	defer s.Close()
	before := s.Snapshot()
	call(t, s, "view.fit", FitParams{Width: 600, Height: 300}, "fit")
	after := s.Snapshot()
	for _, a := range after.SourceAnnotations {
		min, max := SourceAnnotationBounds(a)
		for _, p := range []Point{min, max} {
			x, y := (p.X-after.View.Center.X)*after.View.Zoom+300, (p.Y-after.View.Center.Y)*after.View.Zoom+150
			if x < 0 || x > 600 || y < 0 || y > 300 {
				t.Fatal("fit clips source annotation", p, after.View)
			}
		}
	}
	if before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || before.Document != after.Document {
		t.Fatal("annotation-aware Fit changes physics/history/dirty state")
	}
}

func TestBlockedSourceAnnotationExportCapturesOriginalGlobalTimingAndState(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	cfg := s.Snapshot().TransientSettings
	cfg.HighSeconds, cfg.LowSeconds = .123456, .234567
	call(t, s, "simulation.transient.set", cfg, "timing")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	call(t, s, "simulation.input.set", InputParams{Mode: "charge"}, "hold")
	call(t, s, "simulation.step", StepParams{Duration: .001}, "apply-hold")
	before := s.Snapshot()
	entered, release := blockExportPreparation(t, s)
	defer release()
	path := filepath.Join(t.TempDir(), "original.svg")
	promptLiveCommand(t, s, "export.start", ExportStartParams{Kind: "schematic-svg", Path: path})
	awaitLiveSampleBlock(t, entered)
	cfg.HighSeconds = .4
	call(t, s, "simulation.transient.set", cfg, "later")
	edited := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "completed")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("high 123.456 ms | low 234.567 ms")) || !bytes.Contains(data, []byte("input held: charge")) || bytes.Contains(data, []byte("high 400 ms")) {
		t.Fatal("export source text includes later timing/state")
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var metadata struct {
		Annotations []SourceAnnotation `json:"source_annotations"`
		Settings    TransientSettings  `json:"source_settings"`
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			t.Fatal("SVG metadata absent", err)
		}
		if start, ok := token.(xml.StartElement); ok && start.Name.Local == "metadata" {
			var raw string
			if err := decoder.DecodeElement(&raw, &start); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if !reflect.DeepEqual(metadata.Annotations, before.SourceAnnotations) || math.Abs(metadata.Settings.HighSeconds-.123456) > 1e-15 || after.Document != edited.Document || !reflect.DeepEqual(after.Transient, edited.Transient) {
		t.Fatal("export loses exact/captured source or changes later state")
	}
}
