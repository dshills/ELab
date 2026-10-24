package main

import (
	"fmt"
	"image"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/abilities"
)

func TestScopeScrollForwardsOnlyToItsNamedHostAndRestoresAbilities(t *testing.T) {
	for _, initiallyEnabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("already-unattended=%t", initiallyEnabled), func(t *testing.T) {
			body := core.NewBody("scope-wheel-routing")
			outer := core.NewFrame(body)
			outer.SetName("unrelated-pane")
			host := core.NewFrame(outer)
			host.SetName("scope-scroll")
			plot := core.NewFrame(host)
			plot.SetName("scope-plot")
			host.Styles.SetAbilities(initiallyEnabled, abilities.ScrollableUnattended)
			original := host.Styles.Abilities
			outerCalls, hostCalls := 0, 0
			outer.OnFirst(events.Scroll, func(events.Event) { outerCalls++ })
			event := events.NewScroll(image.Pt(15, 20), math32.Vec2(0, 12), 0)
			host.OnFirst(events.Scroll, func(received events.Event) {
				hostCalls++
				if received != event || !host.AbilityIs(abilities.ScrollableUnattended) {
					t.Fatal("host did not receive the original packet with temporary scroll eligibility")
				}
				received.SetHandled()
			})
			forwardScopeScroll(plot.AsWidget(), event)
			if hostCalls != 1 || outerCalls != 0 || !event.IsHandled() {
				t.Fatal("scope wheel escaped its intended pane", hostCalls, outerCalls, event.IsHandled())
			}
			if host.Styles.Abilities != original {
				t.Fatal("temporary scroll eligibility leaked into the host")
			}
		})
	}
}

func TestScopeScrollWithoutDockSuppressesCanvasZoomAndOtherPanes(t *testing.T) {
	body := core.NewBody("scope-wheel-without-dock")
	other := core.NewFrame(body)
	other.SetName("unrelated-pane")
	plot := core.NewFrame(other)
	plot.SetName("scope-plot")
	original := other.Styles.Abilities
	calls := 0
	other.OnFirst(events.Scroll, func(events.Event) { calls++ })
	event := events.NewScroll(image.Point{}, math32.Vec2(0, -12), 0)
	forwardScopeScroll(plot.AsWidget(), event)
	if calls != 0 || !event.IsHandled() || other.Styles.Abilities != original {
		t.Fatal("an undocked scope wheel reached a different pane or remained available to mesh zoom")
	}
}
