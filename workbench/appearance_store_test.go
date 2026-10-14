package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func appearancePath(t *testing.T, theme string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "appearance.json")
	data, _ := json.Marshal(appearanceFile{Format: "elab.appearance", Version: 1, Theme: theme})
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
func appearanceReady(t *testing.T, s *Service) Snapshot {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	state, err := s.waitAppearance(ctx)
	if err != nil || state.Appearance.Status != "ready" {
		t.Fatalf("appearance: %+v, %v", state.Appearance, err)
	}
	return state
}
func savedAppearance(t *testing.T, path string) string {
	t.Helper()
	loaded, err := readAppearance(context.Background(), path)
	if err != nil || !loaded.exists {
		t.Fatalf("preference read: %+v, %v", loaded, err)
	}
	return loaded.theme
}
func appearanceRelease(t *testing.T, s *Service, release chan struct{}) func() {
	t.Helper()
	var once sync.Once
	done := func() { once.Do(func() { close(release) }) }
	t.Cleanup(func() { done(); s.Close() })
	return done
}

func TestAppearancePersistsFreshSessionsWithoutCircuitChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "appearance.json")
	s := New()
	call(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "configure")
	appearanceReady(t, s)
	before := s.Snapshot()
	call(t, s, "appearance.theme.set", map[string]string{"theme": "dark"}, "dark")
	state := appearanceReady(t, s)
	if !reflect.DeepEqual(state.Document, before.Document) || state.ElectricalRevision != before.ElectricalRevision || state.RunID != before.RunID || state.CanUndo != before.CanUndo {
		t.Fatal("preference save changed circuit/history")
	}
	s.Close()
	if savedAppearance(t, path) != "dark" {
		t.Fatal("theme not saved")
	}
	reopened := New()
	defer reopened.Close()
	call(t, reopened, "appearance.configure", AppearanceConfigureParams{Path: path}, "reopen")
	if state := appearanceReady(t, reopened); state.Theme != "dark" || state.Document.Dirty {
		t.Fatal("fresh session did not load appearance independently")
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm()&0077 != 0 {
		t.Fatal("preference file not private")
	}
}
func TestAppearancePendingLoadCannotOverwriteLaterThemeChoice(t *testing.T) {
	path := appearancePath(t, "dark")
	s := New()
	entered, release := make(chan struct{}), make(chan struct{})
	s.appearanceRead = func(ctx context.Context, path string) (appearanceLoaded, error) {
		close(entered)
		<-release
		return readAppearance(ctx, path)
	}
	done := appearanceRelease(t, s, release)
	call(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "configure")
	<-entered
	call(t, s, "appearance.theme.set", map[string]string{"theme": "system"}, "later-choice")
	call(t, s, "view.pan", map[string]float64{"dx": 10, "dy": 0}, "navigation")
	done()
	if state := appearanceReady(t, s); state.Theme != "system" || savedAppearance(t, path) != "system" {
		t.Fatal("old load replaced later choice")
	}
}
func TestAppearanceWritesCoalesceAndCloseFlushesOutsideEditorLock(t *testing.T) {
	path := appearancePath(t, "light")
	s := New()
	entered, release := make(chan struct{}), make(chan struct{})
	done := appearanceRelease(t, s, release)
	var publishes atomic.Int32
	s.appearancePublish = func(p *preparedFile) error {
		if publishes.Add(1) == 1 {
			close(entered)
			<-release
		}
		return publishPreparedFile(p)
	}
	call(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "configure")
	appearanceReady(t, s)
	call(t, s, "appearance.theme.set", map[string]string{"theme": "dark"}, "dark")
	<-entered
	call(t, s, "appearance.theme.set", map[string]string{"theme": "system"}, "latest")
	call(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 0}, "pan-during-write")
	closed := make(chan struct{})
	go func() { s.Close(); close(closed) }()
	deadline := time.Now().Add(time.Second)
	for !s.Snapshot().Closed {
		if time.Now().After(deadline) {
			t.Fatal("close admission held editor lock")
		}
		time.Sleep(time.Millisecond)
	}
	if s.Snapshot().View.Center.X != 20 {
		t.Fatal("navigation blocked/lost")
	}
	done()
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("close did not finish preference flush")
	}
	if savedAppearance(t, path) != "system" || publishes.Load() != 2 {
		t.Fatal("latest closing preference not flushed")
	}
}
func TestAppearanceRefusesCorruptOversizedLinkedAndExternallyChangedFiles(t *testing.T) {
	for _, data := range []string{`{"format":"elab.appearance","version":1,"theme":"dark","theme":"light"}`, `{"format":"elab.appearance","version":2,"theme":"dark"}`, `{"format":"elab.appearance","version":1,"theme":"sepia"}`, string(make([]byte, MaxAppearanceBytes+1))} {
		t.Run("invalid", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "appearance.json")
			if err := os.WriteFile(path, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			s := New()
			defer s.Close()
			call(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "configure")
			state, err := s.waitAppearance(context.Background())
			if err != nil || state.Appearance.Status != "error" {
				t.Fatal("bad preference accepted")
			}
			call(t, s, "appearance.theme.set", map[string]string{"theme": "dark"}, "session-choice")
			actual, _ := os.ReadFile(path)
			if string(actual) != data {
				t.Fatal("corrupt file overwritten")
			}
		})
	}
	t.Run("link", func(t *testing.T) {
		target := appearancePath(t, "dark")
		path := filepath.Join(t.TempDir(), "link.json")
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		s := New()
		defer s.Close()
		call(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "configure")
		state, _ := s.waitAppearance(context.Background())
		if state.Appearance.Status != "error" || savedAppearance(t, target) != "dark" {
			t.Fatal("linked destination used")
		}
	})
	t.Run("external-change", func(t *testing.T) {
		path := appearancePath(t, "light")
		s := New()
		defer s.Close()
		call(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "configure")
		appearanceReady(t, s)
		external := []byte(`{"format":"elab.appearance","version":1,"theme":"system"}`)
		if err := os.WriteFile(path, external, 0600); err != nil {
			t.Fatal(err)
		}
		call(t, s, "appearance.theme.set", map[string]string{"theme": "dark"}, "choice")
		state, _ := s.waitAppearance(context.Background())
		actual, _ := os.ReadFile(path)
		if state.Appearance.Status != "error" || state.Theme != "dark" || !reflect.DeepEqual(actual, external) {
			t.Fatal("external preference overwritten")
		}
		call(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "reload")
		if state := appearanceReady(t, s); state.Theme != "system" {
			t.Fatal("explicit reload did not resynchronize")
		}
	})
}

