package workbench

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDisplayFramesAreReadOnlyAndDirectionMotionAreExplicit(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	p := DefaultDisplayPreferences()
	p.CurrentCues = true
	p.CurrentMotion = true
	call(t, s, "display.preferences.set", p, "motion")
	before := s.Snapshot()
	query := func(phase float64) DisplayFrame {
		raw, _ := json.Marshal(DisplayFrameParams{Phase: phase})
		r := s.Dispatch(context.Background(), Request{Version: 1, ID: "frame", Command: "display.inspect", Params: raw}, "reader")
		if r.Error != nil {
			t.Fatal(r.Error)
		}
		return r.Result.(DisplayFrame)
	}
	a, b := query(.1), query(.35)
	if len(a.Markers) != 12 || reflect.DeepEqual(a.Markers, b.Markers) || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("illustrative frame phases change physics or do not animate")
	}
	p.Direction = "electron"
	call(t, s, "display.preferences.set", p, "electron")
	electron := query(.1)
	if electron.Markers[0].DX != -a.Markers[0].DX || electron.Markers[0].DY != -a.Markers[0].DY || *electron.Display.Wires[0].Current != *a.Display.Wires[0].Current {
		t.Fatal("electron mode changes measured current or fails to reverse visual direction")
	}
	p.ReducedMotion = true
	call(t, s, "display.preferences.set", p, "reduced")
	reduced := query(.35)
	if reduced.Display.Animate || len(reduced.Markers) != 4 {
		t.Fatal("reduced motion loses static arrows or leaves moving dots")
	}
	for _, marker := range reduced.Markers {
		if marker.Kind != "arrow" {
			t.Fatal("reduced motion dot")
		}
	}
	call(t, s, "display.preferences.set", DefaultDisplayPreferences(), "off")
	if len(query(.1).Markers) != 0 || s.Snapshot().Display.Animate {
		t.Fatal("disabled animation changes retained numerical information")
	}
	if *s.Snapshot().Display.Wires[0].Current != *before.Display.Wires[0].Current {
		t.Fatal("turning off cues loses signed current")
	}
	for _, phase := range []float64{-1, 1.01} {
		raw, _ := json.Marshal(DisplayFrameParams{Phase: phase})
		r := s.Dispatch(context.Background(), Request{Version: 1, ID: "bad", Command: "display.inspect", Params: raw}, "reader")
		if r.Error == nil {
			t.Fatal("invalid phase accepted")
		}
	}
}

func TestDisplayFailureStopsMotionAndMarksAcceptedValuesStale(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	p := DefaultDisplayPreferences()
	p.VoltageColor = true
	p.CurrentCues = true
	p.CurrentMotion = true
	call(t, s, "display.preferences.set", p, "prefs")
	call(t, s, "measurement.pin.add", PinSpec{ID: "v:C1", Position: Point{}}, "pin")
	entered, release := blockFirstLiveSample(t, s, true)
	defer release()
	call(t, s, "simulation.resume", struct{}{}, "resume")
	awaitLiveSampleBlock(t, entered)
	before := s.Snapshot()
	release()
	deadline := time.Now().Add(time.Second)
	for s.Snapshot().Status != "error" {
		if time.Now().After(deadline) {
			t.Fatal("controlled current integration failure not published")
		}
		time.Sleep(time.Millisecond)
	}
	after := s.Snapshot()
	if after.Display.Animate || after.Transient.Sample.Time != before.Transient.Sample.Time || after.PinnedReadings[0].State != "stale" {
		t.Fatal("failed solve animates or publishes fake progress")
	}
	for _, w := range after.Display.Wires {
		if w.State != "stale" || w.Current != nil || w.Color != "#86969f" {
			t.Fatal("failed solve retains fresh current/color", w)
		}
	}
}

func TestDisplayPreferencesOnlyRecoveryAndStaticSVGParity(t *testing.T) {
	s := NewExample()
	settled(t, s)
	call(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "clean.elab")}, "save")
	dir := t.TempDir()
	configureRecovery(t, s, dir, false)
	p := DefaultDisplayPreferences()
	p.VoltageColor = true
	p.CurrentCues = true
	p.CurrentMotion = true
	p.Direction = "electron"
	call(t, s, "display.preferences.set", p, "prefs")
	if s.Snapshot().Document.SchematicDirty || !s.Snapshot().Document.PreferencesDirty {
		t.Fatal("fixture must be preference-only dirty")
	}
	path := filepath.Join(t.TempDir(), "display.svg")
	call(t, s, "export.schematic.svg", ExportParams{Path: path}, "svg")
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), `stroke="#d56a39"`) || !strings.Contains(string(data), `data-current-wire="W1"`) || !strings.Contains(string(data), "electron direction") || !strings.Contains(string(data), "current_motion") {
		t.Fatal("SVG loses visible preferences/current cues/provenance", err)
	}
	call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
	choice := recoveryChoice(t, s)
	s.Close()
	fresh := New()
	defer fresh.Close()
	configureRecovery(t, fresh, dir, false)
	call(t, fresh, "recovery.restore", choice, "restore")
	state := settled(t, fresh)
	if state.Display.Preferences != p || !state.Document.Dirty {
		t.Fatal("recovery loses display preferences")
	}
}

