package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dshills/elab/workbench"
)

type meterParams struct {
	Red   string `json:"red"`
	Black string `json:"black"`
	Trial bool   `json:"trial"`
}

type troubleshootingHint struct {
	Code  string `json:"code"`
	Text  string `json:"text"`
	Trial string `json:"trial,omitempty"`
}

func linkedTerminals(graph workbench.Graph) map[string]bool {
	linked := map[string]bool{}
	for _, wire := range graph.Wires {
		if wire.Connected {
			linked[wire.From], linked[wire.To] = true, true
		}
	}
	return linked
}

func troubleshootTrial(graph workbench.Graph, hint troubleshootingHint, led workbench.Component, source workbench.Component) (workbench.Graph, bool) {
	trial := graph.Clone()
	unique := func(base string) string {
		for i := 1; ; i++ {
			id := fmt.Sprintf("%s%d", base, i)
			if !trial.HasObject(id) {
				return id
			}
		}
	}
	connect := func(a, b string) {
		trial.Wires = append(trial.Wires, workbench.Wire{ID: unique("TrialWire"), From: a, To: b, Connected: true})
	}
	switch hint.Code {
	case "missing-ground":
		if source.ID == "" || len(trial.Sheets) > 0 {
			return trial, false
		}
		occupied := map[workbench.Point]bool{}
		for _, part := range trial.Components {
			occupied[part.Position] = true
		}
		for i := 1; i <= 201; i++ {
			at := workbench.Point{X: source.Position.X, Y: -10000 + float64(i)*workbench.GridSize}
			if occupied[at] {
				continue
			}
			part, err := workbench.MakeComponent(unique("TrialGround"), "ground", at, "")
			if err != nil {
				return trial, false
			}
			trial.Components = append(trial.Components, part)
			connect(source.ID+".negative", part.ID+".reference")
			linked := linkedTerminals(graph)
			if len(graph.Components) == 3 && led.ID != "" && !linked[led.ID+".cathode"] && !linked[source.ID+".negative"] {
				connect(led.ID+".cathode", part.ID+".reference")
			}
			break
		}
	case "open-terminal":
		if len(trial.Sheets) > 0 || len(trial.Components) > 6 || led.ID == "" || source.ID == "" {
			return trial, false
		}
		linked := linkedTerminals(graph)
		var open []string
		for _, part := range graph.Components {
			if part.Kind == "ground" || part.Kind == "net-label" || part.Kind == "junction" {
				continue
			}
			for _, pin := range part.Terminals {
				ref := part.ID + "." + pin
				if !linked[ref] {
					open = append(open, ref)
				}
			}
		}
		if len(open) != 2 {
			return trial, false
		}
		connect(open[0], open[1])
	case "reverse-led":
		if led.ID == "" || len(trial.Sheets) > 0 {
			return trial, false
		}
		flip := func(ref string) string {
			switch ref {
			case led.ID + ".anode":
				return led.ID + ".cathode"
			case led.ID + ".cathode":
				return led.ID + ".anode"
			}
			return ref
		}
		for i := range trial.Wires {
			trial.Wires[i].From = flip(trial.Wires[i].From)
			trial.Wires[i].To = flip(trial.Wires[i].To)
		}
	default:
		return trial, false
	}
	return trial, true
}