func TestAppearanceWorkerQuotaDisableAndRetryAdmission(t *testing.T) {
	s := New()
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	s.appearanceRead = func(ctx context.Context, path string) (appearanceLoaded, error) {
		entered <- struct{}{}
		<-release
		return readAppearance(ctx, path)
	}
	done := appearanceRelease(t, s, release)
	path := appearancePath(t, "dark")
	response := call(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "first")
	<-entered
	revision := uint64(1)
	raw, _ := json.Marshal(AppearanceConfigureParams{Path: path})
	duplicate := s.Dispatch(context.Background(), Request{Version: 1, ID: "retry", Command: "appearance.configure", Params: raw, ExpectedRevision: &revision, Key: "first"}, "test")
	if duplicate.Error != nil || !duplicate.Duplicate || duplicate.Revision != response.Revision {
		done()
		s.Close()
		t.Fatal("configuration retry was not deduplicated")
	}
	call(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "second")
	<-entered
	before := s.Snapshot()
	refuse(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "busy")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		done()
		s.Close()
		t.Fatal("quota refusal changed preferences")
	}
	call(t, s, "appearance.configure", AppearanceConfigureParams{}, "disable")
	done()
	s.Close()
	if state := s.Snapshot(); state.Appearance.Status != "disabled" || state.Theme != "light" || savedAppearance(t, path) != "dark" {
		t.Fatal("old load changed disabled preferences")
	}
}

func TestAppearanceExplicitCurrentThemeSupersedesPendingLoad(t *testing.T) {
	path := appearancePath(t, "dark")
	s := New()
	entered, release := make(chan struct{}), make(chan struct{})
	done := appearanceRelease(t, s, release)
	s.appearanceRead = func(ctx context.Context, path string) (appearanceLoaded, error) {
		close(entered)
		<-release
		return readAppearance(ctx, path)
	}
	call(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "configure")
	<-entered
	// Light is already visible, but selecting it is a deliberate later choice.
	call(t, s, "appearance.theme.set", map[string]string{"theme": "light"}, "explicit-light")
	done()
	if state := appearanceReady(t, s); state.Theme != "light" || savedAppearance(t, path) != "light" {
		t.Fatal("pending load overrode explicit current-value choice")
	}
}
