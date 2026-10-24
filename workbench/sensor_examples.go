package workbench

func SensorExampleGraph(thermostat bool) Graph {
	b := newExampleBuilder()
	b.part("VCC", "dc-source", "5", -850, -300, 0)
	b.ground("GNDV", "VCC.negative", -850, -100)
	b.label("POWER", "VCC", "VCC.positive", -850, -450)
	id, kind, value := "LDR1", "ldr", 100.
	if thermostat {
		id, kind, value = "RT1", "thermistor", 20
	}
	b.part(id, kind, "10k", -600, -100, 90)
	setSensorEnvironment(&b.graph.Components[len(b.graph.Components)-1], value)
	b.label("PSENSOR", "VCC", id+".1", -600, -300)
	b.part("RDIV", "resistor", "10k", -600, 150, 90)
	b.part("JS", "junction", "", -600, 20, 0)
	b.wire(id+".2", "JS.node")
	b.wire("JS.node", "RDIV.1")
	b.ground("GNDS", "RDIV.2", -600, 330)
	b.part("P1", "potentiometer", "10k", -400, 460, 0)
	b.label("PREF", "VCC", "P1.1", -760, 390)
	b.ground("GNDP", "P1.2", -220, 600)
	b.part("U1", "comparator", "200mV", -100, 0, 0)
	b.wire("P1.wiper", "U1.noninverting")
	b.wire("JS.node", "U1.inverting")
	b.label("PC", "VCC", "U1.vcc", -100, -180)
	b.ground("GNDC", "U1.gnd", -100, 170)
	if !thermostat {
		b.part("RL", "resistor", "1k", 180, 0, 0)
		b.wire("U1.out", "RL.1")
		b.part("D1", "led", "", 440, 0, 270)
		b.wire("RL.2", "D1.anode")
		b.ground("GNDD", "D1.cathode", 630, 130)
		return b.graph
	}
	b.part("RB", "resistor", "4.7k", 60, 280, 0)
	b.wire("U1.out", "RB.1")
	b.part("Q1", "npn", "", 280, 280, 0)
	b.wire("RB.2", "Q1.base")
	b.ground("GNDQ", "Q1.emitter", 310, 480)
	b.part("RBE", "resistor", "100k", 120, 450, 90)
	b.wire("Q1.base", "RBE.1")
	b.ground("GNDBE", "RBE.2", 120, 620)
	b.part("K1", "relay", "100mH", 400, -180, 0)
	b.label("PK", "VCC", "K1.coil1", 220, -400)
	b.wire("K1.coil2", "Q1.collector")
	b.part("D2", "diode", "", 110, -180, 180)
	b.wire("K1.coil2", "D2.anode")
	b.wire("K1.coil1", "D2.cathode")
	b.part("VLOAD", "dc-source", "9", 800, -420, 0)
	b.ground("GNDL", "VLOAD.negative", 800, -250)
	b.wire("VLOAD.positive", "K1.common")
	b.part("RL", "resistor", "1k", 810, -130, 0)
	b.wire("K1.no", "RL.1")
	b.part("D1", "led", "", 1060, -130, 270)
	b.wire("RL.2", "D1.anode")
	b.ground("GNDD", "D1.cathode", 1210, 20)
	b.part("RNC", "resistor", "10k", 790, 130, 0)
	b.wire("K1.nc", "RNC.1")
	b.ground("GNDNC", "RNC.2", 1000, 290)
	return b.graph
}
