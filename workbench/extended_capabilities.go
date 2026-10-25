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
		c.Effects = []string{"Admit one silent asynchronous independent experiment; 12 kHz mono; requested start/duration round to nearest sample and metadata reports the actual grid. At most 3000 captured samples plus 12000 pre-roll steps; 30-second cooperative budget. Each solver output retains existing attempt/time/accuracy limits", "Snapshot.audio_preview reports ID, status, source_current, source generation/revision, byte count, digest and failure. No live state, history, project data or dirty changes. No autoplay", "Replacement, electrical/content edit or Close cancels work and stops playback. Keep only the latest preview; an outstanding worker must retire before another render"}
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
	return true
}
