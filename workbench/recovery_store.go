package workbench

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type recoveryLease struct {
	Session string `json:"session"`
	PID     int    `json:"pid"`
}
type recoveryStore struct {
	mu                 sync.Mutex
	directory, session string
	sequence           uint64
	closed             bool
	lastRevision       map[string]uint64
	lastStateRevision  map[string]uint64
	lastHash           map[string][32]byte
	owned              map[string][]string
}

func newRecoveryStore(directory string) (*recoveryStore, error) {
	path, err := documentPath(directory)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, fileError(err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fileError(err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fileError(fmt.Errorf("recovery directory must be a real directory, not a link"))
	}
	if info.Mode().Perm()&0077 != 0 {
		return nil, fileError(fmt.Errorf("recovery directory must be private (permissions 0700)"))
	}
	var nonce [12]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	store := &recoveryStore{directory: path, session: hex.EncodeToString(nonce[:]), lastRevision: map[string]uint64{}, lastStateRevision: map[string]uint64{}, lastHash: map[string][32]byte{}, owned: map[string][]string{}}
	lease, _ := json.Marshal(recoveryLease{Session: store.session, PID: os.Getpid()})
	if err := publishExport(store.leasePath(), lease, false); err != nil {
		return nil, err
	}
	return store, nil
}
func (store *recoveryStore) leasePath() string {
	return filepath.Join(store.directory, "session-"+store.session+".json")
}
func (store *recoveryStore) documentKey(generation uint64) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", store.session, generation)))
	return hex.EncodeToString(hash[:12])
}
func (store *recoveryStore) checkpoint(capture recoveryCapture) (string, string, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.checkpointLocked(capture)
}
func (store *recoveryStore) checkpointLocked(capture recoveryCapture) (string, string, error) {
	if store.closed {
		return "", "", nil
	}
	key := store.documentKey(capture.generation)
	// Compare both command admission and capture order before even a clean
	// retirement. Delayed cleanup must not delete a newer dirty snapshot.
	if capture.commandRevision < store.lastRevision[key] || capture.stateRevision != 0 && capture.stateRevision < store.lastStateRevision[key] {
		return "", "", nil
	}
	store.lastRevision[key] = max(store.lastRevision[key], capture.commandRevision)
	store.lastStateRevision[key] = max(store.lastStateRevision[key], capture.stateRevision)
	if !capture.dirty {
		store.lastRevision[key] = max(store.lastRevision[key], capture.commandRevision)
		return "", "", store.retireLocked(key)
	}
	data, err := json.Marshal(capture.document)
	if err != nil {
		return "", "", err
	}
	digest := sha256.Sum256(data)
	if capture.commandRevision < store.lastRevision[key] || digest == store.lastHash[key] {
		return "", "", nil
	}
	// Directory admission is bounded; retention can replace one of our two
	// existing snapshots without increasing the final entry count.
	entries, err := store.entries()
	if err != nil {
		return "", "", fileError(err)
	}
	if len(entries) >= MaxRecoveryEntries && len(store.owned[key]) < 2 {
		return "", "", fmt.Errorf("recovery directory is full; dismiss or move older snapshots")
	}
	store.sequence++
	id := fmt.Sprintf("rec-%s-%010d", key, store.sequence)
	savedAt := time.Now().UTC().Format(time.RFC3339Nano)
	capsule := recoveryCapsule{Format: RecoveryFormat, Version: RecoveryVersion, ID: id, DocumentKey: key, Session: store.session, Sequence: store.sequence, CommandRevision: capture.commandRevision, SavedAt: savedAt, SourcePath: capture.sourcePath, DocumentSHA256: hex.EncodeToString(digest[:]), Document: data}
	encoded, err := json.Marshal(capsule)
	if err != nil {
		return "", "", err
	}
	encoded = append(encoded, '\n')
	if len(encoded) > MaxRecoveryBytes {
		return "", "", fmt.Errorf("recovery snapshot exceeds its size bound")
	}
	if err := publishExport(store.candidatePath(id), encoded, false); err != nil {
		return "", "", err
	}
	store.lastRevision[key] = capture.commandRevision
	store.lastHash[key] = digest
	store.owned[key] = append(store.owned[key], id)
	// Only prune snapshots written by this store, after the next complete copy exists.
	for len(store.owned[key]) > 2 {
		old := store.owned[key][0]
		if err := os.Remove(store.candidatePath(old)); err != nil && !os.IsNotExist(err) {
			return id, savedAt, fileError(err)
		}
		store.owned[key] = store.owned[key][1:]
	}
	return id, savedAt, nil
}
func (store *recoveryStore) retireLocked(key string) error {
	for _, id := range store.owned[key] {
		if err := os.Remove(store.candidatePath(id)); err != nil && !os.IsNotExist(err) {
			return fileError(err)
		}
	}
	delete(store.owned, key)
	delete(store.lastHash, key)
	return nil
}

