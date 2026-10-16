package workbench

import (
	"context"
	"math"
	"testing"
)

func TestPulseCornersRetainBothSidesAcrossKernels(t *testing.T) {
	for name, graph := range map[string]Graph{"rc": RCExampleGraph(), "npn": NPNPulseExampleGraph(), "mixed": mixedBaseCapGraph()} {
		t.Run(name, func(t *testing.T) {
			compiled, _, diagnostic := graph.Compile()
			if diagnostic != nil {
				t.Fatal(diagnostic)
			}
			kernel, err := newSimulationKernel(context.Background(), kernelPlan{compiled: compiled, settings: DefaultTransientSettings(), mode: "auto"})
			if err != nil {
				t.Fatal(err)
			}
			for i := 1; i <= 150; i++ {
				if err = kernel.advance(context.Background(), float64(i)*.001); err != nil {
					t.Fatal(err)
				}
			}
			left, right := -1., -1.
			voltage := 0.
			inputCurrent := Probe{Unit: "A"}
			for _, part := range graph.Components {
				if part.Kind == "pulse-source" {
					inputCurrent.ID = "i:" + part.ID
				}
			}
			for _, p := range kernel.trace.Samples {
				if math.Abs(p.Time-.15) > 1e-15 {
					continue
				}
				// In the mixed fixture the base capacitor holds VBE across the
				// edge, so LED/collector current is continuous. The input branch
				// current must jump; tiny collector roundoff is not edge evidence.
				current, ok := frameValue(p, inputCurrent)
				if !ok {
					t.Fatal("pulse-side input current is absent", inputCurrent.ID)
				}
				if p.Input == 5 {
					left = current
					voltage = p.CapacitorVoltage
					if name == "rc" {
						voltage = p.Voltage
					}
				}
				if p.Input == 0 {
					right = current
					if left >= 0 && name != "npn" {
						v := p.CapacitorVoltage
						if name == "rc" {
							v = p.Voltage
						}
						if v != voltage {
							t.Fatal("capacitor jumped across rounded source corner")
						}
					}
				}
			}
			if left < 0 || right == -1 || math.Abs(left-right) < 1e-8 {
				t.Fatalf("150ms edge omitted a side: left %g right %g", left, right)
			}
			if name == "mixed" {
				rb, _ := graph.Component("RB")
				if math.Abs(left-right-5/rb.Value) > 1e-10 {
					t.Fatal("held base voltage violates the input resistor's edge law", left, right)
				}
			}
		})
	}
}
func TestPulseClassificationDoesNotSwitchBeforeTheBoundary(t *testing.T) {
	cfg := DefaultTransientSettings()
	for _, edge := range []float64{.05, .1, .15, .3, .35, .4} {
		before := pulseHigh(cfg, edge-1e-12)
		at := pulseHigh(cfg, edge)
		after := pulseHigh(cfg, edge+1e-12)
		if before == at || at != after {
			t.Fatalf("input switched early/late around %g: %v/%v/%v", edge, before, at, after)
		}
		if nextPulseEdge(cfg, edge-1e-12) <= edge-1e-12 {
			t.Fatal("next edge did not advance")
		}
	}
}