func displayWire(t *testing.T, state Snapshot, id string) WireDisplay {
	t.Helper()
	for _, wire := range state.Display.Wires {
		if wire.ID == id {
			return wire
		}
	}
	t.Fatal("missing displayed wire", id)
	return WireDisplay{}
}
func TestDisplayPreferencesPreservePhysicsHistoryAndPortableOwnership(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	path := filepath.Join(t.TempDir(), "display.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	before := s.Snapshot()
	p := DefaultDisplayPreferences()
	p.VoltageColor = true
	p.CurrentCues = true
	p.CurrentMotion = true
	first := call(t, s, "display.preferences.set", p, "prefs").Result.(Snapshot)
	if first.RunID != before.RunID || first.ElectricalRevision != before.ElectricalRevision || !reflect.DeepEqual(first.Result, before.Result) || first.Document.SchematicDirty || !first.Document.PreferencesDirty || first.UndoName != "Change display preferences" || !first.Display.Animate {
		t.Fatal("display preferences changed physics or failed dirty/history semantics")
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	if s.Snapshot().Document.Dirty || s.Snapshot().Display.Preferences != DefaultDisplayPreferences() {
		t.Fatal("undo does not restore saved preferences")
	}
	call(t, s, "edit.redo", HistoryParams{}, "redo")
	copy := s.Snapshot()
	*copy.Display.Wires[0].Voltage = 999
	*copy.Display.Wires[0].Current = 999
	if *s.Snapshot().Display.Wires[0].Voltage == 999 || *s.Snapshot().Display.Wires[0].Current == 999 {
		t.Fatal("display snapshot aliases accepted values")
	}
	call(t, s, "simulation.pause", struct{}{}, "pause")
	if s.Snapshot().Display.Animate {
		t.Fatal("paused simulation animates")
	}
	p.ReducedMotion = true
	call(t, s, "display.preferences.set", p, "reduce")
	call(t, s, "project.save", SaveParams{}, "save-prefs")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	d, err := decodeDocument(data)
	if err != nil || d.DisplayPreferences == nil || *d.DisplayPreferences != p {
		t.Fatal("preference save failed", err)
	}
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	state := settled(t, fresh)
	if state.Document.Dirty || state.Display.Preferences != p || !state.Paused || state.Display.Animate {
		t.Fatal("fresh preference ownership/startup failed")
	}
	for _, raw := range []string{`null`, `{"voltage_color":true}`, `{"voltage_color":true,"voltage_range_v":0,"current_cues":true,"current_motion":false,"reduced_motion":false,"direction":"conventional"}`, `{"voltage_color":true,"voltage_range_v":5,"current_cues":false,"current_motion":true,"reduced_motion":false,"direction":"conventional"}`, `{"voltage_color":true,"voltage_range_v":5,"current_cues":true,"current_motion":false,"reduced_motion":false,"direction":"guess"}`} {
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(data, &fields)
		fields["display_preferences"] = json.RawMessage(raw)
		bad, _ := json.Marshal(fields)
		if _, err := decodeDocument(bad); err == nil {
			t.Fatal("invalid display preferences accepted", raw)
		}
	}
}

func TestWireCutCurrentsMatchLEDAndSignedRCReferences(t *testing.T) {
	s := NewExample()
	settled(t, s)
	for _, w := range s.Snapshot().Display.Wires {
		if w.Current == nil || math.Abs(*w.Current-s.Snapshot().Result.Current) > 1e-12 {
			t.Fatal("series wire does not carry independently accepted branch current", w)
		}
	}
	s.Close()
	s = New()
	defer s.Close()
	call(t, s, "example.rc", ReplaceParams{}, "rc")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	call(t, s, "simulation.step", StepParams{Duration: .01}, "charge")
	state := s.Snapshot()
	want := 5 * math.Exp(-1) / 10000
	for _, w := range state.Display.Wires {
		if w.Current == nil || math.Abs(*w.Current-want) > 1e-12 {
			t.Fatal("RC charge wire current wrong", w)
		}
	}
	call(t, s, "simulation.input.set", InputParams{Mode: "discharge"}, "low")
	call(t, s, "simulation.step", StepParams{Duration: .001}, "step")
	state = s.Snapshot()
	want = -5 * (1 - math.Exp(-1)) * math.Exp(-.1) / 10000
	for _, w := range state.Display.Wires {
		if w.Current == nil || math.Abs(*w.Current-want) > 1e-12 {
			t.Fatal("RC discharge direction/sign wrong", w)
		}
	}
	p := DefaultDisplayPreferences()
	p.VoltageColor = true
	call(t, s, "display.preferences.set", p, "color")
	if displayWire(t, s.Snapshot(), "W2").Color == displayWire(t, s.Snapshot(), "W3").Color {
		t.Fatal("capacitor and ground voltages have indistinguishable color")
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "edit")
	w := displayWire(t, s.Snapshot(), "W2")
	if w.State != "stale" || w.Current != nil || w.Color != "#86969f" {
		t.Fatal("previous result remains a fresh direction/color", w)
	}
}

func TestWireCyclesRefuseAmbiguousCurrentAndAliasesRemainConservative(t *testing.T) {
	g := ExampleGraph()
	j, _ := MakeComponent("J1", "junction", Point{X: -100, Y: -160}, "")
	g.Components = append(g.Components, j)
	g.Wires = append(g.Wires, Wire{ID: "WA", From: "V1.positive", To: "J1.node", Connected: true}, Wire{ID: "WB", From: "J1.node", To: "R1.1", Connected: true})
	s := newGraphService(SolveLED, g)
	defer s.Close()
	state := settled(t, s)
	for _, id := range []string{"W1", "WA", "WB"} {
		w := displayWire(t, state, id)
		if w.Current != nil || !strings.Contains(w.CurrentReason, "not uniquely") {
			t.Fatal("ideal-wire cycle invents a split current", w)
		}
	}
	if displayWire(t, state, "W2").Current == nil {
		t.Fatal("cycle incorrectly suppresses unrelated bridge current")
	}
	g = ExampleGraph()
	a, _ := MakeComponent("LA", "net-label", Point{X: -150, Y: -150}, "SUPPLY")
	b, _ := MakeComponent("LB", "net-label", Point{X: 150, Y: -150}, "SUPPLY")
	g.Components = append(g.Components, a, b)
	g.Wires[0].To = "LA.node"
	g.Wires = append(g.Wires, Wire{ID: "WL", From: "LB.node", To: "R1.1", Connected: true})
	alias := newGraphService(SolveLED, g)
	defer alias.Close()
	state = settled(t, alias)
	for _, id := range []string{"W1", "WL"} {
		w := displayWire(t, state, id)
		if w.Current == nil || math.Abs(*w.Current-state.Result.Current) > 1e-12 {
			t.Fatal("label alias breaks current conservation", w)
		}
	}
}

func TestVoltagePaletteIsSignedBoundedAndUnknownIsDistinct(t *testing.T) {
	p := DefaultDisplayPreferences()
	p.VoltageColor = true
	plus, minus, zero, large := 5., -5., 0., 30.
	if voltageColor(&plus, "live", p) != "#d56a39" || voltageColor(&minus, "live", p) != "#4a6dc7" || voltageColor(&zero, "live", p) != "#70848e" || voltageColor(&large, "live", p) != voltageColor(&plus, "live", p) || voltageColor(nil, "unavailable", p) == voltageColor(&zero, "live", p) {
		t.Fatal("voltage palette loses polarity/bounds/unknown distinction")
	}
}

func TestWireCurrentsCoverNPNMixedAndPotentiometerTerminalConventions(t *testing.T) {
	for _, g := range []Graph{NPNExampleGraph(), NPNPulseExampleGraph(), FlasherExampleGraph(), potGraph(t, "resistor")} {
		s := newGraphService(SolveLED, g)
		state := settled(t, s)
		call(t, s, "simulation.pause", struct{}{}, "pause")
		for _, w := range s.Snapshot().Display.Wires {
			if w.Current == nil || !finite(*w.Current) {
				s.Close()
				t.Fatalf("supported topology missing a conservative wire current: %s, %s", state.Analysis, w.ID)
			}
		}
		if state.NPN != nil {
			w := displayWire(t, s.Snapshot(), "W6")
			if math.Abs(*w.Current-state.NPN.EmitterCurrent) > 1e-12 {
				s.Close()
				t.Fatal("NPN emitter current is not directed out of the device")
			}
		}
		if state.PotentiometerModel != nil {
			var w WireDisplay
			for _, wire := range s.Snapshot().Display.Wires {
				if wire.From == "P1.wiper" {
					w = wire
					break
				}
			}
			if w.Current == nil || math.Abs(*w.Current-.0002) > 1e-10 {
				s.Close()
				t.Fatal("loaded potentiometer wiper wire does not carry its independent 2 V / 10 kΩ load current")
			}
		}
		s.Close()
	}
}
