package main

import "github.com/dshills/vellum/app"

// windowReference captures this screen's original opening on the UI owner.
// A delayed operation must retain that reference instead of resolving a reused
// logical name when its result becomes ready.
func (s *screen) windowReference() (app.WindowRef, error) {
	if s.app.Context().Err() != nil {
		return app.WindowRef{}, app.ErrClosed
	}
	if s.window == (app.WindowRef{}) {
		ref, err := s.app.WindowRef("main")
		if err != nil {
			return app.WindowRef{}, err
		}
		s.window = ref
	}
	if s.window.Context().Err() != nil {
		return app.WindowRef{}, app.ErrWindowClosed
	}
	return s.window, nil
}

// withWindowOwner checks retirement again after acquiring the UI owner. Custom
// canvas callbacks must not apply input to a later opening with the same name.
func withWindowOwner(owner *app.Application, window app.WindowRef, fn func()) error {
	admitted := false
	err := owner.Do(func() {
		if window.Context().Err() != nil {
			return
		}
		admitted = true
		fn()
	})
	if err != nil {
		return err
	}
	if !admitted {
		return app.ErrWindowClosed
	}
	return nil
}
