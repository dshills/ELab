package workbench

// RecordingCursorParams optionally pins a selected capture. Older clients may
// omit OperationID to retain the original current-selection behavior.
type RecordingCursorParams struct {
	OperationID string  `json:"operation_id,omitempty"`
	Cursor      string  `json:"cursor"`
	Time        float64 `json:"time_s"`
}

func addRecordingCursorDetails(id string, schema map[string]any) bool {
	if id != "recording.cursor.set" {
		return false
	}
	inputProperty(schema, "operation_id")["description"] = "Optional exact selected completed recording ID; nonempty mismatches refuse atomically with conflict. Omitted/empty retains current-selection behavior."
	inputProperty(schema, "cursor")["enum"] = []string{"a", "b"}
	inputProperty(schema, "time_s")["x-normalized-si-range"] = map[string]float64{"minimum": 0, "maximum": 10}
	schema["x-inventory"] = "snapshot.recording.id, duration_s, scope.active_cursor and scope.window"
	schema["x-relations"] = []map[string]any{{"operator": "within-selected-captured-duration", "field": "time_s"}}
	schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "recorded-cursor-set", Preconditions: []string{"A retained completed recording is selected; time is within its complete captured duration", "Optional nonempty operation_id matches selected capture; cursor is a or b"}, Effects: []string{"Sets only requested recorded cursor time; nearest stored sample with later/right-hand ties; does not select another capture", "No live cursor/physics/history/project/dirty change; cursor validity also reports its relation to visible interval"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed cursor/time or time outside complete recording"}, {"not-ready", "admission", "No completed selected recording"}, {"conflict", "admission", "Explicit capture ID differs from selected capture"}}, Completion: &commandCompletionDetail{Mode: "immediate session view mutation"}}
	return true
}
