package workbench

import (
	"os"
	"path/filepath"
)

// RecoveryOwnerStatus describes the observed lease/probe, not an ownership
// credential. Unknown foreign owners retain the same protection as active ones.
type RecoveryOwnerStatus string

const (
	RecoveryOwnerActive   RecoveryOwnerStatus = "active"
	RecoveryOwnerInactive RecoveryOwnerStatus = "inactive"
	RecoveryOwnerUnknown  RecoveryOwnerStatus = "unknown"
)

func (store *recoveryStore) sessionOwnerStatus(session string) RecoveryOwnerStatus {
	path := filepath.Join(store.directory, "session-"+session+".json")
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return RecoveryOwnerInactive
		}
		return RecoveryOwnerUnknown
	}
	data, err := readRegularBounded(path, 4096)
	if err != nil {
		return RecoveryOwnerUnknown
	}
	var lease recoveryLease
	if err := DecodeStrict(data, &lease); err != nil || lease.Session != session || lease.PID <= 0 || int64(lease.PID) > 1<<31-1 {
		return RecoveryOwnerUnknown
	}
	return recoveryProcessOwnerStatus(lease.PID)
}

func (store *recoveryStore) recoveryCandidate(capsule recoveryCapsule, document Document, hash string) RecoveryCandidate {
	status := store.sessionOwnerStatus(capsule.Session)
	return RecoveryCandidate{ID: capsule.ID, DocumentKey: capsule.DocumentKey, SavedAt: capsule.SavedAt, SourcePath: capsule.SourcePath, SHA256: hash, CurrentSession: capsule.Session == store.session, ActiveSession: status != RecoveryOwnerInactive, OwnerStatus: status, Recordings: documentRecordingCount(document), Components: len(document.Graph.Components)}
}
