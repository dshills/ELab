package workbench

import "context"

// analysisJobs owns retained analysis assets, their selection and bounded admission.
// It has no independent lock or revision authority. The embedding Service must
// hold Service.mu for all access, including the Locked helpers below. Workers
// publish only through Service, preserving document/source/revision guards.
type analysisJobs struct {
	liveSpectrum          *LiveSpectrumState
	liveSpectrumSerial    uint64
	liveSpectrumResultKey liveSpectrumKey
	liveSpectrumCancel    context.CancelFunc

	tolerance                        *toleranceJob
	toleranceSettings                *ToleranceSettings
	toleranceSerial                  uint64
	toleranceWorkers                 int
	spectrum                         *SpectrumState
	sweeps                           *sweepStore
	sweepSerial                      uint64
	sweepViewerRevision              uint64
	sweepImport                      sweepImportFunc
	sweepWorkers                     int
	sweepCompute                     func(context.Context, sweepPlan, func(int)) (*SweepResult, error)
	operations                       map[string]*verificationOperation
	operationOrder                   []string
	operationSerial                  uint64
	selectedOperation, lastOperation string
	verificationSettings             VerificationSettings
}

func (s *analysisJobs) ensureSweepsLocked() *sweepStore {
	if s.sweeps == nil {
		s.sweeps = &sweepStore{operations: map[string]*sweepJob{}}
	}
	return s.sweeps
}
func (s *analysisJobs) requireSweepSlotLocked() error {
	if s.sweeps == nil || len(s.sweeps.order) < MaxDCSweepOperations {
		return nil
	}
	for _, id := range s.sweeps.order {
		job := s.sweeps.operations[id]
		if !job.keep && !operationActive(job.summary.Status) {
			return nil
		}
	}
	return &actionError{"record-limit", "Eight sweeps are kept; exclude one before creating or importing another"}
}

func (s *analysisJobs) requireRecordingSlotLocked() error {
	if len(s.operationOrder) < MaxVerificationOperations {
		return nil
	}
	for _, id := range s.operationOrder {
		operation := s.operations[id]
		if !operation.keep && !operationActive(operation.summary.Status) {
			return nil
		}
	}
	return &actionError{"record-limit", "The project keeps eight recordings. Exclude one from the project before creating or importing another."}
}

// Admission/publication check capacity first. Eviction never removes kept or
// active work; an expired selection cannot keep pointing at a removed asset.
func (s *analysisJobs) retainSweepJobLocked(job *sweepJob) error {
	if err := s.requireSweepSlotLocked(); err != nil {
		return err
	}
	store := s.ensureSweepsLocked()
	if len(store.order) >= MaxDCSweepOperations {
		for i, id := range store.order {
			old := store.operations[id]
			if old.keep || operationActive(old.summary.Status) {
				continue
			}
			delete(store.operations, id)
			store.order = append(store.order[:i], store.order[i+1:]...)
			if store.selected == id {
				store.selected, store.traceID, store.visible, store.cursor = "", "", false, nil
			}
			break
		}
	}
	store.operations[job.summary.ID] = job
	store.order = append(store.order, job.summary.ID)
	return nil
}
