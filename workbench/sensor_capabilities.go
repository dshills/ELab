package workbench

func addSensorCapabilityDetails(id string, schema map[string]any) bool {
	kind, parameter, unit, minimum, maximum := "thermistor", "temperature_c", "°C", -40., 125.
	if id == "sensor.light.set" {
		kind, parameter, unit, minimum, maximum = "ldr", "illuminance_lux", "lx", 0, 10000
	} else if id != "sensor.temperature.set" {
		return false
	}
	property := inputProperty(schema, parameter)
	property["minimum"], property["maximum"], property["x-unit"] = minimum, maximum, unit
	schema["x-target-inventory"] = "snapshot.components filtered by kind " + kind
	schema["x-model-id"] = SensorModelID
	schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "sensor-environment", Preconditions: []string{"Existing sensor of the matching kind; finite environment within the declared bounds"}, Effects: []string{"Fresh paused/running network transient: stage input_work at accepted time, preserving capacitor voltage, coil current, control memory and run; inspect requested_temperature/requested_light and wait before dependent commands", "Publication saves the environment and one undo entry; model readings use committed environment until completion", "A newer input supersedes pending work; setting the committed value cancels pending work for that sensor. Failure/cancel/supersession preserves committed graph, readings, time, storage and history", "Incomplete/stale/DC drawing: ordinary startup configuration edit/restart semantics; edit.batch configures startup without live staging", "Save/reopen and independent Verify use committed environment; no self-heating or response delay is modeled"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Wrong-kind/unknown target, missing fields, nonfinite or out-of-range environment"}, {"solve-failed", "input_work", "Algebraic event solve or cooperative deadline fails"}, {"canceled", "input_work", "Explicit work cancellation or Pause"}, {"superseded", "input_work", "A newer input/run/document invalidates this staged request"}}, Completion: &commandCompletionDetail{Mode: "Immediate admission, asynchronous atomic input_work when fresh transient; otherwise ordinary schematic edit", Observe: []string{"snapshot.input_work", "snapshot.mixed.sensors", "simulation.wait", "events.poll"}, Cancel: "simulation.work.cancel", CooperativeDeadlineSeconds: int(InputWorkDeadline.Seconds())}}
	if kind == "thermistor" {
		schema["examples"] = []any{TemperatureParams{ID: "RT1", Temperature: 20}}
	} else {
		schema["examples"] = []any{LightParams{ID: "LDR1", Illuminance: 10}}
	}
	return true
}
