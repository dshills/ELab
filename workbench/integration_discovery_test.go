package workbench

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// Discovery must describe the method used by fresh results, rather than the
// earlier backward-Euler engines that remain readable in historical captures.
func TestIntegrationDiscoveryMatchesLiveEngineFamilies(t *testing.T) {
	for _, fixture := range []struct {
		name, engine, method string
		graph                Graph
	}{
		{"mixed-capacitors", MixedEngineID, "SDIRK2", FlasherExampleGraph()},
		{"network-capacitors", NetworkEngineID, "SDIRK2", highPassRCGraph()},
		{"waveform-capacitors", WaveformEngineID, "SDIRK2", waveformRCGraph(t, "finite-pulse-source")},
		{"single-rc", RCEngineID, "exact", RCExampleGraph()},
		{"npn-phases", NPNTransientEngineID, "algebraic", NPNPulseExampleGraph()},
		{"inductors", InductorEngineID, "Radau IIA", RLExampleGraph()},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			s := newGraphService(SolveLED, fixture.graph)
			defer s.Close()
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			before := settled(t, s)
			if before.Transient == nil || before.Transient.Engine != fixture.engine {
				t.Fatal("fixture does not exercise the intended live family", before.Analysis, before.Transient)
			}
			for _, command := range []string{"component.place", "component.set"} {
				detail := sweepDetailSchema(t, command)
				numerics := detail["x-network-numerics"].(map[string]any)
				integration := numerics["time_integration"].(string)
				if !strings.Contains(integration, before.Transient.Engine) || !strings.Contains(integration, fixture.method) {
					t.Error(command, "does not describe the current live integration method", before.Transient.Engine, integration)
				}
				if strings.Contains(integration, "retain backward Euler") || !strings.Contains(integration, "historical") {
					t.Error(command, "confuses fresh results with readable historical captures", integration)
				}
			}
			if !reflect.DeepEqual(before, s.Snapshot()) {
				t.Fatal("detailed discovery changes current circuit or live state")
			}
		})
	}
}

func TestIntegrationDiscoveryStaysOutOfOrdinaryCatalog(t *testing.T) {
	ordinary, err := json.Marshal(Catalog())
	if err != nil {
		t.Fatal(err)
	}
	if len(ordinary) > 64*1024 || strings.Contains(string(ordinary), "x-network-numerics") {
		t.Fatal("detailed numerical guidance expands the ordinary catalog")
	}
}

func TestIntegrationDiscoveryPublishesCPUAndWallPolicy(t *testing.T) {
	numerics := sweepDetailSchema(t, "component.set")["x-network-numerics"].(map[string]any)
	budget := numerics["integration_work_budget"].(map[string]any)
	if budget["attempts_per_sample"] != float64(512) || budget["cpu_ms_per_sample"] != float64(100) || budget["wall_ms_per_sample"] != float64(5000) || budget["time_basis"] != integrationClockBasis {
		t.Fatal("public numerical contract omits actual CPU/wall safeguards", budget)
	}
}
