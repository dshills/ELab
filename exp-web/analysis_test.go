package main

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func TestBrowserACCapturePersistsAcrossServiceLoss(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	f.command("sweep.start", workbench.SweepStartParams{Settings: &workbench.SweepSettings{ComponentID: "V1", Parameter: "frequency", Start: "10 Hz", Stop: "10 kHz", Points: 9, Spacing: "logarithmic", ProbeIDs: []string{"v:C1"}}})
	deadline := time.Now().Add(10 * time.Second)
	var result *workbench.SweepResult
	var id string
	for time.Now().Before(deadline) {
		state := f.state()
		if state.DCSweep.Result != nil {
			result = state.DCSweep.Result
			id = state.DCSweep.Selected
			break
		}
		time.Sleep(time.Millisecond)
	}
	if result == nil {
		t.Fatal("AC capture did not finish")
	}
	if len(result.Probes) != 2 || result.Probes[0].Unit != "dB" || result.Probes[1].Unit != "°" {
		t.Fatal("AC gain/phase channels unavailable")
	}
	for _, point := range result.Points {
		omegaRC := 2 * math.Pi * point.Input * .01
		gain := -10 * math.Log10(1+omegaRC*omegaRC)
		phase := -math.Atan(omegaRC) * 180 / math.Pi
		if math.Abs(point.Values[0]-gain) > 1e-7 || math.Abs(point.Values[1]-phase) > 1e-7 {
			t.Fatal("web AC result differs from RC complex reference", point)
		}
	}
	f.command("sweep.keep.set", workbench.RecordingKeepParams{ID: id, Keep: true})
	// Lose the engine without graceful web-server persistence. The acknowledged
	// keep action itself must already contain the complete captured curve.
	f.server.mu.Lock()
	old := f.server.sessions[f.cookie.Value]
	delete(f.server.sessions, f.cookie.Value)
	f.server.mu.Unlock()
	old.service.Close()
	next := newWebServer(f.server.root)
	t.Cleanup(next.close)
	restored := &webFixture{server: next, handler: next.handler(), cookie: f.cookie, t: t}
	state := restored.state()
	if state.DCSweep == nil || state.DCSweep.Result == nil || !reflect.DeepEqual(state.DCSweep.Result.Points, result.Points) {
		t.Fatal("kept AC points changed after recovery")
	}
	if !state.Document.Dirty {
		t.Fatal("unsaved captured result marked clean after recovery")
	}
	exported := restored.request("POST", "/api/download?"+restored.query("sweep-json"), nil, "")
	if exported.Code != 200 {
		t.Fatal(exported.Body.String())
	}
	imported := restored.request("POST", "/api/import-result?kind=sweep&"+restored.query(""), exported.Body.Bytes(), "")
	if imported.Code != 200 {
		t.Fatal(imported.Body.String())
	}
}
