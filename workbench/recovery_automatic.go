package workbench

import "reflect"

type automaticRecoveryCapture struct {
	recoveryCapture
	catalogRevision uint64
}

// publishAutomaticRecoveryLocked publishes filesystem status without applying
// the captured document to the live editor. The caller holds s.mu.
func (s *Service) publishAutomaticRecoveryLocked(store *recoveryStore, capture automaticRecoveryCapture, result recoveryWorkResult, err error) {
	if s.closed || s.recoveryStore != store {
		return
	}
	// Drawing and live edits do not invalidate session-wide storage status.
	// A newer recovery/save/configuration publication does: its catalog and
	// failure state must not be replaced by this older filesystem observation.
	if capture.catalogRevision != s.recoveryCatalogRevision {
		return
	}
	changed := false
	if result.scanError == nil && (!reflect.DeepEqual(result.candidates, s.recovery.Candidates) || !reflect.DeepEqual(result.problems, s.recovery.Problems)) {
		s.recovery.Candidates, s.recovery.Problems = result.candidates, result.problems
		s.normalizeRecoverySelectionLocked()
		changed = true
	}
	if err != nil {
		if s.recovery.Error != err.Error() {
			s.recovery.Error = err.Error()
			changed = true
		}
	} else {
		if s.recovery.Error != "" {
			s.recovery.Error = ""
			changed = true
		}
		if result.id != "" {
			s.recovery.LastSnapshot = result.id
			s.recovery.LastSavedAt = result.savedAt
			changed = true
		}
	}
	if changed {
		s.recoveryCatalogRevision++
		s.revision++
		s.signalLocked("recovery")
	}
}
