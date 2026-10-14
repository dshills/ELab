package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"
)

func DefaultRecoveryDirectory() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "ELab", "recovery"), nil
}
func (s *Service) recoveryCaptureLocked() recoveryCapture {
	return recoveryCapture{stateRevision: s.revision, document: s.documentLocked(), generation: s.documentGeneration, commandRevision: s.commandRevision, sourcePath: s.documentPath, dirty: s.dirtyLocked()}
}
func (s *Service) configureRecoveryLocked(p RecoveryConfigureParams) error {
	if !finite(p.Interval) || p.Interval < 1 || p.Interval > 60 {
		return fmt.Errorf("autosave interval must be 1–60 seconds")
	}
	directory, err := documentPath(p.Directory)
	if err != nil {
		return err
	}
	if s.recoveryStore != nil && s.recoveryStore.directory == directory {
		if err := s.scanRecoveryLocked(); err != nil {
			return err
		}
		s.recovery.Enabled = p.Enabled
		s.recovery.Interval = p.Interval
		return nil
	}
	store, err := newRecoveryStore(directory)
	if err != nil {
		return err
	}
	candidates, problems, err := store.scan()
	if err != nil {
		store.closeLease()
		return err
	}
	if s.recoveryCancel != nil {
		s.recoveryCancel()
	}
	if s.recoveryStore != nil {
		s.recoveryStore.closeLease()
	}
	s.recoverySelectionCleared = false
	s.recoveryStore = store
	s.recoveryCatalogRevision++
	s.recovery = RecoveryState{Directory: directory, Enabled: p.Enabled, Interval: p.Interval, Candidates: candidates, Problems: problems}
	if len(candidates) > 0 {
		s.recovery.Selected = candidates[0].ID
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.recoveryCancel = cancel
	go s.recoveryLoop(ctx, store)
	return nil
}
func (s *Service) scanRecoveryLocked() error {
	if s.recoveryStore == nil {
		return &actionError{"unavailable", "Configure a recovery directory first"}
	}
	candidates, problems, err := s.recoveryStore.scan()
	if err != nil {
		return err
	}
	s.recoveryCatalogRevision++
	s.recovery.Candidates = candidates
	s.recovery.Problems = problems
	s.normalizeRecoverySelectionLocked()
	return nil
}
func (s *Service) normalizeRecoverySelectionLocked() {
	if s.recoverySelectionCleared && s.recovery.Selected == "" {
		return
	}
	selected := false
	for _, candidate := range s.recovery.Candidates {
		if candidate.ID == s.recovery.Selected {
			selected = true
		}
	}
	if !selected {
		s.recovery.Selected = ""
		if len(s.recovery.Candidates) > 0 {
			s.recovery.Selected = s.recovery.Candidates[0].ID
		}
	}
}
func (s *Service) recoveryLoop(ctx context.Context, store *recoveryStore) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	last := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		s.mu.Lock()
		if s.closed || s.recoveryStore != store {
			s.mu.Unlock()
			return
		}
		enabled, interval := s.recovery.Enabled, s.recovery.Interval
		if !enabled || time.Since(last) < time.Duration(interval*float64(time.Second)) {
			s.mu.Unlock()
			continue
		}
		capture := s.recoveryCaptureLocked()
		s.mu.Unlock()
		last = time.Now()
		id, savedAt, err := store.checkpoint(capture)
		candidates, problems, scanErr := store.scan()
		if err == nil {
			err = scanErr
		}
		// Filesystem work does not hold the editor lock. Late publication cannot
		// replace a newer capture; the store guards per-document command revisions.
		s.mu.Lock()
		if s.closed || s.recoveryStore != store {
			s.mu.Unlock()
			return
		}
		if capture.commandRevision != s.commandRevision {
			s.mu.Unlock()
			continue
		}
		changed := false
		if scanErr == nil && (!reflect.DeepEqual(candidates, s.recovery.Candidates) || !reflect.DeepEqual(problems, s.recovery.Problems)) {
			s.recovery.Candidates, s.recovery.Problems = candidates, problems
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
			if id != "" {
				s.recovery.LastSnapshot = id
				s.recovery.LastSavedAt = savedAt
				changed = true
			}
		}
		if changed {
			s.recoveryCatalogRevision++
			s.revision++
			s.signalLocked("recovery")
		}
		s.mu.Unlock()
	}
}
func (s *Service) recoveryReadLocked(p RecoveryChoiceParams) (recoveryCapsule, Document, string, error) {
	if s.recoveryStore == nil {
		return recoveryCapsule{}, Document{}, "", &actionError{"unavailable", "Configure recovery first"}
	}
	capsule, document, hash, err := s.recoveryStore.read(p.ID)
	if err != nil {
		return capsule, document, hash, &actionError{"invalid-recovery", err.Error()}
	}
	if p.SHA256 == "" || hash != p.SHA256 {
		return capsule, document, hash, &actionError{"file-conflict", "Recovery snapshot changed; scan again before choosing it"}
	}
	return capsule, document, hash, nil
}
func (s *Service) dismissRecoveryLocked(p RecoveryChoiceParams) error {
	if s.recoveryStore == nil || !recoveryID(p.ID) {
		return fmt.Errorf("choose a listed recovery snapshot")
	}
	path := s.recoveryStore.candidatePath(p.ID)
	data, err := readRegularBounded(path, MaxRecoveryBytes)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	if p.SHA256 == "" || hex.EncodeToString(digest[:]) != p.SHA256 {
		return &actionError{"file-conflict", "Recovery snapshot changed; scan again"}
	}
	capsule, _, _, _ := s.recoveryStore.read(p.ID)
	_, sessionErr := hex.DecodeString(capsule.Session)
	if sessionErr == nil && len(capsule.Session) == 24 && capsule.Session != s.recoveryStore.session && s.recoveryStore.sessionActive(capsule.Session) {
		return &actionError{"unavailable", "Another active session owns this snapshot; its recovery data is preserved"}
	}
	if err := os.Remove(path); err != nil {
		return fileError(err)
	}
	return s.scanRecoveryLocked()
}
func (s *Service) recoveryCommandLocked(command string, raw json.RawMessage) error {
	switch command {
	case "recovery.work.start":
		var p RecoveryWorkParams
		if err := params(raw, &p, "kind"); err != nil {
			return err
		}
		return s.startRecoveryWorkLocked(p)
	case "recovery.configure":
		var p RecoveryConfigureParams
		if err := params(raw, &p, "directory", "enabled", "interval_s"); err != nil {
			return err
		}
		return s.configureRecoveryLocked(p)
	case "recovery.scan":
		if err := params(raw, &struct{}{}); err != nil {
			return err
		}
		return s.startRecoveryWorkLocked(RecoveryWorkParams{Kind: "scan"})
	case "recovery.checkpoint":
		if err := params(raw, &struct{}{}); err != nil {
			return err
		}
		return s.startRecoveryWorkLocked(RecoveryWorkParams{Kind: "checkpoint"})
	case "recovery.select":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		if p.ID != "" {
			found := false
			for _, candidate := range s.recovery.Candidates {
				if candidate.ID == p.ID {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("choose a listed recovery ID")
			}
		}
		s.recovery.Selected = p.ID
		s.recoverySelectionCleared = p.ID == ""
		return nil
	case "recovery.restore", "recovery.restore.start", "recovery.dismiss":
		var p RecoveryChoiceParams
		if err := params(raw, &p, "id", "sha256"); err != nil {
			return err
		}
		if command == "recovery.restore" || command == "recovery.restore.start" {
			return s.startRecoveryRestoreLocked(p)
		}
		return s.dismissRecoveryLocked(p)
	}
	return fmt.Errorf("unknown recovery command")
}
func (s *Service) recoveryInspectionLocked(raw json.RawMessage) (RecoveryInspection, error) {
	var p RecoveryChoiceParams
	if err := params(raw, &p, "id", "sha256"); err != nil {
		return RecoveryInspection{}, err
	}
	capsule, document, hash, err := s.recoveryReadLocked(p)
	if err != nil {
		return RecoveryInspection{}, err
	}
	candidate := RecoveryCandidate{ID: capsule.ID, DocumentKey: capsule.DocumentKey, SavedAt: capsule.SavedAt, SourcePath: capsule.SourcePath, SHA256: hash, CurrentSession: capsule.Session == s.recoveryStore.session, ActiveSession: s.recoveryStore.sessionActive(capsule.Session), Recordings: documentRecordingCount(document), Components: len(document.Graph.Components)}
	return RecoveryInspection{Candidate: candidate, Document: inspectDocument(document)}, nil
}

func documentRecordingCount(d Document) int {
	if d.Recordings == nil {
		return 0
	}
	return len(d.Recordings.Entries)
}
func inspectDocument(d Document) DocumentInspection {
	out := DocumentInspection{ExampleID: d.ExampleID, DisplayPreferences: d.DisplayPreferences, PinnedReadings: d.PinnedReadings, WaveformModel: d.WaveformModel, PotentiometerModel: d.PotentiometerModel, DiodeModel: d.DiodeModel, SwitchModel: d.SwitchModel, Format: d.Format, Version: d.Version, Graph: d.Graph.Clone(), View: d.View, ScopeProbes: d.ScopeProbes, TransientSettings: d.TransientSettings, LEDModel: d.LEDModel, NPNModel: d.NPNModel, Paused: d.Paused != nil && *d.Paused, Recordings: []VerificationMetadata{}, SelectedRecording: -1}
	if d.Recordings != nil {
		out.SelectedRecording = d.Recordings.Selected
		out.ViewingRecording = d.Recordings.Viewing
		for _, entry := range d.Recordings.Entries {
			out.Recordings = append(out.Recordings, cloneVerificationMetadata(entry.Data.Metadata))
		}
	}
	return out
}
func (s *Service) recoverySamplesLocked(raw json.RawMessage) (VerificationPage, error) {
	var p RecoverySamplesParams
	if err := params(raw, &p, "id", "sha256", "recording_index"); err != nil {
		return VerificationPage{}, err
	}
	_, document, _, err := s.recoveryReadLocked(RecoveryChoiceParams{ID: p.ID, SHA256: p.SHA256})
	if err != nil {
		return VerificationPage{}, err
	}
	if document.Recordings == nil || p.Recording < 0 || p.Recording >= len(document.Recordings.Entries) {
		return VerificationPage{}, fmt.Errorf("choose an embedded recovery recording index")
	}
	data := document.Recordings.Entries[p.Recording].Data
	return recordedSamplePage(p.ID, data.Metadata.Probes, data.Rows, p.Offset, p.Limit)
}
