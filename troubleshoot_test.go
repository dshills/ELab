package main

import (
	"encoding/json"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestTroubleshootingFlagsReverseLEDAndTriesFixOnCopy(t *testing.T) {
	f := fixture(t)
	f.command("example.load", workbench.ExampleLoadParams{ID: "led", DiscardChanges: true})
	f.command("simulation.pause", struct{}{})
	f.wait()
	meter := f.request("POST", "/api/troubleshoot?"+f.query(""), []byte(`{"red":"D1.cathode","black":"D1.anode"}`), "")
	var reading struct{ Voltage *float64 }
	if meter.Code != 200 || json.Unmarshal(meter.Body.Bytes(), &reading) != nil || reading.Voltage == nil || *reading.Voltage > -1 {
		t.Fatal(meter.Body.String())
	}
	f.command("object.delete", workbench.IDParams{ID: "W2"})
	f.command("object.delete", workbench.IDParams{ID: "W3"})
	f.command("wire.connect", workbench.ConnectParams{From: "R1.2", To: "D1.cathode"})
	f.command("wire.connect", workbench.ConnectParams{From: "D1.anode", To: "GND.reference"})
	f.wait()
	before := f.state()
	response := f.request("POST", "/api/troubleshoot?"+f.query(""), []byte(`{"red":"D1.anode","black":"D1.cathode","trial":true}`), "")
	var out struct {
		Hints   []troubleshootingHint
		Voltage *float64
		Trial   map[string]string
	}
	if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &out) != nil {
		t.Fatal(response.Code, response.Body.String())
	}
	if out.Voltage != nil || len(out.Hints) != 1 || out.Hints[0].Code != "reverse-led" || out.Trial["reading"] == "" {
		t.Fatal(response.Body.String())
	}
	after := f.state()
	if before.CommandRevision != after.CommandRevision || before.Document != after.Document {
		t.Fatal("trial edited the live circuit")
	}
}

func TestTroubleshootingSuggestsOpenWireAndMissingGround(t *testing.T) {
	for _, tc := range []struct{ object, code string }{{"W2", "open-terminal"}, {"GND", "missing-ground"}} {
		t.Run(tc.code, func(t *testing.T) {
			f := fixture(t)
			f.command("example.load", workbench.ExampleLoadParams{ID: "led", DiscardChanges: true})
			f.command("simulation.pause", struct{}{})
			f.command("object.delete", workbench.IDParams{ID: tc.object})
			f.wait()
			response := f.request("POST", "/api/troubleshoot?"+f.query(""), []byte(`{"trial":true}`), "")
			var out struct {
				Hints      []troubleshootingHint
				Suggestion string
				Trial      map[string]string
			}
			if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &out) != nil || len(out.Hints) != 1 || out.Hints[0].Code != tc.code || out.Suggestion == "" || out.Trial["reading"] == "" {
				t.Fatal(response.Body.String())
			}
		})
	}
}
