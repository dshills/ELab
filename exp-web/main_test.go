package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

type webFixture struct {
	server  *webServer
	handler http.Handler
	cookie  *http.Cookie
	t       *testing.T
}

func fixture(t *testing.T) *webFixture {
	t.Helper()
	server := newWebServer(t.TempDir())
	f := &webFixture{server: server, handler: server.handler(), t: t}
	t.Cleanup(server.close)
	response := f.request("GET", "/api/state", nil, "")
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	return f
}
func (f *webFixture) request(method, path string, body []byte, origin string) *httptest.ResponseRecorder {
	f.t.Helper()
	request := httptest.NewRequest(method, "http://127.0.0.1:8787"+path, bytes.NewReader(body))
	if f.cookie != nil {
		request.AddCookie(f.cookie)
	}
	request.Header.Set("Content-Type", "application/json")
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	response := httptest.NewRecorder()
	f.handler.ServeHTTP(response, request)
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "elab-workspace" {
			f.cookie = cookie
		}
	}
	return response
}
func (f *webFixture) state() workbench.Snapshot {
	f.t.Helper()
	r := f.request("GET", "/api/state", nil, "")
	var payload struct{ State workbench.Snapshot }
	if err := json.Unmarshal(r.Body.Bytes(), &payload); err != nil {
		f.t.Fatal(err, r.Body.String())
	}
	return payload.State
}
func (f *webFixture) command(name string, params any) workbench.Response {
	f.t.Helper()
	state := f.state()
	raw, _ := json.Marshal(params)
	r := workbench.Request{Version: 1, ID: token(), Key: token(), Command: name, Params: raw, ExpectedCommandRevision: &state.CommandRevision, ExpectedDocumentGeneration: &state.Document.Generation}
	encoded, _ := json.Marshal(r)
	response := f.request("POST", "/api/command", encoded, "")
	var out workbench.Response
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		f.t.Fatal(err)
	}
	if response.Code != 200 || out.Error != nil {
		f.t.Fatalf("%s: %s", name, response.Body.String())
	}
	return out
}
func (f *webFixture) wait() { f.t.Helper(); f.command("simulation.wait", struct{}{}) }
func (f *webFixture) query(kind string) string {
	state := f.state()
	return url.Values{"kind": {kind}, "revision": {fmt.Sprint(state.CommandRevision)}, "generation": {fmt.Sprint(state.Document.Generation)}, "discard": {"true"}}.Encode()
}

