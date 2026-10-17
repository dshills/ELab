package conformance

import (
	"fmt"

	"github.com/dshills/elab/workbench"
)

var opAmpStorageCaseNames = []string{"opamp-buffered-rlc", "opamp-rc-pmos"}

func opAmpStorageProbes(name string) []string {
	if name == "opamp-buffered-rlc" {
		return []string{"v:C1", "i:L1", "v:U1", "i:U1"}
	}
	return []string{"v:C1", "v:U1", "i:D1", "i:MP1.drain"}
}

func prepareOpAmpStorage(s *session, name string) error {
	if err := s.ask("example.load", workbench.ExampleLoadParams{ID: "opamp-follower"}, true, nil); err != nil {
		return err
	}
	if err := s.ask("simulation.pause", nil, true, nil); err != nil {
		return err
	}
	if err := s.ask("object.delete", workbench.IDParams{ID: "VIN"}, true, nil); err != nil {
		return err
	}
	var fixture circuit
	switch name {
	case "opamp-buffered-rlc":
		fixture = circuit{parts: []part{{"VB", "dc-source", ".2 V", -400, -200}, {"RS", "resistor", "1k", 500, -100}, {"L1", "inductor", "100mH", 700, -100}, {"C1", "capacitor", "1uF", 900, 100}, {"RFILT", "resistor", "1k", 1100, 100}}, wires: [][2]string{{"VB.negative", "GNDIN.reference"}, {"VB.positive", "U1.noninverting"}, {"U1.out", "RS.1"}, {"RS.2", "L1.1"}, {"L1.2", "C1.1"}, {"C1.1", "RFILT.1"}, {"RFILT.2", "GNDL.reference"}, {"C1.2", "GNDL.reference"}}}
	case "opamp-rc-pmos":
		fixture = circuit{parts: []part{{"VRC", "pulse-source", "5 V", -400, -200}, {"RI", "resistor", "10k", -200, -200}, {"C1", "capacitor", "1uF", -100, -200}, {"MP1", "pmos", "", 700, -200}, {"RG", "resistor", "100k", 500, 100}, {"RD", "resistor", "1k", 900, -100}, {"D1", "led", "", 900, 100}}, wires: [][2]string{{"VRC.negative", "GNDIN.reference"}, {"VRC.positive", "RI.1"}, {"RI.2", "C1.1"}, {"C1.1", "U1.noninverting"}, {"C1.2", "GNDIN.reference"}, {"U1.out", "MP1.gate"}, {"MP1.gate", "RG.1"}, {"RG.2", "GNDL.reference"}, {"MP1.source", "VP.positive"}, {"MP1.drain", "RD.1"}, {"RD.2", "D1.anode"}, {"D1.cathode", "GNDL.reference"}}}
	default:
		return fmt.Errorf("unknown op-amp storage fixture %q", name)
	}
	if err := buildElectrical(s, fixture); err != nil {
		return err
	}
	if name == "opamp-buffered-rlc" {
		if err := s.ask("capacitor.initial.set", workbench.ValueParams{ID: "C1", Value: ".05 V"}, true, nil); err != nil {
			return err
		}
		if err := s.ask("inductor.initial.set", workbench.ValueParams{ID: "L1", Value: "-50 uA"}, true, nil); err != nil {
			return err
		}
	}
	return s.ask("scope.probes.set", workbench.ProbeParams{IDs: opAmpStorageProbes(name)}, true, nil)
}
