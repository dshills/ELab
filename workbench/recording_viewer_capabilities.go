package workbench

// Recorded playback is a viewer clock over immutable rows, not a solver run.
// Keep membership is separate from both live physics and session replay.
func addRecordingViewerDetails(id string, schema map[string]any) bool {
	switch id {
	case "recording.view.set", "recording.seek", "recording.step", "recording.speed.set", "recording.play", "recording.pause", "recording.window.set", "recording.probes.set", "recording.cursor.select", "recording.cursor.clear", "recording.keep.set":
	default:
		return false
	}
	contract := commandContractDetail{Version: 1, Scope: "recorded-viewer-and-kept-membership", Preconditions: []string{"A retained completed recording is selected; discover snapshot.recording and operation.inspect metadata"}, Effects: []string{"Preserves live schematic, device state/time, run/electrical revision, edit history and captured rows/model dependencies", "Viewer commands may be used while live scope is shown; they do not implicitly switch to recorded view or select a different operation", "Ordinary command/retry/actor rules apply; viewer clock publications advance presentation revision, not command revision"}, Errors: []commandFailureDetail{{"not-ready", "admission", "No selected completed recording; for viewer commands other than view.set this check precedes command-specific parameter validation"}, {"invalid-params", "admission", "Malformed/unknown/duplicate fields or command-specific value/relationship violation"}}, Completion: &commandCompletionDetail{Mode: "immediate in-memory viewer update; no file IO or solver work", Observe: []string{"project.inspect", "events.poll"}}}
	schema["x-recording-inventory"] = "snapshot.operations; operation.select chooses a retained operation, completed metadata exposes captured probes/duration/rows"
	schema["x-viewer-state"] = "snapshot.recording: id, viewing, playing, seconds_per_wall_second, sample_index, time_s, duration_s, sample_count, last_sample_time_s and scope"
	schema["x-preserved-dirty-policy"] = "Viewer choices do not mark the document dirty. Save/recovery retain selected kept recording/viewing preference; index, playback speed, windows, recorded probe subset and cursors are session-only. Keep membership alone changes recordings_dirty."
	contract.Effects = append(contract.Effects, "Later viewer intent takes precedence over automatic viewer selection by an admitted recording import; the imported asset can still be kept. Inspect the import job outcome")
	failure := func(code, when string) {
		contract.Errors = append(contract.Errors, commandFailureDetail{code, "admission", when})
	}
	switch id {
	case "recording.view.set":
		inputProperty(schema, "mode")["enum"] = []string{"live", "recorded"}
		contract.Preconditions = []string{"live needs no selected capture; recorded requires a selected retained completed recording"}
		contract.Effects = append(contract.Effects, "live stops recorded playback. recorded changes viewing only; it does not start playback or stop/restart the live solver. Returning to recorded retains its current row/window/cursors")
		schema["examples"] = []any{RecordingViewParams{Mode: "recorded"}, RecordingViewParams{Mode: "live"}}
	case "recording.seek":
		inputProperty(schema, "index")["minimum"] = 0
		inputProperty(schema, "index")["description"] = "Zero-based actual stored row, 0 <= index < snapshot.recording.sample_count. Null/omitted is absent; duplicate timestamps remain distinct rows."
		inputProperty(schema, "time_s")["description"] = "Time quantity within complete captured duration; null/omitted is absent. Select nearest stored row, later row on distance ties and last/right-hand row on an exact duplicate timestamp. No interpolation."
		schema["x-relations"] = []map[string]any{{"operator": "exactly-one-non-null", "fields": []string{"index", "time_s"}}, {"operator": "dynamic-selected-recording-bounds", "index_max_exclusive": "snapshot.recording.sample_count", "time_max_inclusive": "snapshot.recording.duration_s"}}
		contract.Effects = append(contract.Effects, "Seeks within complete capture, independently of the visible plot window, and pauses playback; preserves window, probes, cursors and view mode")
		schema["examples"] = []any{map[string]any{"index": 0}, map[string]any{"time_s": "10 ms"}}
	case "recording.step":
		inputProperty(schema, "direction")["enum"] = []string{"next", "previous"}
		contract.Effects = append(contract.Effects, "Moves exactly one actual row and pauses playback, including adjacent rows sharing a timestamp; no wrap, clamping or implicit window change")
		failure("unavailable", "Requested row is before zero or after the final stored row; rejection preserves playback/viewer state")
	case "recording.speed.set", "recording.play":
		speed := inputProperty(schema, "seconds_per_wall_second")
		speed["minimum"], speed["maximum"], speed["x-unit"] = .001, 10, "simulated seconds per wall second"
		speed["description"] = "Required finite numeric rate .001–10; independent of live simulation playback. It is a numeric ratio, not a time quantity."
		if id == "recording.speed.set" {
			contract.Effects = append(contract.Effects, "Does not start paused playback; if playing, reanchors at the current stored row and continues at the new configured rate. Works for a DC point")
		} else {
			contract.Preconditions = append(contract.Preconditions, "Captured duration is positive; DC operating points cannot play")
			contract.Effects = append(contract.Effects, "Starts/continues a separate roughly 33 ms viewer ticker; at the last row, Play restarts from row zero. At captured end it stops on the final stored row", "Ticks choose the last actual row at/before target time; duplicates select their later/right-hand row. No interpolation, looping, solver work or guarantee that every row is displayed", "Playback uses the whole captured duration, independently of the visible plot window. It can advance while recorded view is hidden")
			contract.Completion.Mode = "immediate playback start; later coalesced recordings publications carry current row/playing state"
			contract.Completion.Cancel = "recording.pause"
			failure("unavailable", "Selected recording has zero DC duration")
		}
		schema["examples"] = []any{RecordingPlayParams{Speed: .1}}
	case "recording.pause":
		contract.Effects = append(contract.Effects, "Stops only recorded playback; keeps current row, configured speed, plot window and live solver/pause state. Repeating pause is allowed")
	case "recording.window.set":
		for _, name := range []string{"start_s", "end_s"} {
			inputProperty(schema, name)["description"] = "Required non-null time quantity within the selected captured duration, with existing 1e-12 * max(1,duration_s) boundary roundoff allowance. Near-boundary values clamp into [0,duration_s]."
		}
		schema["x-captured-interval"] = map[string]any{"minimum": 0, "maximum": "snapshot.recording.duration_s", "raw_order": "start_s <= end_s", "transient_after_clamp": "start_s < end_s", "dc_point": "start_s = end_s = 0", "boundary_roundoff_s": "1e-12 * max(1,duration_s)"}
		contract.Effects = append(contract.Effects, "Sets only plot/sample-statistics interval. It does not seek, pause, trim rows, constrain playback or place cursors; empty intervals between stored rows report no exact data", "Positive-duration windows must remain positive after boundary clamping; a DC point permits only normalized [0,0]. Nearest-row cursor lookup and playback indices are independent of the window")
		schema["examples"] = []any{map[string]any{"start_s": "5 ms", "end_s": "15 ms"}}
	case "recording.probes.set":
		ids := inputProperty(schema, "ids")
		ids["minItems"], ids["maxItems"], ids["uniqueItems"] = 0, MaxScopeProbes, true
		ids["description"] = "Required array of unique IDs from selected operation metadata.probes or snapshot.recording.scope.available; null is refused. [] hides every trace. Captured channels cannot be added after the fact."
		contract.Effects = append(contract.Effects, "Sets ordered recorded trace subset; no live-probe change, row reordering, data removal or captured measurement change. New selection resets to all captured probes")
	case "recording.cursor.select", "recording.cursor.clear":
		inputProperty(schema, "cursor")["enum"] = []string{"a", "b"}
		if id == "recording.cursor.select" {
			contract.Effects = append(contract.Effects, "Selects a/b target for recorded plot clicks; does not place, move or clear either cursor")
		} else {
			contract.Effects = append(contract.Effects, "Clears only requested a/b cursor; repeated clear is allowed and does not change active cursor selection, live cursors or plot window")
		}
	case "recording.keep.set":
		contract.Preconditions = []string{"id names any retained completed Verify/imported/project recording; it need not be selected. Sweep membership uses its separate command"}
		inputProperty(schema, "id")["description"] = "Required non-null retained completed recording ID from snapshot.operations; empty/unknown or incomplete target returns not-ready. Does not select that recording."
		inputProperty(schema, "keep")["description"] = "Required non-null membership boolean. Exclusion keeps the session asset available until ordinary retention eviction."
		contract.Effects = []string{"Changes only whether this completed asset is included in project saves/recovery. Does not select it, stop playback, modify captured rows/models, restart live physics or create an undo entry", "recordings_dirty compares kept membership against the last saved baseline; restoring the same saved membership can return to clean. Viewer selection/preferences do not drive this dirty flag", "Exclusion does not immediately free/remove the session operation; later admission may evict old unkept completed records. Kept records are not silently evicted", "No project encoding/size validation is performed here; keeping a record does not guarantee a later Save/checkpoint fits the existing 8 MiB project bound", "Changing kept membership can supersede an admitted Open whose captured membership guard no longer matches; later edits remain unsaved by a captured Save"}
		contract.Errors = []commandFailureDetail{{"invalid-params", "admission", "Malformed, unknown, duplicate, missing or null required fields"}, {"not-ready", "admission", "ID is empty, unknown or not a retained completed recording"}}
	}
	schema["x-command-contract"] = contract
	return true
}
