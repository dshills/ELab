package workbench

// Scope choices are live/session controls. Captured recording commands have
// their own inventory and never silently redirect these commands.
func addScopeCapabilityDetails(id string, schema map[string]any) bool {
	switch id {
	case "scope.visibility.set", "scope.probes.set", "scope.probe.remove", "scope.trace.select", "scope.window.set", "scope.cursor.set", "scope.cursor.select", "scope.cursor.clear", "export.scope.csv":
	default:
		return false
	}
	contract := commandContractDetail{Version: 1, Scope: "live-scope-controls-and-csv", Preconditions: []string{}, Effects: []string{"Preserves graph, device state/time, run/electrical revision, edit history and source dirty membership"}, Errors: []commandFailureDetail{}, Completion: &commandCompletionDetail{Mode: "immediate session/view update"}}
	failure := func(code, phase, when string) {
		contract.Errors = append(contract.Errors, commandFailureDetail{code, phase, when})
	}
	failure("invalid-params", "admission", "Malformed/missing parameters or an invalid command-specific value")
	schema["x-target-scope"] = "snapshot.scope (live); captured recording.scope uses recording.* commands"
	schema["x-max-selected-probes"] = MaxScopeProbes
	schema["x-max-live-history-samples"] = MaxTraceSamples
	switch id {
	case "scope.visibility.set":
		contract.Effects = append(contract.Effects, "Show/hide the dock without pausing acquisition, changing selections/cursors/windows, or stopping independent recorded playback; session only")
		schema["examples"] = []any{ScopeVisibilityParams{Visible: false}}
	case "scope.probes.set":
		p := inputProperty(schema, "ids")
		p["minItems"], p["maxItems"], p["uniqueItems"] = 0, MaxScopeProbes, true
		p["description"] = "Required non-null array of unique current snapshot.scope.available IDs; [] hides all. Inventory is based on drawing parts and does not imply fresh simulated values."
		schema["x-probe-inventory"] = "snapshot.scope.available descriptors, including signed positive/reference and SI unit"
		contract.Preconditions = []string{"Every requested ID belongs to current scope.available, with no duplicates; 0–6 IDs"}
		contract.Effects = append(contract.Effects, "Replace selection in supplied order and fill available traces from retained history without a new solve; hidden channels remain in private bounded frames", "Clear source focus if its trace is no longer selected; explicit Save captures valid probe choices without selection alone marking the document dirty")
		schema["examples"] = []any{ProbeParams{IDs: []string{}}}
	case "scope.probe.remove":
		schema["x-probe-inventory"] = "Requested IDs: union of snapshot.scope.selected and snapshot.scope.unavailable[].id"
		contract.Preconditions = []string{"ID is requested by the current selection, including retained unavailable IDs whose source was removed"}
		contract.Effects = append(contract.Effects, "Remove only that requested channel, retaining other unavailable channels for undo; clear matching source focus")
		failure("not-found", "admission", "ID is not requested; empty ID does not mean clear all")
	case "scope.trace.select":
		schema["x-probe-inventory"] = "Nonempty ID must be both selected and currently available; empty ID clears source focus"
		contract.Preconditions = []string{"A selected available live probe, or empty ID to clear; fresh numerical samples are not required for schematic association"}
		contract.Effects = append(contract.Effects, "Publish trace_focus with stable object/terminal/current-net associations, separately from editing selection", "Move/rotate preserves association; removed source reports unavailable and undo can restore it; recorded viewing suppresses current-schematic highlights with an explanation")
		failure("not-found", "admission", "Nonempty ID is not a selected available live probe")
		schema["examples"] = []any{IDParams{ID: ""}}
	case "scope.cursor.select", "scope.cursor.clear":
		inputProperty(schema, "cursor")["enum"] = []string{"a", "b"}
		contract.Preconditions = []string{"cursor is a or b; no fresh history is required"}
		if id == "scope.cursor.select" {
			contract.Effects = append(contract.Effects, "Choose which live cursor a subsequent plot gesture moves; does not place a cursor")
		} else {
			contract.Effects = append(contract.Effects, "Clear only the named position and paired deltas/readings; clearing an unset position is allowed; active gesture cursor is unchanged")
		}
		schema["examples"] = []any{CursorSelectParams{Cursor: "a"}}
	case "scope.cursor.set":
		inputProperty(schema, "cursor")["enum"] = []string{"a", "b"}
		inputProperty(schema, "time_s")["x-unit"] = "s"
		schema["x-time-inventory"] = "Fresh snapshot.transient.samples[0].time_s through snapshot.transient.sample.time_s, with legacy decimal boundary roundoff tolerance"
		contract.Preconditions = []string{"Fresh retained live transient history and finite requested time inside its bounds (using existing boundary-roundoff tolerance)", "cursor is a or b; the requested time need not be an actual saved row"}
		contract.Effects = append(contract.Effects, "Store requested_s, then snap to nearest retained row; equal-distance/duplicate-time ties choose the latest/right-hand sample; no interpolation", "Successful placement can still be invalid for a fixed visible interval; inspect a/b.valid and reason, not admission alone", "Both valid cursors publish signed B−A time, finite reciprocal 1/abs(delta) only when nonzero, and selected-channel A/B/delta in descriptor SI units; equal times have zero delta and no frequency")
	case "scope.window.set":
		schema["x-time-inventory"] = "Fresh retained transient sample bounds for fixed mode; live mode follows retained history"
		schema["x-normalized-si-relations"] = []map[string]any{{"operator": "start_s<end_s for fixed mode"}, {"operator": "fixed start/end inside retained history with existing boundary roundoff tolerance"}}
		contract.Preconditions = []string{"mode live, or fixed with both finite start_s/end_s inside fresh retained history and start_s<end_s"}
		contract.Effects = append(contract.Effects, "Live clears the fixed interval and follows rolling history; supplied optional times still undergo normal type/time-text validation", "Fixed filters actual saved rows and computes per-selected-trace sample min/max/mean/RMS; expiration or no saved rows makes window_valid false with a reason and no statistics/paired measurements", "Window and cursor positions are session only; electrical restarts clear them, geometry preserves them, and hidden acquisition continues")
		schema["examples"] = []any{map[string]any{"mode": "live"}}
	case "export.scope.csv":
		p := inputProperty(schema, "path")
		p["minLength"], p["maxLength"], p["x-max-utf8-bytes"] = 1, 4096, 4096
		p["description"] = "Nonblank local path without NUL; relative paths resolve against the process working directory; destination must differ from the open schematic path."
		schema["x-project-work-inventory"] = "snapshot.project_work; project.work.inspect retains eight jobs; cancel by exact current ID while cancelable"
		contract.Preconditions = []string{"Fresh live transient, valid visible interval and at least one selected trace with actual rows", "No active project work and filesystem worker capacity available; optional parameter expected_document_generation also matches current document"}
		contract.Effects = append(contract.Effects, "Capture selected live channels/window/rows and signed SI descriptors at admission; recorded viewing does not redirect this export", "Encode captured samples outside editor lock; each row carries run/electrical revision, engine, interval, sample step and timestamp; later live changes cannot substitute data", "Atomic destination publication; explicit overwrite is required for an existing regular file; failure/cancellation does not publish partial CSV or modify source")
		failure("busy", "admission", "An active project operation or worker capacity prevents export admission")
		failure("unavailable", "admission", "No fresh selected rows in the visible interval")
		failure("conflict", "admission", "Parameter document-generation target changed")
		failure("file-exists", "project_work", "Destination exists without explicit overwrite")
		failure("file-conflict", "publication", "Captured destination identity/content changed before publication")
		failure("file-error", "project_work", "Local file encoding/preparation/publication error; arbitrary OS causes are not exhaustive")
		failure("canceled", "wait-or-project_work", "Caller wait ended without canceling admitted work, or exact work cancellation was requested")
		contract.Completion = &commandCompletionDetail{Mode: "waiting alias for captured background scope-csv export; inspect terminal project_work/artifact outcome", Observe: []string{"project.work.inspect", "project.inspect", "simulation.wait", "events.poll"}, Cancel: "project.work.cancel", CooperativeDeadlineSeconds: 30}
	}
	if id == "scope.window.set" || id == "scope.cursor.set" || id == "export.scope.csv" {
		schema["x-visible-window-roundoff"] = map[string]any{"maximum_fraction_of_span": maxScopeWindowRoundoffFraction, "description": "Visible row membership caps legacy boundary tolerance to this fraction of the window span; it must not borrow nearby outside rows. File-grid validation and requested-time admission retain their prior tolerance."}
	}
	schema["x-command-contract"] = contract
	return true
}
