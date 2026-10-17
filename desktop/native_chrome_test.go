package main

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
)

// Managed Scroll now embeds Frame so it can publish partial-paint AX geometry.
// Styling must reach that embedded frame and its existing scrollbar controls.
type chromeScrollFrame struct{ core.Frame }

func TestChromeStylesEmbeddedScrollFrames(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	scene := core.NewScene("embedded-scroll")
	frame := tree.New[chromeScrollFrame](scene)
	bar := core.NewSlider(frame)
	frame.Scrolls[0] = bar
	if !s.styleNativeChrome(scene) {
		t.Fatal("mounted chrome did not install styles")
	}
	for _, widget := range []*core.WidgetBase{frame.AsWidget(), bar.AsWidget()} {
		if styled, _ := widget.Property("elab.chrome.styled").(bool); !styled {
			t.Fatal("embedded scroll or its scrollbar missed native chrome styling")
		}
	}
	if s.styleNativeChrome(scene) {
		t.Fatal("unchanged scroll repeatedly installed styles")
	}
}
