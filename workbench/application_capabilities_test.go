package workbench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var remainingContractCommands = []string{
	"display.inspect", "examples.list", "example.load", "example.current.reset", "diagnostics.details.set",
	"appearance.configure", "appearance.wait", "appearance.theme.set", "display.preferences.set", "capabilities.list",
	"parts.list", "events.poll", "circuit.connection.set", "example.flasher", "example.npn-pulse", "example.npn",
	"example.switch-npn", "example.rectifier", "example.rc", "example.reset",
}

func TestApplicationRemainingDiscoveryCompletePureOwned(t *testing.T) {
	s := New()
	defer s.Close()
	awaitTestInitialization(t, s)
	before := s.Snapshot()
	for _, command := range remainingContractCommands {
		t.Run(command, func(t *testing.T) {
			raw, _ := json.Marshal(CapabilityQuery{Command: command})
			r := s.Dispatch(t.Context(), Request{Version: 1, ID: command, Command: "capabilities.list", Params: raw}, "reader")
			if r.Error != nil {
				t.Fatal(r.Error)
			}
			caps := r.Result.(map[string]any)["capabilities"].([]Capability)
			var schema map[string]any
			if err := json.Unmarshal(caps[0].Input, &schema); err != nil {
				t.Fatal(err)
			}
			contract, ok := schema["x-command-contract"].(map[string]any)
			if !ok || contract["version"] != float64(1) || contract["completion"] == nil || len(contract["effects"].([]any)) == 0 || len(contract["errors"].([]any)) == 0 {
				t.Fatal("missing application contract", command)
			}
			caps[0].Input[0] = '!'
			if fresh := navigationSchema(t, command); fresh["x-command-contract"] == nil {
				t.Fatal("borrowed contract")
			}
		})
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("discovery mutated state")
	}
}

func TestApplicationInspectionPublishedBoundsAndOutcomes(t *testing.T) {
	s := NewExample()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := s.Snapshot()
	for _, phase := range []float64{0, 1, .25} {
		raw, _ := json.Marshal(DisplayFrameParams{Phase: phase})
		r := s.Dispatch(t.Context(), Request{Version: 1, ID: "display", Command: "display.inspect", Params: raw}, "reader")
		if r.Error != nil {
			t.Fatal(r.Error)
		}
		want := phase
		if phase == 1 {
			want = 0
		}
		if r.Result.(DisplayFrame).Phase != want {
			t.Fatal("illustrative phase boundary differs")
		}
	}
	for _, raw := range []string{`{"phase":-0.01}`, `{"phase":1.01}`, `{"phase":"bad"}`} {
		r := s.Dispatch(t.Context(), Request{Version: 1, ID: "bad-display", Command: "display.inspect", Params: json.RawMessage(raw)}, "reader")
		if r.Error == nil || r.Error.Code != "invalid-params" {
			t.Fatal(r.Error)
		}
	}
	events := navigationSchema(t, "events.poll")
	timeout := inputProperty(events, "timeout_ms")
	if timeout["minimum"] != float64(0) || timeout["maximum"] != float64(MaxEventWaitMS) || timeout["default"] != float64(1000) || events["x-max-waiters"] != float64(MaxEventWaiters) {
		t.Fatal("event bounds/defaults absent")
	}
	idle := s.Dispatch(t.Context(), eventRequest(before.Revision, 0), "reader").Result.(EventPollResult)
	if idle.Changed || idle.TimedOut || idle.Snapshot != nil || !idle.Coalesced {
		t.Fatal("nonblocking idle contract differs")
	}
	r := s.Dispatch(t.Context(), eventRequest(before.Revision+1, 0), "reader")
	if r.Error == nil || r.Error.Code != "conflict" {
		t.Fatal("future cursor admitted")
	}
	r = s.Dispatch(t.Context(), eventRequest(before.Revision, MaxEventWaitMS+1), "reader")
	if r.Error == nil || r.Error.Code != "invalid-params" {
		t.Fatal("outside published timeout admitted")
	}
	for _, raw := range []string{`{"command":null}`, `{"command":"missing.command"}`, `{"command":"` + strings.Repeat("µ", 65) + `"}`} {
		r := s.Dispatch(t.Context(), Request{Version: 1, ID: "bad-catalog", Command: "capabilities.list", Params: json.RawMessage(raw)}, "reader")
		code := "invalid-params"
		if strings.Contains(raw, "missing.command") {
			code = "not-found"
		}
		if r.Error == nil || r.Error.Code != code {
			t.Fatal(raw, r.Error)
		}
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("queries/refusals changed state")
	}
}

