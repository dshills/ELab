package workbench

import "strconv"

// OpAmpFollowerExampleGraph uses explicit split supplies and negative feedback.
// The input is a slow bipolar sine; the model has no intrinsic bandwidth.
func OpAmpFollowerExampleGraph() Graph { return opAmpExampleGraph(false) }

// OpAmpInvertingExampleGraph uses 10 kOhm input /20 kOhm feedback resistors.
func OpAmpInvertingExampleGraph() Graph { return opAmpExampleGraph(true) }

func opAmpExampleGraph(inverting bool) Graph {
	g := EmptyGraph()
	add := func(id, kind, value string, point Point, rotation int) {
		p, err := MakeComponent(id, kind, point, value)
		if err != nil {
			panic(err) // Fixed curated template, never caller-provided values.
		}
		p.Rotation = rotation
		g.Components = append(g.Components, p)
	}
	inputY, inputX, amplitude := -20., -220., 1.
	if inverting {
		inputY, inputX, amplitude = 20, -320, .5
	}
	add("VIN", "sine-source", "1 V", Point{inputX, inputY}, 90)
	g.Components[0].Sine.Offset = 0
	g.Components[0].Sine.Amplitude = amplitude
	g.Components[0].Sine.Frequency = 10
	g.Components[0].Value = amplitude
	add("GNDIN", "ground", "", Point{inputX, inputY + 100}, 0)
	add("U1", "opamp", "", Point{}, 0)
	add("RL", "resistor", "10k", Point{260, 60}, 90)
	add("GNDL", "ground", "", Point{260, 150}, 0)
	add("JF", "junction", "", Point{-80, 20}, 0)
	add("JB", "junction", "", Point{-80, 190}, 0)
	add("JO", "junction", "", Point{180, 0}, 0)
	add("JR", "junction", "", Point{180, 190}, 0)
	add("LP", "net-label", "VCC", Point{0, -95}, 0)
	add("LN", "net-label", "VEE", Point{0, 70}, 0)
	add("VP", "dc-source", "5 V", Point{610, -70}, 0)
	add("VN", "dc-source", "5 V", Point{610, 70}, 0)
	add("JM", "junction", "", Point{610, 0}, 0)
	add("GNDP", "ground", "", Point{670, 0}, 0)
	add("LPS", "net-label", "VCC", Point{610, -140}, 0)
	add("LNS", "net-label", "VEE", Point{610, 140}, 0)
	if inverting {
		add("RI", "resistor", "10k", Point{-190, 20}, 0)
		add("RF", "resistor", "20k", Point{80, 190}, 0)
		add("GNDPLUS", "ground", "", Point{-200, -80}, 90)
	}
	connect := func(one, two string) {
		g.Wires = append(g.Wires, Wire{ID: "W" + strconv.Itoa(len(g.Wires)+1), From: one, To: two, Connected: true})
	}
	for _, ends := range [][2]string{
		{"VIN.negative", "GNDIN.reference"}, {"U1.out", "JO.node"},
		{"JO.node", "RL.1"}, {"RL.2", "GNDL.reference"},
		{"JO.node", "JR.node"}, {"JF.node", "U1.inverting"}, {"JF.node", "JB.node"},
		{"U1.vplus", "LP.node"}, {"U1.vminus", "LN.node"},
		{"VP.positive", "LPS.node"}, {"VP.negative", "JM.node"},
		{"VN.positive", "JM.node"}, {"JM.node", "GNDP.reference"}, {"VN.negative", "LNS.node"},
	} {
		connect(ends[0], ends[1])
	}
	if inverting {
		connect("VIN.positive", "RI.1")
		connect("RI.2", "JF.node")
		connect("U1.noninverting", "GNDPLUS.reference")
		connect("JB.node", "RF.1")
		connect("RF.2", "JR.node")
	} else {
		connect("VIN.positive", "U1.noninverting")
		connect("JB.node", "JR.node")
	}
	return g
}
