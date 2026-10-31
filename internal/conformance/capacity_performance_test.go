package conformance

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/internal/testutil"
	"github.com/dshills/elab/workbench"
)

func TestCapacityPerformanceMeasuresFullSizeChangedCaptures(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	factory := ordinaryAgentFactory(t)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	for _, n := range []int{0, 19, 101} {
		if _, err := RunCapacityPerformance(ctx, nil, t.TempDir(), n); err == nil {
			t.Fatal("invalid repetition count accepted")
		}
	}
	for _, fixture := range []struct {
		name, id     string
		low, high    float64
		parts, nodes int
	}{
		{"capacity-100-nodes", "P1", 10000, 20000, 51, 100},
		{"capacity-64-parts", "R1", 1000, 2000, 64, 64},
		{"capacity-rc-mosfet", "RL", 1000, 2000, 64, 60},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			err := electricalSession(ctx, factory, t.TempDir(), func(s *session) error {
				if err := preparePerformance(s, fixture.name); err != nil {
					return err
				}
				before, err := s.state("simulation.wait", nil, false)
				if err != nil {
					return err
				}
				if !capacityPerformanceFresh(before, fixture.name) {
					t.Fatal("prepared full-size state is not fresh")
				}
				raw, err := json.Marshal(before)
				if err != nil {
					t.Fatal(err)
				}
				for _, mutation := range []string{"stale", "wrong-run", "wrong-revision", "wrong-engine", "missing-node", "wrong-parts", "missing-trace", "missing-model"} {
					var changed workbench.Snapshot
					if err := json.Unmarshal(raw, &changed); err != nil {
						t.Fatal(err)
					}
					switch mutation {
					case "stale":
						changed.Mixed.Stale = true
					case "wrong-run":
						changed.Mixed.RunID++
					case "wrong-revision":
						changed.Transient.ElectricalRevision++
					case "wrong-engine":
						changed.Mixed.Engine = "foreign"
					case "missing-node":
						for id := range changed.Mixed.Nodes {
							delete(changed.Mixed.Nodes, id)
							break
						}
					case "wrong-parts":
						changed.SimulatedComponents--
					case "missing-trace":
						changed.Scope.Selected = changed.Scope.Selected[:3]
					case "missing-model":
						if fixture.name == "capacity-rc-mosfet" {
							changed.NMOSModel = nil
						} else if fixture.name == "capacity-100-nodes" {
							changed.PotentiometerModel = nil
						} else {
							changed.Mixed.Resistors = nil
						}
					}
					if capacityPerformanceFresh(changed, fixture.name) {
						t.Fatal("invalid live timing state accepted", mutation)
					}
				}
				previous := ""
				var run, revision uint64
				for i := 0; i < 3; i++ {
					sample, err := measurePerformance(s, fixture.name, i, "test")
					if err != nil {
						return err
					}
					if sample.GraphSHA256 == previous || sample.RunID <= run || sample.ElectricalRevision <= revision {
						t.Fatal("timed no-op or stale identity")
					}
					previous, run, revision = sample.GraphSHA256, sample.RunID, sample.ElectricalRevision
					captured := sample.CapacityVerification
					if captured == nil || len(captured.Rows) != 205 || !reflect.DeepEqual(sample.ProbeIDs, capacityProbes(fixture.name)) || sample.VerifyDuration != .2 || sample.VerifySampleStep != .001 {
						t.Fatal("complete four-channel capture missing")
					}
					graph := captured.Metadata.Graph
					c, _, diagnostic := graph.Compile()
					p, ok := graph.Component(fixture.id)
					want := fixture.high
					if i%2 == 1 {
						want = fixture.low
					}
					if !ok || p.Value != want || diagnostic != nil || workbench.SimulatedComponentCount(graph) != fixture.parts || len(c.Nets) != fixture.nodes {
						t.Fatal("size or real edit differs", p.Value, want)
					}
					if fixture.name == "capacity-rc-mosfet" {
						current := captured.Rows[50].Values[3]
						if i%2 == 0 && (current < .001 || current > .002) || i%2 == 1 && (current < .002 || current > .003) {
							t.Fatal("changed load did not change LED current", current)
						}
					}
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
