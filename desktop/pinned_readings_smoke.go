package main

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"math"
	"os"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func nativePinnedReadingsSmoke(ctx context.Context, a *app.Application, s *screen) error {
	ready := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if r := s.invoke("example.rc", workbench.ReplaceParams{DiscardChanges: true}); r.Error != nil {
			ready <- fmt.Errorf("pin example: %s", r.Error.Message)
			return
		}
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{{"simulation.pause", nil}, {"simulation.restart", nil}, {"pin-reading-probe", &ui.ChangeEvent{SelectedID: "v:C1"}}, {"pin-X-v:C1", &ui.ChangeEvent{Number: 350}}, {"pin-Y-v:C1", &ui.ChangeEvent{Number: 250}}, {"pin-reading-probe", &ui.ChangeEvent{SelectedID: "i:C1"}}, {"pin-X-i:C1", &ui.ChangeEvent{Number: 350}}, {"pin-Y-i:C1", &ui.ChangeEvent{Number: 310}}} {
			s.view()
			if err := smokeUI(s, action.id, action.event); err != nil {
				ready <- err
				return
			}
		}
		if r := s.invoke("simulation.step.start", workbench.StepParams{Duration: .01}); r.Error != nil {
			ready <- fmt.Errorf("pin step: %s", r.Error.Message)
			return
		}
		ready <- nil
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	state, err := s.service.Wait(ctx)
	if err != nil || len(state.PinnedReadings) != 2 || state.PinnedReadings[0].Value == nil || math.Abs(*state.PinnedReadings[0].Value-5*(1-math.Exp(-1))) > 1e-12 || state.PinnedReadings[1].Value == nil || *state.PinnedReadings[1].Value <= 0 {
		return fmt.Errorf("native pinned RC readings fail: %v", err)
	}
	if err := a.PostWait(ctx, func() {
		if err := smokeUI(s, "view.fit", nil); err != nil {
			ready <- err
			return
		}
		s.view()
		ready <- s.drawing.RebuildResources()
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		if err := a.PostWait(ctx, func() {
			pixels, err := s.drawing.Snapshot()
			if err != nil {
				ready <- err
				return
			}
			view := *s.drawing.View
			sx, sy := float64(pixels.Bounds().Dx())/view.Width, float64(pixels.Bounds().Dy())/view.Height
			state := s.service.Snapshot()
			cardColor := paletteForTheme(state.Theme).Surface
			for _, reading := range state.PinnedReadings {
				min, max := workbench.PinnedReadingBounds(reading)
				a, b := view.Screen(canvas.Point{X: min.X, Y: min.Y}), view.Screen(canvas.Point{X: max.X, Y: max.Y})
				cards := 0
				textCoverage := 0.
				// Fit now includes source annotations. At smaller zoom glyphs are
				// antialiased, so count equivalent ink coverage inside the card,
				// excluding its border; exact-color pixels are not text coverage.
				textA := view.Screen(canvas.Point{X: min.X + 8, Y: min.Y + 7})
				textB := view.Screen(canvas.Point{X: max.X - 8, Y: max.Y - 7})
				for y := int(math.Max(0, a.Y*sy)); y < int(math.Min(float64(pixels.Bounds().Dy()), b.Y*sy)); y++ {
					for x := int(math.Max(0, a.X*sx)); x < int(math.Min(float64(pixels.Bounds().Dx()), b.X*sx)); x++ {
						p := pixels.NRGBAAt(x, y)
						if p == cardColor {
							cards++
						}
						if p.A > 250 && float64(x) >= textA.X*sx && float64(x) < textB.X*sx && float64(y) >= textA.Y*sy && float64(y) < textB.Y*sy {
							textCoverage += readingTextCoverage(p, cardColor, color.NRGBA{R: 39, G: 110, B: 126, A: 255})
						}
					}
				}
				if cards < 100 || textCoverage < 20 {
					if path := os.Getenv("ELAB_SMOKE_PIN_IMAGE"); path != "" {
						file, saveErr := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
						if saveErr == nil {
							saveErr = png.Encode(file, pixels)
							closeErr := file.Close()
							if saveErr == nil {
								saveErr = closeErr
							}
						}
						if saveErr != nil {
							ready <- saveErr
							return
						}
					}
					ready <- fmt.Errorf("fresh %s reading card/text absent: %d card pixels / %.1f equivalent text pixels", reading.Definition.ID, cards, textCoverage)
					return
				}
			}
			ready <- nil
		}); err != nil {
			return err
		}
		err := <-ready
		if errors.Is(err, cogent.ErrFrameNotReady) {
			continue
		}
		return err
	}
}

// GPU multisample resolution blends in linear light; decoded sRGB pixels are
// not a straight interpolation of byte values. Reject unrelated colors and
// measure glyph coverage rather than only fully covered samples.
func readingTextCoverage(pixel, background, ink color.NRGBA) float64 {
	linear := func(v uint8) float64 {
		c := float64(v) / 255
		if c <= .04045 {
			return c / 12.92
		}
		return math.Pow((c+.055)/1.055, 2.4)
	}
	if pixel.A < 251 || background.R == ink.R {
		return 0
	}
	coverage := (linear(background.R) - linear(pixel.R)) / (linear(background.R) - linear(ink.R))
	if coverage <= 0 || coverage > 1.001 {
		return 0
	}
	for _, channels := range [][3]uint8{{pixel.G, background.G, ink.G}, {pixel.B, background.B, ink.B}} {
		expected := linear(channels[1]) + coverage*(linear(channels[2])-linear(channels[1]))
		if math.Abs(linear(channels[0])-expected) > .01 {
			return 0
		}
	}
	return math.Min(1, coverage)
}
