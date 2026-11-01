package main

import (
	"bytes"
	"encoding/csv"
	"math"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func TestBrowserSpectrumSourcesCursorsAndDistortionRemainIndependent(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	f.command("simulation.restart", struct{}{})
	f.wait()
	id := captureForControls(t, f, .2, []string{"v:C1"})
	f.command("operation.select", workbench.IDParams{ID: id})
	before := f.state()
	f.command("spectrum.analyze", map[string]any{"id": id, "probe_id": "v:C1", "start_s": "0 ms", "end_s": "200 ms", "window": "hann", "samples": 128, "fundamental_hz": "10 Hz", "harmonics": 5})
	result := f.state().Spectrum.Result
	if result == nil || result.THD == nil || len(result.Bins) != 65 || len(result.Harmonics) != 5 {
		t.Fatal("missing native FFT/THD result", result)
	}
	f.command("spectrum.cursor.set", workbench.SpectrumCursorParams{Source: "recorded", Cursor: "a", ResultID: result.ID, Peak: 1})
	frequency := 20.
	f.command("spectrum.cursor.set", workbench.SpectrumCursorParams{Source: "recorded", Cursor: "b", ResultID: result.ID, Frequency: &frequency})
	m := f.state().Spectrum.Measurements
	if m.A == nil || !m.A.Valid || m.A.Reading.Frequency != 10 || m.B == nil || !m.B.Valid || m.B.Reading.Frequency != 20 || m.DeltaHz == nil || *m.DeltaHz != 10 {
		t.Fatal("native peaks/cursors not exposed", m)
	}
	f.command("spectrum.live.set", workbench.LiveSpectrumParams{Enabled: false, Hold: true, ProbeID: "v:C1.1", Window: "hann"})
	after := f.state()
	live := after.Spectrum.Live
	if live == nil || live.Settings.Enabled || !live.Settings.Hold || live.Settings.ProbeID != "v:C1.1" || live.Settings.Window != "hann" {
		t.Fatal("live FFT settings were not applied", live)
	}
	if !reflect.DeepEqual(result, after.Spectrum.Result) || !reflect.DeepEqual(before.Transient, after.Transient) || before.Document.Dirty != after.Document.Dirty {
		t.Fatal("live spectrum controls changed captured FFT or live circuit")
	}
	export := f.request("POST", "/api/download?"+f.query("spectrum-csv"), nil, "")
	if export.Code != 200 || !bytes.Contains(export.Body.Bytes(), []byte("db_reference_1_unit")) || !bytes.Contains(export.Body.Bytes(), []byte("v:C1")) {
		t.Fatal("recorded spectrum export unavailable", export.Body.String())
	}
	rows, err := csv.NewReader(bytes.NewReader(export.Body.Bytes())).ReadAll()
	if err != nil {
		t.Fatal("FFT export has malformed rows", err)
	}
	if len(rows) != len(result.Bins)+1 {
		t.Fatalf("FFT export has %d rows, want %d", len(rows), len(result.Bins)+1)
	}
	if !reflect.DeepEqual(rows[0], []string{"recording_sha256", "probe_id", "frequency_hz", "peak_V", "db_reference_1_unit"}) {
		t.Fatal("FFT export has the wrong schema", rows[0])
	}
	for i, bin := range result.Bins {
		row := rows[i+1]
		if row[0] != result.SourceSHA256 || row[1] != result.Probe.ID {
			t.Fatal("FFT export mixed source provenance", row)
		}
		for j, expected := range []float64{bin.Frequency, bin.Peak, bin.DB} {
			value, err := strconv.ParseFloat(row[j+2], 64)
			if err != nil || value != expected {
				t.Fatal("FFT export changed an exact native bin", row)
			}
		}
	}
}

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
