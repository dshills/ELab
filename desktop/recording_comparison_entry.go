package main

import "github.com/dshills/elab/workbench"

type recordedComparisonRejection struct {
	generation      uint64
	id, referenceID string
	value           float64
	text            string
}

func (s *screen) changeRecordedComparisonTime(state workbench.Snapshot, text string) {
	r := state.Recording
	cfg := r.Comparison.Settings
	response := s.invokeAtCommandRevision("recording.compare.set", textFieldParams(cfg, "time_s", text), state.CommandRevision, state.Document.Generation)
	if response.Error == nil {
		s.recordedComparisonRejected = nil
		return
	}
	if response.Error.Code != "invalid-params" || !s.fieldContextCurrent(s.fieldContext) {
		return
	}
	current := s.service.Snapshot()
	if current.Recording == nil || !current.Recording.Viewing || current.Recording.ID != r.ID || current.Recording.Comparison == nil {
		return
	}
	accepted := current.Recording.Comparison.Settings
	if accepted.ReferenceID != cfg.ReferenceID || accepted.Time != cfg.Time {
		return
	}
	s.recordedComparisonRejected = &recordedComparisonRejection{
		generation: state.Document.Generation, id: r.ID, referenceID: cfg.ReferenceID,
		value: cfg.Time, text: text,
	}
}

func (s *screen) pruneRecordedComparisonRejection(state workbench.Snapshot) {
	rejected := s.recordedComparisonRejected
	if rejected == nil {
		return
	}
	r := state.Recording
	if r == nil || !r.Viewing || r.Comparison == nil || rejected.generation != state.Document.Generation ||
		rejected.id != r.ID || rejected.referenceID != r.Comparison.Settings.ReferenceID || rejected.value != r.Comparison.Settings.Time {
		s.recordedComparisonRejected = nil
	}
}
