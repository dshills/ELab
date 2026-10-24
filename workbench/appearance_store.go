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
	latest               *appearanceJob
}
type appearanceReadFunc func(context.Context, string) (appearanceLoaded, error)

func DefaultAppearanceFile() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "ELab", "appearance.json"), nil
}
func validTheme(theme string) bool {
	return theme == "light" || theme == "dark" || theme == "system" || theme == "apollo"
}
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
		s.detachAppearanceLocked("canceled")
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
	ctx, cancel := context.WithCancel(context.Background())
	store := &appearanceStore{path: path, initialThemeRevision: s.themeRevision, ctx: ctx, cancel: cancel, wake: make(chan struct{}, 1)}
	job, err := s.newAppearanceWorkLocked(store, "appearance.configure", "")
	if err != nil {
		cancel()
		return err
	}
	s.detachAppearanceLocked("superseded")
	store.latest, s.appearanceJob = job, job
	s.appearanceStore = store
	s.appearance = AppearanceState{Path: path, Status: "loading"}
	s.appearanceWorkers++
	s.liveWorkers.Add(1)
	read, publish := s.appearanceRead, s.appearancePublish
	go s.runAppearanceStore(store, read, publish)
	return nil
}
func (s *Service) setAppearanceThemeLocked(theme string) error {
	if store := s.appearanceStore; store != nil && s.appearance.Status != "error" {
		job, err := s.newAppearanceWorkLocked(store, "appearance.theme.set", theme)
		if err != nil {
			return err
		}
		s.interruptAppearanceJobLocked(store.latest, "superseded")
		store.latest, s.appearanceJob = job, job
		if s.appearance.Status != "loading" {
			s.appearance.Status = "saving"
		}
		select {
		case store.wake <- struct{}{}:
		default:
		}
	}
	s.theme = theme
	s.themeRevision++
	return nil
}
func (s *Service) appearanceSignalLocked() { s.revision++; s.signalLocked("preferences", "jobs") }
func (s *Service) runAppearanceStore(store *appearanceStore, read appearanceReadFunc, publish projectPublishFunc) {
	defer func() { store.cancel(); s.mu.Lock(); s.appearanceWorkers--; s.mu.Unlock(); s.liveWorkers.Done() }()
	s.mu.Lock()
	if s.appearanceStore != store || store.ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	reading := store.latest
	reading.inProgress = true
	reading.work.Status, reading.work.Phase = "running", "reading"
	s.appearanceSignalLocked()
	s.mu.Unlock()
	loaded, err := read(store.ctx, store.path)
	s.mu.Lock()
	reading.inProgress = false
	if reading.reason != "" {
		s.finishAppearanceReasonLocked(reading)
		s.appearanceSignalLocked()
	}
	if s.appearanceStore != store || store.ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	if err != nil {
		s.failAppearanceStoreLocked(store, reading, err)
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
		job := store.latest
		needsSave := !loaded.exists || loaded.theme != theme
		if !needsSave {
			if operationActive(job.work.Status) {
				job.work.SourceSHA256 = fmt.Sprintf("%x", loaded.hash)
				s.completeAppearanceJobLocked(job, theme, fmt.Sprintf("%x", loaded.hash))
				s.appearanceSignalLocked()
			}
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
		job.inProgress = true
		job.work.Status, job.work.Phase = "running", "preparing"
		if loaded.exists {
			job.work.SourceSHA256 = fmt.Sprintf("%x", loaded.hash)
		}
		s.appearanceSignalLocked()
		s.mu.Unlock()
		var keepRunning bool
		loaded, keepRunning = s.runAppearanceSave(store, job, loaded, theme, publish)
		if !keepRunning {
			return
		}
	}
}

func (s *Service) failAppearanceStoreLocked(store *appearanceStore, job *appearanceJob, err error) {
	s.failAppearanceJobLocked(job, err)
	s.failAppearanceJobLocked(store.latest, err)
	s.appearance.Error, s.appearance.Status = err.Error(), "error"
	s.appearanceSignalLocked()
}

// Prepare and publish outside the editor lock. The locked publishing transition
// is the cancellation boundary; detached publishers retain their actual result.
func (s *Service) runAppearanceSave(store *appearanceStore, job *appearanceJob, loaded appearanceLoaded, theme string, publish projectPublishFunc) (appearanceLoaded, bool) {
	data, _ := json.MarshalIndent(appearanceFile{Format: "elab.appearance", Version: 1, Theme: theme}, "", "  ")
	data = append(data, '\n')
	ctx, cancel := context.WithTimeout(job.ctx, 5*time.Second)
	defer cancel()
	err := os.MkdirAll(filepath.Dir(store.path), 0700)
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
	s.mu.Lock()
	current := s.appearanceStore == store && store.ctx.Err() == nil && job.reason == "" && operationActive(job.work.Status)
	if current && err == nil {
		job.publishing = true
		job.work.Phase, job.work.Cancelable = "publishing", false
		s.appearanceSignalLocked()
	}
	s.mu.Unlock()
	published := false
	if current && err == nil {
		err = publish(prepared)
		published = err == nil
	}
	var cleanupWarning string
	if prepared != nil {
		if cleanup := os.Remove(prepared.temp); cleanup != nil && !os.IsNotExist(cleanup) {
			cleanupWarning = cleanup.Error()
			if err == nil {
				err = cleanup
			}
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	job.publishing, job.inProgress = false, false
	job.work.CleanupWarning = cleanupWarning
	if published {
		job.work.Published, job.work.Bytes = true, len(data)
		job.work.ResultTheme, job.work.ArtifactSHA256 = theme, fmt.Sprintf("%x", sha256.Sum256(data))
		loaded = appearanceLoaded{theme: theme, exists: true, hash: sha256.Sum256(data)}
	}
	if job.reason != "" {
		s.finishAppearanceReasonLocked(job)
	} else if err != nil {
		s.failAppearanceJobLocked(job, err)
	} else if published {
		s.completeAppearanceJobLocked(job, theme, job.work.ArtifactSHA256)
	}
	s.appearanceSignalLocked()
	if s.appearanceStore != store || store.ctx.Err() != nil {
		return loaded, false
	}
	if err != nil && job.reason == "" {
		s.failAppearanceStoreLocked(store, job, err)
		return loaded, false
	}
	if published {
		s.appearance.SavedTheme = theme
	}
	return loaded, true
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