func TestApplicationPreferenceContractsMatchPersistenceAndHistory(t *testing.T) {
	s := NewExample()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "project.elab")}, "save")
	before := s.Snapshot()
	call(t, s, "diagnostics.details.set", DiagnosticDetailsParams{Visible: true}, "details")
	if state := s.Snapshot(); !state.DiagnosticPresentation.DetailsVisible || state.Document != before.Document || !reflect.DeepEqual(state.UndoEntry, before.UndoEntry) {
		t.Fatal("session details affected document")
	}
	prefs := navigationSchema(t, "display.preferences.set")
	rangeV := inputProperty(prefs, "voltage_range_v")
	if len(prefs["required"].([]any)) != 6 || prefs["allOf"] == nil {
		t.Fatal("full preference requirement missing")
	}
	for i, voltage := range []float64{rangeV["minimum"].(float64), rangeV["maximum"].(float64)} {
		p := DefaultDisplayPreferences()
		p.VoltageRange = voltage
		p.CurrentCues = true
		p.CurrentMotion = true
		p.ReducedMotion = true
		p.Direction = "electron"
		call(t, s, "display.preferences.set", p, fmt.Sprint("preferences", i))
		after := s.Snapshot()
		if after.Display.Preferences != p || after.Display.Animate || !after.Document.Dirty || after.UndoEntry.Name != "Change display preferences" || after.RunID != before.RunID || !reflect.DeepEqual(after.Result, before.Result) {
			t.Fatal("published display effects differ")
		}
	}
	p := DefaultDisplayPreferences()
	p.CurrentMotion = true
	refuse(t, s, "display.preferences.set", p, "invalid-params")
	p.CurrentCues = true
	p.VoltageRange = 30.01
	refuse(t, s, "display.preferences.set", p, "invalid-params")
	path := filepath.Join(t.TempDir(), "appearance.json")
	call(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "configure")
	appearanceReady(t, s)
	configured := s.Snapshot()
	for _, theme := range inputProperty(navigationSchema(t, "appearance.theme.set"), "theme")["enum"].([]any) {
		call(t, s, "appearance.theme.set", map[string]any{"theme": theme}, fmt.Sprint("theme", theme))
		ready := appearanceReady(t, s)
		if ready.Theme != theme || savedAppearance(t, path) != theme || ready.Document != configured.Document || !reflect.DeepEqual(ready.UndoEntry, configured.UndoEntry) {
			t.Fatal("theme persistence affected project or missed choice")
		}
	}
	bytesBefore, _ := os.ReadFile(path)
	theme := s.Snapshot().Theme
	call(t, s, "appearance.configure", AppearanceConfigureParams{}, "disable")
	bytesAfter, _ := os.ReadFile(path)
	if !bytes.Equal(bytesBefore, bytesAfter) || s.Snapshot().Theme != theme || s.Snapshot().Appearance.Status != "disabled" {
		t.Fatal("disable changed file/theme")
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("not JSON"), 0600); err != nil {
		t.Fatal(err)
	}
	call(t, s, "appearance.configure", AppearanceConfigureParams{Path: bad}, "bad-store")
	raw := json.RawMessage(`{}`)
	r := s.Dispatch(t.Context(), Request{Version: 1, ID: "wait", Command: "appearance.wait", Params: raw}, "reader")
	if r.Error != nil || r.Result.(Snapshot).Appearance.Status != "error" || r.Result.(Snapshot).Appearance.Error == "" {
		t.Fatal("worker failure must be inspectable in successful wait", r.Error)
	}
	call(t, s, "appearance.theme.set", map[string]string{"theme": "dark"}, "session-after-error")
	data, _ := os.ReadFile(bad)
	if string(data) != "not JSON" {
		t.Fatal("corrupt preference silently overwritten")
	}
	// Retire the intentionally failed private store so test cleanup has no failed
	// persistence outcome to acknowledge; no user preference path was configured.
	call(t, s, "appearance.configure", AppearanceConfigureParams{}, "disable-failed-store")
}

