package workbench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// requestDiagnosticLocked has checked revision/document context. Only a
// strict cursor intent may now describe its live run or selected recording.
func (s *Service) cursorTimeRequestHelpLocked(r Request, help *DiagnosticHelp) {
	var p struct {
		Cursor      string          `json:"cursor"`
		Time        json.RawMessage `json:"time_s"`
		OperationID string          `json:"operation_id,omitempty"`
	}
	if params(r.Params, &p, "cursor", "time_s") != nil || p.Cursor != "a" && p.Cursor != "b" {
		return
	}
	// operation_id is a recording-only guard, including when explicitly empty.
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(r.Params, &fields)
	if _, hasOperation := fields["operation_id"]; r.Command == "scope.cursor.set" && hasOperation {
		return
	}
	at, typed, parsed := requestGuidanceTime(p.Time)
	if !typed {
		return
	}
	start, end := 0., 0.
	if r.Command == "recording.cursor.set" {
		record, err := s.selectedRecordLocked()
		if err != nil || p.OperationID != "" && p.OperationID != s.selectedOperation {
			return
		}
		end = record.metadata.Duration
	} else {
		if s.transient == nil || s.transient.Stale || len(s.transient.Samples) == 0 {
			help.Summary = "Live cursor measurements need fresh transient samples."
			help.Action = "Restart or step a transient circuit, then enter a cursor time. Existing cursor positions and circuit state are unchanged."
			return
		}
		start, end = s.transient.Samples[0].Time, s.transient.Sample.Time
	}
	if parsed && scopeTimeInside(at, start, end) {
		return // The value is valid: this is not a time-domain refusal.
	}
	help.Summary = fmt.Sprintf("Choose a valid time for cursor %s.", strings.ToUpper(p.Cursor))
	help.Action = fmt.Sprintf("Enter a time from %g s to %g s. Units such as ms and us are accepted; bare numbers are seconds. Existing cursor positions and circuit state are unchanged.", start, end)
}

func requestGuidanceTime(raw json.RawMessage) (value float64, typed, parsed bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return 0, false, false
	}
	if raw[0] == '"' {
		var text string
		if json.Unmarshal(raw, &text) != nil {
			return 0, false, false
		}
		value, err := ParseTimeSeconds(text)
		return value, true, err == nil
	}
	if raw[0] != '-' && (raw[0] < '0' || raw[0] > '9') {
		return 0, false, false
	}
	err := json.Unmarshal(raw, &value)
	return value, true, err == nil && finite(value)
}
