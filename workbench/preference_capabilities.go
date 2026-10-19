package workbench

func addPreferenceCapabilityDetails(id string, schema map[string]any) bool {
	switch id {
	case "diagnostics.details.set", "appearance.configure", "appearance.wait", "appearance.theme.set", "appearance.work.inspect", "appearance.work.cancel", "display.preferences.set":
	default:
		return false
	}
	c := commandContractDetail{Version: 1, Scope: "application-preferences", Preconditions: []string{"Ordinary mutation guards/retry retention apply to setters; appearance.wait is a read-only wait"}, Effects: []string{"Preserves components, wires and accepted simulation/run/time/charge; preference persistence is distinct from project saving"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed, missing, null required, extra, wrong-type or invalid preference/path values"}}, Completion: &commandCompletionDetail{Mode: "immediate accepted state change; no solver job", Observe: []string{"project.inspect", "events.poll"}}}
	if id != "diagnostics.details.set" && id != "display.preferences.set" {
		schema["x-appearance-work"] = map[string]any{
			"retained_jobs": MaxAppearanceOperations, "max_workers": MaxAppearanceWorkers,
			"statuses":    []string{"queued", "running", "canceling", "completed", "failed", "canceled", "superseded"},
			"observation": "snapshot.appearance_work is latest only; retain the admitted ID and use appearance.work.inspect. events.poll coalesces preferences/jobs transitions and may skip phases",
			"retention":   "Oldest terminal work is evicted on admission; active/canceling and publishing work retain their slots",
			"outcomes":    "Actor, command/input revisions, document/electrical/run identity, path and requested theme are captured. Result theme/hash describe the last read/published artifact, not current disk contents after external changes. current_at_completion is a completion-time observation, not an ongoing association",
			"publication": "published reports successful atomic publication even when a newer request detached the store. superseded flags displaced work; a publishing job remains running until its actual artifact/error is known. Preparation cancellation does not publish; OS calls can unwind later",
		}
	}
	switch id {
	case "diagnostics.details.set":
		inputProperty(schema, "visible")["description"] = "Required boolean; expands/collapses detailed diagnostic presentation only"
		c.Effects = append(c.Effects, "Session-only visibility in snapshot.diagnostic_presentation.details_visible; preserves document dirty state/history and does not suppress diagnostic detection or structured help", "New/example load/project restoration can reset this presentation flag")
	case "appearance.theme.set":
		inputProperty(schema, "theme")["enum"] = []string{"system", "light", "dark"}
		c.Effects = append(c.Effects, "Sets session theme without changing project dirty state or undo history; an explicit choice, even the current theme, supersedes a pending preference load", "If persistence is configured and not already in error, admits a distinct appearance_work ID and coalesces preparation to the latest theme. An already saved value can complete without writing. Disabled/error stores keep the choice session-only and admit no work; compare appearance_work.command_revision with the response command_revision", "A successful setter is admission, not disk completion. Inspect its ID; earlier publishing work may still produce its captured artifact before the latest choice is saved")
		c.Errors = append(c.Errors, commandFailureDetail{"busy", "admission", "All retained appearance work slots are active"})
		c.Completion = &commandCompletionDetail{Mode: "immediate session choice plus optional identified persistence admission", Observe: []string{"appearance.work.inspect", "project.inspect", "events.poll"}, Cancel: "appearance.work.cancel before publishing; does not undo the selected theme", CooperativeDeadlineSeconds: 5}
		schema["examples"] = []any{map[string]any{"theme": "dark"}}
	case "appearance.configure":
		p := inputProperty(schema, "path")
		p["maxLength"], p["x-max-utf8-bytes"] = 4096, 4096
		p["description"] = "Required exact empty string disables persistence without deleting the old file or changing session theme. Otherwise a nonblank, NUL-free local path, resolved absolute against process cwd; no default user file is chosen by this command"
		schema["x-max-preference-file-bytes"] = MaxAppearanceBytes
		schema["x-max-appearance-workers"] = MaxAppearanceWorkers
		c.Effects = append(c.Effects, "Configures an opt-in preference store, not a project. Missing file is created privately with current theme; an existing bounded regular elab.appearance/version 1 file may load its theme", "Read/save happens outside the editor mutex; nonempty configuration returns a distinct snapshot.appearance_work with stable ID, captured provenance and progress/outcome. Global appearance.path/status/saved_theme/error remain available for the existing UI/wait", "Reconfigure supersedes prior work; empty path admits no new job and disables persistence while preserving the session theme/file. Preparing work cancels cooperatively; publishing work may finish with a retained artifact/error. Actual workers keep their quota until they return", "Preserves project dirty state/history. Writes use captured source hashes and atomic publish with a cooperative five-second preparation deadline; OS calls may return later. Shutdown joins actual workers and reports persistence failures")
		c.Errors = append(c.Errors, commandFailureDetail{"busy", "admission", "Two appearance workers are still active or all retained work slots are active"})
		c.Completion = &commandCompletionDetail{Mode: "immediate identified configuration admission, or immediate disable without a new job", Observe: []string{"appearance.work.inspect", "appearance.wait", "project.inspect", "events.poll"}, Cancel: "appearance.work.cancel targets exact current work before publication; configure with empty path disables the store even when an older publisher must finish", CooperativeDeadlineSeconds: 5}
		schema["x-work-outcome"] = "Worker IO failures appear in snapshot.appearance.status/error, not as a fabricated command failure code; session theme remains usable"
		schema["examples"] = []any{AppearanceConfigureParams{Path: "/absolute/private/appearance.json"}, AppearanceConfigureParams{Path: ""}}
	case "appearance.wait":
		c.Preconditions = []string{"Empty query params; no mutation guard/key; waits only for global appearance loading/saving"}
		c.Effects = append(c.Effects, "Returns immediately unless global status is loading or saving; ready/error/disabled are terminal observations for this legacy wait. Status=error is a successful query response containing the persistence error", "Timeout/caller cancellation returns timeout and does not cancel the store. This global wait does not wait for an older detached publisher/canceling job; inspect the captured appearance_work ID for that operation's terminal outcome")
		c.Errors = append(c.Errors, commandFailureDetail{"timeout", "wait", "Five-second wait bound or caller context ended before loading/saving finished"})
		c.Completion = &commandCompletionDetail{Mode: "read-only wait up to five seconds; no operation ID", Observe: []string{"project.inspect", "events.poll"}, CooperativeDeadlineSeconds: 5}
	case "appearance.work.inspect":
		c.Preconditions = []string{"Required non-null string id from a retained appearance_work; readable across document replacement and after Close while the transport remains available"}
		c.Effects = []string{"Pure independently owned metadata for that exact operation, including original actor/input/path, cancellation/progress and actual artifact/error. No file IO, wait, mutation or worker admission"}
		c.Errors = append(c.Errors, commandFailureDetail{"operation-unavailable", "admission", "ID unknown, empty or evicted"})
		c.Completion = &commandCompletionDetail{Mode: "immediate exact-ID observation", Observe: []string{"appearance.work.inspect", "events.poll"}}
	case "appearance.work.cancel":
		c.Preconditions = []string{"Exact latest active cancelable appearance_work ID with ordinary mutation guards; publishing/cleanup must not have begun"}
		c.Effects = []string{"Disable that current store and request cancellation; preserves the visible session theme and prior file. Reading/preparation can remain canceling until OS work returns; queued work can cancel immediately", "An older/terminal/unknown ID cannot cancel a newer store or theme request. An older already-publishing operation may still complete separately with its own retained artifact/error; no published file is rolled back"}
		c.Errors = append(c.Errors, commandFailureDetail{"operation-unavailable", "admission", "ID is unknown, evicted, terminal, superseded or no longer the current cancelable operation"}, commandFailureDetail{"operation-committing", "admission", "Atomic publication/cleanup has begun"})
		c.Completion = &commandCompletionDetail{Mode: "immediate cancellation request; inspect until terminal", Observe: []string{"appearance.work.inspect", "project.inspect", "events.poll"}}
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
	schema["x-details-scope"] = "preference domains, bounded appearance job identity/lifecycle and captured persistence outcomes; not an exhaustive OS error taxonomy"
	return true
}
