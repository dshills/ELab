package workbench

func addSpectrumCapabilityDetails(id string, schema map[string]any) bool {
	if addSpectrumMeasurementDetails(id, schema) || addLiveSpectrumCapabilityDetails(id, schema) {
		return true
	}
	if id != "spectrum.analyze" && id != "spectrum.settings.set" && id != "spectrum.export" {
		return false
	}
	c := commandContractDetail{Version: 1, Scope: "captured-spectrum-analysis", Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed or out-of-range interval, probe, grid, window, fundamental or export format; accepted state is preserved"}, {"not-ready", "admission", "A retained completed recording/current exact spectrum result is required"}}, Effects: []string{"Session-only captured-data analysis; does not change live physics, time, graph or Undo history"}}
	if id == "spectrum.export" {
		inputProperty(schema, "format")["enum"] = []string{"json", "csv"}
		c.Preconditions = []string{"id equals snapshot.spectrum.result.id of the exact current completed spectrum; normal project-file destination and generation guards apply"}
		c.Completion = &commandCompletionDetail{Mode: "immediate file-work admission or waiting compatibility path", Observe: []string{"project.inspect"}, Cancel: "project.work.cancel"}
	} else {
		inputProperty(schema, "window")["enum"] = []string{"", "rectangular", "hann"}
		inputProperty(schema, "samples")["enum"] = []int{0, 32, 64, 128, 256, 512, 1024, 2048, 4096}
		inputProperty(schema, "harmonics")["enum"] = []int{0, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		frequency := inputProperty(schema, "fundamental_hz")
		delete(frequency, "type")
		delete(frequency, "$ref")
		frequency["anyOf"] = []any{map[string]any{"type": "number", "minimum": 0, "maximum": 1e6}, map[string]any{"type": "string", "minLength": 1, "maxLength": 128}}
		frequency["description"] = "Finite frequency 0–1e6 Hz, numeric SI or decimal text with Hz/kHz/mHz; zero leaves THD unset"
		for _, key := range []string{"start_s", "end_s"} {
			inputProperty(schema, key)["minimum"] = 0
		}
		schema["x-defaults"] = map[string]any{"window": "rectangular", "samples": 0, "harmonics": 5, "end_s": "recording end", "id": "selected completed operation", "probe_id": "first captured probe"}
		schema["x-sampling"] = "Power-of-two uniform end-exclusive grid, never finer than the largest recorded sample gap; duplicate timestamps choose right side; mean removed; window coherent gain corrected; no claimed noise floor or THD+N"
		schema["x-thd"] = "Declared fundamental must span at least two integer cycles and requested harmonics must be strictly below Nyquist; otherwise FFT succeeds with explicit THD-unavailable reason"
		c.Preconditions = []string{"Analyze needs a retained completed recording, at least 32 distinct sample intervals and a captured probe; settings can be drafted before a recording exists"}
		c.Effects = append(c.Effects, "Empty analyze params use current spectrum settings; a nonempty parameter object is a complete replacement with declared defaults; snapshot.spectrum.settings retains draft auto/end choices; result.settings records the actual normalized grid and interval")
	}
	schema["x-command-contract"] = c
	return true
}
