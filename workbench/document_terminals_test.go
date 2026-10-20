package workbench

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func terminalShapeDocument(t *testing.T) Document {
	t.Helper()
	g := EmptyGraph()
	for i, part := range Library() {
		component, err := MakeComponent(fmt.Sprintf("PART%d", i+1), part.Kind, Point{X: float64(i) * 100}, "")
		if err != nil {
			t.Fatal(err)
		}
		g.Components = append(g.Components, component)
	}
	s := newGraphService(SolveLED, g)
	defer s.Close()
	s.mu.Lock()
	document := s.documentLocked()
	s.mu.Unlock()
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeDocument(data); err != nil {
		t.Fatal("valid complete library metadata must remain readable", err)
	}
	return document
}

func TestMalformedTerminalOpenPreservesDirtyProjectAndSavedFile(t *testing.T) {
	for _, pins := range []bool{false, true} {
		t.Run(fmt.Sprint(pins), func(t *testing.T) {
			s, path := savedLED(t)
			defer s.Close()
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var d Document
			if err := json.Unmarshal(original, &d); err != nil {
				t.Fatal(err)
			}
			for i := range d.Graph.Components {
				if d.Graph.Components[i].Kind == "resistor" {
					d.Graph.Components[i].Terminals = []string{}
				}
			}
			if pins {
				d.PinnedReadings = &PinPreferences{Items: []PinSpec{}}
			}
			bad, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			badPath := filepath.Join(t.TempDir(), "damaged.elab")
			if err := os.WriteFile(badPath, bad, 0600); err != nil {
				t.Fatal(err)
			}
			call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "dirty-edit")
			refuse(t, s, "project.load", LoadParams{Path: badPath, DiscardChanges: true}, "invalid-document")
			for name, expected := range map[string][]byte{path: original, badPath: bad} {
				actual, err := os.ReadFile(name)
				if err != nil || !bytes.Equal(actual, expected) {
					t.Fatal("refused Open changed a file", name, err)
				}
			}
			call(t, s, "project.load", LoadParams{Path: path, DiscardChanges: true}, "valid-open-after-refusal")
			if state := settled(t, s); state.Document.Dirty || state.Circuit.Resistance != 1000 {
				t.Fatal("valid Open did not recover after a malformed file refusal")
			}
		})
	}
}

func TestRecoveryMalformedTerminalsRemainProblemsWithoutDeletingCapsule(t *testing.T) {
	document := terminalShapeDocument(t)
	for i := range document.Graph.Components {
		if document.Graph.Components[i].Kind == "capacitor" {
			document.Graph.Components[i].Terminals = []string{}
		}
	}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	key := strings.Repeat("a", 24)
	capsule := recoveryCapsule{Format: RecoveryFormat, Version: RecoveryVersion,
		ID: "rec-" + key + "-0000000001", DocumentKey: key, Session: strings.Repeat("b", 24),
		Sequence: 1, SavedAt: time.Unix(0, 0).UTC().Format(time.RFC3339Nano),
		DocumentSHA256: hex.EncodeToString(digest[:]), Document: data}
	encoded, err := json.Marshal(capsule)
	if err != nil {
		t.Fatal(err)
	}
	store, err := newRecoveryStore(filepath.Join(t.TempDir(), "recovery"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.closeLease()
	path := store.candidatePath(capsule.ID)
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := store.read(capsule.ID); err == nil || !strings.Contains(err.Error(), "terminal metadata") {
		t.Fatal("valid capsule checksum did not reach terminal validation", err)
	}
	candidates, problems, err := store.scan()
	if err != nil || len(candidates) != 0 || len(problems) != 1 {
		t.Fatal("malformed capsule must remain an inspectable recovery problem", candidates, problems, err)
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, encoded) {
		t.Fatal("recovery validation deleted or rewrote the damaged capsule", err)
	}
}

func TestDocumentRejectsMalformedTerminalsBeforeProbeInventory(t *testing.T) {
	baseline := terminalShapeDocument(t)
	for _, kind := range []string{"resistor", "capacitor", "inductor", "led", "diode", "switch"} {
		for _, count := range []int{0, 1} {
			for _, pins := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/pins-%t/terminals-%d", kind, pins, count), func(t *testing.T) {
					defer func() {
						if failure := recover(); failure != nil {
							t.Errorf("malformed file must return an error rather than panic: %v", failure)
						}
					}()
					d := baseline
					d.Graph = baseline.Graph.Clone()
					for i := range d.Graph.Components {
						if d.Graph.Components[i].Kind == kind {
							d.Graph.Components[i].Terminals = append([]string{}, d.Graph.Components[i].Terminals[:count]...)
						}
					}
					if pins {
						d.PinnedReadings = &PinPreferences{Items: []PinSpec{}}
					}
					data, err := json.Marshal(d)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := decodeDocument(data); err == nil {
						t.Fatal("malformed terminal metadata was admitted")
					}
				})
			}
		}
	}
}