// finish serializes the final capture and lease removal with background writes.
// Once closed, an already captured worker cannot recreate discarded snapshots.
func (store *recoveryStore) finish(capture recoveryCapture, discard, checkpoint bool) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	var err error
	if discard {
		err = store.retireLocked(store.documentKey(capture.generation))
	} else if checkpoint {
		_, _, err = store.checkpointLocked(capture)
	}
	store.closed = true
	if removeErr := os.Remove(store.leasePath()); err == nil && removeErr != nil && !os.IsNotExist(removeErr) {
		err = fileError(removeErr)
	}
	return err
}
func (store *recoveryStore) closeLease() {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.closed = true
	_ = os.Remove(store.leasePath())
}
func (store *recoveryStore) candidatePath(id string) string {
	return filepath.Join(store.directory, id+".elab-recovery")
}
func recoveryID(id string) bool { return ValidID(id) && strings.HasPrefix(id, "rec-") }
func (store *recoveryStore) read(id string) (recoveryCapsule, Document, string, error) {
	var capsule recoveryCapsule
	var document Document
	if !recoveryID(id) {
		return capsule, document, "", fmt.Errorf("invalid recovery ID")
	}
	data, err := readRegularBounded(store.candidatePath(id), MaxRecoveryBytes)
	if err != nil {
		return capsule, document, "", err
	}
	digest := sha256.Sum256(data)
	hash := hex.EncodeToString(digest[:])
	if err := params(data, &capsule, "format", "version", "id", "document_key", "session", "sequence", "command_revision", "saved_at", "document_sha256", "document"); err != nil {
		return capsule, document, hash, err
	}
	if capsule.Format != RecoveryFormat || capsule.Version != RecoveryVersion || capsule.ID != id || capsule.Sequence == 0 || len(capsule.DocumentKey) != 24 || len(capsule.Session) != 24 {
		return capsule, document, hash, fmt.Errorf("unsupported or inconsistent recovery identity")
	}
	if _, err := hex.DecodeString(capsule.DocumentKey); err != nil {
		return capsule, document, hash, err
	}
	if _, err := hex.DecodeString(capsule.Session); err != nil {
		return capsule, document, hash, err
	}
	if capsule.ID != fmt.Sprintf("rec-%s-%010d", capsule.DocumentKey, capsule.Sequence) {
		return capsule, document, hash, fmt.Errorf("recovery filename does not match sequence")
	}
	if _, err := time.Parse(time.RFC3339Nano, capsule.SavedAt); err != nil {
		return capsule, document, hash, err
	}
	if capsule.SourcePath != "" && (!filepath.IsAbs(capsule.SourcePath) || strings.ContainsRune(capsule.SourcePath, 0) || len(capsule.SourcePath) > 4096) {
		return capsule, document, hash, fmt.Errorf("invalid recovery source path")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, capsule.Document); err != nil {
		return capsule, document, hash, err
	}
	documentDigest := sha256.Sum256(compact.Bytes())
	if hex.EncodeToString(documentDigest[:]) != capsule.DocumentSHA256 {
		return capsule, document, hash, fmt.Errorf("recovery document checksum does not match")
	}
	document, err = decodeDocument(capsule.Document)
	return capsule, document, hash, err
}
func readRegularBounded(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fileError(err)
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fileError(fmt.Errorf("file must be regular and within its size bound"))
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fileError(err)
	}
	defer file.Close()
	actual, err := file.Stat()
	if err != nil {
		return nil, fileError(err)
	}
	if !os.SameFile(info, actual) || !actual.Mode().IsRegular() {
		return nil, fileError(fmt.Errorf("file changed while opening"))
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fileError(err)
	}
	if int64(len(data)) > limit {
		return nil, fileError(fmt.Errorf("file exceeds size bound"))
	}
	return data, nil
}
func (store *recoveryStore) sessionActive(session string) bool {
	data, err := readRegularBounded(filepath.Join(store.directory, "session-"+session+".json"), 4096)
	if err != nil {
		return false
	}
	var lease recoveryLease
	if err := DecodeStrict(data, &lease); err != nil || lease.Session != session || lease.PID <= 0 {
		return false
	}
	return recoveryProcessAlive(lease.PID)
}
func (store *recoveryStore) scan() ([]RecoveryCandidate, []RecoveryProblem, error) {
	entries, err := store.entries()
	if err != nil {
		return nil, nil, err
	}
	latest := map[string]recoveryCapsule{}
	candidates := map[string]RecoveryCandidate{}
	problems := []RecoveryProblem{}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".elab-recovery") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".elab-recovery")
		if !recoveryID(id) {
			continue
		}
		capsule, document, hash, err := store.read(id)
		if err != nil {
			problems = append(problems, RecoveryProblem{ID: id, SHA256: hash, Message: recoveryMessage(err)})
			continue
		}
		previous, exists := latest[capsule.DocumentKey]
		if exists && capsule.Sequence <= previous.Sequence {
			continue
		}
		latest[capsule.DocumentKey] = capsule
		candidates[capsule.DocumentKey] = RecoveryCandidate{ID: id, DocumentKey: capsule.DocumentKey, SavedAt: capsule.SavedAt, SourcePath: capsule.SourcePath, SHA256: hash, CurrentSession: capsule.Session == store.session, ActiveSession: store.sessionActive(capsule.Session), Recordings: documentRecordingCount(document), Components: len(document.Graph.Components)}
	}
	out := []RecoveryCandidate{}
	for _, candidate := range candidates {
		out = append(out, candidate)
	}
	sort.Slice(out, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339Nano, out[i].SavedAt)
		b, _ := time.Parse(time.RFC3339Nano, out[j].SavedAt)
		if a.Equal(b) {
			return out[i].ID > out[j].ID
		}
		return a.After(b)
	})
	sort.Slice(problems, func(i, j int) bool { return problems[i].ID < problems[j].ID })
	return out, problems, nil
}

func (store *recoveryStore) entries() ([]os.DirEntry, error) {
	directory, err := os.Open(store.directory)
	if err != nil {
		return nil, fileError(err)
	}
	entries, err := directory.ReadDir(MaxRecoveryEntries + 1)
	_ = directory.Close()
	if err != nil && err != io.EOF {
		return nil, fileError(err)
	}
	if len(entries) > MaxRecoveryEntries {
		return nil, fmt.Errorf("recovery directory exceeds %d entries; preserve and move older snapshots before scanning", MaxRecoveryEntries)
	}
	return entries, nil
}

// Keep damaged-file diagnostics bounded even if a JSON field name is enormous.
func recoveryMessage(err error) string {
	message := []rune(err.Error())
	if len(message) > 512 {
		return string(message[:512]) + "…"
	}
	return string(message)
}
