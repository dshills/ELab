package workbench

import (
	"context"
	"errors"
	"fmt"
)

const MaxAppearanceOperations = 32

// AppearanceWork identifies one admitted preference load/synchronization or
// theme save. It never denotes a circuit edit or a new simulation run.
type AppearanceWork struct {
	ID                  string   `json:"id"`
	Command             string   `json:"command"`
	Status              string   `json:"status"`
	Phase               string   `json:"phase"`
	Actor               string   `json:"actor"`
	Path                string   `json:"path"`
	InputRevision       uint64   `json:"input_revision"`
	CommandRevision     uint64   `json:"command_revision"`
	DocumentGeneration  uint64   `json:"document_generation"`
	ElectricalRevision  uint64   `json:"electrical_revision"`
	RunID               uint64   `json:"input_run_id"`
	InputTheme          string   `json:"input_theme"`
	RequestedTheme      string   `json:"requested_theme,omitempty"`
	ResultTheme         string   `json:"result_theme,omitempty"`
	SourceSHA256        string   `json:"source_sha256,omitempty"`
	ArtifactSHA256      string   `json:"artifact_sha256,omitempty"`
	Bytes               int      `json:"bytes"`
	Cancelable          bool     `json:"cancelable"`
	Superseded          bool     `json:"superseded"`
	Published           bool     `json:"published"`
	CurrentAtCompletion bool     `json:"current_at_completion"`
	CleanupWarning      string   `json:"cleanup_warning,omitempty"`
	Error               *Failure `json:"error,omitempty"`
}

type appearanceJob struct {
	work                   AppearanceWork
	store                  *appearanceStore
	ctx                    context.Context
	cancel                 context.CancelFunc
	reason                 string
	inProgress, publishing bool
}

func cloneAppearanceWork(work *AppearanceWork) *AppearanceWork {
	if work == nil {
		return nil
	}
	out := *work
	if work.Error != nil {
		failure := *work.Error
		out.Error = &failure
	}
	return &out
}

func (s *Service) appearanceWorkLocked() *AppearanceWork {
	if s.appearanceJob == nil {
		return nil
	}
	return cloneAppearanceWork(&s.appearanceJob.work)
}

func (s *Service) newAppearanceWorkLocked(store *appearanceStore, command, theme string) (*appearanceJob, error) {
	for len(s.appearanceJobs) >= MaxAppearanceOperations {
		removed := false
		for i, id := range s.appearanceOrder {
			if !operationActive(s.appearanceJobs[id].work.Status) {
				delete(s.appearanceJobs, id)
				s.appearanceOrder = append(s.appearanceOrder[:i], s.appearanceOrder[i+1:]...)
				removed = true
				break
			}
		}
		if !removed {
			return nil, &actionError{"busy", "Wait for active appearance operations to finish"}
		}
	}
	if s.appearanceJobs == nil {
		s.appearanceJobs = make(map[string]*appearanceJob)
	}
	s.appearanceSerial++
	ctx, cancel := context.WithCancel(store.ctx)
	job := &appearanceJob{store: store, ctx: ctx, cancel: cancel, work: AppearanceWork{
		ID: fmt.Sprintf("appearance-%d", s.appearanceSerial), Command: command, Status: "queued", Phase: "queued",
		Path: store.path, InputRevision: s.revision, CommandRevision: s.commandRevision + 1,
		DocumentGeneration: s.documentGeneration, ElectricalRevision: s.electrical, RunID: s.run,
		InputTheme: s.theme, RequestedTheme: theme, Cancelable: true,
	}}
	s.appearanceJobs[job.work.ID] = job
	s.appearanceOrder = append(s.appearanceOrder, job.work.ID)
	return job, nil
}

func (s *Service) interruptAppearanceJobLocked(job *appearanceJob, reason string) {
	if job == nil || !operationActive(job.work.Status) {
		return
	}
	job.work.Superseded = job.work.Superseded || reason == "superseded" || job.publishing
	job.cancel()
	if job.publishing {
		// Publication has crossed its irreversible admission point. Retain a
		// running job until its actual artifact/error is known, even detached.
		return
	}
	if job.reason == "" {
		job.reason = reason
	}
	job.work.Cancelable = false
	if job.inProgress {
		job.work.Status = "canceling"
	} else {
		s.finishAppearanceReasonLocked(job)
	}
}

func (s *Service) finishAppearanceReasonLocked(job *appearanceJob) {
	if !operationActive(job.work.Status) || job.reason == "" {
		return
	}
	job.inProgress = false
	job.work.Status, job.work.Phase = job.reason, job.reason
	job.work.Cancelable = false
	job.work.Error = &Failure{Code: job.reason, Message: "Appearance work ended before file publication"}
	job.cancel()
}

func (s *Service) detachAppearanceLocked(reason string) {
	if store := s.appearanceStore; store != nil {
		store.cancel()
		for _, job := range s.appearanceJobs {
			if job.store == store {
				s.interruptAppearanceJobLocked(job, reason)
			}
		}
	}
	s.appearanceStore = nil
}

func (s *Service) cancelAppearanceWorkLocked(id string) error {
	job := s.appearanceJobs[id]
	if job == nil || !operationActive(job.work.Status) {
		return &actionError{"operation-unavailable", "Appearance operation expired, finished or is unknown"}
	}
	if job.publishing {
		return &actionError{"operation-committing", "The appearance file is publishing atomically and can no longer be canceled"}
	}
	if s.appearanceStore != job.store || job.store.latest != job || !job.work.Cancelable {
		return &actionError{"operation-unavailable", "Choose the current cancelable appearance operation"}
	}
	s.detachAppearanceLocked("canceled")
	s.appearance = AppearanceState{Path: s.appearance.Path, Status: "disabled", SavedTheme: s.appearance.SavedTheme}
	return nil
}

func (s *Service) failAppearanceJobLocked(job *appearanceJob, err error) {
	if job == nil || !operationActive(job.work.Status) {
		return
	}
	job.inProgress, job.publishing = false, false
	job.work.Status, job.work.Phase, job.work.Cancelable = "failed", "failed", false
	code := "appearance-failed"
	var action *actionError
	if errors.As(err, &action) {
		code = action.code
	}
	job.work.Error = &Failure{Code: code, Message: err.Error()}
	job.cancel()
}

func (s *Service) completeAppearanceJobLocked(job *appearanceJob, theme, digest string) {
	if job == nil || !operationActive(job.work.Status) {
		return
	}
	job.inProgress, job.publishing = false, false
	job.work.Status, job.work.Phase, job.work.Cancelable = "completed", "completed", false
	job.work.ResultTheme, job.work.ArtifactSHA256 = theme, digest
	job.work.CurrentAtCompletion = s.appearanceStore == job.store && job.store.latest == job && s.theme == theme
	job.cancel()
}
