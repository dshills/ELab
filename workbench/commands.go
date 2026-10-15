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
	Version                    int             `json:"version"`
	ID                         string          `json:"id"`
	Command                    string          `json:"command"`
	Params                     json.RawMessage `json:"params"`
	ExpectedRevision           *uint64         `json:"expected_revision,omitempty"`
	ExpectedCommandRevision    *uint64         `json:"expected_command_revision,omitempty"`
	ExpectedDocumentGeneration *uint64         `json:"expected_document_generation,omitempty"`
	Key                        string          `json:"key,omitempty"`
}
type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type Response struct {
	Help      *DiagnosticHelp `json:"help,omitempty"`
	Version   int             `json:"version"`
	ID        string          `json:"id"`
	Revision  uint64          `json:"revision"`
	Duplicate bool            `json:"duplicate,omitempty"`
	Result    any             `json:"result,omitempty"`
	Error     *Failure        `json:"error,omitempty"`
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
		output = json.RawMessage(`{"type":"object","required":["version","schema_version","schemas","capabilities","retry_retention"],"properties":{"version":{"const":1},"schema_version":{"const":2},"schemas":{"type":"object","additionalProperties":{"type":"object"}},"capabilities":{"type":"array","items":{"type":"object"}},"events":{"type":"object"},"retry_retention":{"type":"string"},"mutation_preconditions":{"type":"object"}}}`)
	}
	if id == "parts.list" {
		data, _ := json.Marshal(typeSchema(reflect.TypeOf([]Part{})))
		output = data
	}
	var input map[string]any
	_ = json.Unmarshal([]byte(schema), &input)
	encoded, _ := json.Marshal(timeInputSchema(id, input))
	return Capability{id, description, mutates, encoded, "Snapshot mutations; query data otherwise; local schemas", append(json.RawMessage(nil), output...)}
}

func typedCapability(id, description string, t reflect.Type) Capability {
	data, _ := json.Marshal(typeSchema(t))
	return capability(id, description, true, string(data))
}

const ObjectIDSchemaID = "urn:elab:object-id:1"

const VerificationMetadataSchemaID = "urn:elab:verification-metadata:1"

const SnapshotSchemaID = "urn:elab:snapshot:1"
const DiscoverySchemaVersion = 2

var commonSchemas = []struct {
	Type reflect.Type
	ID   string
}{
	{reflect.TypeOf(ExportStartParams{}), "urn:elab:export-start:1"},
	{reflect.TypeOf(SchematicPage{}), "urn:elab:schematic-page:1"},
	{reflect.TypeOf(ExportParams{}), "urn:elab:export-path:1"},
	{reflect.TypeOf(PDFExportParams{}), "urn:elab:pdf-export:1"},
	{reflect.TypeOf(PNGExportParams{}), "urn:elab:png-export:1"},
	{reflect.TypeOf(PreviewParams{}), "urn:elab:preview-start:1"},
	{reflect.TypeOf(PreviewImageParams{}), "urn:elab:preview-image:1"},
	{reflect.TypeOf(ReplaceParams{}), "urn:elab:replace-document:1"},
	{reflect.TypeOf(IDParams{}), ObjectIDSchemaID},
	{reflect.TypeOf(CursorSelectParams{}), "urn:elab:cursor-select:1"},
	{reflect.TypeOf(ProbeParams{}), "urn:elab:probe-selection:1"},
	{reflect.TypeOf(RecordingPlayParams{}), "urn:elab:playback:1"},
	{reflect.TypeOf(InputParams{}), "urn:elab:input-mode:1"},
	{reflect.TypeOf(ValueParams{}), "urn:elab:component-value:1"},
	{reflect.TypeOf(PrintPreview{}), "urn:elab:print-preview:1"},
	{reflect.TypeOf(DisplayPreferences{}), "urn:elab:display-preferences:1"},
	{reflect.TypeOf(VerificationPage{}), "urn:elab:verification-page:1"},
	{reflect.TypeOf(SweepSettings{}), "urn:elab:sweep-settings:1"},
	{reflect.TypeOf(SweepState{}), "urn:elab:sweep-state:1"},
	{reflect.TypeOf(SweepResult{}), "urn:elab:sweep-result:1"},
	{reflect.TypeOf(SweepMetadata{}), "urn:elab:sweep-metadata:1"},
	{reflect.TypeOf(DCCircuit{}), "urn:elab:dc-circuit:1"},
	{reflect.TypeOf(Probe{}), "urn:elab:probe:1"},
	{reflect.TypeOf(Component{}), "urn:elab:component:1"},
	{reflect.TypeOf(Graph{}), "urn:elab:graph:1"},
	{reflect.TypeOf(CircuitDisplay{}), "urn:elab:circuit-display:1"},
	{reflect.TypeOf(VerificationExportParams{}), "urn:elab:record-export:1"},
	{reflect.TypeOf(RecordingImportParams{}), "urn:elab:record-import:1"},
	{reflect.TypeOf(RecoveryConfigureParams{}), "urn:elab:recovery-configure:1"},
	{reflect.TypeOf(SweepSummary{}), "urn:elab:sweep-summary:1"},
}

