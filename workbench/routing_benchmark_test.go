package workbench

import (
	"crypto/sha256"
	"encoding/json"
	"os"
	"testing"
)

// Measure presentation routing alone, without a solver or UI renderer. Run in
// its own benchmark process: the cold case clears that process's route cache.
// These elapsed means do not measure command acknowledgment or visible latency.
func BenchmarkCapacityWireRoutes(b *testing.B) {
	data, err := os.ReadFile("../internal/conformance/testdata/capacity-rc-mosfet.json")
	if err != nil {
		b.Fatal(err)
	}
	var fixture struct{ Graph Graph }
	if err := json.Unmarshal(data, &fixture); err != nil {
		b.Fatal(err)
	}
	g := fixture.Graph
	if len(g.Components) == 0 || len(g.Wires) == 0 {
		b.Fatal("empty capacity routing fixture")
	}
	b.Logf("public capacity fixture: %d drawing objects, %d wires, SHA-256 %x", len(g.Components), len(g.Wires), sha256.Sum256(data))
	for _, name := range []string{"cold", "warm"} {
		b.Run(name, func(b *testing.B) {
			for _, wire := range g.Wires {
				WirePath(g, wire)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if name == "cold" {
					b.StopTimer()
					routeCache.Lock()
					routeCache.paths, routeCache.order = nil, nil
					routeCache.Unlock()
					b.StartTimer()
				}
				for _, wire := range g.Wires {
					if len(WirePath(g, wire)) < 2 {
						b.Fatal("route lost its endpoints", wire.ID)
					}
				}
			}
			b.ReportMetric(float64(len(g.Wires)), "wires/op")
		})
	}
}