func TestBrowserCommandsUseNativeSolverAndScene(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	f.command("simulation.restart", struct{}{})
	f.wait()
	f.command("simulation.step", workbench.StepParams{Duration: .01})
	state := f.state()
	if state.Example.Info.ID != "rc" || math.Abs(state.Transient.Sample.Voltage-5*(1-math.Exp(-1))) > 1e-9 {
		t.Fatalf("RC differs from native result: %+v", state.Transient)
	}
	scene := makeScene(state)
	if len(scene.Parts) != 4 || len(scene.Wires) != 4 {
		t.Fatal("missing graph geometry")
	}
	for _, part := range scene.Parts {
		if len(part.Paths) == 0 || len(part.Pins) != len(part.Terminals) {
			t.Fatal("missing symbol or terminals", part.ID)
		}
	}
	old := state
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "20k"})
	raw, _ := json.Marshal(workbench.Request{Version: 1, ID: token(), Key: token(), Command: "component.set", Params: json.RawMessage(`{"id":"R1","value":"3k"}`), ExpectedCommandRevision: &old.CommandRevision, ExpectedDocumentGeneration: &old.Document.Generation})
	r := f.request("POST", "/api/command", raw, "")
	var refused workbench.Response
	_ = json.Unmarshal(r.Body.Bytes(), &refused)
	if refused.Error == nil || refused.Error.Code != "conflict" {
		t.Fatal("stale editor was allowed", r.Body.String())
	}
}
func TestBrowserHBridgeRecordingDownloadAndReopen(t *testing.T) {
	f := fixture(t)
	f.command("example.load", workbench.ExampleLoadParams{ID: "discrete-hbridge", DiscardChanges: true})
	f.command("simulation.pause", struct{}{})
	f.command("simulation.restart", struct{}{})
	f.wait()
	before := f.state()
	if before.Mixed.Resistors[0].Voltage > -4.72 {
		t.Fatal("missing negative polarity")
	}
	f.command("simulation.step", workbench.StepParams{Duration: .021})
	if f.state().Mixed.Resistors[0].Voltage < 4.72 {
		t.Fatal("missing reversal")
	}
	f.command("simulation.verify", map[string]any{"duration_s": .045, "sample_step_s": .001, "probe_ids": []string{"v:DIR.positive", "v:RL", "i:RL"}})
	id := f.state().LastOperation
	deadline := time.Now().Add(10 * time.Second)
	var state workbench.Snapshot
	for time.Now().Before(deadline) {
		state = f.state()
		if len(state.Operations) > 0 && state.Operations[0].Status == "completed" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if len(state.Operations) == 0 || state.Operations[0].Status != "completed" {
		t.Fatal("capture did not complete")
	}
	f.command("recording.keep.set", map[string]any{"id": id, "keep": true})
	captured := f.command("operation.samples", workbench.VerificationPageParams{ID: id, Limit: 200}).Result
	blob := f.request("POST", "/api/download?"+f.query("project"), nil, "")
	if blob.Code != 200 {
		t.Fatal(blob.Body.String())
	}
	var document map[string]any
	if err := json.Unmarshal(blob.Body.Bytes(), &document); err != nil || document["format"] != "elab.schematic" {
		t.Fatal("invalid native project", err)
	}
	f.command("project.clear", workbench.ReplaceParams{DiscardChanges: true})
	upload := f.request("POST", "/api/upload?"+f.query(""), blob.Body.Bytes(), "")
	if upload.Code != 200 {
		t.Fatal(upload.Body.String())
	}
	f.wait()
	restored := f.state()
	if len(restored.Operations) != 1 || len(restored.Components) != len(before.Components) || restored.Document.Dirty {
		t.Fatal("project or recording changed during browser round trip")
	}
	reopened := f.command("operation.samples", workbench.VerificationPageParams{ID: restored.Operations[0].ID, Limit: 200}).Result
	originalRows := captured.(map[string]any)
	restoredRows := reopened.(map[string]any)
	for _, field := range []string{"rows", "probes", "total"} {
		if !reflect.DeepEqual(originalRows[field], restoredRows[field]) {
			t.Fatalf("recorded %s changed during browser round trip", field)
		}
	}
	svg := f.request("POST", "/api/download?"+f.query("svg"), nil, "")
	if svg.Code != 200 || !bytes.Contains(svg.Body.Bytes(), []byte("MNA")) {
		t.Fatal("schematic export missing MOSFET", svg.Body.String())
	}
}
func TestBrowserSessionsAndFileBoundaries(t *testing.T) {
	f := fixture(t)
	foreign := f.request("GET", "/api/state", nil, "https://unrelated.example")
	if foreign.Code != 403 {
		t.Fatal("foreign origin allowed")
	}
	for _, name := range []string{"project.load", "project.load.start", "project.save.start", "verification.import.start", "sweep.import.start", "export.start"} {
		raw, _ := json.Marshal(workbench.Request{Version: 1, ID: "file", Command: name, Params: json.RawMessage(`{"path":"/tmp/owner-project.elab"}`)})
		if r := f.request("POST", "/api/command", raw, ""); r.Code != 400 {
			t.Fatal("arbitrary filesystem command allowed", name)
		}
	}
	state := f.state()
	etag := f.request("GET", fmt.Sprintf("/api/state?after=%d", state.Revision), nil, "")
	if etag.Code != 304 && etag.Code != 200 {
		t.Fatal("invalid polling response")
	}
	second := &webFixture{server: f.server, handler: f.handler, t: t}
	second.request("GET", "/api/state", nil, "")
	if second.cookie.Value == f.cookie.Value {
		t.Fatal("workspaces share a cookie")
	}
	f.command("component.select", workbench.IDParams{ID: "C1"})
	if second.state().Selected == "C1" {
		t.Fatal("different browser session was changed")
	}
	for _, part := range workbench.Library() {
		if len(symbolPaths(workbench.Component{Kind: part.Kind})) == 0 {
			t.Fatal("missing library symbol", part.Kind)
		}
	}
}

// Every concrete command wired to a browser control must exist in this engine.
func TestFrontendUsesAvailableCommands(t *testing.T) {
	js, err := os.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	commands := newWebServer(t.TempDir()).commands
	for _, match := range regexp.MustCompile(`command\('([a-z.]+)'`).FindAllSubmatch(js, -1) {
		name := string(match[1])
		if _, ok := commands[name]; !ok || !permitted(name) {
			t.Errorf("unavailable browser command %s", name)
		}
	}
}

func TestBrowserBuildEditAndUndo(t *testing.T) {
	f := fixture(t)
	f.command("project.clear", workbench.ReplaceParams{DiscardChanges: true})
	for _, p := range []workbench.PlaceParams{
		{ID: "VS", Kind: "dc-source", Position: workbench.Point{X: -180}, Value: "5V"},
		{ID: "RS", Kind: "resistor", Position: workbench.Point{Y: -90}, Value: "1k"},
		{ID: "LED", Kind: "led", Position: workbench.Point{X: 180}},
		{ID: "GD", Kind: "ground", Position: workbench.Point{Y: 100}},
	} {
		f.command("component.place", p)
	}
	for _, pins := range [][2]string{{"VS.positive", "RS.1"}, {"RS.2", "LED.anode"}, {"LED.cathode", "GD.reference"}, {"GD.reference", "VS.negative"}} {
		f.command("wire.connect", workbench.ConnectParams{From: pins[0], To: pins[1]})
	}
	f.wait()
	before := f.state()
	if before.Status != "running" || before.Result == nil || before.Result.Stale || before.Result.Current <= 0 {
		t.Fatalf("LED drawing did not solve: %s %+v", before.Status, before.Diagnostics)
	}
	f.command("component.move", workbench.MoveParams{ID: "RS", Position: workbench.Point{Y: -110}})
	f.command("edit.undo", struct{}{})
	if !reflect.DeepEqual(before.Components, f.state().Components) {
		t.Fatal("move undo changed graph")
	}
	f.command("component.set", workbench.ValueParams{ID: "RS", Value: "2k"})
	f.wait()
	if f.state().Result.Current >= before.Result.Current {
		t.Fatal("higher resistance did not reduce LED current")
	}
	f.command("object.delete", workbench.IDParams{ID: f.state().Wires[0].ID})
	if len(f.state().Wires) != 3 {
		t.Fatal("wire was not removed")
	}
	f.command("edit.undo", struct{}{})
	f.wait()
	if len(f.state().Wires) != 4 || f.state().Status != "running" {
		t.Fatal("wire undo did not restore circuit")
	}
}

func TestLocalNavigationAndForeignAPI(t *testing.T) {
	f := fixture(t)
	for _, path := range []string{"/", "/api/state"} {
		req := httptest.NewRequest("GET", "http://127.0.0.1:8787"+path, nil)
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		out := httptest.NewRecorder()
		f.handler.ServeHTTP(out, req)
		want := 200
		if path == "/api/state" {
			want = 403
		}
		if out.Code != want {
			t.Fatalf("%s: got %d want %d", path, out.Code, want)
		}
	}
}

func TestBrowserCommonAdvancedSettings(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	f.command("capacitor.initial.set", workbench.ValueParams{ID: "C1", Value: "2 V"})
	f.wait()
	find := func(id string) workbench.Component {
		t.Helper()
		for _, p := range f.state().Components {
			if p.ID == id {
				return p
			}
		}
		t.Fatal("missing component", id)
		return workbench.Component{}
	}
	if p := find("C1"); p.InitialVoltage == nil || *p.InitialVoltage != 2 {
		t.Fatal("startup voltage not applied")
	}
	f.command("passive.configure", map[string]any{"id": "C1", "enabled": true, "parameters": map[string]any{"version": "elab.passive-loss.v1", "esr_ohm": 2, "leakage_resistance_ohm": 1e6, "winding_resistance_ohm": 0}})
	if p := find("C1"); p.Device == nil || p.Device.Passive == nil || p.Device.Passive.ESR != 2 {
		t.Fatal("loss model not applied")
	}
	f.command("simulation.pulse.timing.set", map[string]any{"id": "V1", "frequency_hz": "25 Hz", "duty_percent": "40 %"})
	cfg := f.state().TransientSettings
	if math.Abs(cfg.HighSeconds-.016) > 1e-12 || math.Abs(cfg.LowSeconds-.024) > 1e-12 {
		t.Fatal("pulse timing not applied", cfg)
	}
	f.command("simulation.transient.set", map[string]any{"step_s": "500 us", "high_s": cfg.HighSeconds, "low_s": cfg.LowSeconds, "initial_voltage_v": 3, "seconds_per_wall_second": .01})
	f.command("simulation.restart", struct{}{})
	f.wait()
	f.command("simulation.step", map[string]any{"duration_s": "21 ms"})
	cfg = f.state().TransientSettings
	if cfg.StepSeconds != .0005 || cfg.InitialVoltage != 3 || cfg.HighSeconds != .016 {
		t.Fatal("sampling edit reset another field", cfg)
	}
	f.command("capacitor.initial.set", workbench.ValueParams{ID: "C1", Value: "default"})
	if find("C1").InitialVoltage != nil {
		t.Fatal("default capacitor startup did not reset override")
	}
	// Use real web payload shapes, including unit-bearing strings in nested forms.
	f.command("component.place", workbench.PlaceParams{ID: "VS", Kind: "sine-source", Position: workbench.Point{X: 400}})
	f.command("source.sine.set", map[string]any{"id": "VS", "waveform": map[string]any{"amplitude_v": "500 mV", "offset_v": "1 V", "frequency_hz": "100 Hz", "phase_degrees": "90"}})
	if p := find("VS"); p.Sine.Amplitude != .5 || p.Sine.Frequency != 100 || p.Sine.Phase != 90 {
		t.Fatal("sine settings not applied", p.Sine)
	}
	f.command("component.place", workbench.PlaceParams{ID: "VP", Kind: "finite-pulse-source", Position: workbench.Point{X: 600}})
	f.command("source.pulse.set", map[string]any{"id": "VP", "waveform": map[string]any{"high_v": "4 V", "low_v": "1 V", "high_s": "20 ms", "low_s": "30 ms", "rise_s": "1 ms", "fall_s": "2 ms"}})
	if p := find("VP"); p.Pulse.HighSeconds != .02 || p.Pulse.FallSeconds != .002 {
		t.Fatal("finite pulse settings not applied")
	}
	f.command("component.place", workbench.PlaceParams{ID: "L1", Kind: "inductor", Position: workbench.Point{X: 800}})
	f.command("inductor.initial.set", workbench.ValueParams{ID: "L1", Value: "-5 mA"})
	if p := find("L1"); p.InitialCurrent == nil || *p.InitialCurrent != -.005 {
		t.Fatal("inductor startup not applied")
	}
}
