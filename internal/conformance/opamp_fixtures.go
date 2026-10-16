package conformance

import "github.com/dshills/elab/workbench"

func opAmpCaseProbes(name string) []string {
	if name == "opamp-rc-npn" {
		return []string{"v:C1", "v:U1", "i:U1", "i:D1", "v:Q1.base"}
	}
	if name == "opamp-rc-mosfet" {
		return []string{"v:C1", "v:U1", "v:U2", "i:D1"}
	}
	return []string{"v:VIN.positive", "v:U1", "i:U1", "v:U1.input-difference"}
}

func prepareOpAmpCase(s *session, name string) error {
	id := name
	if name == "opamp-rc-npn" || name == "opamp-rc-mosfet" {
		id = "opamp-follower"
	}
	if err := s.ask("example.load", workbench.ExampleLoadParams{ID: id}, true, nil); err != nil {
		return err
	}
	if err := s.ask("simulation.pause", nil, true, nil); err != nil {
		return err
	}
	if id == name {
		return nil
	}
	if err := s.ask("object.delete", workbench.IDParams{ID: "VIN"}, true, nil); err != nil {
		return err
	}
	parts := []part{{"VRC", "pulse-source", "5 V", -400, -200}, {"RI", "resistor", "10k", -200, -200}, {"C1", "capacitor", "1uF", -100, -200}, {"RD", "resistor", "1k", 700, -100}, {"D1", "led", "", 700, 100}}
	wires := [][2]string{{"VRC.negative", "GNDIN.reference"}, {"VRC.positive", "RI.1"}, {"RI.2", "C1.1"}, {"C1.1", "U1.noninverting"}, {"C1.2", "GNDIN.reference"}, {"VP.positive", "RD.1"}, {"RD.2", "D1.anode"}}
	if name == "opamp-rc-npn" {
		parts = append(parts, part{"RB", "resistor", "10k", 500, 220}, part{"Q1", "npn", "", 700, 220})
		wires = append(wires, [2]string{"U1.out", "RB.1"}, [2]string{"RB.2", "Q1.base"}, [2]string{"D1.cathode", "Q1.collector"}, [2]string{"Q1.emitter", "GNDL.reference"})
	} else {
		parts = append(parts, part{"U2", "opamp", "", 500, 220}, part{"M1", "nmos", "", 700, 220}, part{"RG", "resistor", "100k", 500, 400})
		wires = append(wires, [2]string{"U1.out", "U2.noninverting"}, [2]string{"U2.out", "U2.inverting"}, [2]string{"U2.vplus", "VP.positive"}, [2]string{"U2.vminus", "VN.negative"}, [2]string{"U2.out", "M1.gate"}, [2]string{"U2.out", "RG.1"}, [2]string{"RG.2", "GNDL.reference"}, [2]string{"D1.cathode", "M1.drain"}, [2]string{"M1.source", "GNDL.reference"})
	}
	return buildElectrical(s, circuit{parts: parts, wires: wires})
}
