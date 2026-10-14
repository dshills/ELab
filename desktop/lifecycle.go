package main

import "github.com/dshills/elab/workbench"

// requestClose runs on the owner for both native window policy and menu/keys.
// The service owns clean/dirty admission; the main loop tears down the window
// only after the shared close command succeeds. Never replace an active choice.
func (s *screen) requestClose() bool {
	if s.fileBusy || s.pendingCommand != "" {
		s.app.Invalidate("main")
		return false
	}
	s.requestReplace("app.close", workbench.ReplaceParams{})
	return false
}
