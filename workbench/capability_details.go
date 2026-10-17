package workbench

import (
	"encoding/json"
	"strings"
)

// CapabilityQuery preserves full discovery for {} or an empty command.
type CapabilityQuery struct {
	Command string `json:"command,omitempty"`
}

// Detailed schemas are produced on demand, keeping the ordinary catalog within
// its transport budget. Only inputs gain bounds; numeric SI output schemas do
// not acquire the command-only unit-text grammar.
func detailedCapabilities(id string) ([]Capability, error) {
	if len(id) > 128 {
		return nil, &actionError{"invalid-params", "command must be at most 128 bytes"}
	}
	for _, cap := range Catalog() {
		if cap.ID != id {
			continue
		}
		var schema map[string]any
		if err := json.Unmarshal(cap.Input, &schema); err != nil {
			return nil, err
		}
		if ref, ok := schema["$ref"].(string); ok {
			schema = nil
			if err := json.Unmarshal(SchemaRegistry()[ref], &schema); err != nil {
				return nil, err
			}
			delete(schema, "$id")
		}
		addCapabilityDetails(id, schema)
		data, err := json.Marshal(schema)
		if err != nil {
			return nil, err
		}
		cap.Input = data
		return []Capability{cap}, nil
	}
	return nil, &actionError{"not-found", "unknown command; choose an ID from capabilities.list"}
}

func inputProperty(schema map[string]any, path string) map[string]any {
	for _, part := range strings.Split(path, ".") {
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			return nil
		}
		schema, ok = properties[part].(map[string]any)
		if !ok {
			return nil
		}
	}
	return schema
}

type inputRelation struct {
	Operator string   `json:"operator"`
	Fields   []string `json:"fields"`
	Limit    float64  `json:"limit,omitempty"`
	WhenKind string   `json:"when_component_kind,omitempty"`
}

