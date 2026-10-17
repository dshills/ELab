package workbench

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestRecoveryUnreadableOwnershipCannotAuthorizeOtherSessionDismissal(t *testing.T) {
	source := NewExample()
	defer source.Close()
	directory := recoverySetupDirectory(t)
	configureRecovery(t, source, directory, false)
	call(t, source, "recovery.checkpoint", struct{}{}, "owner-copy")
	choice := recoveryChoice(t, source)
	source.mu.Lock()
	lease := source.recoveryStore.leasePath()
	source.mu.Unlock()
	if err := os.WriteFile(lease, []byte("damaged lease\n"), 0600); err != nil {
		t.Fatal(err)
	}
	other := New()
	defer other.Close()
	configureRecovery(t, other, directory, false)
	refuse(t, other, "recovery.dismiss", choice, "unavailable")
	if _, err := os.Stat(filepath.Join(directory, choice.ID+".elab-recovery")); err != nil {
		t.Fatal("protected snapshot missing", err)
	}
}

func TestRecoveryOwnerClassificationAndForeignCopyProtection(t *testing.T) {
	for _, kind := range []string{"malformed", "missing-session", "wrong-session", "zero-pid", "oversized-pid", "extra-field", "oversized", "symlink", "directory", "unreadable"} {
		t.Run(kind, func(t *testing.T) {
			source := NewExample()
			defer source.Close()
			directory := recoverySetupDirectory(t)
			configureRecovery(t, source, directory, false)
			call(t, source, "recovery.checkpoint", struct{}{}, "checkpoint")
			choice := recoveryChoice(t, source)
			source.mu.Lock()
			lease := source.recoveryStore.leasePath()
			session := source.recoveryStore.session
			source.mu.Unlock()
			original, err := os.ReadFile(lease)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Remove(lease); _ = os.WriteFile(lease, original, 0600) }()
			data := []byte("damaged")
			switch kind {
			case "missing-session":
				data = []byte(`{"pid":1}`)
			case "wrong-session":
				data = []byte(`{"session":"wrong","pid":1}`)
			case "zero-pid":
				data, _ = json.Marshal(recoveryLease{Session: session, PID: 0})
			case "oversized-pid":
				data = []byte(`{"session":"` + session + `","pid":4294967296}`)
			case "extra-field":
				data = []byte(`{"session":"` + session + `","pid":1,"extra":true}`)
			case "oversized":
				data = bytes.Repeat([]byte("x"), 4097)
			case "symlink":
				target := filepath.Join(t.TempDir(), "lease-copy")
				if err := os.WriteFile(target, original, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(lease); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, lease); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Remove(lease); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(lease, 0700); err != nil {
					t.Fatal(err)
				}
			case "unreadable":
				if runtime.GOOS == "windows" {
					t.Skip("POSIX lease permissions")
				}
				if err := os.Chmod(lease, 0000); err != nil {
					t.Fatal(err)
				}
				if _, err := os.ReadFile(lease); err == nil {
					t.Skip("this user can read mode-zero files")
				}
			}
			if kind != "symlink" && kind != "directory" && kind != "unreadable" {
				if err := os.WriteFile(lease, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			other := New()
			defer other.Close()
			configureRecovery(t, other, directory, false)
			candidate := other.Snapshot().Recovery.Candidates[0]
			if candidate.OwnerStatus != RecoveryOwnerUnknown || !candidate.ActiveSession || candidate.CurrentSession {
				t.Fatal("unknown owner was reported unprotected", candidate)
			}
			read := call(t, other, "recovery.inspect", choice, "read").Result.(RecoveryInspection)
			if read.Candidate.OwnerStatus != RecoveryOwnerUnknown || !read.Candidate.ActiveSession {
				t.Fatal("query disagrees with scan")
			}
			path := filepath.Join(directory, choice.ID+".elab-recovery")
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			snapshot := other.Snapshot()
			refuse(t, other, "recovery.dismiss", choice, "unavailable")
			after := other.Snapshot()
			if after.ProjectWork == nil || after.ProjectWork.RecoveryDismissed == nil || *after.ProjectWork.RecoveryDismissed || !reflect.DeepEqual(snapshot.Components, after.Components) || snapshot.Document != after.Document {
				t.Fatal("refused deletion changed source/outcome")
			}
			current, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, current) {
				t.Fatal("unverified owner lost bytes", err)
			}
			call(t, other, "recovery.restore", choice, "restore-copy")
			if !other.Snapshot().Document.Dirty || len(other.Snapshot().Components) != 4 {
				t.Fatal("unknown ownership prevented copy restoration")
			}
			current, _ = os.ReadFile(path)
			if !bytes.Equal(before, current) {
				t.Fatal("restore changed original capsule")
			}
		})
	}
}

