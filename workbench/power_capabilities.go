package workbench

func addPowerCapabilityDetails(id string, schema map[string]any) bool {
	if id != "battery.configure" {
		return false
	}
	for _, d := range []struct {
		name, unit string
		min, max   float64
	}{{"capacity_ah", "Ah", 1e-6, 100}, {"internal_resistance_ohm", "Ω", .01, 1000}, {"initial_charge_fraction", "fraction", 0, 1}} {
		p := inputProperty(schema, d.name)
		p["minimum"], p["maximum"], p["x-unit"] = d.min, d.max, d.unit
	}
	schema["x-target-inventory"] = "snapshot.components filtered by kind battery"
	schema["x-model-id"] = PowerModelID
	schema["examples"] = []any{BatteryParams{ID: "B1", CapacityAh: .0002, Resistance: 10, Charge: .75}}
	schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "battery-startup-configuration", Preconditions: []string{"Existing battery; all parameters finite and within the stated domains"}, Effects: []string{"Ordinary saved, undoable electrical edit; running circuits restart and paused circuits become stale until Restart/Resume", "Initial charge is a startup condition, not a live charge refill. Accepted circuit integration depletes charge using delivered current, with internal voltage sag and explicit stored energy", "Startup parameters persist through save/reopen, duplicates, edit.batch, recovery and independent Verify; live discharge is not silently saved as startup", "mixed.batteries reports live SOC, remaining Ah, OCV, terminal voltage, delivered current, ESR heat and stored energy. v:ID and i:ID use the authored battery ports", "Generic linear OCV/charge reservoir; external charging is an ideal reversible reservoir, not a chemical charger model; live values outside nominal SOC range are explicitly reported"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Missing fields, wrong target kind or invalid parameters preserve graph/history/physics"}}, Completion: &commandCompletionDetail{Mode: "Immediate configuration admission; existing asynchronous initialization/readiness applies", Observe: []string{"project.inspect", "simulation.wait", "events.poll"}}}
	return true
}
