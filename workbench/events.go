package workbench

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const MaxEventWaiters = 4
const MaxEventWaitMS = 5000

// EventPollParams resumes from a revision observed in this same process.
// An omitted/null timeout defaults to one second; zero is nonblocking.
type EventPollParams struct {
	AfterRevision uint64 `json:"after_revision"`
	TimeoutMS     *int   `json:"timeout_ms,omitempty"`
}
type StatePublication struct {
	Topic    string `json:"topic"`
	Revision uint64 `json:"revision"`
}

// EventPollResult is coalesced latest-state observation, not a transition log.
// Snapshot is omitted during idle polls and includes all state on change/close.
type EventPollResult struct {
	AfterRevision   uint64             `json:"after_revision"`
	Revision        uint64             `json:"revision"`
	CommandRevision uint64             `json:"command_revision"`
	Changed         bool               `json:"changed"`
	TimedOut        bool               `json:"timed_out"`
	Closed          bool               `json:"closed"`
	Coalesced       bool               `json:"coalesced"`
	Publications    []StatePublication `json:"publications"`
	Snapshot        *Snapshot          `json:"snapshot,omitempty"`
}

var eventTopics = []string{"state", "project", "editor", "view", "simulation", "diagnostics", "jobs", "measurements", "recordings", "recovery", "preferences", "lifecycle", "library", "preview", "sweeps", "printing"}

func commandEventTopics(command string, documentChanged bool) []string {
	topics := []string{}
	switch {
	case strings.HasPrefix(command, "selection."):
		topics = append(topics, "editor", "view")
	case strings.HasPrefix(command, "component."), strings.HasPrefix(command, "wire."), strings.HasPrefix(command, "drawing."), strings.HasPrefix(command, "editor."), strings.HasPrefix(command, "edit."), strings.HasPrefix(command, "capacitor."), strings.HasPrefix(command, "source."), strings.HasPrefix(command, "switch."), strings.HasPrefix(command, "potentiometer."):
		topics = append(topics, "editor")
	case strings.HasPrefix(command, "view."):
		topics = append(topics, "view")
	case strings.HasPrefix(command, "simulation."):
		topics = append(topics, "simulation", "jobs")
	case command == "scope.visibility.set":
		topics = append(topics, "view", "preferences")
	case strings.HasPrefix(command, "scope."), strings.HasPrefix(command, "measurement."):
		topics = append(topics, "measurements")
	case strings.HasPrefix(command, "diagnostics."):
		topics = append(topics, "diagnostics")
	case strings.HasPrefix(command, "recording."), strings.HasPrefix(command, "operation."), strings.HasPrefix(command, "verification."):
		topics = append(topics, "recordings", "jobs")
	case strings.HasPrefix(command, "recovery."):
		topics = append(topics, "recovery", "jobs")
	case strings.HasPrefix(command, "sweep."):
		topics = append(topics, "sweeps", "jobs", "recordings")
	case strings.HasPrefix(command, "print.preview."):
		topics = append(topics, "preview")
	case strings.HasPrefix(command, "print."):
		topics = append(topics, "printing", "jobs")
	case strings.HasPrefix(command, "project."), strings.HasPrefix(command, "export."):
		topics = append(topics, "project", "jobs")
	case strings.HasPrefix(command, "example."):
		topics = append(topics, "project", "view", "simulation", "diagnostics", "recordings")
	case strings.HasPrefix(command, "display."), strings.HasPrefix(command, "appearance."):
		topics = append(topics, "preferences")
	case strings.HasPrefix(command, "parts."):
		topics = append(topics, "library")
	case strings.HasPrefix(command, "app.close"):
		topics = append(topics, "lifecycle")
	}
	if documentChanged {
		topics = append(topics, "project")
	}
	return topics
}
func projectEventTopics(job *projectJob) []string {
	topics := []string{"jobs"}
	if job.work.Status != "completed" {
		return topics
	}
	switch {
	case job.save != nil:
		topics = append(topics, "project", "recovery")
	case job.export != nil:
		topics = append(topics, "project")
	case job.importing != nil:
		topics = append(topics, "project", "recordings", "sweeps")
	case job.recoveryWork != nil || job.recoveryConfigure != nil || job.recoveryDismiss != nil:
		topics = append(topics, "recovery")
	default:
		topics = append(topics, "project", "editor", "simulation", "diagnostics", "recordings")
	}
	return topics
}

