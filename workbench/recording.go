package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"
)

func (s *Service) selectRecordingLocked(id string) {
	s.viewerRevision++
	s.recording.comparison = nil
	s.recording.timing = nil
	s.recording.statistics = nil
	s.recording.stepResponse = nil
	s.selectedOperation = id
	s.recording.playing = false
	s.recording.index = 0
	s.recording.window = nil
	s.recording.probes = nil
	s.recording.a = nil
	s.recording.b = nil
	s.recording.activeCursor = "a"
	if id == "" {
		s.recording.viewing = false
	}
}
func (s *Service) selectedRecordLocked() (*verificationRecord, error) {
	operation := s.operations[s.selectedOperation]
	if operation == nil || operation.record == nil {
		return nil, &actionError{"not-ready", "Choose a completed recorded run first"}
	}
	return operation.record, nil
}
func (s *Service) recordingCommandLocked(command string, raw json.RawMessage) error {
	if command == "recording.view.set" {
		var p RecordingViewParams
		if err := params(raw, &p, "mode"); err != nil {
			return err
		}
		if p.Mode != "live" && p.Mode != "recorded" {
			return fmt.Errorf("view mode must be live or recorded")
		}
		if p.Mode == "recorded" {
			if _, err := s.selectedRecordLocked(); err != nil {
				return err
			}
		}
		s.recording.viewing = p.Mode == "recorded"
		if !s.recording.viewing {
			s.recording.playing = false
		}
		return nil
	}
	record, err := s.selectedRecordLocked()
	if err != nil {
		return err
	}
	switch command {
	case "recording.seek":
		var p RecordingSeekParams
		if err := params(raw, &p); err != nil {
			return err
		}
		if (p.Index == nil) == (p.Time == nil) {
			return fmt.Errorf("seek requires exactly one of index or time_s")
		}
		index := 0
		if p.Index != nil {
			index = *p.Index
			if index < 0 || index >= len(record.rows) {
				return fmt.Errorf("sample index is outside the recording")
			}
		} else {
			if !finite(*p.Time) || !scopeTimeInside(*p.Time, 0, record.metadata.Duration) {
				return fmt.Errorf("seek time is outside the recording")
			}
			index = nearestRecordingIndex(record.rows, *p.Time)
		}
		s.recording.index = index
		s.recording.playing = false
	case "recording.step":
		var p RecordingStepParams
		if err := params(raw, &p, "direction"); err != nil {
			return err
		}
		index := s.recording.index
		switch p.Direction {
		case "next":
			index++
		case "previous":
			index--
		default:
			return fmt.Errorf("recorded step direction must be next or previous")
		}
		if index < 0 || index >= len(record.rows) {
			return &actionError{"unavailable", "Recorded sample boundary reached"}
		}
		s.recording.index = index
		s.recording.playing = false
	case "recording.play", "recording.speed.set":
		var p RecordingPlayParams
		if err := params(raw, &p, "seconds_per_wall_second"); err != nil {
			return err
		}
		if !finite(p.Speed) || p.Speed < .001 || p.Speed > 10 {
			return fmt.Errorf("recorded playback speed must be .001–10 simulated seconds per wall second")
		}
		if command == "recording.speed.set" {
			s.recording.speed = p.Speed
			if s.recording.playing {
				s.recording.wall = time.Now()
				s.recording.anchor = record.rows[s.recording.index].Time
			}
			return nil
		}
		if record.metadata.Duration == 0 {
			return &actionError{"unavailable", "DC recording has one operating point; no playback duration"}
		}
		if s.recording.index >= len(record.rows)-1 {
			s.recording.index = 0
		}
		s.recording.speed = p.Speed
		s.recording.playing = true
		s.recording.wall = time.Now()
		s.recording.anchor = record.rows[s.recording.index].Time
		s.startRecordingClockLocked()
	case "recording.pause":
		if err := params(raw, &struct{}{}); err != nil {
			return err
		}
		s.recording.playing = false
	case "recording.window.set":
		var p RecordingWindowParams
		if err := params(raw, &p, "start_s", "end_s"); err != nil {
			return err
		}
		p, err = normalizeRecordingWindow(p, record.metadata.Duration)
		if err != nil {
			return err
		}
		s.recording.window = &p
	case "recording.probes.set":
		var p ProbeParams
		if err := params(raw, &p, "ids"); err != nil {
			return err
		}
		if p.IDs == nil || len(p.IDs) > MaxScopeProbes {
			return fmt.Errorf("recorded probes require an array of up to six captured IDs")
		}
		seen := map[string]bool{}
		available := map[string]bool{}
		for _, probe := range record.metadata.Probes {
			available[probe.ID] = true
		}
		for _, id := range p.IDs {
			if !available[id] || seen[id] {
				return fmt.Errorf("recorded probe is unknown or duplicated")
			}
			seen[id] = true
		}
		s.recording.probes = append([]string{}, p.IDs...)
	case "recording.cursor.set":
		var p RecordingCursorParams
		if err := params(raw, &p, "cursor", "time_s"); err != nil {
			return err
		}
		if p.OperationID != "" && p.OperationID != s.selectedOperation {
			return &actionError{"conflict", "Selected recording changed; refresh before placing a cursor"}
		}
		if (p.Cursor != "a" && p.Cursor != "b") || !finite(p.Time) || !scopeTimeInside(p.Time, 0, record.metadata.Duration) {
			return fmt.Errorf("recorded cursor must be a/b with time inside the recording")
		}
		if p.Cursor == "a" {
			s.recording.a = &p.Time
		} else {
			s.recording.b = &p.Time
		}
	case "recording.cursor.select", "recording.cursor.clear":
		var p CursorSelectParams
		if err := params(raw, &p, "cursor"); err != nil {
			return err
		}
		if p.Cursor != "a" && p.Cursor != "b" {
			return fmt.Errorf("recorded cursor must be a/b")
		}
		if command == "recording.cursor.select" {
			s.recording.activeCursor = p.Cursor
		} else if p.Cursor == "a" {
			s.recording.a = nil
		} else {
			s.recording.b = nil
		}
	default:
		return fmt.Errorf("unknown recorded command")
	}
	return nil
}

