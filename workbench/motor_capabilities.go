package workbench

func addMotorCapabilityDetails(id string, schema map[string]any) bool {
	if id != "motor.configure" && id != "motor.load.set" {
		return false
	}
	schema["x-model-id"] = PowerModelID
	schema["x-target-inventory"] = "snapshot.components filtered by kind dc-motor"
	contract := commandContractDetail{Version: 1, Scope: "motor-configuration", Preconditions: []string{"Existing brushed DC motor; finite parameters inside the declared domains"}, Effects: []string{"Reciprocal SI coupling uses Ke=Kt: winding RL/back-EMF drives rotor inertia and viscous friction/load. Accepted storage uses the existing adaptive collocation solver", "mixed.motors reports RPM, rad/s, armature current, back EMF, torque, blocked-rotor reference current, winding/rotor energy and losses", "Ordinary configuration is a saved, undoable startup edit; running circuits restart and paused results stay stale until Restart/Resume. Locked rotor is an exact zero-speed constraint with a static reaction torque; configuring lock restarts rather than deleting live rotational energy", "Save/reopen/Verify use configured startup at rest and zero armature current. Internal parts/nets are private and consume existing solver resource budgets. v:ID and i:ID use the authored terminals; positive current enters positive terminal"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Wrong target, missing fields, nonfinite/out-of-range parameters"}}, Completion: &commandCompletionDetail{Mode: "Immediate configuration admission; asynchronous initialization remains observable", Observe: []string{"project.inspect", "simulation.wait", "events.poll"}}}
	if id == "motor.configure" {
		schema["x-parameter-domains"] = map[string]any{"armature_resistance_ohm": []float64{.1, 100}, "armature_inductance_h": []float64{1e-5, 1}, "ke_v_s_per_rad": []float64{.001, 1}, "inertia_kg_m2": []float64{1e-7, .001}, "friction_n_m_s_per_rad": []float64{1e-7, .01}, "load_n_m_s_per_rad": []float64{0, .01}, "locked_rotor": "Boolean; startup rest/locked condition"}
		schema["examples"] = []any{MotorParams{ID: "MTR1", Parameters: motorDefault()}}
	} else {
		p := inputProperty(schema, "load_n_m_s_per_rad")
		p["minimum"], p["maximum"], p["x-unit"] = 0, .01, "N·m·s/rad"
		contract.Scope = "motor-live-load"
		contract.Effects = append(contract.Effects, "Fresh transient: stage input_work at accepted time, preserve winding current, shaft speed, all capacitor voltages and battery charge. Pending request is requested_motor_load; committed graph/readings/history change only on success", "Failure/cancel/supersession preserves accepted physics/history. A newer input supersedes pending work; returning to committed load cancels a pending motor request. One successful publication creates one saved undo entry", "Incomplete/stale/DC drawings and edit.batch use ordinary startup configuration. This command changes only viscous load; other motor parameters and lock state are untouched")
		contract.Errors = append(contract.Errors, commandFailureDetail{"solve-failed", "input_work", "Cooperative event solve fails"}, commandFailureDetail{"canceled", "input_work", "Explicit cancel/Pause"}, commandFailureDetail{"superseded", "input_work", "A newer input/run/document retires this request"})
		contract.Completion.Mode = "Fresh transient: asynchronous atomic input_work; otherwise ordinary startup edit"
		contract.Completion.Cancel = "simulation.work.cancel"
		contract.Completion.CooperativeDeadlineSeconds = int(InputWorkDeadline.Seconds())
		schema["examples"] = []any{MotorLoadParams{ID: "MTR1", Load: .0001}}
	}
	schema["x-command-contract"] = contract
	return true
}
