package workbench

import (
	"math"
	"testing"
)

func TestOpAmpLEDResolvesOffTransistorLeakageWithoutChangingLegacyPath(t *testing.T) {
	network := &mixedNetwork{opamps: []Component{{Kind: "opamp"}}}
	legacy := &mixedNetwork{}
	for _, current := range []float64{1e-22, 1e-20, 1e-16, 1e-14, 1e-9, .003} {
		voltage := ledDrop(current)
		got, slope := network.ledCurrent(voltage)
		if math.Abs(got-current) > 1e-12*current || !finite(slope) || slope <= 0 {
			t.Fatal("powered circuit cannot resolve declared LED leakage law", current, got, slope)
		}
		one, two := legacy.ledCurrent(voltage)
		want, derivative := diodeCurrent(voltage)
		if one != want || two != derivative {
			t.Fatal("legacy no-op-amp LED arithmetic changed")
		}
	}
}
