package workbench

// The detailed catalog describes the same state checks and work limits used by
// user controls. Common envelope/revision/retry failures remain separate.
func addLiveControlDetails(id string, schema map[string]any) bool {
	switch id {
	case "simulation.transient.set", "simulation.input.set", "simulation.pause", "simulation.resume", "simulation.restart", "simulation.step", "simulation.step.start", "simulation.work.cancel", "simulation.wait":
	default:
		return false
	}
	contract := commandContractDetail{Version: 1, Scope: "live-simulation-controls", Preconditions: []string{}, Effects: []string{}, Errors: []commandFailureDetail{}, Completion: &commandCompletionDetail{Mode: "immediate command result; inspect state for physical readiness"}}
	failure := func(code, phase, when string) {
		contract.Errors = append(contract.Errors, commandFailureDetail{code, phase, when})
	}
	failure("invalid-params", "admission", "Malformed or missing input, invalid values or relationships")
	observe := []string{"project.inspect", "simulation.wait", "events.poll"}
	switch id {
	case "simulation.transient.set":
		rangeSI := func(field, unit string, lo, hi float64) {
			p := inputProperty(schema, field)
			p["minimum"], p["maximum"], p["x-unit"] = lo, hi, unit
			if unit == "s" {
				p["x-normalized-si-range"] = map[string]float64{"minimum": lo, "maximum": hi}
			}
		}
		rangeSI("step_s", "s", 1e-6, 1)
		rangeSI("high_s", "s", 1e-6, 10)
		rangeSI("low_s", "s", 1e-6, 10)
		rangeSI("initial_voltage_v", "V", 0, 12)
		rangeSI("seconds_per_wall_second", "simulated_s/wall_s", .001, 1)
		schema["x-normalized-si-relations"] = []inputRelation{
			{Operator: "ratio_maximum", Fields: []string{"seconds_per_wall_second", "step_s"}, Limit: MaxLiveSamplesPerWallSecond},
			{Operator: "less_or_equal", Fields: []string{"step_s", "high_s"}, WhenKind: "pulse-source"},
			{Operator: "less_or_equal", Fields: []string{"step_s", "low_s"}, WhenKind: "pulse-source"},
		}
		contract.Preconditions = []string{"All five settings required; incomplete graphs may be configured", "Live playback speed/step <=1000 samples per wall second; ideal pulse-source high/low phases must each cover one step, even if graph is incomplete"}
		contract.Effects = []string{"Changed settings form one undoable schematic edit and invalidate/restart the live run from configured startup; identical settings preserve physical state/history", "While paused, changed settings leave results stale until Resume or explicit Restart; per-capacitor signed startup overrides take precedence over the 0–12 V global fallback", "Per-source finite pulse/sine waveforms keep their separate settings; phase/step restriction here applies to legacy ideal pulse-source"}
		contract.Completion = &commandCompletionDetail{Mode: "settings commit immediately; fresh initialization may be asynchronous or deferred while paused", Observe: observe, Cancel: "simulation.pause"}
		schema["examples"] = []any{DefaultTransientSettings()}
		schema["x-rounding-policy"] = "Playback limit compares speed <= rounded(1000*step_s), avoiding quotient-rounding rejection at the exact boundary; next larger representable speed refuses."
	case "simulation.input.set":
		inputProperty(schema, "mode")["enum"] = []string{"auto", "charge", "discharge", "high", "low"}
		schema["x-input-mode-inventory"] = "snapshot.live_input_modes for a complete supported graph; membership does not imply fresh-result readiness"
		schema["x-target-source"] = "snapshot.transient.source_id; network override affects only the compiled primary source"
		contract.Preconditions = []string{"Complete supported transient graph and a fresh initialized transient result", "mode belongs to snapshot.live_input_modes: RC auto/charge/discharge; NPN, mixed, waveform and network transients auto/high/low; DC/unsupported analyses have no live modes"}
		contract.Effects = []string{"Paused input queues pending_input_mode without changing applied input, simulated time or capacitor charge; Step or Resume applies it", "Running exact RC/specialized NPN input is applied immediately; running network input uses atomic background input_work while the live clock waits", "New input supersedes active input/manual-step work; no partial physical publication, source edit, dirty membership or undo entry for mode actuation", "Auto follows configured waveforms at current simulated time; high/low override the primary source rather than all supplies"}
		failure("unavailable", "admission", "Circuit is incomplete, DC/unsupported, uninitialized or stale")
		failure("solve-failed", "input_work", "Background algebraic input application fails or its cooperative deadline expires; committed time/charge/input remain")
		failure("canceled", "input_work", "Explicit/Pause cancellation preserves committed physical state")
		failure("superseded", "input_work", "Newer input/source/run/document operation makes the captured input obsolete")
		contract.Completion = &commandCompletionDetail{Mode: "paused queue, immediate exact-kernel application, or background network input_work according to current state; inspect actual outcome", Observe: observe, Cancel: "simulation.work.cancel", CooperativeDeadlineSeconds: int(InputWorkDeadline.Seconds())}
		schema["examples"] = []any{InputParams{Mode: "auto"}}
	case "simulation.step", "simulation.step.start":
		p := inputProperty(schema, "duration_s")
		p["x-unit"] = "s"
		p["x-normalized-si-range"] = map[string]any{"exclusiveMinimum": 0, "maximum": MaxManualStepSeconds}
		schema["x-max-output-steps"] = MaxManualStepSamples
		schema["x-normalized-si-relations"] = []map[string]any{
			{"operator": "duration_s<=binary64_product(limit,snapshot.transient_settings.step_s)", "limit": MaxManualStepSamples},
			{"operator": "snapshot.transient.sample.time_s + duration_s > snapshot.transient.sample.time_s"},
			{"operator": "duration_s>=minimum for network analyses with capacitors", "minimum": MinCapacitorStepSeconds},
		}
		schema["x-rounding-policy"] = "Duration compares against rounded(1000*step_s); next larger representable duration refuses. The ceil-derived total-sample estimate is capped at 1000 at that boundary."
		contract.Preconditions = []string{"Paused, complete supported transient with fresh physical state; network analysis also needs a fresh network state", "No active input_work or step_work, including canceling; duration is positive, <=1 s, and <= the rounded product of 1000 and step", "Duration advances floating-point time; networks containing capacitors require at least 1e-10 s"}
		contract.Effects = []string{"Capture graph/settings/current charge/time and queued input into isolated work; apply queued input before advancing", "Publish the complete requested advance atomically on success; progress reports attempted output samples without exposing partial physics", "Failure/cancel/supersession preserves committed physical state and pending input; one latest step_work is retained; it is not a kept recording or saved project data"}
		failure("unavailable", "admission", "Not paused, incomplete/DC/unsupported, stale or network state not ready")
		failure("busy", "admission", "Active input or manual step, including canceling")
		failure("solve-failed", "admission-or-step_work", "Network capacitor step below 1e-10 s, solver failure or cooperative worker deadline")
		failure("integration-attempt-limit", "step_work", "Per-output-sample mixed integration attempts exceed the unchanged kernel budget")
		failure("integration-time-limit", "step_work", "Per-output-sample mixed integration wall time exceeds the unchanged kernel budget")
		failure("canceled", "step_work-or-wait", "Explicit/Pause cancellation; waiting caller disconnect stops only the wait after admission")
		failure("superseded", "step_work", "A newer source/input/run/document operation makes the captured work obsolete")
		mode := "immediate admission; retry returns owned original admission; inspect step_work until terminal"
		if id == "simulation.step" {
			mode = "compatibility waiting alias: may wait for pending initialization before admission, then waits for its same job's final response; retry observes original result"
			failure("operation-unavailable", "wait", "Original job expired before its waiting response could be joined")
		}
		contract.Completion = &commandCompletionDetail{Mode: mode, Observe: observe, Cancel: "simulation.work.cancel", CooperativeDeadlineSeconds: int(ManualStepDeadline.Seconds())}
		schema["examples"] = []any{StepParams{Duration: .001}}
	case "simulation.work.cancel":
		schema["x-id-inventory"] = "snapshot.input_work.id or snapshot.step_work.id with queued/running/canceling status; exact current ID required"
		contract.Preconditions = []string{"Exact ID of the current active input or manual-step job"}
		contract.Effects = []string{"Request cooperative cancellation and expose canceling; inspect until terminal", "No partial physical publication or automatic reversal of completed work; terminal/old/unknown IDs refuse"}
		failure("operation-unavailable", "admission", "ID is unknown, replaced or terminal")
		contract.Completion = &commandCompletionDetail{Mode: "immediate cancellation request; actual completion is asynchronous", Observe: observe}
	case "simulation.pause":
		contract.Effects = []string{"Set paused immediately; cancel automatic initialization/clock and active manual-step/input work", "Preserve committed source/results/time/charge and queued input; do not wait for worker termination or mark source/history dirty"}
		contract.Completion = &commandCompletionDetail{Mode: "pause admission immediately; previously active jobs may still be canceling", Observe: observe}
	case "simulation.resume":
		contract.Effects = []string{"Already running/updating is idempotent without restarting the clock", "Fresh transient continues current charge/time; queued input is applied before progression, using background input_work for a network", "Stale/uninitialized/DC results recalculate through the existing initializer; incomplete graphs retain diagnostics rather than inventing a fresh result"}
		failure("unavailable", "admission", "Queued network input cannot capture fresh network state")
		failure("solve-failed", "input_work", "Background queued-input application fails; inspect retained job outcome")
		contract.Completion = &commandCompletionDetail{Mode: "immediate running/initializing admission; inspect readiness and any input work before depending on readings", Observe: observe, Cancel: "simulation.pause"}
	case "simulation.restart":
		contract.Effects = []string{"Start a new live run at configured startup/time zero, retaining pause state; supersede active input/step work and clear live cursors/window", "Consume compatible queued mode according to existing engine startup policy; incompatible analysis modes fall back to Auto", "Preserve graph/settings/edit history/dirty membership and independent retained recordings; incomplete graphs report diagnostics"}
		contract.Completion = &commandCompletionDetail{Mode: "new-run admission; fresh initialization can be asynchronous, including a paused Restart", Observe: observe, Cancel: "simulation.pause"}
	case "simulation.wait":
		contract.Effects = []string{"Wait for status updating plus active project, manual-step and input work to settle, then return an owned current snapshot", "Ready live transients return immediately without waiting for a future sample; does not wait for Verify, independent recovery-read, sweep, printer, appearance or shutdown jobs; recovery work represented by project_work is included", "No computation is launched or canceled by this query; inspect each job/result outcome because settled does not imply success"}
		failure("timeout", "wait", "Five-second bound or earlier caller cancellation ends observation; admitted work continues")
		contract.Completion = &commandCompletionDetail{Mode: "bounded query; re-inspect/re-wait after timeout", Observe: []string{"project.inspect", "events.poll"}, CooperativeDeadlineSeconds: int(SimulationWaitTimeout.Seconds())}
	}
	schema["x-command-contract"] = contract
	return true
}
