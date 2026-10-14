package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const MaxAppearanceBytes = 4096
const MaxAppearanceWorkers = 2

type AppearanceState struct {
	Path       string `json:"path"`
	Status     string `json:"status"`
	SavedTheme string `json:"saved_theme,omitempty"`
	Error      string `json:"error,omitempty"`
}
type AppearanceConfigureParams struct {
	Path string `json:"path"`
}
type appearanceFile struct {
	Format  string `json:"format"`
	Version int    `json:"version"`
	Theme   string `json:"theme"`
}
type appearanceLoaded struct {
	theme  string
	hash   [32]byte
	exists bool
}
type appearanceStore struct {
	path                 string
	initialThemeRevision uint64
	ctx                  context.Context
	cancel               context.CancelFunc
	wake                 chan struct{}
}
type appearanceReadFunc func(context.Context, string) (appearanceLoaded, error)

func DefaultAppearanceFile() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "ELab", "appearance.json"), nil
}
func validTheme(theme string) bool { return theme == "light" || theme == "dark" || theme == "system" }
func readAppearance(ctx context.Context, path string) (appearanceLoaded, error) {
	if err := ctx.Err(); err != nil {
		return appearanceLoaded{}, err
	}
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return appearanceLoaded{}, nil
	} else if err != nil {
		return appearanceLoaded{}, err
	}
	data, err := readRegularBounded(path, MaxAppearanceBytes)
	if err != nil {
		return appearanceLoaded{}, err
	}
	var value appearanceFile
	if err = params(data, &value, "format", "version", "theme"); err != nil {
		return appearanceLoaded{}, err
	}
	if value.Format != "elab.appearance" || value.Version != 1 || !validTheme(value.Theme) {
		return appearanceLoaded{}, fmt.Errorf("invalid appearance preference format, version or theme")
	}
	return appearanceLoaded{theme: value.Theme, hash: sha256.Sum256(data), exists: true}, ctx.Err()
}
func (s *Service) configureAppearanceLocked(p AppearanceConfigureParams) error {
	if p.Path == "" {
		if s.appearanceStore != nil {
			s.appearanceStore.cancel()
		}
		s.appearanceStore = nil
		s.appearance = AppearanceState{Path: s.appearance.Path, Status: "disabled", SavedTheme: s.appearance.SavedTheme}
		return nil
	}
	if s.appearanceWorkers >= MaxAppearanceWorkers {
		return &actionError{"busy", "Wait for pending appearance workers to finish"}
	}
	path, err := documentPath(p.Path)
	if err != nil {
		return err
	}
	if s.appearanceStore != nil {
		s.appearanceStore.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	store := &appearanceStore{path: path, initialThemeRevision: s.themeRevision, ctx: ctx, cancel: cancel, wake: make(chan struct{}, 1)}
	s.appearanceStore = store
	s.appearance = AppearanceState{Path: path, Status: "loading"}
	s.appearanceWorkers++
	s.liveWorkers.Add(1)
	read, publish := s.appearanceRead, s.appearancePublish
	go s.runAppearanceStore(store, read, publish)
	return nil
}
func (s *Service) appearanceChangedLocked() {
	s.themeRevision++
	if store := s.appearanceStore; store != nil && s.appearance.Status != "error" {
		if s.appearance.Status != "loading" && s.theme != s.appearance.SavedTheme {
			s.appearance.Status = "saving"
		}
		select {
		case store.wake <- struct{}{}:
		default:
		}
	}
}
func (s *Service) appearanceSignalLocked() { s.revision++; s.signalLocked("preferences") }
func (s *Service) runAppearanceStore(store *appearanceStore, read appearanceReadFunc, publish projectPublishFunc) {
	defer func() { store.cancel(); s.mu.Lock(); s.appearanceWorkers--; s.mu.Unlock(); s.liveWorkers.Done() }()
	loaded, err := read(store.ctx, store.path)
	s.mu.Lock()
	if s.appearanceStore != store || store.ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	if err != nil {
		s.appearance.Error = err.Error()
		s.appearance.Status = "error"
		s.appearanceSignalLocked()
		s.mu.Unlock()
		return
	}
	if loaded.exists && s.themeRevision == store.initialThemeRevision {
		s.theme = loaded.theme
		s.themeRevision++
	}
	s.appearance.SavedTheme = loaded.theme
	s.appearance.Status = "saving"
	s.appearanceSignalLocked()
	s.mu.Unlock()
	for {
		s.mu.Lock()
		if s.appearanceStore != store || store.ctx.Err() != nil {
			s.mu.Unlock()
			return
		}
		theme := s.theme
		needsSave := !loaded.exists || loaded.theme != theme
		if !needsSave {
			ready := AppearanceState{Path: store.path, Status: "ready", SavedTheme: theme}
			if s.appearance != ready {
				s.appearance = ready
				s.appearanceSignalLocked()
			}
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return
			}
			select {
			case <-store.ctx.Done():
				return
			case <-store.wake:
			}
			continue
		}
		s.appearance.Status = "saving"
		s.mu.Unlock()
		data, _ := json.MarshalIndent(appearanceFile{Format: "elab.appearance", Version: 1, Theme: theme}, "", "  ")
		data = append(data, '\n')
		ctx, cancel := context.WithTimeout(store.ctx, 5*time.Second)
		err = os.MkdirAll(filepath.Dir(store.path), 0700)
		var prepared *preparedFile
		if err == nil {
			source := ""
			if loaded.exists {
				source = store.path
			}
			prepared, err = prepareFileData(ctx, data, store.path, source, loaded.hash, false, ".elab-appearance-*", func(string, int) {})
		}
		if err == nil {
			err = ctx.Err()
		}
		if err == nil {
			err = publish(prepared)
		}
		if prepared != nil {
			if cleanup := os.Remove(prepared.temp); err == nil && cleanup != nil && !os.IsNotExist(cleanup) {
				err = cleanup
			}
		}
		cancel()
		s.mu.Lock()
		if s.appearanceStore != store || store.ctx.Err() != nil {
			s.mu.Unlock()
			return
		}
		if err != nil {
			s.appearance.Error = err.Error()
			s.appearance.Status = "error"
			s.appearanceSignalLocked()
			s.mu.Unlock()
			return
		}
		loaded = appearanceLoaded{theme: theme, exists: true, hash: sha256.Sum256(data)}
		s.appearance.SavedTheme = theme
		s.appearanceSignalLocked()
		s.mu.Unlock()
	}
}
func (s *Service) waitAppearance(ctx context.Context) (Snapshot, error) {
	for {
		s.mu.Lock()
		state := s.snapshotLocked()
		changed := s.changed
		s.mu.Unlock()
		if state.Appearance.Status != "loading" && state.Appearance.Status != "saving" {
			return state, nil
		}
		select {
		case <-ctx.Done():
			return state, ctx.Err()
		case <-changed:
		}
	}
}
