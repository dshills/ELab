package main

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
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

func TestSidePanelHostsDoNotCreateNestedScrollbars(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	scene := core.NewScene("owned-side-panels")
	parts := core.NewSplits(scene)
	parts.SetName("parts-split")
	left, center := core.NewFrame(parts), core.NewFrame(parts)
	scroll := core.NewFrame(left)
	scroll.Styler(func(style *styles.Style) { style.Overflow.Set(styles.OverflowAuto) })
	inspector := core.NewSplits(center)
	inspector.SetName("inspector-split")
	sheet, right := core.NewFrame(inspector), core.NewFrame(inspector)
	navigation := core.NewFrame(scene)
	navigation.SetName("simulation-tool-navigation")
	rail := core.NewFrame(scene)
	rail.SetName("inspector-rail")
	owned := []*core.Frame{left, right, navigation, rail}
	for _, name := range []string{"drawer-controls", "parts-drawer-header", "inspector-drawer-header"} {
		f := core.NewFrame(scene)
		f.SetName(name)
		owned = append(owned, f)
	}
	s.styleNativeChrome(scene)
	scene.StyleTree()
	for _, w := range owned {
		if w.Styles.Overflow.X != styles.OverflowHidden || w.Styles.Overflow.Y != styles.OverflowHidden {
			t.Fatal("side host retained a second scroller")
		}
	}
	for _, w := range []*core.Frame{center, sheet, scroll} {
		if w.Styles.Overflow.X != styles.OverflowAuto || w.Styles.Overflow.Y != styles.OverflowAuto {
			t.Fatal("adapter changed a real scroll or center pane")
		}
	}
	if s.styleNativeChrome(scene) {
		t.Fatal("unchanged native hosts reinstalled styles")
	}
}
