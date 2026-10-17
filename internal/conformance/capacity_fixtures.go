package conformance

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/dshills/elab/workbench"
)

var capacityCaseNames = []string{"capacity-100-nodes", "capacity-64-parts", "capacity-rc-mosfet"}

// Captured from public blank-sheet construction, not a private solver fixture.
//
//go:embed testdata/capacity-rc-mosfet.json
var capacityRCMOSFET []byte

func capacityFixture(name string) (workbench.Graph, workbench.TransientSettings, error) {
	settings := workbench.DefaultTransientSettings()
	settings.HighSeconds, settings.LowSeconds = .1, .1
	if name == "capacity-rc-mosfet" {
		var fixture struct {
			Graph    workbench.Graph             `json:"graph"`
			Settings workbench.TransientSettings `json:"settings"`
		}
		if err := json.Unmarshal(capacityRCMOSFET, &fixture); err != nil {
			return workbench.Graph{}, settings, err
		}
		return fixture.Graph, fixture.Settings, nil
	}
	if name != "capacity-100-nodes" && name != "capacity-64-parts" {
		return workbench.Graph{}, settings, fmt.Errorf("unknown capacity fixture %q", name)
	}
	g := workbench.EmptyGraph()
	add := func(id, kind, value string, x, y float64) error {
		p, err := workbench.MakeComponent(id, kind, workbench.Point{X: x, Y: y}, value)
		g.Components = append(g.Components, p)
		return err
	}
	connect := func(one, two string) {
		g.Wires = append(g.Wires, workbench.Wire{ID: fmt.Sprintf("W%d", len(g.Wires)+1), From: one, To: two, Connected: true})
	}
	if err := add("V1", "pulse-source", "5 V", -300, -200); err != nil {
		return g, settings, err
	}
	if err := add("GND", "ground", "", -300, 1600); err != nil {
		return g, settings, err
	}
	count, kind, prefix, value := 63, "resistor", "R", "1k"
	if name == "capacity-100-nodes" {
		count, kind, prefix, value = 50, "potentiometer", "P", "10k"
	}
	previous := "V1.positive"
	for i := 1; i <= count; i++ {
		id := fmt.Sprintf("%s%d", prefix, i)
		if err := add(id, kind, value, float64((i-1)%8)*180, float64((i-1)/8)*160); err != nil {
			return g, settings, err
		}
		connect(previous, id+".1")
		previous = id + ".2"
	}
	if name == "capacity-100-nodes" {
		connect("P50.wiper", "P50.1")
	}
	connect(previous, "GND.reference")
	connect("V1.negative", "GND.reference")
	return g, settings, nil
}

func capacityProbes(name string) []string {
	switch name {
	case "capacity-100-nodes":
		return []string{"v:V1", "v:P1.wiper", "v:P25.wiper", "i:P49.1"}
	case "capacity-64-parts":
		return []string{"v:V1", "v:R32.1", "v:R63.1", "i:R63"}
	default:
		return []string{"v:C1", "v:C2", "v:C4", "i:D1"}
	}
}

func capacitySize(name string) (parts, nodes int) {
	switch name {
	case "capacity-100-nodes":
		return 51, 100
	case "capacity-64-parts":
		return 64, 64
	default:
		return 64, 60
	}
}

func prepareCapacity(s *session, name string) (workbench.Snapshot, error) {
	graph, settings, err := capacityFixture(name)
	if err != nil {
		return workbench.Snapshot{}, err
	}
	fixture := circuit{}
	for _, p := range graph.Components {
		value := ""
		if p.Unit != "" {
			value = strconv.FormatFloat(p.Value, 'g', -1, 64) + " " + p.Unit
		}
		fixture.parts = append(fixture.parts, part{p.ID, p.Kind, value, p.Position.X, p.Position.Y})
	}
	for _, w := range graph.Wires {
		fixture.wires = append(fixture.wires, [2]string{w.From, w.To})
	}
	if err := buildElectrical(s, fixture); err != nil {
		return workbench.Snapshot{}, err
	}
	if err := s.ask("simulation.transient.set", settings, true, nil); err != nil {
		return workbench.Snapshot{}, err
	}
	if err := s.ask("scope.probes.set", workbench.ProbeParams{IDs: capacityProbes(name)}, true, nil); err != nil {
		return workbench.Snapshot{}, err
	}
	return restart(s)
}
