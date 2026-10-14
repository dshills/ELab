package workbench

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func waveformRCGraph(t *testing.T, kind string) Graph {
	t.Helper()
	g := RCExampleGraph()
	source, err := MakeComponent("V1", kind, Point{-200, 0}, "")
	if err != nil {
		t.Fatal(err)
	}
	g.Components[0] = source
	return g
}
func sineRCReference(at, amplitude, offset, frequency, phase, tau, initial float64) float64 {
	omega := 2 * math.Pi * frequency
	x := omega * tau
	phi := phase * math.Pi / 180
	steady := func(t float64) float64 {
		return offset + amplitude*(math.Sin(omega*t+phi)-x*math.Cos(omega*t+phi))/(1+x*x)
	}
	return steady(at) + (initial-steady(0))*math.Exp(-at/tau)
}
func TestSineWaveformAndRCIndependentReference(t *testing.T) {
	s := newGraphService(SolveLED, waveformRCGraph(t, "sine-source"))
	defer s.Close()
	settled(t, s)
	cfg := SineWaveform{Amplitude: 2, Offset: 1, Frequency: 20, Phase: 30}
	call(t, s, "source.sine.set", SineParams{ID: "V1", Waveform: cfg}, "sine")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	for i := 1; i <= 30; i++ {
		call(t, s, "simulation.step", StepParams{Duration: .001}, fmtWire(i))
		state := s.Snapshot()
		at := state.Transient.Sample.Time
		want := sineRCReference(at, 2, 1, 20, 30, .01, 0)
		voltage := sourceVoltageAt(state.Components[0], s.transient, at, true)
		if state.Transient.Engine != WaveformEngineID || math.Abs(state.Mixed.Capacitors[0].Voltage-want) > .003 || math.Abs(state.Mixed.Sources[0].Voltage-voltage) > 1e-9 || math.Abs(state.Transient.Sample.Input-voltage) > 1e-9 || state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
			t.Fatalf("sine t=%g got %g reference %g input %g/%g", at, state.Mixed.Capacitors[0].Voltage, want, state.Transient.Sample.Input, voltage)
		}
	}
	before := s.Snapshot()
	duration := .03
	_, record := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:V1.positive", "v:C1", "i:R1"}}, "verify")
	if record.Metadata.WaveformModel == nil || record.Metadata.Engine != WaveformEngineID || math.Abs(record.Metadata.Mixed.Capacitors[0].Voltage-before.Mixed.Capacitors[0].Voltage) > 1e-10 {
		t.Fatal("sine live/Verify/model differs")
	}
}
func TestFinitePulseRampsCornersAndRCReference(t *testing.T) {
	g := waveformRCGraph(t, "finite-pulse-source")
	pulse := PulseWaveform{HighVoltage: 5, LowVoltage: 0, HighSeconds: .007, LowSeconds: .013, RiseSeconds: .002, FallSeconds: .003}
	g.Components[0].Pulse = &pulse
	source := g.Components[0]
	trace := &Transient{Mode: "auto", Settings: DefaultTransientSettings()}
	for _, p := range []struct{ at, want float64 }{{0, 0}, {.001, 2.5}, {.002, 5}, {.007, 5}, {.0085, 2.5}, {.01, 0}, {.02, 0}, {.021, 2.5}} {
		if got := sourceVoltageAt(source, trace, p.at, true); math.Abs(got-p.want) > 1e-10 {
			t.Fatalf("pulse at %g: %g != %g", p.at, got, p.want)
		}
	}
	s := newGraphService(SolveLED, g)
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	call(t, s, "simulation.step", StepParams{Duration: .005}, "charge")
	state := s.Snapshot()
	// Exact linear-ramp followed by constant-input first-order RC response.
	want := 5 * (1 - (.01/.002)*(math.Exp(-(.005-.002)/.01)-math.Exp(-.005/.01)))
	if math.Abs(state.Mixed.Capacitors[0].Voltage-want) > .003 {
		t.Fatalf("ramp RC got %g reference %g", state.Mixed.Capacitors[0].Voltage, want)
	}
	call(t, s, "simulation.step", StepParams{Duration: .007}, "fall")
	state = s.Snapshot()
	for _, corner := range []float64{.002, .007, .01} {
		found := false
		for _, event := range state.Transient.Events {
			if event.Mode == "pulse:V1:corner" && math.Abs(event.Time-corner) < 1e-12 {
				found = true
			}
		}
		if !found {
			t.Fatalf("pulse corner %g omitted", corner)
		}
	}
	if math.Abs(state.Mixed.Sources[0].Voltage) > 1e-12 {
		t.Fatal("finite pulse failed low hold")
	}
}
func TestIndependentPulsePeriodsAndWholeSampleBudget(t *testing.T) {
	g := waveformRCGraph(t, "finite-pulse-source")
	other, err := MakeComponent("V2", "finite-pulse-source", Point{-400, 0}, "")
	if err != nil {
		t.Fatal(err)
	}
	second := PulseWaveform{HighVoltage: 2, LowVoltage: 1, HighSeconds: .004, LowSeconds: .006, RiseSeconds: .001, FallSeconds: .001}
	other.Pulse = &second
	other.Value = 2
	resistor, err := MakeComponent("R2", "resistor", Point{-400, 120}, "10k")
	if err != nil {
		t.Fatal(err)
	}
	g.Components = append(g.Components, other, resistor)
	for i, pair := range [][2]string{{"V2.positive", "R2.1"}, {"R2.2", "GND.reference"}, {"V2.negative", "GND.reference"}} {
		g.Wires = append(g.Wires, Wire{ID: "X" + fmtWire(i), From: pair[0], To: pair[1], Connected: true})
	}
	s := newGraphService(SolveLED, g)
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	call(t, s, "simulation.step", StepParams{Duration: .006}, "step")
	state := s.Snapshot()
	if state.Mixed.Sources[0].Voltage != 5 || state.Mixed.Sources[1].Voltage != 1 {
		t.Fatal("per-source levels/periods were collapsed into the global pulse")
	}
	event := false
	for _, e := range state.Transient.Events {
		if e.Mode == "pulse:V2:corner" && math.Abs(e.Time-.004) < 1e-12 {
			event = true
		}
	}
	if !event {
		t.Fatal("second-source corner absent")
	}
	// A deliberately coarse memoryless output sample crosses >512 pulse corners.
	// The staged call must refuse and preserve all original physical state.
	g = waveformRCGraph(t, "finite-pulse-source")
	g.Components = append(g.Components[:2], g.Components[3:]...)
	g.Wires = []Wire{{ID: "A", From: "V1.positive", To: "R1.1", Connected: true}, {ID: "B", From: "R1.2", To: "GND.reference", Connected: true}, {ID: "C", From: "V1.negative", To: "GND.reference", Connected: true}}
	fast := PulseWaveform{HighVoltage: 5, LowVoltage: 0, HighSeconds: .0005, LowSeconds: .0005, RiseSeconds: .0001, FallSeconds: .0001}
	g.Components[0].Pulse = &fast
	compiled, _, d := g.Compile()
	if d != nil {
		t.Fatal(d)
	}
	n := newMixedNetwork(compiled)
	tr := &Transient{Mode: "auto", Settings: DefaultTransientSettings()}
	initial, err := n.solve(nil, nil, 0, tr, 0)
	if err != nil {
		t.Fatal(err)
	}
	original, _ := json.Marshal(initial.result)
	_, _, err = advanceMixed(initial, tr, .2)
	after, _ := json.Marshal(initial.result)
	if err == nil || !reflect.DeepEqual(after, original) || len(tr.Events) != 0 || len(tr.Samples) != 0 {
		t.Fatal("coarse waveform sample bypassed budget or published partial state")
	}
}
func TestWaveformStrictCommandsModelAndProjectRecords(t *testing.T) {
	s := newGraphService(SolveLED, waveformRCGraph(t, "finite-pulse-source"))
	defer s.Close()
	settled(t, s)
	pulse := PulseWaveform{HighVoltage: -1, LowVoltage: -3, HighSeconds: .02, LowSeconds: .03, RiseSeconds: .001, FallSeconds: .002}
	call(t, s, "source.pulse.set", PulseParams{ID: "V1", Waveform: pulse}, "set")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	duration := .005
	verify(t, s, VerifyParams{Duration: &duration}, "verify")
	path := filepath.Join(t.TempDir(), "pulse.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	loaded := fresh.Snapshot()
	if loaded.WaveformModel == nil || loaded.Document.RecordingCount != 1 || loaded.Components[0].Pulse == nil || *loaded.Components[0].Pulse != pulse {
		t.Fatal("negative-level waveform or recording lost in project")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(map[string]any){
		func(d map[string]any) { delete(d, "waveform_model") },
		func(d map[string]any) { d["waveform_model"].(map[string]any)["maximum_magnitude_v"] = 100 },
		func(d map[string]any) {
			d["graph"].(map[string]any)["components"].([]any)[0].(map[string]any)["pulse"].(map[string]any)["rise_s"] = 0
		},
		func(d map[string]any) {
			delete(d["graph"].(map[string]any)["components"].([]any)[0].(map[string]any)["pulse"].(map[string]any), "low_v")
		},
	} {
		var d map[string]any
		if err := json.Unmarshal(data, &d); err != nil {
			t.Fatal(err)
		}
		change(d)
		bad, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		badPath := filepath.Join(t.TempDir(), "bad.elab")
		if err := os.WriteFile(badPath, bad, 0600); err != nil {
			t.Fatal(err)
		}
		refuse(t, fresh, "project.load", LoadParams{Path: badPath}, "invalid-document")
	}
	badPulse := pulse
	badPulse.RiseSeconds = 0
	refuse(t, s, "source.pulse.set", PulseParams{ID: "V1", Waveform: badPulse}, "invalid-params")
	refuse(t, s, "source.sine.set", SineParams{ID: "V1", Waveform: DefaultSineWaveform()}, "invalid-params")
}

func TestWaveformFitSamplingValidationAndCopies(t *testing.T) {
	s := newGraphService(SolveLED, waveformRCGraph(t, "finite-pulse-source"))
	defer s.Close()
	settled(t, s)
	before := s.Snapshot()
	if len(before.SourceWarnings) != 1 {
		t.Fatal("coarse finite-ramp output sampling was not explained")
	}
	call(t, s, "simulation.sampling.fit", struct{}{}, "fit")
	state := s.Snapshot()
	if state.TransientSettings.StepSeconds != .00025 || len(state.SourceWarnings) != 0 {
		t.Fatal("shared fit sampling did not resolve the source warning")
	}
	for i := range state.Components {
		if state.Components[i].Pulse != nil {
			state.Components[i].Pulse.RiseSeconds = .1
		}
	}
	if *s.Snapshot().Components[0].Pulse != *before.Components[0].Pulse {
		t.Fatal("waveform snapshot aliases source parameters")
	}
	sine := DefaultSineWaveform()
	sine.Frequency = 1000
	g := waveformRCGraph(t, "sine-source")
	g.Components[0].Sine = &sine
	other := newGraphService(SolveLED, g)
	defer other.Close()
	call(t, other, "simulation.sampling.fit", struct{}{}, "fit")
	fitted := other.Snapshot().TransientSettings
	if fitted.StepSeconds != 1./32000 || fitted.PlaybackSpeed/fitted.StepSeconds > 1000 {
		t.Fatal("sine sampling fit broke playback/sample limit")
	}
	bad := sine
	bad.Offset = 12
	refuse(t, other, "source.sine.set", SineParams{ID: "V1", Waveform: bad}, "invalid-params")
	cfg := other.Snapshot().TransientSettings
	cfg.StepSeconds = .1
	call(t, other, "simulation.transient.set", cfg, "coarse") // no legacy pulse phase coupling
}
func TestFinitePulseNPNSwitchMatchesDCPhases(t *testing.T) {
	g := NPNExampleGraph()
	for i, p := range g.Components {
		if p.ID == "VIN" {
			source, err := MakeComponent(p.ID, "finite-pulse-source", p.Position, "5V")
			if err != nil {
				t.Fatal(err)
			}
			g.Components[i] = source
		}
	}
	s := newGraphService(SolveLED, g)
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	call(t, s, "simulation.input.set", InputParams{Mode: "high"}, "high")
	call(t, s, "simulation.step", StepParams{Duration: .001}, "on")
	on := s.Snapshot()
	reference := newGraphService(SolveLED, NPNExampleGraph())
	defer reference.Close()
	dc := settled(t, reference)
	if on.Mixed == nil || math.Abs(on.Mixed.LEDs[0].Current-dc.NPN.CollectorCurrent) > 1e-9 {
		t.Fatal("finite source/NPN high phase differs from DC reference")
	}
	call(t, s, "simulation.input.set", InputParams{Mode: "low"}, "low")
	call(t, s, "simulation.step", StepParams{Duration: .001}, "off")
	if s.Snapshot().Mixed.LEDs[0].Current > 1e-8 {
		t.Fatal("finite source/NPN low phase failed cutoff")
	}
}

func TestSineCustomParametersRoundTripAndForeignFieldRefusal(t *testing.T) {
	s := newGraphService(SolveLED, waveformRCGraph(t, "sine-source"))
	defer s.Close()
	settled(t, s)
	cfg := SineWaveform{Amplitude: 2.3, Offset: -.5, Frequency: 17, Phase: -120.5}
	call(t, s, "source.sine.set", SineParams{ID: "V1", Waveform: cfg}, "set")
	path := filepath.Join(t.TempDir(), "sine.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	if got := fresh.Snapshot().Components[0].Sine; got == nil || *got != cfg {
		t.Fatal("sine parameters did not round trip")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, foreign := range []bool{false, true} {
		var d map[string]any
		if err := json.Unmarshal(data, &d); err != nil {
			t.Fatal(err)
		}
		parts := d["graph"].(map[string]any)["components"].([]any)
		if foreign {
			parts[1].(map[string]any)["sine"] = parts[0].(map[string]any)["sine"]
		} else {
			delete(parts[0].(map[string]any)["sine"].(map[string]any), "phase_degrees")
		}
		bad, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		badPath := filepath.Join(t.TempDir(), "bad.elab")
		if err := os.WriteFile(badPath, bad, 0600); err != nil {
			t.Fatal(err)
		}
		refuse(t, fresh, "project.load", LoadParams{Path: badPath}, "invalid-document")
	}
}
func TestConstantWaveformAvoidsArtificialWorkAndSamplingWarning(t *testing.T) {
	p, err := MakeComponent("V1", "finite-pulse-source", Point{}, "")
	if err != nil {
		t.Fatal(err)
	}
	flat := PulseWaveform{HighVoltage: 1, LowVoltage: 1, HighSeconds: .0005, LowSeconds: .0005, RiseSeconds: .0001, FallSeconds: .0001}
	p.Pulse = &flat
	p.Value = 1
	tr := &Transient{Mode: "auto", Settings: DefaultTransientSettings()}
	if !math.IsInf(nextSourceCorner(p, tr, 0, true), 1) || !math.IsInf(waveformRecommendedStep(p), 1) || sourceVoltageAt(p, tr, .789, true) != 1 {
		t.Fatal("constant waveform generated artificial corners or detail warnings")
	}
}
