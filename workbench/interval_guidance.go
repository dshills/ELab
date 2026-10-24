package workbench

import (
	"encoding/json"
	"fmt"
)

// requestDiagnosticLocked has checked revision/document context. Explain only
// strict interval intent and leave other refusal causes with generic help.
func (s *Service) intervalTimeRequestHelpLocked(r Request, help *DiagnosticHelp) {
	var p struct {
		Mode  string          `json:"mode,omitempty"`
		Start json.RawMessage `json:"start_s"`
		End   json.RawMessage `json:"end_s"`
	}
	if params(r.Params, &p, "start_s", "end_s") != nil {
		return
	}
	recorded := r.Command == "recording.window.set"
	if !recorded && p.Mode != "fixed" {
		return
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(r.Params, &fields)
	if _, hasMode := fields["mode"]; recorded && hasMode {
		return
	}
	start, startTyped, startParsed := requestGuidanceTime(p.Start)
	end, endTyped, endParsed := requestGuidanceTime(p.End)
	if !startTyped || !endTyped {
		return
	}
	lo, hi := 0., 0.
	subject := "scope"
	if recorded {
		record, err := s.selectedRecordLocked()
		if err != nil {
			return
		}
		hi, subject = record.metadata.Duration, "recorded"
		if _, err := normalizeRecordingWindow(RecordingWindowParams{Start: start, End: end}, hi); startParsed && endParsed && err == nil {
			return
		}
		if hi == 0 {
			help.Summary = "This recording has one time point."
			help.Action = "Use start 0 s and end 0 s for its interval. Existing samples and circuit state are unchanged."
			return
		}
	} else {
		if s.transient == nil || s.transient.Stale || len(s.transient.Samples) == 0 {
			help.Summary = "Live interval measurements need fresh transient samples."
			help.Action = "Restart or step a transient circuit, then enter the interval start and end. Existing interval and circuit state are unchanged."
			return
		}
		lo, hi = s.transient.Samples[0].Time, s.transient.Sample.Time
		if startParsed && endParsed && validFixedScopeWindow(ScopeWindow{Mode: "fixed", Start: start, End: end}, lo, hi) {
			return
		}
		if lo == hi {
			help.Summary = "A live interval needs more transient history."
			help.Action = "Step or resume the circuit to record a later time, then choose a start before the end. Existing interval and circuit state are unchanged."
			return
		}
	}
	help.Summary = fmt.Sprintf("Choose a valid %s interval.", subject)
	help.Action = fmt.Sprintf("Enter start and end times from %g s to %g s, with start before end. Units such as ms and us are accepted; bare numbers are seconds. Existing interval, samples and circuit state are unchanged.", lo, hi)
}
