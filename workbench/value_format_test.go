package workbench

import (
	"math"
	"math/rand/v2"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInspectorValueDraftPreservesExactStoredBits(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "component.set", ValueParams{ID: "C1", Value: "1.234567890123456 µF"}, "precision")
	call(t, s, "component.select", IDParams{ID: "C1"}, "select")
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "redo-seed")
	call(t, s, "edit.undo", HistoryParams{}, "undo-seed")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	call(t, s, "simulation.step", StepParams{Duration: .001}, "sample")
	call(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "precision.elab")}, "save")
	before := s.Snapshot()
	component, _ := (Graph{Components: before.Components}).Component("C1")
	parsed, err := ParseValue(before.Drawing.Value, component.Unit)
	if err != nil || math.Float64bits(parsed) != math.Float64bits(component.Value) {
		t.Fatalf("unchanged Inspector draft %q changes %.17g into %.17g: %v", before.Drawing.Value, component.Value, parsed, err)
	}
	call(t, s, "component.set", ValueParams{ID: "C1", Value: before.Drawing.Value}, "unchanged-apply")
	after := s.Snapshot()
	if after.ElectricalRevision != before.ElectricalRevision || after.RunID != before.RunID || after.Document != before.Document || !reflect.DeepEqual(after.Transient, before.Transient) || !reflect.DeepEqual(after.UndoEntry, before.UndoEntry) || !reflect.DeepEqual(after.RedoEntry, before.RedoEntry) {
		t.Fatal("unchanged apply restarted physical state")
	}
}

func TestEditableValueFormattingAcrossFiniteDomains(t *testing.T) {
	random := rand.New(rand.NewPCG(2026, 1015))
	for _, domain := range []struct {
		unit     string
		min, max float64
	}{{"V", 0, 12}, {"V", -12, 0}, {"Ω", 10, 100000}, {"F", 1e-12, 1}} {
		values := []float64{domain.min, domain.max, math.Nextafter(domain.min, domain.max), math.Nextafter(domain.max, domain.min)}
		for i := 0; i < 2000; i++ {
			values = append(values, domain.min+(domain.max-domain.min)*random.Float64())
		}
		if domain.unit == "V" {
			values = append(values, math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64, math.Copysign(0, -1), 1e-320)
		}
		if domain.unit == "F" {
			for i := 0; i < 2000; i++ {
				values = append(values, math.Exp(-random.Float64()*12*math.Ln10))
			}
		}
		for _, value := range values {
			text := FormatValue(value, domain.unit)
			parsed, err := ParseValue(text, domain.unit)
			if err != nil || math.Float64bits(parsed) != math.Float64bits(value) {
				t.Fatalf("%s %.17g -> %q -> %.17g: %v", domain.unit, value, text, parsed, err)
			}
			if FormatValue(parsed, domain.unit) != text {
				t.Fatal("editable text is not stable", text)
			}
		}
	}
	for _, test := range []struct {
		value      float64
		unit, text string
	}{{1e-6, "F", "1 µF"}, {1000, "Ω", "1 kΩ"}, {100000, "Ω", "100 kΩ"}, {.5, "V", "0.5 V"}, {0, "V", "0 V"}} {
		if got := FormatValue(test.value, test.unit); got != test.text {
			t.Fatal("common value lost readable form", test, got)
		}
	}
}
func TestReopenedValueDraftKeepsNumericSIAndLabelSemantics(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "component.set", ValueParams{ID: "C1", Value: "1.234567890123456 µF"}, "precise")
	original, _ := (Graph{Components: s.Snapshot().Components}).Component("C1")
	path := filepath.Join(t.TempDir(), "precise.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	loaded := New()
	defer loaded.Close()
	call(t, loaded, "project.load", LoadParams{Path: path}, "open")
	call(t, loaded, "component.select", IDParams{ID: "C1"}, "select")
	before := loaded.Snapshot()
	call(t, loaded, "component.set", ValueParams{ID: "C1", Value: before.Drawing.Value}, "unchanged")
	after := loaded.Snapshot()
	c, _ := (Graph{Components: after.Components}).Component("C1")
	if math.Float64bits(c.Value) != math.Float64bits(original.Value) || before.Document != after.Document || before.ElectricalRevision != after.ElectricalRevision || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("reopened inspector apply changed saved data")
	}
	call(t, loaded, "component.place", PlaceParams{ID: "Label", Kind: "net-label", Position: Point{X: 500}, Value: "VCC/+"}, "label")
	if loaded.Snapshot().Drawing.Value != "VCC/+" {
		t.Fatal("numeric formatter changed label draft")
	}
}
