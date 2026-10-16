package workbench

import (
	"fmt"
	"testing"
)

func TestRotatedInductorRoutesClearCaptionMasks(t *testing.T) {
	for _, rotation := range []int{0, 90, 180, 270} {
		t.Run(fmt.Sprint(rotation), func(t *testing.T) {
			g := RLExampleGraph()
			for i := range g.Components {
				if g.Components[i].ID == "L1" {
					g.Components[i].Rotation = rotation
				}
			}
			assertLabelMasksLeaveWiresContinuous(t, g, []string{"R1", "L1"})
		})
	}
}
