package main

import "github.com/dshills/elab/workbench"

// requestClose runs on the owner for both native window policy and menu/keys.
// The service owns clean/dirty admission; the main loop tears down the window
// only after shared cleanup succeeds or its failure is acknowledged. Never replace an active choice.
func (s *screen) requestClose() bool {
	if state := s.service.Snapshot(); state.Closed {
		if state.Shutdown != nil && state.Shutdown.Status == "failed" && !state.Shutdown.ExitReady {
			s.invoke("app.close.acknowledge", workbench.IDParams{ID: state.Shutdown.ID})
		}
		return false
	}
	if s.fileBusy || s.pendingCommand != "" {
		s.app.Invalidate("main")
		return false
	}
	s.requestReplace("app.close", workbench.ReplaceParams{})
	return false
}
