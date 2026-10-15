package workbench

import (
	"context"
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
func (s *Service) recoveryCommandLocked(command string, raw json.RawMessage) error {
	switch command {
	case "recovery.work.start":
		var p RecoveryWorkParams
		if err := params(raw, &p, "kind"); err != nil {
			return err
		}
		return s.startRecoveryWorkLocked(p)
	case "recovery.configure", "recovery.configure.start":
		var p RecoveryConfigureParams
		if err := params(raw, &p, "directory", "enabled", "interval_s"); err != nil {
			return err
		}
		return s.startRecoveryConfigurationLocked(p)
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
	case "recovery.restore", "recovery.restore.start", "recovery.dismiss", "recovery.dismiss.start":
		var p RecoveryChoiceParams
		if err := params(raw, &p, "id", "sha256"); err != nil {
			return err
		}
		if command == "recovery.restore" || command == "recovery.restore.start" {
			return s.startRecoveryRestoreLocked(p)
		}
		return s.startRecoveryDismissLocked(p)
	}
	return fmt.Errorf("unknown recovery command")
}
func documentRecordingCount(d Document) int {
	count := 0
	if d.Recordings != nil {
		count += len(d.Recordings.Entries)
	}
	if d.Sweeps != nil {
		count += len(d.Sweeps.Entries)
	}
	return count
}
func inspectDocument(d Document) DocumentInspection {
	out := DocumentInspection{ExampleID: d.ExampleID, DisplayPreferences: d.DisplayPreferences, PinnedReadings: d.PinnedReadings, WaveformModel: d.WaveformModel, PotentiometerModel: d.PotentiometerModel, DiodeModel: d.DiodeModel, SwitchModel: d.SwitchModel, NMOSModel: d.NMOSModel, PMOSModel: d.PMOSModel, Format: d.Format, Version: d.Version, Graph: d.Graph.Clone(), View: d.View, ScopeProbes: d.ScopeProbes, TransientSettings: d.TransientSettings, LEDModel: d.LEDModel, NPNModel: d.NPNModel, Paused: d.Paused != nil && *d.Paused, Recordings: []VerificationMetadata{}, SelectedRecording: -1}
	if d.Recordings != nil {
		out.SelectedRecording = d.Recordings.Selected
		out.ViewingRecording = d.Recordings.Viewing
		for _, entry := range d.Recordings.Entries {
			out.Recordings = append(out.Recordings, cloneVerificationMetadata(entry.Data.Metadata))
		}
	}
	return out
}
