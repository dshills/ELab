package workbench

func addPreferenceCapabilityDetails(id string, schema map[string]any) bool {
	switch id {
	case "diagnostics.details.set", "appearance.configure", "appearance.wait", "appearance.theme.set", "display.preferences.set":
	default:
		return false
	}
	c := commandContractDetail{Version: 1, Scope: "application-preferences", Preconditions: []string{"Ordinary mutation guards/retry retention apply to setters; appearance.wait is a read-only wait"}, Effects: []string{"Preserves components, wires and accepted simulation/run/time/charge; preference persistence is distinct from project saving"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed, missing, null required, extra, wrong-type or invalid preference/path values"}}, Completion: &commandCompletionDetail{Mode: "immediate accepted state change; no solver job", Observe: []string{"project.inspect", "events.poll"}}}
	switch id {
	case "diagnostics.details.set":
		inputProperty(schema, "visible")["description"] = "Required boolean; expands/collapses detailed diagnostic presentation only"
		c.Effects = append(c.Effects, "Session-only visibility in snapshot.diagnostic_presentation.details_visible; preserves document dirty state/history and does not suppress diagnostic detection or structured help", "New/example load/project restoration can reset this presentation flag")
	case "appearance.theme.set":
		inputProperty(schema, "theme")["enum"] = []string{"system", "light", "dark"}
		c.Effects = append(c.Effects, "Sets session theme without changing project dirty state or undo history; an explicit choice, even the current theme, supersedes a pending preference load", "If persistence is configured and not already in error, queues/coalesces saving; otherwise choice remains session-only. Inspect appearance outcome separately; a successful setter does not guarantee disk persistence")
		schema["examples"] = []any{map[string]any{"theme": "dark"}}
	case "appearance.configure":
		p := inputProperty(schema, "path")
		p["maxLength"], p["x-max-utf8-bytes"] = 4096, 4096
		p["description"] = "Required exact empty string disables persistence without deleting the old file or changing session theme. Otherwise a nonblank, NUL-free local path, resolved absolute against process cwd; no default user file is chosen by this command"
		schema["x-max-preference-file-bytes"] = MaxAppearanceBytes
		schema["x-max-appearance-workers"] = MaxAppearanceWorkers
		c.Effects = append(c.Effects, "Configures an opt-in preference store, not a project. Missing file is created privately with current theme; an existing bounded regular elab.appearance/version 1 file may load its theme", "Read/save happens outside the editor mutex; snapshots expose appearance.path/status/saved_theme/error. Corrupt, linked, oversized or externally changed files produce appearance.status=error and are not silently overwritten", "No per-operation ID exists for this legacy store: inspect global appearance state and current path. Reconfigure/disable supersedes the store cooperatively; actual pending workers still count until they return", "Preserves project dirty state/history. Writes use captured source hashes and atomic publish with a cooperative five-second preparation deadline; OS calls may return later. Shutdown joins actual workers and reports persistence failures")
		c.Errors = append(c.Errors, commandFailureDetail{"busy", "admission", "Two appearance workers are still active"})
		c.Completion = &commandCompletionDetail{Mode: "immediate configuration/disable admission; inspect global appearance status for IO outcome", Observe: []string{"appearance.wait", "project.inspect", "events.poll"}, Cancel: "appearance.configure with empty path disables/supersedes the store; already published files are not undone", CooperativeDeadlineSeconds: 5}
		schema["x-work-outcome"] = "Worker IO failures appear in snapshot.appearance.status/error, not as a fabricated command failure code; session theme remains usable"
		schema["examples"] = []any{AppearanceConfigureParams{Path: "/absolute/private/appearance.json"}, AppearanceConfigureParams{Path: ""}}
	case "appearance.wait":
		c.Preconditions = []string{"Empty query params; no mutation guard/key; waits only for global appearance loading/saving"}
		c.Effects = append(c.Effects, "Returns immediately unless status is loading or saving; ready/error/disabled are terminal observations for this wait. Status=error is a successful query response containing the persistence error", "Timeout/caller cancellation returns timeout and does not cancel the store. Existing workers can still publish; wait is not a new job or a persistence retry")
		c.Errors = append(c.Errors, commandFailureDetail{"timeout", "wait", "Five-second wait bound or caller context ended before loading/saving finished"})
		c.Completion = &commandCompletionDetail{Mode: "read-only wait up to five seconds; no operation ID", Observe: []string{"project.inspect", "events.poll"}, CooperativeDeadlineSeconds: 5}
	case "display.preferences.set":
		p := inputProperty(schema, "voltage_range_v")
		p["minimum"], p["maximum"], p["x-unit"] = .01, 30, "V"
		inputProperty(schema, "direction")["enum"] = []string{"conventional", "electron"}
		schema["required"] = []string{"voltage_color", "voltage_range_v", "current_cues", "current_motion", "reduced_motion", "direction"}
		schema["allOf"] = []any{map[string]any{"if": map[string]any{"properties": map[string]any{"current_motion": map[string]any{"const": true}}, "required": []string{"current_motion"}}, "then": map[string]any{"properties": map[string]any{"current_cues": map[string]any{"const": true}}}}}
		schema["x-default-preferences"] = DefaultDisplayPreferences()
		c.Preconditions = append(c.Preconditions, "All six fields are required and nonnull; current_motion=true requires current_cues=true")
		c.Effects = append(c.Effects, "Replaces the full display preference set. Changed values are persisted project preferences, contribute to dirty state and one ordinary undo entry; no electrical restart", "Electron direction affects drawing cues only, never signed electrical measurements. Reduced motion suppresses animated dots; stale/unavailable/nonunique wire currents do not invent direction", "Changing voltage color range clips illustrative color magnitude, not numerical voltage. An identical full preference set preserves history/dirty state")
		schema["examples"] = []any{DefaultDisplayPreferences()}
	}
	schema["x-command-contract"] = c
	schema["x-details-scope"] = "existing preference domains/state and persistence outcomes; no new job IDs or exhaustive OS error taxonomy"
	return true
}