func (s *Service) signalLocked(topics ...string) {
	if s.eventRevisions == nil {
		s.eventRevisions = map[string]uint64{}
	}
	s.eventRevisions["state"] = s.revision
	// Only fixed topic names can enter this bounded latest-publication table.
	for _, topic := range topics {
		for _, known := range eventTopics {
			if topic == known {
				s.eventRevisions[topic] = s.revision
				break
			}
		}
	}
	close(s.changed)
	s.changed = make(chan struct{})
}
func (s *Service) eventPollResultLocked(p EventPollParams, timedOut bool) Response {
	result := EventPollResult{AfterRevision: p.AfterRevision, Revision: s.revision, CommandRevision: s.commandRevision, Changed: s.revision > p.AfterRevision, Closed: s.closed, Coalesced: true, Publications: []StatePublication{}}
	result.TimedOut = timedOut && !result.Changed && !s.closed
	for _, topic := range eventTopics {
		if revision := s.eventRevisions[topic]; revision > p.AfterRevision {
			result.Publications = append(result.Publications, StatePublication{topic, revision})
		}
	}
	if result.Changed || result.Closed {
		state := s.snapshotLocked()
		result.Snapshot = &state
	}
	return Response{Version: ProtocolVersion, Revision: s.revision, Result: result}
}
func (s *Service) pollEvents(ctx context.Context, r Request) Response {
	if r.Version != ProtocolVersion || r.ID == "" || len(r.ID) > 128 {
		return s.failure(r, "invalid-request", "version 1 and a nonempty ID (max 128 bytes) required")
	}
	var p EventPollParams
	if err := params(r.Params, &p, "after_revision"); err != nil {
		return s.failure(r, "invalid-params", err.Error())
	}
	waitMS := 1000
	if p.TimeoutMS != nil {
		waitMS = *p.TimeoutMS
	}
	if waitMS < 0 || waitMS > MaxEventWaitMS {
		return s.failure(r, "invalid-params", "event timeout_ms must be 0–5000 (omission defaults to 1000)")
	}
	if err := ctx.Err(); err != nil {
		return s.failure(r, "canceled", err.Error())
	}
	s.mu.Lock()
	if p.AfterRevision > s.revision {
		revision := s.revision
		s.mu.Unlock()
		return s.failure(r, "conflict", fmt.Sprintf("event cursor %d is ahead of current revision %d; inspect and resynchronize this process", p.AfterRevision, revision))
	}
	if s.revision > p.AfterRevision || s.closed || waitMS == 0 {
		response := s.eventPollResultLocked(p, false)
		s.mu.Unlock()
		response.ID = r.ID
		return response
	}
	if s.eventWaiters >= MaxEventWaiters {
		s.mu.Unlock()
		return s.failure(r, "busy", "four event waits are already active; retry after one finishes")
	}
	s.eventWaiters++
	changed := s.changed
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.eventWaiters--; s.mu.Unlock() }()
	timer := time.NewTimer(time.Duration(waitMS) * time.Millisecond)
	defer timer.Stop()
	for {
		timedOut := false
		select {
		case <-ctx.Done():
			return s.failure(r, "canceled", ctx.Err().Error())
		case <-timer.C:
			timedOut = true
		case <-changed:
		}
		if err := ctx.Err(); err != nil {
			return s.failure(r, "canceled", err.Error())
		}
		s.mu.Lock()
		if timedOut || s.revision > p.AfterRevision || s.closed {
			response := s.eventPollResultLocked(p, timedOut)
			s.mu.Unlock()
			response.ID = r.ID
			return response
		}
		changed = s.changed
		s.mu.Unlock()
	}
}

const EventPollSchemaID = "urn:elab:events:1"
