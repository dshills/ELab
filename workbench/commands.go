package workbench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"unicode/utf8"
)

const ProtocolVersion = 1

type Request struct {
	Version          int             `json:"version"`
	ID               string          `json:"id"`
	Command          string          `json:"command"`
	Params           json.RawMessage `json:"params"`
	ExpectedRevision *uint64         `json:"expected_revision,omitempty"`
	Key              string          `json:"key,omitempty"`
}
type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type Response struct {
	Version   int      `json:"version"`
	ID        string   `json:"id"`
	Revision  uint64   `json:"revision"`
	Duplicate bool     `json:"duplicate,omitempty"`
	Result    any      `json:"result,omitempty"`
	Error     *Failure `json:"error,omitempty"`
}
type Capability struct {
	ID           string          `json:"id"`
	Description  string          `json:"description"`
	Mutation     bool            `json:"mutation"`
	Input        json.RawMessage `json:"input_schema"`
	Output       string          `json:"output"`
	OutputSchema json.RawMessage `json:"output_schema"`
}

func capability(id, description string, mutates bool, schema string) Capability {
	output := snapshotSchema
	if id == "capabilities.list" {
		output = json.RawMessage(`{"type":"object","required":["version","schema_version","schemas","capabilities","retry_retention"],"properties":{"version":{"const":1},"schema_version":{"const":2},"schemas":{"type":"object","additionalProperties":{"type":"object"}},"capabilities":{"type":"array","items":{"type":"object"}},"retry_retention":{"type":"string"}}}`)
	}
	if id == "parts.list" {
		data, _ := json.Marshal(typeSchema(reflect.TypeOf([]Part{})))
		output = data
	}
	return Capability{id, description, mutates, json.RawMessage(schema), "Snapshot for mutations; structured query result otherwise; resolve schema refs through capabilities.list.schemas", append(json.RawMessage(nil), output...)}
}

func typedCapability(id, description string, t reflect.Type) Capability {
	data, _ := json.Marshal(typeSchema(t))
	return capability(id, description, true, string(data))
}

const VerificationMetadataSchemaID = "urn:elab:verification-metadata:1"

const SnapshotSchemaID = "urn:elab:snapshot:1"
const DiscoverySchemaVersion = 2

var snapshotSchema = json.RawMessage(`{"$ref":"urn:elab:snapshot:1"}`)
var snapshotDefinition = func() json.RawMessage {
	schema := typeSchema(reflect.TypeOf(Snapshot{}))
	schema["$id"] = SnapshotSchemaID
	data, _ := json.Marshal(schema)
	return data
}()

// Schema references are local identifiers, never URLs to fetch. Returning a
// fresh registry keeps in-process clients from changing future discovery data.
func SchemaRegistry() map[string]json.RawMessage {
	metadata := structSchema(reflect.TypeOf(VerificationMetadata{}))
	metadata["$id"] = VerificationMetadataSchemaID
	data, _ := json.Marshal(metadata)
	return map[string]json.RawMessage{SnapshotSchemaID: append(json.RawMessage(nil), snapshotDefinition...), VerificationMetadataSchemaID: data}
}

