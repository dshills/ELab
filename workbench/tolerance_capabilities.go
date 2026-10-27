package workbench

func addToleranceCapabilityDetails(id string, schema map[string]any) bool {
	if id != "tolerance.settings.set" && id != "tolerance.run" && id != "tolerance.cancel" && id != "tolerance.export" {
		return false
	}
	c := commandContractDetail{Version: 1, Scope: "independent-tolerance-analysis", Effects: []string{"No graph, live time, readings, project dirty flag or history change; captured results are session-only; JSON/CSV export preserves exact source and per-trial provenance"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed or out-of-range settings or unsupported graph/probe preserve accepted state"}, {"busy", "admission", "Only one tolerance worker; canceled workers retain slot until retirement"}}}
	if id == "tolerance.cancel" {
		c.Effects = append(c.Effects, "Exact ID required; queued/running/canceling run is cooperatively canceled; completed result remains unchanged")
	} else if id == "tolerance.export" {
		if p := inputProperty(schema, "format"); p != nil {
			p["enum"] = []string{"json", "csv"}
		}
		c.Effects = append(c.Effects, "Exact current completed result ID required; shared captured atomic file worker supplies overwrite, alias, destination and generation protection")
		c.Completion = &commandCompletionDetail{Mode: "captured file work; immediate admission or waiting compatibility path", Observe: []string{"project.inspect"}, Cancel: "project.work.cancel"}
	} else {
		if p := inputProperty(schema, "mode"); p != nil {
			p["enum"] = []string{"dc", "ac-gain"}
		}
		for _, f := range []struct {
			name     string
			min, max float64
		}{{"trials", 2, 64}, {"seed", 0, 9007199254740991}, {"frequency_hz", 0, 100000}, {"resistance_fraction", 0, .5}, {"capacitance_fraction", 0, .5}, {"beta_fraction", 0, .5}} {
			p := inputProperty(schema, f.name)
			if p != nil {
				p["minimum"], p["maximum"] = f.min, f.max
			}
		}
		c.Effects = append(c.Effects, "Empty params use current settings; nonempty object replaces the full settings. Mode dc opens capacitors and may omit frequency_hz; ac-gain requires 0.01–100000 Hz. Mode ac-gain measures dB voltage gain at frequency_hz about independently solved configured DC bias", "Stable SplitMix64 top-53-bit uniform draws in captured component order vary authored resistor/capacitor values and explicit profile forward beta; generic transistors do not vary. Existing component/solver limits apply. Every failed trial is retained with diagnostic; statistics count successes and failures explicitly", "Source generation/electrical identity, exact captured model-bearing document and SHA-256 are exposed; later live edits mark source_current=false without rewriting captured results")
		if id == "tolerance.run" {
			c.Completion = &commandCompletionDetail{Mode: "immediate admission; independent asynchronous worker", Observe: []string{"project.inspect", "events.poll"}, Cancel: "tolerance.cancel", CooperativeDeadlineSeconds: 30}
		}
	}
	schema["x-command-contract"] = c
	schema["x-engine-id"] = ToleranceEngineID
	return true
}
