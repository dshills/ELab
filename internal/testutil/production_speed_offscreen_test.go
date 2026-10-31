//go:build offscreen

package testutil

import "testing"

// The child cannot run this named test unless it preserves the parent's tag.
func TestProductionSpeedRetainsOffscreenTag(t *testing.T) {
	if RunAtProductionSpeed(t) {
		return
	}
}