var snapshotSchema = json.RawMessage(`{"$ref":"urn:elab:snapshot:1"}`)
var snapshotDefinition = func() json.RawMessage {
	schema := structSchema(reflect.TypeOf(Snapshot{}))
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
	eventSchema := structSchema(reflect.TypeOf(EventPollResult{}))
	eventSchema["$id"] = EventPollSchemaID
	eventData, _ := json.Marshal(eventSchema)
	registry := map[string]json.RawMessage{SnapshotSchemaID: append(json.RawMessage(nil), snapshotDefinition...), VerificationMetadataSchemaID: data, EventPollSchemaID: eventData, TimeInputSchemaID: json.RawMessage(`{"$id":"urn:elab:time:1","type":["number","string"],"maxLength":128,"description":"Seconds as a finite number or decimal/scientific text; bare text is seconds; case-sensitive s/ms/us/µs/μs/ns/ps. Command bounds still apply.","examples":[0.01,"10 ms","250 us"]}`)}
	for _, common := range commonSchemas {
		schema := structSchema(common.Type)
		// This is an input-only type. Preserve unit-text support once in its shared
		// definition, without widening numeric RecoveryState output fields.
		if common.Type == reflect.TypeOf(RecoveryConfigureParams{}) {
			schema = timeInputSchema("recovery.configure", schema)
		}
		schema["$id"] = common.ID
		encoded, _ := json.Marshal(schema)
		registry[common.ID] = encoded
	}
	return registry
}

