package workbench

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

var capturedExportCommands = []string{"export.page.set", "export.start", "export.schematic.svg", "export.schematic.pdf", "export.schematic.png", "verification.export"}

func TestExportContractsPreserveDefaultCatalogAndOwnedPageDefinitions(t *testing.T) {
	before, _ := json.Marshal(Catalog())
	registry, _ := json.Marshal(SchemaRegistry())
	for _, id := range capturedExportCommands {
		details, err := detailedCapabilities(id)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(details[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		c := schema["x-command-contract"].(map[string]any)
		if c["scope"] != "captured-file-export" || len(c["effects"].([]any)) == 0 || len(c["errors"].([]any)) == 0 {
			t.Fatal("missing export state contract", id)
		}
		if id == "export.schematic.png" || id == "export.start" {
			if schema["x-preview-png"] == nil || schema["x-png-page-selection"] == nil || inputProperty(schema, "dpi")["default"] != float64(150) {
				t.Fatal("export details dropped existing tiled/preview PNG policy", id)
			}
		}
		details[0].Input[0] = '!'
		fresh, err := detailedCapabilities(id)
		if err != nil || !json.Valid(fresh[0].Input) {
			t.Fatal("detailed export schema aliases caller bytes", err)
		}
	}
	after, _ := json.Marshal(Catalog())
	output, _ := json.Marshal(SchemaRegistry())
	if !bytes.Equal(before, after) || !bytes.Equal(registry, output) {
		t.Fatal("on-demand export details changed catalog/shared output definitions")
	}
}

func TestExportPageReplacementNullDefaultsAndLayoutAdmission(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	call(t, s, "export.page.set", SchematicPage{Paper: "a4", Orientation: "portrait", Tile: true, Scale: ptrFloat(4), OverlapMM: ptrFloat(20), MarginMM: ptrFloat(40), Color: true, Measurements: true}, "set")
	changed := s.Snapshot()
	if changed.Document != before.Document || changed.RunID != before.RunID || !reflect.DeepEqual(changed.UndoEntry, before.UndoEntry) || !reflect.DeepEqual(changed.Transient, before.Transient) {
		t.Fatal("output defaults changed circuit/history/physics")
	}
	call(t, s, "export.page.set", map[string]any{"paper": nil, "orientation": "", "margin_mm": nil, "scale": nil, "color": nil}, "defaults")
	page := s.Snapshot().SchematicPage
	if page.Paper != "letter" || page.Orientation != "landscape" || page.MarginMM == nil || *page.MarginMM != 10 || page.Scale != nil || page.Tile || page.OverlapMM != nil || page.Color || page.Measurements {
		t.Fatal("omitted page fields patched old defaults instead of replacing them", page)
	}
	call(t, s, "export.page.set", map[string]any{"margin_mm": 0}, "zero-margin")
	if *s.Snapshot().SchematicPage.MarginMM != 0 {
		t.Fatal("zero margin was treated as omission")
	}
	for _, bad := range []any{map[string]any{"tile": true}, map[string]any{"tile": true, "scale": nil}, map[string]any{"scale": 0}, map[string]any{"margin_mm": 41}, map[string]any{"overlap_mm": 1}, map[string]any{"paper": "legal"}, map[string]any{"measurements": "false"}} {
		before := s.Snapshot()
		refuse(t, s, "export.page.set", bad, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("invalid output defaults changed source/session state", bad)
		}
	}
}
