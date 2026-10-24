package workbench

import (
	"fmt"
	"strings"
)

func ButtonCounterExampleGraph(debounce bool) Graph {
	if debounce {
		return ButtonDebounceExampleGraph()
	}
	base := DisplayCounterExampleGraph()
	b := newExampleBuilder()
	removed := map[string]bool{"U1": true, "RA": true, "RB": true, "C1": true, "CC": true, "JD": true, "JT": true, "GND1": true, "GNDC": true, "GNDU": true, "GNDR": true}
	for _, p := range base.Components {
		if !removed[p.ID] {
			switch p.ID {
			case "V1":
				p.Position = Point{-750, -280}
			case "GNDV":
				p.Position = Point{-750, -100}
			case "JV":
				p.Position = Point{-550, -400}
			case "POWER":
				p.Position = Point{-550, -500}
			}
			b.graph.Components = append(b.graph.Components, p)
		}
	}
	for _, w := range base.Wires {
		from, _, _ := base.Terminal(w.From)
		to, _, _ := base.Terminal(w.To)
		if !removed[from.ID] && !removed[to.ID] {
			b.wire(w.From, w.To)
		}
	}
	kind := "pushbutton"
	if debounce {
		kind = "bouncing-pushbutton"
	}
	b.part("SCOUNT", kind, "", -450, 0, 0)
	b.label("PBPOWER", "VCC", "SCOUNT.1", -620, -70)
	b.wire("SCOUNT.2", "U2.clk")
	b.part("RCLK", "resistor", "10k", -240, 170, 90)
	b.wire("SCOUNT.2", "RCLK.1")
	b.ground("GNDP", "RCLK.2", -240, 340)
	b.part("SRESET", "pushbutton", "", -450, 480, 0)
	b.label("PRPOWER", "VCC", "SRESET.1", -620, 410)
	b.part("RRESET", "resistor", "10k", -140, 580, 90)
	b.wire("SRESET.2", "RRESET.1")
	b.wire("SRESET.2", "U2.reset")
	b.ground("GNDR", "RRESET.2", -140, 740)

	// Preserve unambiguous stable IDs even after replacing the timer subcircuit.
	for k := range b.graph.Wires {
		b.graph.Wires[k].ID = fmt.Sprintf("W%d", k+1)
	}
	return b.graph
}
func buttonEventTarget(event string) *ButtonParams {
	v := strings.Split(event, ":")
	if len(v) != 3 || v[0] != "button" {
		return nil
	}
	return &ButtonParams{ID: v[1], Pressed: v[2] != "release"}
}

func ButtonDebounceExampleGraph() Graph {
	b := newExampleBuilder()
	b.part("V1", "dc-source", "5", -700, -300, 0)
	b.ground("GNDV", "V1.negative", -700, -140)
	b.part("SCOUNT", "bouncing-pushbutton", "", -400, -50, 0)
	b.wire("V1.positive", "SCOUNT.1")
	b.part("RCLK", "resistor", "10k", -250, 100, 90)
	b.wire("SCOUNT.2", "RCLK.1")
	b.ground("GNDP", "RCLK.2", -120, 180)
	b.part("SRESET", "pushbutton", "", -600, 450, 0)
	b.wire("V1.positive", "SRESET.1")
	b.part("RRESET", "resistor", "10k", -380, 550, 90)
	b.wire("SRESET.2", "RRESET.1")
	b.ground("GNDR", "RRESET.2", -380, 720)
	b.part("RDEB", "resistor", "1k", -180, 420, 0)
	b.part("U6", "logic-not", "", -470, 300, 0)
	b.part("U7", "logic-not", "", -280, 300, 0)
	b.wire("SCOUNT.2", "U6.a")
	b.wire("U6.out", "U7.a")
	b.wire("U7.out", "RDEB.1")
	for _, id := range []string{"U6", "U7"} {
		x := -470.
		if id == "U7" {
			x = -280
		}
		b.label("PWR"+id, "VCC", id+".vcc", x, 180)
		b.ground("GND"+id, id+".gnd", x, 410)
	}
	b.part("CDEB", "capacitor", "10u", -20, 650, 0)
	b.wire("RDEB.2", "CDEB.1")
	b.ground("GNDC", "CDEB.2", -20, 830)
	for k, id := range []string{"U2", "U4"} {
		y := float64(k) * 500
		b.part(id, "decade-counter", "", 200, y, 0)
		b.label("POWER"+id, "VCC", id+".vcc", 200, y-230)
		b.label("ENABLE"+id, "VCC", id+".enable", 20, y-70)
		b.wire(id+".reset", "SRESET.2")
		b.ground("GND"+id, id+".gnd", 350, y+200)
		r, d := fmt.Sprintf("RL%d", k+1), fmt.Sprintf("D%d", k+1)
		b.part(r, "resistor", "1k", 450, y-60, 0)
		b.part(d, "led", "", 650, y-60, 270)
		b.wire(id+".q0", r+".1")
		b.wire(r+".2", d+".anode")
		b.ground("GND"+d, d+".cathode", 800, y+20)
	}
	b.label("PWR", "VCC", "V1.positive", -700, -450)
	b.wire("SCOUNT.2", "U2.clk")
	b.wire("CDEB.1", "U4.clk")
	return b.graph
}