func typeSchema(t reflect.Type) map[string]any {
	if t == reflect.TypeOf([]byte{}) {
		return map[string]any{"type": "string", "contentEncoding": "base64"}
	}
	for _, common := range commonSchemas {
		if t == common.Type {
			return map[string]any{"$ref": common.ID}
		}
	}
	if t == reflect.TypeOf(EventPollResult{}) {
		return map[string]any{"$ref": EventPollSchemaID}
	}
	if t == reflect.TypeOf(Snapshot{}) {
		return map[string]any{"$ref": SnapshotSchemaID}
	}
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
	inputSchema := typeSchema(input)
	if id == "events.poll" {
		inputSchema["properties"].(map[string]any)["timeout_ms"] = map[string]any{"type": []string{"integer", "null"}, "minimum": 0, "maximum": MaxEventWaitMS, "default": 1000}
	}
	data, _ := json.Marshal(inputSchema)
	cap := capability(id, description, false, string(data))
	cap.OutputSchema, _ = json.Marshal(typeSchema(output))
	return cap
}
func Catalog() []Capability {
	return []Capability{
		typedCapability("simulation.sampling.fit", "Reduce output step to recommended waveform resolution (at least 1 µs), reducing playback speed if needed for the 1000 samples/wall-second bound; ordinary settings restart semantics", reflect.TypeOf(struct{}{})),
		typedCapability("source.sine.set", "Set per-source sine amplitude, offset, frequency and phase; ordinary schematic restart semantics", reflect.TypeOf(SineParams{})),
		typedCapability("source.pulse.set", "Set per-source finite pulse levels, phase lengths and linear rise/fall; ordinary schematic restart semantics", reflect.TypeOf(PulseParams{})),
		typedCapability("potentiometer.wiper.set", "Set a linear potentiometer wiper position 0–1; fresh transient algebraic work is staged outside the editor lock; inspect input_work/wait; preserves charge/time/run on success", reflect.TypeOf(WiperParams{})),
		typedCapability("switch.set", "Set desired SPST startup state; fresh transients queue/coalesce actuation for the next accepted step, exposing desired/applied/pending state and preserving charge/run; DC/config edits restart", reflect.TypeOf(SwitchParams{})),
		typedCapability("wire.junction.insert", "Atomically insert one explicit junction into 1–8 named connected wire routes; position required for multi-wire joins, single wire defaults to a free interior point", reflect.TypeOf(JunctionInsertParams{})),
		typedCapability("recording.keep.set", "Include/exclude a completed recording in project saves and recovery; exclusion keeps session replay/export; kept records are never silently evicted", reflect.TypeOf(RecordingKeepParams{})),
		typedCapability("verification.import.start", "Read/validate a recorded JSON asset in background; inspect project_work; keeps live circuit/state and later viewer choice", reflect.TypeOf(RecordingImportParams{})),
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
		typedCapability("recovery.configure", "Wait for recovery setup; private local directory, automatic snapshots and 1–60 s interval; existing recovery stays active until ready", reflect.TypeOf(RecoveryConfigureParams{})),
		typedCapability("recovery.configure.start", "Prepare recovery setup in background; inspect/cancel project_work; preserves current recovery on failure/cancel; no editor-lock filesystem work", reflect.TypeOf(RecoveryConfigureParams{})),
		typedCapability("recovery.work.start", "Start background scan or checkpoint with shared project job inspection/cancellation; checkpoint publication is irreversible", reflect.TypeOf(RecoveryWorkParams{})),
		capability("recovery.scan", "Validate and list newest complete snapshots per document, with corrupt-file diagnostics", true, `{"type":"object","additionalProperties":false}`),
		capability("recovery.checkpoint", "Write a complete recovery snapshot now (or retire owned snapshots when clean); does not save/associate the project", true, `{"type":"object","additionalProperties":false}`),
		typedCapability("recovery.select", "Select a listed recovery candidate; empty ID clears", reflect.TypeOf(IDParams{})),
		typedCapability("recovery.restore.start", "Read/hash-validate a recovery copy in background; protects original generation/unsaved work and intervening edits", reflect.TypeOf(RecoveryChoiceParams{})),
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
		typedCapability("export.schematic.svg", "Atomically render the complete schematic and pinned reading cards/provenance as portable SVG using shared wire routes; works headlessly without a solve", reflect.TypeOf(ExportParams{})),
		typedCapability("export.schematic.pdf", "Export captured complete schematic to printable A4/Letter PDF with page options; macOS CoreGraphics required; no printer submission or project/physics changes", reflect.TypeOf(PDFExportParams{})),
		typedCapability("export.schematic.png", "Export complete schematic on white A4/Letter page as PNG; shared page options; dpi omitted/0=150 or 72–300; macOS CoreGraphics required; preserves source/physics", reflect.TypeOf(PNGExportParams{})),
		typedCapability("sweep.settings.set", "Configure DC bias, resistance or wiper range with unit text, 2–201 inclusive points and 1–6 probes; no live mutation", reflect.TypeOf(SweepSettings{})),
		typedCapability("sweep.start", "Capture an independent DC sweep; optional settings override; inspect dc_sweep/jobs; one worker, 30 s cooperative deadline", reflect.TypeOf(SweepStartParams{})),
		typedCapability("sweep.cancel", "Cancel exact active sweep ID; no partial points are published", reflect.TypeOf(IDParams{})),
		queryCapability("sweep.inspect", "Read retained sweep capture and complete result; copied data; no live mutation", reflect.TypeOf(IDParams{}), reflect.TypeOf(SweepInspection{})),
		typedCapability("sweep.select", "Select a retained sweep ID without stopping current work", reflect.TypeOf(IDParams{})),
		typedCapability("sweep.visibility.set", "Show/hide sweep plot; preserves live acquisition; showing closes print preview", reflect.TypeOf(SweepVisibilityParams{})),
		typedCapability("sweep.trace.select", "Choose one captured probe for the sweep plot; signed SI values", reflect.TypeOf(IDParams{})),
		typedCapability("sweep.cursor.set", "Select exact sweep point index; null clears; no interpolation", reflect.TypeOf(SweepCursorParams{})),
		typedCapability("sweep.import.start", "Read/validate captured sweep JSON in background; project_work inspection/cancellation; preserves live circuit and later viewer choices", reflect.TypeOf(RecordingImportParams{})),
		typedCapability("sweep.import", "Wait for captured sweep import; kept recorded data, not a live solve; preserves graph/physics and later view choices", reflect.TypeOf(RecordingImportParams{})),
		typedCapability("sweep.export", "Export complete captured DC curve as json or SI csv; atomic file guards; no live mutation", reflect.TypeOf(VerificationExportParams{})),
		typedCapability("sweep.keep.set", "Include/exclude a completed sweep in project storage", reflect.TypeOf(RecordingKeepParams{})),
		typedCapability("print.preview.start", "Capture page/source and render preview in background; one worker; preserves circuit/physics; no printer submission", reflect.TypeOf(PreviewParams{})),
		typedCapability("print.preview.cancel", "Cancel active preview preparation by exact ID", reflect.TypeOf(IDParams{})),
		typedCapability("print.preview.close", "Close exact preview ID and release pixels; cancel active preparation", reflect.TypeOf(IDParams{})),
		queryCapability("print.preview.image", "Read owned PNG chunks from visible completed preview; 1–65536 bytes; default 65536; base64 output; exact ID prevents stale image reads", reflect.TypeOf(PreviewImageParams{}), reflect.TypeOf(PreviewImageChunk{})),
		typedCapability("export.page.set", "Set session PDF/PNG paper/orientation/margin/scale/color/measurement defaults; omitted fields reset; no circuit/history/physics changes", reflect.TypeOf(SchematicPage{})),
		capability("scope.window.set", "Set mode live or fixed with start_s/end_s inside fresh retained history; plot and interval statistics use only that window", true, `{"type":"object","required":["mode"],"properties":{"mode":{"enum":["live","fixed"]},"start_s":{"type":"number"},"end_s":{"type":"number"}},"allOf":[{"if":{"properties":{"mode":{"const":"fixed"}}},"then":{"required":["start_s","end_s"]}}],"additionalProperties":false}`),
		typedCapability("parts.search.set", "Filter components by case-insensitive words in names, hobbyist aliases, kind, pins, units, defaults or model IDs; session only", reflect.TypeOf(SearchParams{})),
		typedCapability("component.duplicate", "Copy one component's value, orientation and startup voltage into an unconnected new component; optional exact position", reflect.TypeOf(DuplicateParams{})),
		typedCapability("edit.batch", "Atomically apply 1–100 supported schematic edits as one named undo action; no file, simulation or nested batch commands", reflect.TypeOf(BatchParams{})),
		queryCapability("display.inspect", "Read exact wire voltages, unique conventional wire currents, palette and illustrative marker geometry at phase 0–1; no mutation or physics advancement", reflect.TypeOf(DisplayFrameParams{}), reflect.TypeOf(DisplayFrame{})),
		queryCapability("examples.list", "Discover immutable example questions, expected template behavior, experiments and model limitations", reflect.TypeOf(struct{}{}), reflect.TypeOf([]ExampleInfo{})),
		typedCapability("example.load", "Load a known example ID into a new editable document; protects unsaved work and optional document generation", reflect.TypeOf(ExampleLoadParams{})),
		typedCapability("example.current.reset", "Restore the current example template, not another example; protects unsaved work and optional expected_example_id/document generation", reflect.TypeOf(ExampleResetParams{})),
		typedCapability("scope.visibility.set", "Show/hide the scope dock for this session; preserves acquisition, recorded playback and measurements", reflect.TypeOf(ScopeVisibilityParams{})),
		typedCapability("diagnostics.details.set", "Show/hide technical diagnostic details; session-only presentation, no physics/history/dirty change; related targets are in diagnostic_presentation", reflect.TypeOf(DiagnosticDetailsParams{})),
		typedCapability("appearance.configure", "Load and autosave bounded app theme preferences in background; empty path disables; inspect appearance", reflect.TypeOf(AppearanceConfigureParams{})),
		capability("appearance.wait", "Wait up to 5 s for theme preference loading/saving; inspect appearance status/error", false, `{"type":"object","additionalProperties":false}`),
		capability("appearance.theme.set", "Theme: light, dark, system; autosaves if configured; preserves circuit/history", true, `{"type":"object","required":["theme"],"properties":{"theme":{"type":"string","enum":["light","dark","system"]}},"additionalProperties":false}`),
		typedCapability("display.preferences.set", "Set persisted voltage color/range, current cues/motion, reduced motion and conventional/electron visualization; signed values always use conventional current; preserves physics", reflect.TypeOf(DisplayPreferences{})),
		typedCapability("measurement.select", "Select current object, terminal, anchored net or probe for live inspection; empty kind/ID returns to the editing selection; session only, preserves physics/history/dirty state", reflect.TypeOf(MeasurementSelectParams{})),
		queryCapability("measurement.inspect", "Read the selected live measurement context, or a supplied current target, with references, signed values and accepted-result provenance; no mutation", reflect.TypeOf(MeasurementQueryParams{}), reflect.TypeOf(MeasurementInspection{})),
		typedCapability("measurement.pin.add", "Pin a current voltage/current probe at a grid position; at most eight; persisted undoable presentation edit, no simulation restart", reflect.TypeOf(PinSpec{})),
		typedCapability("measurement.pin.move", "Move a pinned reading to a grid position without changing physics; previous_position refuses stale gestures", reflect.TypeOf(MoveParams{})),
		typedCapability("measurement.pin.remove", "Remove a pinned reading including missing sources; undoable presentation edit", reflect.TypeOf(IDParams{})),
		typedCapability("scope.trace.select", "Inspect a selected live trace's current schematic source; empty ID clears focus; source associations are in snapshot.trace_focus and preserve physics", reflect.TypeOf(IDParams{})),
		typedCapability("scope.probe.remove", "Remove one selected probe, including an unavailable removed-source probe; preserves other requested probes, graph and simulation", reflect.TypeOf(IDParams{})),
		typedCapability("scope.probes.set", "Choose up to six unique IDs from snapshot.scope.available; [] hides all; session only, no simulation restart", reflect.TypeOf(ProbeParams{})),
		typedCapability("scope.cursor.set", "Set a/b time inside retained fresh history; snap to nearest sample, ties choose latest/right-hand; inspect scope for timing and trace deltas", reflect.TypeOf(CursorParams{})),
		typedCapability("scope.cursor.select", "Select a/b for plot click/drag placement", reflect.TypeOf(CursorSelectParams{})),
		typedCapability("scope.cursor.clear", "Clear cursor a/b and its measurements", reflect.TypeOf(CursorSelectParams{})),
		capability("capabilities.list", "List commands; optional command returns one detailed input schema", false, `{"type":"object","properties":{"command":{"type":"string","maxLength":128}},"additionalProperties":false}`),
		capability("project.inspect", "Read circuit, connectivity, selection, view, model, diagnostics and results", false, `{"type":"object","additionalProperties":false}`),
		capability("simulation.wait", "Wait up to 5 s for project/initialization/step/input work; inspect corresponding work state for outcomes; ready transients return immediately", false, `{"type":"object","additionalProperties":false}`),
		capability("parts.list", "List available kinds, terminals, models and default values", false, `{"type":"object","additionalProperties":false}`),
		typedCapability("project.clear", "Start blank; unsaved changes require discard_changes; undo restores drawing", reflect.TypeOf(ReplaceParams{})),
		queryCapability("events.poll", "Coalesced same-process observation; timeout 0–5000 ms, default 1000; four waits; idle omits snapshot", reflect.TypeOf(EventPollParams{}), reflect.TypeOf(EventPollResult{})),
		queryCapability("project.work.inspect", "Inspect an operation by stable ID; retain eight project jobs and preserve active operations", reflect.TypeOf(IDParams{}), reflect.TypeOf(ProjectWork{})),
		typedCapability("export.start", "Capture schematic-svg/pdf/png, scope-csv, verification-json/csv or sweep-json/csv export and return background admission; inspect/cancel via project.work commands", reflect.TypeOf(ExportStartParams{})),
		typedCapability("project.save.start", "Capture a project and return background save admission; inspect project_work; cancellation is available before publishing", reflect.TypeOf(SaveParams{})),
		typedCapability("project.save", "Wait for project.save.start semantics; keeps records/models/preferences; later edits stay dirty; path defaults to current file; overwrite permits replacement", reflect.TypeOf(SaveParams{})),
		typedCapability("project.load.start", "Start background bounded local project reading/validation; inspect project_work; replacement refuses intervening edits", reflect.TypeOf(LoadParams{})),
		typedCapability("project.work.cancel", "Cancel exact current project work before publication; cancelable is false during atomic Save publication/cleanup; refusal code operation-committing", reflect.TypeOf(IDParams{})),
		typedCapability("project.load", "Wait for the same background open job as project.load.start; restores saved pause/view with fresh physics; unsaved changes require discard_changes", reflect.TypeOf(LoadParams{})),
		typedCapability("component.place", "Place a library part on the 10-unit grid; optional ID must be unused; 100 drawing-object limit including helpers; simulation envelope in snapshot.limits", reflect.TypeOf(PlaceParams{})),
		typedCapability("component.move", "Move a part to a grid position within ±10000; preserve all wires and electrical results", reflect.TypeOf(MoveParams{})),
		typedCapability("component.rotate", "Rotate a part 90 degrees clockwise without changing its pin identities or connectivity", reflect.TypeOf(IDParams{})),
		typedCapability("object.delete", "Delete a part and its incident wires, or one wire, as one undoable edit", reflect.TypeOf(IDParams{})),
		typedCapability("wire.connect", "Join named terminals explicitly; crossings never connect; 300 wires maximum", reflect.TypeOf(ConnectParams{})),
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
		typedCapability("example.switch-npn", "Load a manual SPST-controlled transistor LED with a 100 kΩ base pull-down; finite switch resistance; protects unsaved work", reflect.TypeOf(ReplaceParams{})),
		typedCapability("example.rectifier", "Load a 5 V peak/10 Hz half-wave silicon-diode rectifier with 100 Ω series resistance, 10 µF smoothing and 10 kΩ load; protects unsaved work", reflect.TypeOf(ReplaceParams{})),
		typedCapability("example.rc", "Load a 5 V pulse / 10 kΩ / 1 µF RC example; unsaved changes require discard_changes", reflect.TypeOf(ReplaceParams{})),
		typedCapability("simulation.transient.set", "Set transient sampling, pulse durations and playback speed (initial_voltage_v supplies default capacitor startup voltages); undoable and restarts from initial conditions", reflect.TypeOf(TransientSettings{})),
		typedCapability("simulation.input.set", "Actuate RC input (auto/charge/discharge) NPN base input, or selected mixed source (auto/high/low); queue input while paused", reflect.TypeOf(InputParams{})),
		typedCapability("simulation.step", "Advance a paused transient atomically using a cancellable background worker and wait for its terminal result; at most 1 s / 1000 samples", reflect.TypeOf(StepParams{})),
		typedCapability("simulation.step.start", "Start a cancellable atomic manual advance and return immediately; inspect snapshot.step_work or simulation.wait; at most 1 s / 1000 samples", reflect.TypeOf(StepParams{})),
		typedCapability("simulation.work.cancel", "Cancel the exact active manual step or input-work ID; no staged physical/graph update is committed", reflect.TypeOf(IDParams{})),
		typedCapability("example.reset", "Restore LED example; unsaved changes require discard_changes", reflect.TypeOf(ReplaceParams{})),
		typedCapability("edit.undo", "Undo the latest schematic edit; optional expected_entry_revision guards the exact history tip; other actors require allow_other_actor:true", reflect.TypeOf(HistoryParams{})),
		typedCapability("edit.redo", "Redo the latest undone edit; optional exact history tip and explicit other-actor acknowledgment", reflect.TypeOf(HistoryParams{})),
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

func paramsNested(raw json.RawMessage, field string, dst any, required ...string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return err
	}
	return params(object[field], dst, required...)
}
