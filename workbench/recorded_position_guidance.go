package workbench

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Interpret only known seek intent. Index must be numeric (fractional or
// unrepresentable numbers still need whole-index guidance); time may be a
// number or unit string. Null/omitted fields retain their absent semantics.
func (s *Service) recordedPositionRequestHelpLocked(r Request, help *DiagnosticHelp) {
	var p struct {
		Index json.RawMessage `json:"index,omitempty"`
		Time  json.RawMessage `json:"time_s,omitempty"`
	}
	if params(r.Params, &p) != nil {
		return
	}
	index, at := bytes.TrimSpace(p.Index), bytes.TrimSpace(p.Time)
	if len(index) > 0 && !bytes.Equal(index, []byte("null")) && index[0] != '-' && (index[0] < '0' || index[0] > '9') {
		return
	}
	if len(at) > 0 && !bytes.Equal(at, []byte("null")) {
		if _, typed, _ := requestGuidanceTime(at); !typed {
			return
		}
	}
	record, err := s.selectedRecordLocked()
	if err != nil {
		return
	}
	// A valid position does not explain another error, such as request
	// metadata. This normalization and strict decode are pure input checks.
	if normalized, err := normalizeTimeInput("recording.seek", r.Params); err == nil {
		var value RecordingSeekParams
		if params(normalized, &value) == nil {
			validIndex := value.Index != nil && value.Time == nil && *value.Index >= 0 && *value.Index < len(record.rows)
			validTime := value.Index == nil && value.Time != nil && finite(*value.Time) && scopeTimeInside(*value.Time, 0, record.metadata.Duration)
			if validIndex || validTime {
				return
			}
		}
	}
	help.Summary = "Choose a valid position in the recording."
	help.Action = fmt.Sprintf("Use a whole sample index from 0 to %d, or a time from 0 s to %g s. Time units such as ms and us are accepted. Your circuit is unchanged.", len(record.rows)-1, record.metadata.Duration)
	if record.metadata.Duration == 0 {
		help.Summary = "This recording has one sample."
		help.Action = "Use sample index 0 or time 0 s. Your circuit is unchanged."
	}
}