func normalizeRecordingWindow(p RecordingWindowParams, duration float64) (RecordingWindowParams, error) {
	if !finite(p.Start) || !finite(p.End) || !scopeTimeInside(p.Start, 0, duration) || !scopeTimeInside(p.End, 0, duration) || p.Start > p.End {
		return p, fmt.Errorf("recorded window must be an ordered interval inside the complete recording")
	}
	p.Start = math.Max(0, math.Min(duration, p.Start))
	p.End = math.Max(0, math.Min(duration, p.End))
	if duration > 0 && p.Start >= p.End {
		return p, fmt.Errorf("recorded window must retain positive width after boundary normalization")
	}
	return p, nil
}
func (s *Service) startRecordingClockLocked() {
	if s.recording.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.recording.cancel = cancel
	s.liveWorkers.Add(1)
	go func() {
		defer s.liveWorkers.Done()
		ticker := time.NewTicker(33 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				s.mu.Lock()
				if s.closed {
					s.mu.Unlock()
					return
				}
				record, err := s.selectedRecordLocked()
				if !s.recording.playing || err != nil {
					s.mu.Unlock()
					continue
				}
				// A tick queued before seek/speed changes must never move playback backward.
				elapsed := math.Max(0, now.Sub(s.recording.wall).Seconds())
				target := s.recording.anchor + elapsed*s.recording.speed
				index := sort.Search(len(record.rows), func(i int) bool { return record.rows[i].Time > target }) - 1
				index = max(0, index)
				done := target >= record.metadata.Duration
				if done {
					index = len(record.rows) - 1
					s.recording.playing = false
				}
				if index != s.recording.index || done {
					s.recording.index = index
					s.revision++
					s.signalLocked("recordings")
				}
				s.mu.Unlock()
			}
		}
	}()
}
