package main

import "github.com/dshills/elab/workbench"

type recordedStatisticsRejection struct {
	generation           uint64
	recordingID, probeID string
	value                float64
	text                 string
}

func statisticsEndpoint(settings workbench.RecordingStatisticsSettings, side string) float64 {
	if side == "start" {
		return settings.Start
	}
	return settings.End
}

func (s *screen) changeRecordedStatisticsTime(state workbench.Snapshot, side, text string) {
	r := state.Recording
	cfg := r.Statistics.Settings
	response := s.invokeAtCommandRevision("recording.statistics.set", textFieldParams(cfg, side+"_s", text), state.CommandRevision, state.Document.Generation)
	if response.Error == nil {
		delete(s.recordedStatisticsRejected, side)
		s.pruneRecordedStatisticsRejections(response.Result.(workbench.Snapshot))
		return
	}
	if !s.fieldContextCurrent(s.fieldContext) || (response.Error.Code != "invalid-params" && response.Error.Code != "unavailable") {
		return
	}
	current := s.service.Snapshot()
	if current.Recording == nil || !current.Recording.Viewing || current.Recording.ID != r.ID || current.Recording.Statistics == nil ||
		current.Recording.Statistics.Settings.ProbeID != cfg.ProbeID || statisticsEndpoint(current.Recording.Statistics.Settings, side) != statisticsEndpoint(cfg, side) {
		return
	}
	if s.recordedStatisticsRejected == nil {
		s.recordedStatisticsRejected = make(map[string]recordedStatisticsRejection)
	}
	s.recordedStatisticsRejected[side] = recordedStatisticsRejection{
		generation: state.Document.Generation, recordingID: r.ID, probeID: cfg.ProbeID,
		value: statisticsEndpoint(cfg, side), text: text,
	}
}

func (s *screen) pruneRecordedStatisticsRejections(state workbench.Snapshot) {
	r := state.Recording
	for side, rejected := range s.recordedStatisticsRejected {
		if r == nil || !r.Viewing || r.Statistics == nil || rejected.generation != state.Document.Generation ||
			rejected.recordingID != r.ID || rejected.probeID != r.Statistics.Settings.ProbeID ||
			rejected.value != statisticsEndpoint(r.Statistics.Settings, side) {
			delete(s.recordedStatisticsRejected, side)
		}
	}
}
