package workbench

// PartActuationHint is the concise explanation shared by native controls and
// detailed command discovery. State requirements and outcomes are expanded in
// the command contract; the model itself is unchanged.
func PartActuationHint(kind string) string {
	switch kind {
	case "switch":
		return "Current transient readings: changes apply after the next sample; paused changes wait for Step or Resume. Otherwise the saved startup state changes."
	case "potentiometer":
		return "Current transient readings: movement updates at the same time after calculation, preserving capacitor voltage. Paused stale readings refresh on Restart or Resume."
	}
	return ""
}

func addActuationCapabilityDetails(id string, schema map[string]any) bool {
	if addPowerCapabilityDetails(id, schema) {
		return true
	}
	if addSensorCapabilityDetails(id, schema) {
		return true
	}
	if id == "button.set" || id == "button.click" {
		return addButtonCapabilityDetails(id, schema)
	}
	kind := ""
	switch id {
	case "switch.set":
		kind = "switch"
	case "potentiometer.wiper.set":
		kind = "potentiometer"
	default:
		return false
	}
	target := inputProperty(schema, "id")
	target["minLength"], target["maxLength"] = 1, 64
	target["pattern"] = "^[A-Za-z][A-Za-z0-9_-]{0,63}$"
	target["description"] = "Exact current component ID with kind " + kind
	schema["x-target-inventory"] = "snapshot.components filtered by kind " + kind
	schema["x-interaction-summary"] = PartActuationHint(kind)
	contract := commandContractDetail{Version: 1, Scope: "interactive-part-actuation", Preconditions: []string{"Target is an existing component of the declared kind; incomplete/stale/DC drawings may be configured", "Fresh network transient means snapshot.transient is initialized and not stale, with a fresh network state; running and paused both qualify"}, Effects: []string{"Ordinary mutation guards and retry retention apply; a replay returns the original owned response, not current readiness", "Inside edit.batch, these are ordinary startup configuration edits, not live actuation; changed running edits restart and paused edits stale results; undo/redo follow ordinary schematic history"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed/missing input, wrong or unknown target kind/ID, or invalid value"}}, Completion: &commandCompletionDetail{Observe: []string{"project.inspect", "simulation.wait", "events.poll"}}}
	if kind == "switch" {
		schema["x-state-inventory"] = "snapshot.switch_states: desired_closed, optional applied_closed, pending; absent applied state means no fresh applied reading"
		inputProperty(schema, "closed")["description"] = "Required Boolean saved desired startup state; false opens, true closes"
		contract.Effects = append(contract.Effects,
			"A fresh network transient commits the saved desired state and one undo entry immediately, preserving accepted time/charge/run/electrical revision; actuation revision changes separately",
			"All pending switches apply as one staged group after the next accepted output sample advances the old applied circuit; a failed/canceled advance publishes neither partial time nor partial switch application",
			"Paused changes remain pending until Step/Resume; simulation.wait alone does not advance time or apply a pending switch",
			"Repeated desired values are no-ops; toggles coalesce to the final desired Boolean; returning to the applied state cancels that pending change",
			"No separate input_work is launched; a changed desired state supersedes active input/manual-step work. Restart consumes the saved desired startup states at time zero",
			"DC, stale or incomplete changes use ordinary electrical edit/restart/paused-stale behavior. Save/Verify capture desired startup states; live symbol/readings follow applied state")
		contract.Errors = append(contract.Errors, commandFailureDetail{"solve-failed", "step_work-or-live-simulation", "Later sample/event solve fails; inspect the resulting work/diagnostic rather than attributing failure to desired-state admission"}, commandFailureDetail{"integration-attempt-limit", "step_work-or-live-simulation", "Later sample exceeds the shared integration-attempt budget"}, commandFailureDetail{"integration-time-limit", "step_work-or-live-simulation", "Later sample exceeds the shared integration CPU budget or wall-time backstop"})
		contract.Completion.Mode = "changed fresh transient: desired-state admission immediately; application waits for a subsequent successful sample. Otherwise ordinary startup edit/initialization; unchanged values preserve the applied state. Inspect switch_states and readiness for the actual path"
		schema["examples"] = []any{SwitchParams{ID: "S1", Closed: true}}
	} else {
		position := inputProperty(schema, "position")
		position["minimum"], position["maximum"], position["x-unit"] = 0, 1, "fraction"
		position["description"] = "Finite fraction: 0 toward pin 2, 1 toward pin 1; 0.5 represents 50%. Native percentage controls normalize to this fraction"
		schema["x-work-inventory"] = "Latest snapshot.input_work only; require matching ID, command and requested_wiper before attributing a retained outcome. Fallback edits do not create a new input job"
		schema["x-model-id"] = PotentiometerModelID
		contract.Effects = append(contract.Effects,
			"A fresh network transient admits background input_work even while paused: solve at the committed simulated time with current capacitor voltages and previously applied switch states",
			"Admission preserves committed graph/readings/history; requested_wiper is separate until successful atomic publication. Success commits the wiper, one actor-attributed undo entry and both sides of its event, preserving charge/time/run/electrical revision",
			"New changed requests supersede active input/manual-step work. Setting the already committed position is a no-op except that it supersedes an active wiper request for the same potentiometer",
			"Failure/cancel/supersession leaves committed wiper/charge/time/history intact; only the latest input_work is retained; old retired jobs cannot publish",
			"Inspect input_work until completed before a dependent Step. Cancel through simulation.work.cancel with the exact active input_work.id; Pause also requests cancellation",
			"DC, stale or incomplete drawings use ordinary electrical edit/restart/paused-stale behavior. Save/Verify capture committed wiper/startup, not a staged request or live movement history")
		contract.Errors = append(contract.Errors, commandFailureDetail{"solve-failed", "input_work", "Algebraic staging fails or its cooperative deadline expires"}, commandFailureDetail{"canceled", "input_work", "Explicit/Pause cancellation preserves committed values"}, commandFailureDetail{"superseded", "input_work", "A newer request, committed-position cancellation, switch/input/source edit, run or document invalidates the captured job"})
		contract.Completion.Mode = "changed fresh transient: immediate admission with asynchronous atomic input_work outcome; unchanged values may only supersede a matching pending request. Otherwise ordinary edit admission; inspect which path actually occurred"
		contract.Completion.Cancel = "simulation.work.cancel"
		contract.Completion.CooperativeDeadlineSeconds = int(InputWorkDeadline.Seconds())
		schema["examples"] = []any{WiperParams{ID: "P1", Position: .5}}
	}
	schema["x-command-contract"] = contract
	return true
}

func addButtonCapabilityDetails(id string, schema map[string]any) bool {
	target := inputProperty(schema, "id")
	target["minLength"], target["maxLength"] = 1, 64
	target["pattern"] = "^[A-Za-z][A-Za-z0-9_-]{0,63}$"
	schema["x-target-inventory"] = "snapshot.components filtered by pushbutton or bouncing-pushbutton"
	schema["x-model-id"] = ControlModelID
	schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "momentary-button-actuation", Preconditions: []string{"Existing momentary button, fresh initialized network transient, running or paused", "No active input work; click additionally requires a released button whose bounce has finished"}, Effects: []string{"Admit staged input_work; observe requested_button and button_click together with work ID/command", "Press/release commits at the accepted simulation time, preserving capacitor voltage, coil current, run, document dirtiness and undo history", "button.click schedules automatic release after 50 ms of simulation time; button.set holds until explicitly released", "The bouncing kind toggles four times at 1 ms intervals after press and release; clean buttons do not bounce", "Step or Resume advances timed contact events; Restart/reopen clears runtime button history. Verify starts an independent released circuit", "Ordinary revision/document/retry guards apply; runtime actuation is excluded from edit.batch"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Missing/wrong-kind ID or invalid pressed boolean"}, {"unavailable", "admission", "Stale/incomplete circuit, active input work, or click while pressed/bouncing"}, {"solve-failed", "input_work", "Event solve fails or cooperative work deadline expires; committed runtime state is preserved"}, {"canceled", "input_work", "Explicit work cancellation or Pause before publication"}, {"superseded", "input_work", "A newer run/document/input invalidates this staged solve"}}, Completion: &commandCompletionDetail{Mode: "Immediate admission, asynchronous atomic input_work completion; unchanged already-held/released set is a no-op", Observe: []string{"snapshot.input_work", "simulation.wait", "events.poll", "snapshot.mixed.controls"}, Cancel: "simulation.work.cancel", CooperativeDeadlineSeconds: int(InputWorkDeadline.Seconds())}}
	return true
}