func TestRecoveryOwnerTransitionsAndOwnUnknownDismissal(t *testing.T) {
	source := NewExample()
	defer source.Close()
	directory := recoverySetupDirectory(t)
	configureRecovery(t, source, directory, false)
	call(t, source, "recovery.checkpoint", struct{}{}, "checkpoint")
	choice := recoveryChoice(t, source)
	source.mu.Lock()
	lease := source.recoveryStore.leasePath()
	source.mu.Unlock()
	other := New()
	defer other.Close()
	configureRecovery(t, other, directory, false)
	if other.Snapshot().Recovery.Candidates[0].OwnerStatus != RecoveryOwnerActive {
		t.Fatal("live owner not active")
	}
	original, err := os.ReadFile(lease)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lease, []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	call(t, other, "recovery.scan", struct{}{}, "unknown")
	if other.Snapshot().Recovery.Candidates[0].OwnerStatus != RecoveryOwnerUnknown {
		t.Fatal("bad lease not unknown")
	}
	if err := os.WriteFile(lease, original, 0600); err != nil {
		t.Fatal(err)
	}
	call(t, other, "recovery.scan", struct{}{}, "repaired")
	if other.Snapshot().Recovery.Candidates[0].OwnerStatus != RecoveryOwnerActive {
		t.Fatal("repaired live lease not active")
	}
	source.Close()
	call(t, other, "recovery.scan", struct{}{}, "closed")
	if other.Snapshot().Recovery.Candidates[0].OwnerStatus != RecoveryOwnerInactive || other.Snapshot().Recovery.Candidates[0].ActiveSession {
		t.Fatal("confirmed close not inactive")
	}
	call(t, other, "recovery.dismiss", choice, "dismiss-inactive")
	own := NewExample()
	defer own.Close()
	configureRecovery(t, own, recoverySetupDirectory(t), false)
	call(t, own, "recovery.checkpoint", struct{}{}, "checkpoint")
	ownChoice := recoveryChoice(t, own)
	own.mu.Lock()
	ownLease := own.recoveryStore.leasePath()
	own.mu.Unlock()
	if err := os.WriteFile(ownLease, []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	call(t, own, "recovery.dismiss", ownChoice, "dismiss-own")
}

func TestRecoveryDismissRechecksLeaseThatBecomesUnverifiable(t *testing.T) {
	s, choice, path := recoveryDismissService(t)
	defer s.Close()
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var capsule recoveryCapsule
	if err := json.Unmarshal(before, &capsule); err != nil {
		t.Fatal(err)
	}
	entered, release := blockRecoveryDismissPreparation(t, s)
	defer release()
	promptLiveCommand(t, s, "recovery.dismiss.start", choice)
	awaitLiveSampleBlock(t, entered)
	lease := filepath.Join(filepath.Dir(path), "session-"+capsule.Session+".json")
	if err := os.WriteFile(lease, []byte("unverifiable late owner"), 0600); err != nil {
		t.Fatal(err)
	}
	release()
	after := awaitProjectStatus(t, s, "failed")
	if after.ProjectWork.Error.Code != "unavailable" || *after.ProjectWork.RecoveryDismissed {
		t.Fatal("late unknown owner bypassed commit check")
	}
	current, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, current) {
		t.Fatal("late unknown owner lost snapshot")
	}
}

func TestRecoveryOwnerStatusSchemaAndKnownCurrentProcess(t *testing.T) {
	schema := typeSchema(reflect.TypeOf(RecoveryOwnerUnknown))
	if schema["type"] != "string" || !reflect.DeepEqual(schema["enum"], []RecoveryOwnerStatus{RecoveryOwnerActive, RecoveryOwnerInactive, RecoveryOwnerUnknown}) {
		t.Fatal("ownership enum not discoverable", schema)
	}
	if recoveryProcessOwnerStatus(os.Getpid()) != RecoveryOwnerActive {
		t.Fatal("current process is not active")
	}
}
