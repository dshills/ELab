package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"cogentcore.org/core/core"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/canvas"
)

type viewportReport struct {
	width, height        float64
	revision, generation uint64
	widget               *core.WidgetBase
}

// Prepare observes actual Cogent allocation. View construction also retries the
// last measured size after a stale report, even if mesh preparation is cached.
// Publication always runs after the owner callback, never during current paint.
func (s *screen) observeSchematicViewport(v canvas.Viewport, state workbench.Snapshot) {
	if s.drawing == nil || s.drawing.widget == nil || state.Closed || sourcePaneHidden(state) || s.fileBusy || s.pendingCommand != "" {
		return
	}
	if math.IsNaN(v.Width) || math.IsNaN(v.Height) || math.IsInf(v.Width, 0) || math.IsInf(v.Height, 0) || v.Width < 64 || v.Height < 64 || v.Width > 100000 || v.Height > 100000 {
		return
	}
	if s.viewportPending == nil && state.Viewport.Registered && state.Viewport.Width == v.Width && state.Viewport.Height == v.Height {
		return
	}
	plan := &viewportReport{v.Width, v.Height, state.CommandRevision, state.Document.Generation, s.drawing.widget}
	if s.viewportPending != nil {
		s.viewportPending = plan
		return
	}
	s.viewportPending = plan
	publish := func() {
		plan := s.viewportPending
		s.viewportPending = nil
		if plan == nil {
			return
		}
		current := s.service.Snapshot()
		if current.Closed || sourcePaneHidden(current) || s.fileBusy || s.pendingCommand != "" {
			return
		}
		if s.drawing.widget != plan.widget || current.CommandRevision != plan.revision || current.Document.Generation != plan.generation {
			s.app.Invalidate("main")
			return
		}
		if current.Viewport.Registered && current.Viewport.Width == plan.width && current.Viewport.Height == plan.height {
			return
		}
		s.viewportSerial++
		id := fmt.Sprintf("native-viewport-%d", s.viewportSerial)
		raw, _ := json.Marshal(workbench.ViewportParams{Width: plan.width, Height: plan.height})
		// A racing agent command refuses this report. A later view/frame retries;
		// observation never replaces the user's error or file-completion messages.
		result := s.service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: id, Command: "view.viewport.set", Params: raw, ExpectedCommandRevision: &plan.revision, ExpectedDocumentGeneration: &plan.generation, Key: id}, "user")
		if accepted, ok := result.Result.(workbench.Snapshot); result.Error == nil && ok && s.requestHelp != nil && s.requestHelpRevision == plan.revision {
			// Showing a refusal can itself resize the canvas. Continue only
			// that refusal's context through this accepted layout observation;
			// a later agent/user command still invalidates it in view().
			s.requestHelpRevision = accepted.CommandRevision
		}
		s.app.Invalidate("main")
	}
	if err := s.app.Post(publish); err != nil {
		if errors.Is(err, app.ErrQueueFull) {
			// One waiting delivery per canvas; shutdown cancels admission waiting.
			// PostWait admission is not callback completion.
			go func() { _ = s.app.PostWait(s.app.Context(), publish) }()
		} else {
			s.viewportPending = nil
		}
	}
}
