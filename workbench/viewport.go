package workbench

import "fmt"

// ViewportParams reports a client's drawable schematic area, not window size.
type ViewportParams struct {
	Width  float64 `json:"width_px"`
	Height float64 `json:"height_px"`
}

// ViewportState is session-only. Saved projects retain View, not this client size.
type ViewportState struct {
	Width      float64  `json:"width_px"`
	Height     float64  `json:"height_px"`
	Registered bool     `json:"registered"`
	Following  bool     `json:"following"`
	FitFailure *Failure `json:"fit_failure,omitempty"`
}

func cloneViewport(v ViewportState) ViewportState {
	if v.FitFailure != nil {
		f := *v.FitFailure
		v.FitFailure = &f
	}
	return v
}
func validViewportSize(width, height float64) bool {
	return finite(width) && finite(height) && width >= 64 && height >= 64 && width <= 100000 && height <= 100000
}
func (s *Service) stopViewportFollowingLocked() {
	s.viewport.Following = false
	s.viewport.FitFailure = nil
}
func (s *Service) fitViewportLocked() {
	s.viewport.FitFailure = nil
	if !s.viewport.Registered || !s.viewport.Following {
		return
	}
	view, err := s.fitViewLocked(FitParams{Width: s.viewport.Width, Height: s.viewport.Height})
	if err != nil {
		s.viewport.FitFailure = &Failure{Code: "view-limit", Message: err.Error()}
		return
	}
	s.view = view
}
func (s *Service) setViewportLocked(p ViewportParams) error {
	if !validViewportSize(p.Width, p.Height) {
		return fmt.Errorf("viewport dimensions must be 64–100000 pixels")
	}
	if s.viewport.Registered && s.viewport.Width == p.Width && s.viewport.Height == p.Height {
		return nil
	}
	s.viewport.Width, s.viewport.Height, s.viewport.Registered = p.Width, p.Height, true
	s.fitViewportLocked()
	return nil
}