func (s *webServer) troubleshoot(w http.ResponseWriter, r *http.Request, ws *workspace) {
	var params meterParams
	if !readExperimentRequest(w, r, &params) {
		return
	}
	if !ws.experimentMu.TryLock() {
		webError(w, 409, "Wait for the current experiment to finish.")
		return
	}
	defer ws.experimentMu.Unlock()
	state, ok := observedExperiment(w, r, ws)
	if !ok {
		return
	}
	graph := experimentGraph(state)
	var sources, leds []workbench.Component
	hasGround := false
	for _, part := range graph.Components {
		switch part.Kind {
		case "dc-source":
			sources = append(sources, part)
		case "led":
			leds = append(leds, part)
		case "ground":
			hasGround = true
		}
	}
	var source, led workbench.Component
	if len(sources) == 1 {
		source = sources[0]
	}
	if len(leds) == 1 {
		led = leds[0]
	}
	compiled, _, diagnostic := graph.Compile()
	ctx := r.Context()
	hints := []troubleshootingHint{}
	if !hasGround && len(graph.Components) > 0 {
		hints = append(hints, troubleshootingHint{Code: "missing-ground", Text: "There is no ground reference. For a simple circuit with one DC supply, start at its negative terminal.", Trial: "Add a ground reference to the DC supply"})
	} else if diagnostic != nil && (diagnostic.Code == "open-terminal" || diagnostic.Code == "unconnected-primitive" || diagnostic.Code == "missing-parts") {
		hints = append(hints, troubleshootingHint{Code: "open-terminal", Text: "Check for an open connection. The circuit checks highlight the affected terminals.", Trial: "Connect the two open terminals"})
	} else if diagnostic != nil && reversedSeriesLED(compiled, diagnostic) {
		hints = append(hints, troubleshootingHint{Code: "reverse-led", Text: "The LED is wired backward for this supply. Its cathode faces the positive side and its anode faces ground.", Trial: "Swap the LED connections"})
	} else if diagnostic != nil {
		hints = append(hints, troubleshootingHint{Code: diagnostic.Code, Text: diagnostic.Message})
	}
	if diagnostic == nil && led.ID != "" {
		v, err := workbench.MeasureStartupVoltage(ctx, graph, state.TransientSettings, led.ID+".anode", led.ID+".cathode")
		if err == nil && v < -.05 {
			hints = append(hints, troubleshootingHint{Code: "reverse-led", Text: "The meter reads a lower voltage on the LED’s anode than its cathode at startup. Check the LED direction.", Trial: "Swap the LED connections"})
		}
	}
	var voltage *float64
	var reason string
	if params.Red != "" || params.Black != "" {
		v, err := workbench.MeasureStartupVoltage(ctx, graph, state.TransientSettings, params.Red, params.Black)
		if err != nil {
			reason = err.Error()
		} else {
			voltage = &v
		}
	}
	response := map[string]any{"hints": hints, "voltage": voltage, "reason": reason}
	if len(hints) > 0 {
		trial, ok := troubleshootTrial(graph, hints[0], led, source)
		if ok && hints[0].Trial != "" {
			if _, _, problem := trial.Compile(); problem == nil {
				response["suggestion"] = hints[0].Trial
				if params.Trial {
					addTrialResult(ctx, response, trial, state, led, source, hints[0].Trial)
				}
			}
		}
	}
	jsonReply(w, 200, response)
}

func reversedSeriesLED(c workbench.Compiled, diagnostic *workbench.Diagnostic) bool {
	if diagnostic.Code != "unsupported-topology" || c.LED.ID == "" || c.Circuit.Voltage <= 0 {
		return false
	}
	r1, r2 := c.TerminalNet[c.Resistor.ID+".1"], c.TerminalNet[c.Resistor.ID+".2"]
	between := func(a, b string) bool { return r1 == a && r2 == b || r1 == b && r2 == a }
	return c.Anode == c.Ground && between(c.Supply, c.Cathode) || c.Cathode == c.Supply && between(c.Anode, c.Ground)
}

func addTrialResult(ctx context.Context, response map[string]any, trial workbench.Graph, state workbench.Snapshot, led, source workbench.Component, title string) {
	measurement := ""
	if led.ID != "" {
		capture, err := workbench.CaptureExperiment(ctx, trial, state.TransientSettings, "i:"+led.ID, state.TransientSettings.StepSeconds)
		if err == nil {
			measurement = fmt.Sprintf("%s startup current: %s", led.ID, workbench.FormatValue(capture.Rows[0].Values[0], "A"))
		} else {
			response["trial_error"] = err.Error()
			return
		}
	} else if source.ID != "" {
		if value, err := workbench.MeasureStartupVoltage(ctx, trial, state.TransientSettings, source.ID+".positive", source.ID+".negative"); err == nil {
			measurement = "Supply voltage: " + workbench.FormatValue(value, "V")
		} else {
			response["trial_error"] = err.Error()
			return
		}
	}
	response["trial"] = map[string]string{"title": title, "reading": measurement}
}
