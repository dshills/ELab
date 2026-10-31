package workbench

func addExtendedCapabilityDetails(id string, schema map[string]any) bool {
	bounds := func(name, unit string, min, max float64) {
		p := inputProperty(schema, name)
		if p != nil {
			p["minimum"], p["maximum"], p["x-unit"] = min, max, unit
		}
	}
	c := commandContractDetail{Version: 1, Scope: "hobbyist-device-configuration", Preconditions: []string{"Ordinary revision, document-generation, retry and closed-service guards apply"}, Effects: []string{}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Missing, extra, wrong-type, null-required, out-of-range or wrong-target fields preserve accepted state"}}}
	switch id {
	case "recording.step-response", "recording.step-response.set":
		c.Scope = "captured-step-response"
		bounds("initial_value", "captured probe SI unit", -1e6, 1e6)
		bounds("final_value", "captured probe SI unit", -1e6, 1e6)
		bounds("tolerance_fraction", "fraction of step amplitude", 0, .5)
		if p := inputProperty(schema, "tolerance_fraction"); p != nil {
			p["x-enabled-minimum"] = .0001
			p["description"] = "0 only when clearing with empty probe/all-zero settings; otherwise 0.0001–0.5"
		}
		schema["x-inventory"] = map[string]string{"ids": "retained completed operations; set requires snapshot.recording.id", "probe_ids": "operation.inspect metadata.probes", "time_bounds": "actual saved first/last timestamps inside nominal duration"}
		c.Preconditions = []string{"Retained completed recording, captured probe and fully covered positive interval"}
		c.Errors = append(c.Errors, commandFailureDetail{"not-ready", "admission", "Capture absent or incomplete"}, commandFailureDetail{"not-found", "admission", "Probe absent"}, commandFailureDetail{"unavailable", "admission", "Actual timestamps do not cover the interval"}, commandFailureDetail{"conflict", "admission", "Set names a changed selected capture"})
		c.Completion = &commandCompletionDetail{Mode: "bounded in-memory saved-row analysis; no file IO/worker", Observe: []string{"recording.step-response", "project.inspect"}}
		c.Effects = []string{"Pure query or session-only selected recording settings, no graph/physics/history mutation. Requires fully covered positive interval, explicit distinct initial/final references (±1e6 SI), tolerance 0.0001–0.5 of step amplitude and 0 < minimum_hold_s ≤ interval. Full fields required; set with empty probe and all-zero settings disables.", "Direction-normalized overshoot and last arrival into supplied final-value band, linearly interpolated saved segments including duplicate-time jumps. Observed settling is reported only if the trace then stays in band for the requested hold through end_s. Unsettled/short captures have a reason, no fabricated settling value. No inferred steady-state reference, unseen peaks, future guarantee or hardware qualification. Retained recording identity/engine and settings accompany every result."}
		if id == "recording.step-response.set" {
			c.Preconditions = append(c.Preconditions, "Exact currently selected retained recording ID; stale controls/replaced document refuse")
		}
	case "thyristor.configure":
		c.Effects = []string{thyristorDescription, "Complete saved undoable startup edit; all fields required. Holding current must be strictly below latching current. Edits restart the experiment from blocking state; gate inputs use ordinary sources/buttons, no hidden control command. Rejected edits preserve accepted source/history. Captures retain model/config, conduction/latching/polarity state and all pin currents."}
	case "photodiode.configure", "photodiode.light.set":
		bounds("incident_light_uw", "µW", 0, 1000)
		c.Effects = []string{photodiodeDescription, "Configuration is a complete saved undoable startup edit. Light command stages atomic input_work at accepted time for a fresh transient, preserves voltage/charge/run/time and publishes one saved environment/history edit; newer inputs supersede pending work and failures preserve the committed state. Inspect requested_optical_light and simulation.wait. Incomplete/stale drawings use ordinary startup edit semantics. Parameter light in the existing sweep workflow solves independent DC optical operating points."}
		c.Completion = &commandCompletionDetail{Mode: "configuration edit or asynchronous atomic input_work for fresh light change", Observe: []string{"project.inspect", "simulation.wait", "snapshot.input_work"}, Cancel: "simulation.work.cancel", CooperativeDeadlineSeconds: int(InputWorkDeadline.Seconds())}
	case "thermal.configure":
		c.Effects = []string{thermalDescription, "All fields required even when disabling; enabled=false restores the exact original law. Supported resistor, silicon diode, NPN and PNP only. Ambient -40..125 °C; accepted self-heated state -40..150 °C. Settings are saved, undoable electrical startup edits. Thermal readings are a separate heat domain, not electrical capacitor energy."}
	case "jfet.configure":
		c.Effects = []string{jfetDescription, "Complete saved undoable electrical edit, all fields required. Cutoff sign must match N/P kind. Up to two JFETs within the four-FET network budget."}
	case "current-source.configure", "load.configure":
		c.Effects = []string{currentDeviceDescription, "Complete saved undoable startup edit; all fields and pulse timing required even in DC. Source 0–1 A/compliance 1–24 V; load cc 0–1 A or cp 0–24 W, minimum 0.1–12 V, current limit 0.001–1 A. Low level must not exceed high. Pulse phase/edge rules match finite source timing. No solver-limit changes."}
	case "passive.configure":
		schema["x-passive-version"] = PassiveLossModelID
		c.Effects = []string{"Saved undoable capacitor/inductor configuration; all fields required, including when disabling. Unused kind-specific fields must be zero. Enabled=false removes parameters and restores the exact original ideal model. Version elab.passive-loss.v1.", "Capacitor ESR is series with ideal storage; leakage is parallel across external terminals. Initial voltage and energy refer to internal storage, external probes report total terminal voltage/current. Inductor winding loss is series resistance. Results include storage voltage/current and each nonnegative heat loss; captures validate them. No extra storage slots or solver-limit changes."}
	case "speaker.configure":
		c.Effects = []string{"Complete saved undoable passive loudspeaker configuration, version elab.loudspeaker-impedance.v1. Series coil R/L and parallel motional R/L/C, with C derived from resonance_hz and motional_inductance_h. Zero-energy startup; no calibrated acoustic output or manufacturer rating", "Two private inductors, one capacitor, two resistors and two nodes count against existing budgets. AC and transient use the same passive equivalent. Voltage/current, terminal power, coil and motional heat, stored energy and internal state are captured and validated", "Simulation is silent. Existing explicit audio.render/play/stop/read and WAV export support a normalized terminal-voltage illustration, not SPL/cone/radiation prediction"}
		schema["x-speaker-version"] = SpeakerModelID
	case "transistor.profile.set":
		schema["x-profile-version"] = TransistorProfileModelID
		bounds("parameters.forward_beta", "ratio", 1, 1000)
		bounds("parameters.reverse_beta", "ratio", .1, 100)
		bounds("parameters.saturation_current_a", "A", 1e-18, 1e-10)
		schema["x-early-voltage"] = "0 disables; otherwise 1–1000 V, as defined by the referenced profile schema"
		bounds("parameters.base_emitter_capacitance_f", "F", 0, 1e-6)
		bounds("parameters.base_collector_capacitance_f", "F", 0, 1e-6)
		c.Effects = []string{"Complete saved, undoable NPN/PNP educational profile. All fields required and validated even when disabling. Enabled=false restores exact generic-v1 law. Early voltage zero disables Early term; positive values must be at least 1 V. No manufacturer identity or full SPICE-card fidelity is claimed", "Optional constant CBE/CBC use biased startup and consume the existing four-capacitor and total-part budgets, including other private device storage. Transient and AC stamps include charge, terminal currents and stored energy; edits restart the experiment. Profile in device_parameters and per-result model parameters are captured and validated"}
	case "pwm.configure":
		bounds("parameters.frequency_hz", "Hz", 10, 1000)
		bounds("parameters.duty_fraction", "fraction", .1, .9)
		c.Effects = []string{"Complete saved PWM configuration; edits are undoable electrical startup edits. Running circuits restart; paused readings stay stale until Restart/Resume", "Generic clock drives physical finite-resistance output paths; its edges participate in integration and output sampling advice. Frequency/duty do not depend on the experiment's primary input mode"}
	case "opamp.dynamics.set":
		bounds("parameters.gain_bandwidth_hz", "Hz", 100, 1e6)
		bounds("parameters.slew_v_per_s", "V/s", 100, 1e7)
		c.Effects = []string{"Saved undoable startup edit; enabled=false removes optional dynamics and restores the original memoryless law. Both config values remain required and range-validated even when disabling", "Enabled circuits declare one private compensation capacitor/node per op-amp within existing storage/part/node limits. The base model identity and all existing circuits without this option remain unchanged"}
	case "sound.configure":
		bounds("parameters.capacitance_f", "F", 1e-9, 1e-6)
		bounds("parameters.frequency_hz", "Hz", 100, 4000)
		c.Effects = []string{"Saved undoable startup edit for an existing piezo or active-buzzer; both fields required. Piezo uses capacitance, active buzzer uses preview frequency; the unused field is retained but has no effect for that kind", "No rendering or playback occurs. Passive piezo capacitance is included in existing storage/constraint limits; insert series resistance before an ideal source"}
	case "audio.render":
		c.Scope = "explicit-audio-preview"
		bounds("start_s", "s", 0, 1)
		bounds("duration_s", "s", .02, .25)
		bounds("gain", "fraction", 0, .25)
		c.Effects = []string{"Admit one silent asynchronous independent experiment; 12 kHz mono; requested start/duration round to nearest sample and metadata reports the actual grid. At most 3000 captured samples plus 12000 pre-roll steps; 30-second cooperative budget. Each solver output retains existing attempt/time/accuracy limits", "Snapshot.audio_preview reports ID, status, source_current, source generation/revision, byte count, digest and failure. No live state, history, project data or dirty changes. No autoplay. Loudspeaker previews use 100 Hz high-pass terminal voltage, peak-normalized before gain; piezo/buzzer behavior is unchanged", "Replacement, electrical/content edit or Close cancels work and stops playback. Keep only the latest preview; an outstanding worker must retire before another render"}
		c.Completion = &commandCompletionDetail{Mode: "Immediate admission; completion is asynchronous", Observe: []string{"project.inspect", "audio.read", "events.poll"}, Cancel: "audio.stop", CooperativeDeadlineSeconds: 30}
	case "audio.play":
		c.Errors = append(c.Errors, commandFailureDetail{"audio-unavailable", "admission", "Playback backend cannot start; existing accepted playback and captured WAV remain unchanged"})
		c.Scope = "explicit-audio-preview"
		c.Effects = []string{"Explicitly play the exact current completed preview ID once; stops any previous owned playback. No autoplay or loops. macOS AVAudioPlayer required; unsupported builds can read/export WAV", "Completion means playback scheduled, not hardware audibility. Snapshot playing clears on completion, Stop, source change or Close"}
	case "audio.stop":
		c.Scope = "explicit-audio-preview"
		c.Effects = []string{"Cancel rendering and stop playback for the exact current ID; repeated Stop is harmless. Completed bytes remain readable and replayable while source_current"}
	case "audio.read":
		c.Scope = "explicit-audio-preview"
		c.Effects = []string{"Pure owned metadata/WAV read, no disk or playback; at most 6044 bytes base64. Exact ID rejects replaced previews; stale bytes remain readable with source_current=false", "Export through export.start kind=audio-wav, id=the exact current completed preview. Shared captured file worker supplies overwrite/alias guards, digest, completion and cancellation"}
	default:
		return false
	}
	schema["x-command-contract"] = c
	schema["x-model-id"] = ExtendedModelID
	if id == "recording.step-response" || id == "recording.step-response.set" {
		delete(schema, "x-model-id")
		schema["x-method-id"] = StepResponseMethod
	}
	if id == "thyristor.configure" {
		schema["x-model-id"] = ThyristorModelID
	}
	if id == "photodiode.configure" || id == "photodiode.light.set" {
		schema["x-model-id"] = PhotodiodeModelID
	}
	if id == "thermal.configure" {
		schema["x-model-id"] = ThermalModelID
	}
	if id == "jfet.configure" {
		schema["x-model-id"] = JFETModelID
	}
	if id == "current-source.configure" || id == "load.configure" {
		schema["x-model-id"] = CurrentDeviceModelID
	}
	if id == "passive.configure" {
		schema["x-model-id"] = PassiveLossModelID
	}
	if id == "speaker.configure" {
		schema["x-model-id"] = SpeakerModelID
	}
	if id == "transistor.profile.set" {
		schema["x-model-id"] = TransistorProfileModelID
	}
	return true
}

func addTransistorProfileSchema(schema map[string]any) {
	if p := inputProperty(schema, "version"); p != nil {
		p["enum"] = []string{TransistorProfileModelID}
	}
	for _, f := range []struct {
		name, unit string
		min, max   float64
	}{{"forward_beta", "ratio", 1, 1000}, {"reverse_beta", "ratio", .1, 100}, {"saturation_current_a", "A", 1e-18, 1e-10}, {"base_emitter_capacitance_f", "F", 0, 1e-6}, {"base_collector_capacitance_f", "F", 0, 1e-6}} {
		p := inputProperty(schema, f.name)
		if p == nil {
			continue
		}
		p["minimum"], p["maximum"], p["x-unit"] = f.min, f.max, f.unit
	}
	p := inputProperty(schema, "early_voltage_v")
	if p == nil {
		return
	}
	delete(p, "type")
	p["anyOf"] = []any{map[string]any{"type": "number", "enum": []int{0}}, map[string]any{"type": "number", "minimum": 1, "maximum": 1000}}
	p["x-unit"] = "V"
}

func addSpeakerSchema(schema map[string]any) {
	if p := inputProperty(schema, "version"); p != nil {
		p["enum"] = []string{SpeakerModelID}
	}
	for _, f := range []struct {
		name, unit string
		min, max   float64
	}{{"coil_resistance_ohm", "ohm", 1, 100}, {"coil_inductance_h", "H", 1e-5, .1}, {"resonance_hz", "Hz", 20, 1000}, {"motional_resistance_ohm", "ohm", 1, 1000}, {"motional_inductance_h", "H", .0001, 1}} {
		if p := inputProperty(schema, f.name); p != nil {
			p["minimum"], p["maximum"], p["x-unit"] = f.min, f.max, f.unit
		}
	}
}

func addPassiveSchema(schema map[string]any) {
	if p := inputProperty(schema, "version"); p != nil {
		p["enum"] = []string{PassiveLossModelID}
	}
	for _, v := range []struct {
		name     string
		min, max float64
	}{{"esr_ohm", .001, 1e5}, {"leakage_resistance_ohm", 1000, 1e12}, {"winding_resistance_ohm", .001, 1e5}} {
		if p := inputProperty(schema, v.name); p != nil {
			delete(p, "type")
			p["anyOf"] = []any{map[string]any{"type": "number", "enum": []int{0}}, map[string]any{"type": "number", "minimum": v.min, "maximum": v.max}}
			p["x-unit"] = "ohm"
		}
	}
	schema["x-unused-fields"] = "capacitor requires winding_resistance_ohm=0; inductor requires esr_ohm=0 and leakage_resistance_ohm=0"
}

func addCurrentSchema(schema map[string]any, kind string) {
	bounds := func(name, unit string, min, max float64) {
		if p := inputProperty(schema, name); p != nil {
			p["minimum"], p["maximum"], p["x-unit"] = min, max, unit
		}
	}
	if kind == "timing" {
		for _, name := range []string{"high_s", "low_s", "rise_s", "fall_s"} {
			bounds(name, "s", 1e-6, 10)
		}
		schema["x-relations"] = "period ≥ 1 ms; rise ≤ high phase; fall ≤ low phase. All fields required even when pulsed=false."
		return
	}
	if p := inputProperty(schema, "version"); p != nil {
		p["enum"] = []string{CurrentDeviceModelID}
	}
	if kind == "source" {
		bounds("current_a", "A", 0, 1)
		bounds("low_current_a", "A", 0, 1)
		bounds("compliance_v", "V", 1, 24)
	} else {
		if p := inputProperty(schema, "mode"); p != nil {
			p["enum"] = []string{"cc", "cp"}
		}
		bounds("level", "A in cc; W in cp", 0, 24)
		bounds("low_level", "A in cc; W in cp", 0, 24)
		bounds("minimum_voltage_v", "V", .1, 12)
		bounds("maximum_current_a", "A", .001, 1)
		schema["x-mode-bound"] = "CC levels ≤ 1 A; CP levels ≤ 24 W"
	}
	schema["x-level-relation"] = "low ≤ high; DC uses high level, pulse timing is independent of primary input mode"
}

func addJFETSchema(schema map[string]any) {
	if p := inputProperty(schema, "version"); p != nil {
		p["enum"] = []string{JFETModelID}
	}
	for _, f := range []struct {
		name, unit string
		min, max   float64
	}{{"idss_a", "A", 1e-5, .1}, {"cutoff_v", "V", -12, 12}, {"lambda_per_v", "1/V", 0, .1}} {
		if p := inputProperty(schema, f.name); p != nil {
			p["minimum"], p["maximum"], p["x-unit"] = f.min, f.max, f.unit
		}
	}
	schema["x-cutoff-sign"] = "N: -12 to -0.1 V; P: +0.1 to +12 V; zero and |cutoff|<0.1 V refused"
}

func addThermalSchema(schema map[string]any) {
	inputProperty(schema, "version")["enum"] = []string{ThermalModelID}
	for _, f := range []struct {
		name, unit string
		min, max   float64
	}{{"ambient_c", "°C", -40, 125}, {"thermal_resistance_k_per_w", "K/W", .1, 2000}, {"thermal_capacitance_j_per_k", "J/K", .001, 10}, {"resistance_coefficient_per_k", "1/K", -.005, .005}} {
		p := inputProperty(schema, f.name)
		p["minimum"], p["maximum"], p["x-unit"] = f.min, f.max, f.unit
	}
	schema["x-kind-relation"] = "resistance_coefficient_per_k must be zero unless the target is a resistor"
}

func addPhotodiodeSchema(schema map[string]any) {
	inputProperty(schema, "version")["enum"] = []string{PhotodiodeModelID}
	for _, f := range []struct {
		name, unit string
		min, max   float64
	}{{"responsivity_a_per_w", "A/W", .01, 1}, {"incident_light_uw", "µW", 0, 1000}, {"dark_saturation_current_a", "A", 1e-15, 1e-6}, {"junction_capacitance_f", "F", 0, 1e-6}} {
		p := inputProperty(schema, f.name)
		p["minimum"], p["maximum"], p["x-unit"] = f.min, f.max, f.unit
	}
}

func addThyristorSchema(schema map[string]any) {
	inputProperty(schema, "version")["enum"] = []string{ThyristorModelID}
	for _, f := range []struct {
		name, unit string
		min, max   float64
	}{{"gate_trigger_current_a", "A", 1e-5, .01}, {"latching_current_a", "A", .0001, 1}, {"holding_current_a", "A", 1e-5, .05}, {"on_resistance_ohm", "Ω", .1, 100}, {"forward_drop_v", "V", 0, 2}, {"gate_resistance_ohm", "Ω", 10, 10000}} {
		p := inputProperty(schema, f.name)
		p["minimum"], p["maximum"], p["x-unit"] = f.min, f.max, f.unit
	}
	schema["x-relations"] = []map[string]any{{"operator": "less-than", "fields": []string{"holding_current_a", "latching_current_a"}}}
}