func TestApplicationExampleAliasAndResetContracts(t *testing.T) {
	aliases := map[string]string{"example.reset": "led", "example.rc": "rc", "example.npn": "npn", "example.npn-pulse": "blink", "example.flasher": "flasher", "example.switch-npn": "switch-npn", "example.rectifier": "rectifier"}
	for command, id := range aliases {
		t.Run(command, func(t *testing.T) {
			if navigationSchema(t, command)["x-example-id"] != id {
				t.Fatal("legacy alias mapping changed")
			}
			s := New()
			defer s.Close()
			call(t, s, command, ReplaceParams{}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			state := s.Snapshot()
			if state.Example == nil || state.Example.Info.ID != id {
				t.Fatal("published alias loaded wrong template")
			}
			call(t, s, "component.move", MoveParams{ID: state.Components[0].ID, Position: Point{X: 900, Y: 700}}, "edit-template")
			refuse(t, s, command, ReplaceParams{}, "unsaved-changes")
			wrong := state.Document.Generation + 1
			refuse(t, s, command, ReplaceParams{DiscardChanges: true, ExpectedGeneration: &wrong}, "conflict")
		})
	}
	s := New()
	defer s.Close()
	refuse(t, s, "example.current.reset", ExampleResetParams{}, "unavailable")
	call(t, s, "example.load", ExampleLoadParams{ID: "rc"}, "rc")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	original := s.Snapshot()
	call(t, s, "component.move", MoveParams{ID: "R1", Position: Point{X: 800, Y: 700}}, "move")
	refuse(t, s, "example.current.reset", ExampleResetParams{ExpectedID: "led", DiscardChanges: true}, "conflict")
	refuse(t, s, "example.current.reset", ExampleResetParams{ExpectedID: "rc"}, "unsaved-changes")
	call(t, s, "example.current.reset", ExampleResetParams{ExpectedID: "rc", DiscardChanges: true}, "reset")
	if state := s.Snapshot(); !state.Paused || !reflect.DeepEqual(state.Components, original.Components) || state.Document.Generation <= original.Document.Generation {
		t.Fatal("current reset failed to preserve pause/fresh generation")
	}
	call(t, s, "example.reset", ReplaceParams{DiscardChanges: true}, "led")
	if state := s.Snapshot(); state.Paused || state.Example.Info.ID != "led" {
		t.Fatal("example.reset is LED load, not current reset")
	}
}

func TestApplicationCompatibilityConnectionContract(t *testing.T) {
	s := NewExample()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := s.Snapshot()
	call(t, s, "circuit.connection.set", map[string]bool{"connected": true}, "existing")
	if state := s.Snapshot(); !reflect.DeepEqual(state.Components, before.Components) || !reflect.DeepEqual(state.Wires, before.Wires) || state.RunID != before.RunID || state.Document != before.Document || !reflect.DeepEqual(state.UndoEntry, before.UndoEntry) {
		t.Fatal("existing wire wasn't no-op")
	}
	call(t, s, "circuit.connection.set", map[string]bool{"connected": false}, "disconnect")
	if len(s.Snapshot().Wires) != len(before.Wires)-1 {
		t.Fatal("did not remove exact fixture wire")
	}
	disconnected := s.Snapshot()
	call(t, s, "circuit.connection.set", map[string]bool{"connected": false}, "already-disconnected")
	if state := s.Snapshot(); !reflect.DeepEqual(state.Wires, disconnected.Wires) || state.RunID != disconnected.RunID || !reflect.DeepEqual(state.UndoEntry, disconnected.UndoEntry) {
		t.Fatal("absent disconnect changed history/physics")
	}
	call(t, s, "circuit.connection.set", map[string]bool{"connected": true}, "reconnect")
	if len(s.Snapshot().Wires) != len(before.Wires) {
		t.Fatal("failed to reconnect fixture")
	}
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "clear")
	for _, connected := range []bool{false, true} {
		refuse(t, s, "circuit.connection.set", map[string]bool{"connected": connected}, "invalid-params")
	}
}

// New public commands inherit the same discovery obligation; this guard is
// intentionally only annotation presence/ownership, not behavior acceptance.
func TestApplicationCatalogRequiresContractAnnotations(t *testing.T) {
	for _, cap := range Catalog() {
		detailed, err := detailedCapabilities(cap.ID)
		if err != nil || len(detailed) != 1 {
			t.Fatal(cap.ID, err)
		}
		var schema map[string]any
		if err := json.Unmarshal(detailed[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		contract, ok := schema["x-command-contract"].(map[string]any)
		if !ok || contract["version"] != float64(1) || contract["scope"] == "" {
			t.Errorf("public command %s lacks versioned semantic annotations", cap.ID)
		}
		if cap.ID != detailed[0].ID || cap.Mutation != detailed[0].Mutation || !bytes.Equal(cap.OutputSchema, detailed[0].OutputSchema) {
			t.Fatal("filtered input discovery changed command classification/output", cap.ID)
		}
	}
}