func addCapabilityDetails(id string, schema map[string]any) {
	if addProjectCapabilityDetails(id, schema) {
		return
	}
	if addSamplingCapabilityDetails(id, schema) {
		return
	}
	if addWorkspaceCapabilityDetails(id, schema) {
		return
	}
	if addViewportCapabilityDetails(id, schema) {
		return
	}
	if addSchematicCapabilityDetails(id, schema) {
		return
	}
	if addTiledPageDetails(id, schema) {
		return
	}
	if addPreviewCapabilityDetails(id, schema) {
		return
	}
	if addExportCapabilityDetails(id, schema) {
		return
	}
	if addIdealPulseTimingDetails(id, schema) {
		return
	}
	if addPulseTimingDetails(id, schema) {
		return
	}
	if addMeasurementCapabilityDetails(id, schema) {
		return
	}
	if addScopeCapabilityDetails(id, schema) {
		return
	}
	if addActuationCapabilityDetails(id, schema) {
		return
	}
	if addLiveControlDetails(id, schema) {
		return
	}
	if addShutdownDetails(id, schema) {
		return
	}
	if addRecoveryReadDetails(id, schema) {
		return
	}
	if addRecordingViewerDetails(id, schema) {
		return
	}
	if addRecordingCursorDetails(id, schema) {
		return
	}
	if addPrintingCapabilityDetails(id, schema) {
		return
	}
	if addVerificationCapabilityDetails(id, schema) {
		return
	}
	if addValueCapabilityDetails(id, schema) {
		return
	}
	if addRecordingStatisticsDetails(id, schema) {
		return
	}
	if addRecordingTimingDetails(id, schema) {
		return
	}
	if addRecordingComparisonDetails(id, schema) {
		return
	}
	if addSweepCapabilityDetails(id, schema) {
		return
	}
	rangeSI := func(path, unit string, min, max float64) {
		p := inputProperty(schema, path)
		if p != nil {
			p["minimum"], p["maximum"], p["x-unit"] = min, max, unit
			if strings.HasSuffix(path, "_s") || len(waveformQuantityFields[id]) > 0 {
				p["x-normalized-si-range"] = map[string]float64{"minimum": min, "maximum": max}
			}
		}
	}
	rules := []inputRelation{}
	switch id {
	case "recovery.configure", "recovery.configure.start":
		rangeSI("interval_s", "s", 1, 60)
		inputProperty(schema, "directory")["minLength"] = 1
		inputProperty(schema, "directory")["maxLength"] = 4096
		inputProperty(schema, "directory")["x-max-utf8-bytes"] = 4096
		inputProperty(schema, "directory")["description"] = "Nonblank local path without NUL. A new store requires a private (0700) real directory. Relative paths resolve against the process working directory. Disabling the already configured canonical path updates scheduling without opening/scanning that folder; last-known catalog/error remain."
		schema["x-existing-store-disable"] = map[string]any{"folder_access": false, "recovery_catalog_applied": false, "preserves_last_known_catalog_and_error": true, "in_flight_checkpoint_may_finish": true, "reenable": "Scans storage normally and can fail; disabling does not validate availability"}
		schema["examples"] = []any{RecoveryConfigureParams{Directory: "/absolute/path/recovery", Enabled: false, Interval: 2}}
	case "export.schematic.png":
		p := inputProperty(schema, "dpi")
		p["anyOf"] = []any{map[string]any{"const": 0}, map[string]any{"minimum": 72, "maximum": 300}}
		p["default"], p["x-unit"] = 150, "pixels/inch"
		schema["examples"] = []any{PNGExportParams{Path: "/absolute/path/schematic.png", DPI: 150}}
	case "source.sine.set":
		rangeSI("waveform.amplitude_v", "V", 0, 12)
		rangeSI("waveform.offset_v", "V", -12, 12)
		rangeSI("waveform.frequency_hz", "Hz", .1, 1000)
		rangeSI("waveform.phase_degrees", "degrees", -360, 360)
		rules = append(rules, inputRelation{Operator: "sum_absolute_maximum", Fields: []string{"waveform.offset_v", "waveform.amplitude_v"}, Limit: 12})
		schema["examples"] = []any{SineParams{ID: "V1", Waveform: DefaultSineWaveform()}}
	case "source.pulse.set":
		rangeSI("waveform.low_v", "V", -12, 12)
		rangeSI("waveform.high_v", "V", -12, 12)
		for _, field := range []string{"high_s", "low_s", "rise_s", "fall_s"} {
			rangeSI("waveform."+field, "s", 1e-6, 10)
		}
		rules = append(rules,
			inputRelation{Operator: "less_or_equal", Fields: []string{"waveform.low_v", "waveform.high_v"}},
			inputRelation{Operator: "less_or_equal", Fields: []string{"waveform.rise_s", "waveform.high_s"}},
			inputRelation{Operator: "less_or_equal", Fields: []string{"waveform.fall_s", "waveform.low_s"}},
			inputRelation{Operator: "sum_minimum", Fields: []string{"waveform.high_s", "waveform.low_s"}, Limit: .001})
		schema["examples"] = []any{PulseParams{ID: "V1", Waveform: DefaultPulseWaveform()}}
	case "example.load":
		ids := []string{}
		for _, info := range ExampleCatalog() {
			ids = append(ids, info.ID)
		}
		inputProperty(schema, "id")["enum"] = ids
		schema["examples"] = []any{ExampleLoadParams{ID: "switch-npn"}}
	default:
		return
	}
	if len(rules) != 0 {
		schema["x-normalized-si-relations"] = rules
	}
	if addWaveformSourceContract(id, schema) {
		return
	}
	// This is explicit coverage of parameter restrictions, not an exhaustive
	// catalog of stateful admission preconditions or all possible failure codes.
	schema["x-details-scope"] = "parameter-bounds-and-examples; stateful-preconditions-and-errors-not-exhaustive"
}
