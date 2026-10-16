package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func keepTestApplicationOpen(t *testing.T, s *screen) {
	t.Helper()
	if err := s.app.Open(app.Window{ID: "keeper", Title: "Lifetime fixture", Width: 200, Height: 100, View: func() ui.Node { return ui.Node{ID: "keeper", Kind: ui.Column} }}); err != nil {
		t.Fatal(err)
	}
}

func TestScreenWindowReferenceDoesNotAdoptReusedName(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	keepTestApplicationOpen(t, s)
	openCloseTestWindow(t, s)
	original, err := s.windowReference()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.app.CloseWindow(original); err != nil {
		t.Fatal(err)
	}
	openCloseTestWindow(t, s)
	replacement, err := s.app.WindowRef("main")
	if err != nil || replacement == original || original.Context().Err() == nil || replacement.Context().Err() != nil {
		t.Fatal("fixture did not create an independent live opening", err)
	}
	if _, err := s.windowReference(); !errors.Is(err, app.ErrWindowClosed) {
		t.Fatal("old screen adopted the replacement opening", err)
	}
	called := false
	if err := s.app.PostWindow(original, func() { called = true }); !errors.Is(err, app.ErrWindowClosed) {
		t.Fatal("old callback was admitted for replacement", err)
	}
	s.app.Drain()
	if err := withWindowOwner(s.app, original, func() { called = true }); !errors.Is(err, app.ErrWindowClosed) {
		t.Fatal("old canvas input was admitted for replacement", err)
	}
	if called {
		t.Fatal("old callback reached replacement")
	}
	if err := withWindowOwner(s.app, replacement, func() { called = true }); err != nil || !called {
		t.Fatal("current canvas input was refused", err)
	}
	fresh := &screen{app: s.app}
	if ref, err := fresh.windowReference(); err != nil || ref != replacement {
		t.Fatal("new screen cannot deliberately capture current opening", err)
	}
}

func TestNativeFilePickerUsesOriginalWindowContext(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	keepTestApplicationOpen(t, s)
	openCloseTestWindow(t, s)
	entered := make(chan context.Context, 1)
	returned := make(chan struct{})
	s.chooseFile = func(ctx context.Context, _ app.FileOptions) (string, error) {
		entered <- ctx
		<-ctx.Done()
		close(returned)
		return "", app.ErrCanceled
	}
	uiAction(t, s, "file.save-as", nil)
	var ctx context.Context
	select {
	case ctx = <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("file picker did not start")
	}
	original := s.window
	if err := s.app.CloseWindow(original); err != nil {
		t.Fatal(err)
	}
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("retiring the window did not cancel its file picker")
	}
	if ctx.Err() == nil || s.app.Context().Err() != nil {
		t.Fatal("picker used the application lifetime instead of its window")
	}
}
