package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"

	"github.com/dshills/vellum/ui"
)

func TestNativeThemePersistenceSetupAndSessionOnlyControl(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	path := filepath.Join(t.TempDir(), "appearance.json")
	if err := setupAppearance(s.service, path); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "theme-menu", &ui.ChangeEvent{SelectedID: "dark"})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// Use the same public preference wait an external agent sends.
	result := s.service.Dispatch(ctx, workbench.Request{Version: 1, ID: "appearance-wait", Command: "appearance.wait", Params: json.RawMessage(`{}`)}, "agent")
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	if s.service.Snapshot().Appearance.SavedTheme != "dark" {
		t.Fatal("native choice was not saved")
	}
	uiAction(t, s, "appearance-session-only", nil)
	uiAction(t, s, "theme-menu", &ui.ChangeEvent{SelectedID: "light"})
	if s.service.Snapshot().Appearance.Status != "disabled" {
		t.Fatal("session-only choice bypassed shared command")
	}
	uiAction(t, s, "appearance-enable", nil)
	result = s.service.Dispatch(ctx, workbench.Request{Version: 1, ID: "reenable", Command: "appearance.wait", Params: json.RawMessage(`{}`)}, "agent")
	if result.Error != nil || s.service.Snapshot().Theme != "dark" {
		t.Fatal("re-enable did not use saved profile")
	}
	fresh, clean := testScreen(t)
	defer clean()
	if err := setupAppearance(fresh.service, path); err != nil {
		t.Fatal(err)
	}
	result = fresh.service.Dispatch(ctx, workbench.Request{Version: 1, ID: "appearance-wait", Command: "appearance.wait", Params: json.RawMessage(`{}`)}, "agent")
	if result.Error != nil || fresh.service.Snapshot().Theme != "dark" {
		t.Fatal("session-only override changed stored preference")
	}
}

func TestNativePaletteSnapshotsAreIndependentDuringThemeChanges(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	light, dark := resolvePalette("light", false), resolvePalette("dark", false)
	var wg sync.WaitGroup
	failure := make(chan appPalette, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			p := s.nativeColors()
			if p != light && p != dark {
				select {
				case failure <- p:
				default:
				}
				return
			}
		}
	}()
	for i := 0; i < 50; i++ {
		mode := "light"
		if i%2 == 0 {
			mode = "dark"
		}
		uiAction(t, s, "theme-menu", &ui.ChangeEvent{SelectedID: mode})
		s.app.Do(func() { s.view() })
	}
	wg.Wait()
	select {
	case p := <-failure:
		t.Fatalf("partially published palette: %+v", p)
	default:
	}
}