func typeSchema(t reflect.Type) map[string]any {
	if t == reflect.TypeOf(VerificationMetadata{}) {
		return map[string]any{"$ref": VerificationMetadataSchemaID}
	}
	if t == reflect.TypeOf(json.RawMessage{}) {
		return map[string]any{"type": "object"}
	}
	if t.Kind() == reflect.Pointer {
		return map[string]any{"anyOf": []any{typeSchema(t.Elem()), map[string]any{"type": "null"}}}
	}
	switch t.Kind() {
	case reflect.Struct:
		return structSchema(t)
	case reflect.Slice:
		return map[string]any{"type": "array", "items": typeSchema(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": typeSchema(t.Elem())}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Uint64:
		return map[string]any{"type": "integer"}
	default:
		return map[string]any{"type": "number"}
	}
}
func queryCapability(id, description string, input, output reflect.Type) Capability {
	data, _ := json.Marshal(typeSchema(input))
	cap := capability(id, description, false, string(data))
	cap.OutputSchema, _ = json.Marshal(typeSchema(output))
	return cap
}
func Catalog() []Capability {
	return []Capability{
		typedCapability("switch.set", "Set a generic SPST open/closed state; fresh network transient actuation preserves capacitor voltage, time and run, and records an input event", reflect.TypeOf(SwitchParams{})),
		typedCapability("wire.junction.insert", "Atomically insert one explicit junction into 1–8 named connected wire routes; position required for multi-wire joins, single wire defaults to a free interior point", reflect.TypeOf(JunctionInsertParams{})),
		typedCapability("recording.keep.set", "Include/exclude a completed recording in project saves and recovery; exclusion keeps session replay/export; kept records are never silently evicted", reflect.TypeOf(RecordingKeepParams{})),
		typedCapability("verification.import", "Validate and load a complete elab.verification v1 JSON record into an independent recorded viewer; never changes live circuit/state", reflect.TypeOf(RecordingImportParams{})),
		typedCapability("recording.view.set", "Show live or recorded scope; recorded requires a selected completed operation", reflect.TypeOf(RecordingViewParams{})),
		typedCapability("recording.seek", "Pause and seek exact sample index or nearest recorded time (exactly one); time ties select latest/right-hand edge sample", reflect.TypeOf(RecordingSeekParams{})),
		typedCapability("recording.step", "Pause and move one actual recorded row next/previous, preserving duplicate timestamps", reflect.TypeOf(RecordingStepParams{})),
		typedCapability("recording.speed.set", "Set recorded playback speed .001–10 without starting playback", reflect.TypeOf(RecordingPlayParams{})),
		typedCapability("recording.play", "Play selected complete recording at .001–10 simulated seconds per wall second, without interpolation or solver work", reflect.TypeOf(RecordingPlayParams{})),
		capability("recording.pause", "Pause recorded playback without changing live pause/time", true, `{"type":"object","additionalProperties":false}`),
		typedCapability("recording.window.set", "Set recorded plot/statistics interval inside complete recording; no reduction of exact data", reflect.TypeOf(RecordingWindowParams{})),
		typedCapability("recording.probes.set", "Select up to six unique captured probes; [] hides traces", reflect.TypeOf(ProbeParams{})),
		typedCapability("recording.cursor.set", "Set recorded a/b cursor inside complete recording; nearest sample/right-hand ties, independent of live cursors", reflect.TypeOf(CursorParams{})),
		typedCapability("recording.cursor.select", "Choose recorded a/b cursor for plot clicks", reflect.TypeOf(CursorSelectParams{})),
		typedCapability("recording.cursor.clear", "Clear recorded a/b cursor", reflect.TypeOf(CursorSelectParams{})),
		typedCapability("recovery.configure", "Configure a private local recovery directory, automatic snapshots and 1–60 s interval; saved project files are never overwritten", reflect.TypeOf(RecoveryConfigureParams{})),
		capability("recovery.scan", "Validate and list newest complete snapshots per document, with corrupt-file diagnostics", true, `{"type":"object","additionalProperties":false}`),
		capability("recovery.checkpoint", "Write a complete recovery snapshot now (or retire owned snapshots when clean); does not save/associate the project", true, `{"type":"object","additionalProperties":false}`),
		typedCapability("recovery.select", "Select a listed recovery candidate; empty ID clears", reflect.TypeOf(IDParams{})),
		typedCapability("recovery.restore", "Validate hash and complete snapshot, then restore as a dirty unassociated copy; protects unsaved work and document generation", reflect.TypeOf(RecoveryChoiceParams{})),
		typedCapability("recovery.dismiss", "Remove exactly the hash-confirmed recovery snapshot; other active sessions are protected", reflect.TypeOf(RecoveryChoiceParams{})),
		queryCapability("recovery.samples", "Read exact embedded recovery-recording rows without restoring; hash-confirmed candidate and recording index, pages 1–200", reflect.TypeOf(RecoverySamplesParams{}), reflect.TypeOf(VerificationPage{})),
		queryCapability("recovery.inspect", "Read fully validated recovery configuration and captured recording metadata; use recovery.samples for bounded exact rows", reflect.TypeOf(RecoveryChoiceParams{}), reflect.TypeOf(RecoveryInspection{})),
		typedCapability("verification.export", "Atomically export a completed recorded verification as json (captured graph/models/settings/full data) or csv (SI units/input identity); never substitutes live data", reflect.TypeOf(VerificationExportParams{})),
		typedCapability("simulation.verify", "Start an independent recorded verification from explicit/captured startup, returning last_operation ID; <=10 s and 1000 output steps; DC duration zero; one active operation", reflect.TypeOf(VerifyParams{})),
		typedCapability("verification.settings.set", "Set session Verify duration, sample step (0 follows project), source mode and optional initial overrides without altering the schematic/live state", reflect.TypeOf(VerificationSettings{})),
		typedCapability("operation.cancel", "Request cancellation of queued/running verification; inspect until canceled/failed/completed", reflect.TypeOf(IDParams{})),
		typedCapability("operation.select", "Select a retained operation for its recorded preview; empty ID clears it; does not alter live simulation", reflect.TypeOf(IDParams{})),
		queryCapability("operation.inspect", "Read progress/state/error and captured metadata/results of a retained verification", reflect.TypeOf(IDParams{}), reflect.TypeOf(OperationView{})),
		queryCapability("operation.samples", "Read completed verification samples in pages, preserving duplicate edge timestamps; limit 1–200 (default 100)", reflect.TypeOf(VerificationPageParams{}), reflect.TypeOf(VerificationPage{})),
		typedCapability("export.scope.csv", "Atomically export selected fresh scope-window samples with SI units, engine and run/revision identity; preserves schematic", reflect.TypeOf(ExportParams{})),
		typedCapability("export.schematic.svg", "Atomically render the complete schematic as portable SVG using the shared wire routes; works headlessly without a solve", reflect.TypeOf(ExportParams{})),
		capability("scope.window.set", "Set mode live or fixed with start_s/end_s inside fresh retained history; plot and interval statistics use only that window", true, `{"type":"object","required":["mode"],"properties":{"mode":{"enum":["live","fixed"]},"start_s":{"type":"number"},"end_s":{"type":"number"}},"allOf":[{"if":{"properties":{"mode":{"const":"fixed"}}},"then":{"required":["start_s","end_s"]}}],"additionalProperties":false}`),
		typedCapability("parts.search.set", "Filter the component tray by case-insensitive name, kind, pin, unit or model text; session only", reflect.TypeOf(SearchParams{})),
		typedCapability("component.duplicate", "Copy one component's value, orientation and startup voltage into an unconnected new component; optional exact position", reflect.TypeOf(DuplicateParams{})),
		typedCapability("edit.batch", "Atomically apply 1–100 supported schematic edits as one named undo action; no file, simulation or nested batch commands", reflect.TypeOf(BatchParams{})),
		typedCapability("scope.probes.set", "Choose up to six unique IDs from snapshot.scope.available; [] hides all; session only, no simulation restart", reflect.TypeOf(ProbeParams{})),
		typedCapability("scope.cursor.set", "Set a/b time inside retained fresh history; snap to nearest sample, ties choose latest/right-hand; inspect scope for timing and trace deltas", reflect.TypeOf(CursorParams{})),
		typedCapability("scope.cursor.select", "Select a/b for plot click/drag placement", reflect.TypeOf(CursorSelectParams{})),
		typedCapability("scope.cursor.clear", "Clear cursor a/b and its measurements", reflect.TypeOf(CursorSelectParams{})),
		capability("capabilities.list", "Discover versioned commands and schemas", false, `{"type":"object","additionalProperties":false}`),
		capability("project.inspect", "Read circuit, connectivity, selection, view, model, diagnostics and results", false, `{"type":"object","additionalProperties":false}`),
		capability("simulation.wait", "Wait for current DC calculation (at most 5 seconds), or read ready transient state; future transient time uses paused simulation.step", false, `{"type":"object","additionalProperties":false}`),
		capability("parts.list", "List available kinds, terminals, models and default values", false, `{"type":"object","additionalProperties":false}`),
		typedCapability("project.clear", "Start blank; unsaved changes require discard_changes; undo restores drawing", reflect.TypeOf(ReplaceParams{})),
		typedCapability("project.save", "Atomically save a version-4 .elab project with kept recordings, dependency hashes and paused preference; omit path to save current file; captures selected probes; overwrite explicitly replaces another or externally changed file", reflect.TypeOf(SaveParams{})),
		typedCapability("project.load", "Validate and load a local .elab file, restore view, and resume simulation; unsaved changes require discard_changes", reflect.TypeOf(LoadParams{})),
		typedCapability("component.place", "Place a library part on the 10-unit grid; optional ID must be unused; 50 component limit", reflect.TypeOf(PlaceParams{})),
		typedCapability("component.move", "Move a part to a grid position within ±10000; preserve all wires and electrical results", reflect.TypeOf(MoveParams{})),
		typedCapability("component.rotate", "Rotate a part 90 degrees clockwise without changing its pin identities or connectivity", reflect.TypeOf(IDParams{})),
		typedCapability("object.delete", "Delete a part and its incident wires, or one wire, as one undoable edit", reflect.TypeOf(IDParams{})),
		typedCapability("wire.connect", "Join two existing named terminals explicitly; crossings never connect; 150 wire limit", reflect.TypeOf(ConnectParams{})),
		capability("wire.begin", "Begin a visible wire gesture from a named terminal", true, `{"type":"object","required":["terminal"],"properties":{"terminal":{"type":"string"}},"additionalProperties":false}`),
		typedCapability("drawing.tool.set", "Select tool: select, place, wire, or pan; optionally choose a part kind", reflect.TypeOf(ToolParams{})),
		typedCapability("drawing.position.set", "Set the keyboard placement position on the 10-unit grid within ±10000", reflect.TypeOf(PositionParams{})),
		typedCapability("wire.terminals.set", "Set the named From/To choices without connecting yet; empty string clears a choice", reflect.TypeOf(TerminalsParams{})),
		typedCapability("editor.value.set", "Set the inspector value draft (up to 128 bytes); does not change the circuit until component.set", reflect.TypeOf(DraftParams{})),
		capability("drawing.cancel", "Cancel an unfinished placement or wire gesture and return to selection", true, `{"type":"object","additionalProperties":false}`),
		typedCapability("component.set", "Set an existing source voltage (0–12 V), resistance (10–100000 Ω), or capacitance (1 pF–1 F) using exact unit-aware text", reflect.TypeOf(ValueParams{})),
		capability("circuit.connection.set", "Legacy fixture alias for R1.2 to D1.anode; new drawings use wire.connect/object.delete", true, `{"type":"object","required":["connected"],"properties":{"connected":{"type":"boolean"}},"additionalProperties":false}`),
		typedCapability("component.select", "Select any component or wire by stable ID; empty ID clears selection", reflect.TypeOf(IDParams{})),
		typedCapability("capacitor.initial.set", "Set a capacitor voltage at time zero (pin 1 minus pin 2), within ±12 V; value default uses experiment initial_voltage_v; saved and undoable", reflect.TypeOf(ValueParams{})),
		typedCapability("example.flasher", "Load a DC-powered two-NPN / two-capacitor astable LED flasher with an explicit 0.1 V C1 startup condition; protects unsaved work", reflect.TypeOf(ReplaceParams{})),
		typedCapability("example.npn-pulse", "Load a blinking NPN LED switch with 5 V pulse base drive; instantaneous junction response, no device delay; unsaved changes require discard_changes", reflect.TypeOf(ReplaceParams{})),
		typedCapability("example.npn", "Load a generic NPN low-side LED switch (5 V supply / 5 V base drive / 10 kΩ base / 1 kΩ load); unsaved changes require discard_changes", reflect.TypeOf(ReplaceParams{})),
		typedCapability("example.rc", "Load a 5 V pulse / 10 kΩ / 1 µF RC example; unsaved changes require discard_changes", reflect.TypeOf(ReplaceParams{})),
		typedCapability("simulation.transient.set", "Set transient sampling, pulse durations and playback speed (initial_voltage_v supplies default capacitor startup voltages); undoable and restarts from initial conditions", reflect.TypeOf(TransientSettings{})),
		typedCapability("simulation.input.set", "Actuate RC input (auto/charge/discharge) NPN base input, or selected mixed source (auto/high/low); queue input while paused", reflect.TypeOf(InputParams{})),
		typedCapability("simulation.step", "Advance a paused valid transient run by duration_s, at most 1 second / 1000 samples", reflect.TypeOf(StepParams{})),
		typedCapability("example.reset", "Restore LED example; unsaved changes require discard_changes", reflect.TypeOf(ReplaceParams{})),
		capability("edit.undo", "Undo the most recent schematic edit; presentation undo preserves solver state", true, `{"type":"object","additionalProperties":false}`),
		capability("edit.redo", "Redo an undone schematic edit", true, `{"type":"object","additionalProperties":false}`),
		capability("simulation.pause", "Freeze transient time or pause automatic DC calculation; keep identified readings", true, `{"type":"object","additionalProperties":false}`),
		capability("simulation.resume", "Continue a fresh transient run at its current time, or recalculate stale/DC state", true, `{"type":"object","additionalProperties":false}`),
		capability("simulation.restart", "Restart transient time at zero or recalculate DC; retain paused state and current input mode", true, `{"type":"object","additionalProperties":false}`),
		capability("view.pan", "Pan by world-coordinate delta without restarting simulation", true, `{"type":"object","required":["dx","dy"],"properties":{"dx":{"type":"number"},"dy":{"type":"number"}},"additionalProperties":false}`),
		capability("view.zoom", "Zoom by 0.25–4 around an optional world point; final zoom clamped to 0.01–4", true, `{"type":"object","required":["factor"],"properties":{"factor":{"type":"number","minimum":0.25,"maximum":4},"anchor":{"type":"object","required":["x","y"],"properties":{"x":{"type":"number"},"y":{"type":"number"}},"additionalProperties":false}},"additionalProperties":false}`),
		typedCapability("view.fit", "Fit component bounds into width_px/height_px (64–100000); defaults 650 × 400", reflect.TypeOf(FitParams{})),
		typedCapability("app.close", "Close session; unsaved changes require discard_changes", reflect.TypeOf(ReplaceParams{})),
	}
}

// Admission only needs command identity and mutation classification. Avoid
// rebuilding schema documents for every pointer or text-entry event.
var capabilityIndex = func() map[string]Capability {
	index := map[string]Capability{}
	for _, c := range Catalog() {
		index[c.ID] = c
	}
	return index
}()

// DecodeStrict also rejects duplicate keys, preventing ambiguous wire commands.
func DecodeStrict(data []byte, dst any) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("JSON must be valid UTF-8")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 32 {
			return fmt.Errorf("JSON exceeds nesting limit")
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		if delimiter, ok := t.(json.Delim); ok {
			switch delimiter {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					key, err := d.Token()
					if err != nil {
						return err
					}
					name := key.(string)
					if seen[name] {
						return fmt.Errorf("duplicate field %q", name)
					}
					seen[name] = true
					if err := walk(depth + 1); err != nil {
						return err
					}
				}
			case '[':
				for d.More() {
					if err := walk(depth + 1); err != nil {
						return err
					}
				}
			default:
				return fmt.Errorf("unexpected JSON delimiter")
			}
			_, err = d.Token()
			return err
		}
		return nil
	}
	if err := walk(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("one JSON value required")
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := exactFields(data, reflect.TypeOf(dst)); err != nil {
		return err
	}
	return d.Decode(dst)
}

// Go's JSON decoder normally accepts case aliases; the wire contract uses only
// the canonical schema field names, so id and ID cannot shadow each other.
func exactFields(data []byte, t reflect.Type) error {
	if t == nil || t == reflect.TypeOf(json.RawMessage{}) {
		return nil
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() == reflect.Slice {
		var values []json.RawMessage
		if err := json.Unmarshal(data, &values); err != nil {
			return err
		}
		for _, value := range values {
			if err := exactFields(value, t.Elem()); err != nil {
				return err
			}
		}
		return nil
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	fields := map[string]reflect.Type{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" || field.Tag.Get("json") == "-" {
			continue
		}
		name := field.Tag.Get("json")
		if comma := bytes.IndexByte([]byte(name), ','); comma >= 0 {
			name = name[:comma]
		}
		if name == "" {
			name = field.Name
		}
		if name != "-" {
			fields[name] = field.Type
		}
	}
	for name, value := range object {
		field, ok := fields[name]
		if !ok {
			return fmt.Errorf("unknown field %q", name)
		}
		if err := exactFields(value, field); err != nil {
			return err
		}
	}
	return nil
}
func params(raw json.RawMessage, dst any, required ...string) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return fmt.Errorf("params must be a JSON object")
	}
	if err := DecodeStrict(raw, dst); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for _, name := range required {
		v, ok := fields[name]
		if !ok || bytes.Equal(v, []byte("null")) {
			return fmt.Errorf("required field %q is missing", name)
		}
	}
	return nil
}

func structSchema(t reflect.Type) map[string]any {
	properties := map[string]any{}
	required := []string{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" || field.Tag.Get("json") == "-" {
			continue
		}
		tag := field.Tag.Get("json")
		name := tag
		if comma := bytes.IndexByte([]byte(tag), ','); comma >= 0 {
			name = tag[:comma]
		} else {
			required = append(required, name)
		}
		properties[name] = typeSchema(field.Type)
	}
	return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
}
