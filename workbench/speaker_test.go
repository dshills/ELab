package workbench

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"math/cmplx"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func speakerTestGraph() Graph {
	b := newExampleBuilder()
	b.part("VIN", "sine-source", ".5", -300, 0, 0)
	b.graph.Components[0].Sine = &SineWaveform{Amplitude: .5, Frequency: 100}
	b.ground("GI", "VIN.negative", -300, 100)
	b.part("RS", "resistor", "10", -100, 0, 0)
	b.part("SP1", "speaker", "", 100, 0, 0)
	b.wire("VIN.positive", "RS.1")
	b.wire("RS.2", "SP1.positive")
	b.ground("GS", "SP1.negative", 100, 100)
	return b.graph
}
func TestSpeakerACAgainstIndependentPassiveImpedance(t *testing.T) {
	g := speakerTestGraph()
	cfg := acSettings("VIN", "v:SP1")
	cfg.Start, cfg.Stop, cfg.Points, cfg.Spacing = "10 Hz", "1000 Hz", 3, "logarithmic"
	r := sweepForTest(t, g, cfg)
	v := DefaultSpeakerConfig()
	cap := 1 / (math.Pow(2*math.Pi*v.Resonance, 2) * v.MotionalInductance)
	if len(r.Points) != 3 {
		t.Fatal("missing AC points")
	}
	for _, p := range r.Points {
		if len(p.AC) != 1 {
			t.Fatal("missing response")
		}
		w := 2 * math.Pi * p.Input
		zm := 1 / (complex(1/v.MotionalResistance, 0) + 1/complex(0, w*v.MotionalInductance) + complex(0, w*cap))
		z := complex(v.Resistance, w*v.Inductance) + zm
		want := z / (10 + z)
		got := complex(p.AC[0].Real, p.AC[0].Imaginary)
		if cmplx.Abs(want-got) > 1e-9 {
			t.Fatal("independent speaker impedance", p.Input, want, got)
		}
	}
	if r.Points[1].Values[0] <= r.Points[0].Values[0] {
		t.Fatal("resonance did not raise load impedance")
	}
}
func speakerReferenceStep(x [3]float64, at, h float64, c SpeakerConfig) [3]float64 {
	capacitance := 1 / (math.Pow(2*math.Pi*c.Resonance, 2) * c.MotionalInductance)
	derivative := func(y [3]float64, t float64) [3]float64 {
		return [3]float64{(.5*math.Sin(2*math.Pi*100*t) - (10+c.Resistance)*y[0] - y[2]) / c.Inductance, y[2] / c.MotionalInductance, (y[0] - y[1] - y[2]/c.MotionalResistance) / capacitance}
	}
	add := func(a, b [3]float64, scale float64) [3]float64 {
		return [3]float64{a[0] + b[0]*scale, a[1] + b[1]*scale, a[2] + b[2]*scale}
	}
	k1 := derivative(x, at)
	k2 := derivative(add(x, k1, h/2), at+h/2)
	k3 := derivative(add(x, k2, h/2), at+h/2)
	k4 := derivative(add(x, k3, h), at+h)
	for i := range x {
		x[i] += h * (k1[i] + 2*k2[i] + 2*k3[i] + k4[i]) / 6
	}
	return x
}
func TestSpeakerTransientAgainstIndependentODEAndConservation(t *testing.T) {
	g := speakerTestGraph()
	state, tr := initializeInductorTest(t, g)
	if len(state.result.Speakers) != 1 || state.result.Speakers[0].Energy != 0 {
		t.Fatal("zero energy startup", state.result)
	}
	reference := [3]float64{}
	cfg := DefaultSpeakerConfig()
	for i := 0; i < 5000; i++ {
		reference = speakerReferenceStep(reference, float64(i)*1e-6, 1e-6, cfg)
	}
	for step := 1; step <= 50; step++ {
		var err error
		state, err = state.advance(tr, float64(step)*.0001)
		if err != nil {
			t.Fatal(err)
		}
		r := state.result.Speakers[0]
		if r.CoilHeat < 0 || r.MotionalHeat < 0 || r.Energy < 0 || math.Abs(state.result.PowerResidual) > 1e-7 || math.Abs(r.Current-r.MotionalCurrent-r.CapacitorCurrent-r.MotionalVoltage/cfg.MotionalResistance) > 1e-9 {
			t.Fatal("speaker conservation", r, state.result.PowerResidual)
		}
	}
	r := state.result.Speakers[0]
	for _, pair := range [][2]float64{{r.Current, reference[0]}, {r.MotionalCurrent, reference[1]}, {r.MotionalVoltage, reference[2]}} {
		if math.Abs(pair[0]-pair[1]) > 2e-7+2e-4*math.Abs(pair[1]) {
			t.Fatal("independent transient reference", pair)
		}
	}
	sample := mixedSample(state.net.c, state.result, 0)
	if math.Abs(sample.Energy-r.Energy) > 1e-15 {
		t.Fatal("scope energy excludes speaker", sample.Energy, r.Energy)
	}
}
func TestSpeakerSharedCapturePersistenceAudioAndRefusals(t *testing.T) {
	s := New()
	defer s.Close()
	player := &fakeAudioPlayer{}
	s.audioPlayer = player
	call(t, s, "example.load", ExampleLoadParams{ID: "speaker-amplifier"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	before := settled(t, s)
	if before.Mixed == nil || len(before.Mixed.Speakers) != 1 || before.DeviceModel == nil || before.DeviceModel.Speaker != speakerDescription || before.Audio != nil || player.plays != 0 {
		t.Fatal("speaker example", before.Diagnostics)
	}
	duration, step := .002, .0001
	_, op := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:SP1", "i:SP1"}}, "verify")
	s.mu.Lock()
	record := s.operations[op.Operation.ID].record
	data, err := json.Marshal(VerificationDocument{Metadata: record.metadata, Rows: record.rows})
	s.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeRecording(data)
	if err != nil {
		t.Fatal("speaker captured state", err)
	}
	if len(decoded.metadata.Mixed.Speakers) != 1 {
		t.Fatal("missing recorded speaker")
	}
	path := filepath.Join(t.TempDir(), "speaker.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	other := New()
	defer other.Close()
	call(t, other, "project.load", LoadParams{Path: path}, "open")
	if !reflect.DeepEqual(before.Components, other.Snapshot().Components) {
		t.Fatal("speaker params changed on load")
	}
	call(t, s, "audio.render", AudioRenderParams{ID: "SP1", Duration: .02, Gain: .1}, "render")
	audio := waitAudio(t, s)
	if audio.Status != "completed" || audio.Playing || player.plays != 0 {
		t.Fatal("speaker preview failed or autoplay", audio)
	}
	if !reflect.DeepEqual(before.Mixed, s.Snapshot().Mixed) {
		t.Fatal("preview changed live circuit")
	}
	call(t, s, "audio.play", IDParams{ID: audio.ID}, "play")
	call(t, s, "audio.stop", IDParams{ID: audio.ID}, "stop")
	peak := 0
	for i := 44; i+1 < len(player.data); i += 2 {
		v := int(int16(binary.LittleEndian.Uint16(player.data[i : i+2])))
		if v < 0 {
			v = -v
		}
		if v > peak {
			peak = v
		}
	}
	if peak < 1000 || peak > 3277 || audio.Signal == "" {
		t.Fatal("speaker normalization/metadata", peak, audio)
	}
	if player.plays != 1 || len(player.data) != 524 {
		t.Fatal("explicit preview playback")
	}
	config := DefaultSpeakerConfig()
	config.Resonance = 150
	call(t, s, "speaker.configure", SpeakerParams{ID: "SP1", Parameters: config}, "configure")
	if !s.Snapshot().Document.Dirty {
		t.Fatal("speaker change not saved intent")
	}
	call(t, s, "edit.undo", struct{}{}, "undo")
	config.Resistance = 0
	refuse(t, s, "speaker.configure", SpeakerParams{ID: "SP1", Parameters: config}, "invalid-params")
	var forged VerificationDocument
	if err := json.Unmarshal(data, &forged); err != nil {
		t.Fatal(err)
	}
	forged.Metadata.Mixed.Speakers[0].Energy += 1
	bad, _ := json.Marshal(forged)
	if _, err := decodeRecording(bad); err == nil {
		t.Fatal("forged speaker energy accepted")
	}
	snapshot := s.Snapshot()
	svg := string(schematicSVG(Graph{Components: snapshot.Components, Wires: snapshot.Wires}, 0, 0))
	if !strings.Contains(svg, "loudspeaker") {
		t.Fatal("speaker export omitted model")
	}
}

func TestOpAmpInductorStartupPreservesIndependentInputCurrentPath(t *testing.T) {
	b := newExampleBuilder()
	b.part("VIN", "dc-source", "1", -300, 0, 0)
	b.ground("GI", "VIN.negative", -300, 100)
	b.part("VP", "dc-source", "5", 300, -100, 0)
	b.ground("GP", "VP.negative", 300, 0)
	b.part("VN", "dc-source", "5", 300, 100, 0)
	b.ground("GN", "VN.positive", 300, 200)
	b.part("U", "opamp", "", 0, 0, 0)
	b.part("RL", "resistor", "1k", 200, 0, 0)
	b.part("L", "inductor", "1H", -100, 0, 0)
	b.wire("VIN.positive", "L.1")
	b.wire("L.2", "U.noninverting")
	b.wire("L.2", "U.inverting")
	b.wire("VP.positive", "U.vplus")
	b.wire("VN.negative", "U.vminus")
	b.wire("U.out", "RL.1")
	b.ground("GL", "RL.2", 200, 100)
	for i, p := range b.graph.Components {
		if p.ID == "L" {
			current := .001
			b.graph.Components[i].InitialCurrent = &current
		}
	}
	c, _, d := b.graph.Compile()
	if d != nil {
		t.Fatal(d)
	}
	n := newMixedNetwork(c)
	_, err := n.solveContext(t.Context(), nil, nil, 0, &Transient{Mode: "auto", Settings: DefaultTransientSettings()}, 0)
	if err == nil || !strings.Contains(err.Error(), "incompatible inductor startup currents") {
		t.Fatal("input current incorrectly joined to op-amp supply/output", err)
	}
}

func TestClassABSpeakerWorkLimitAtomicity(t *testing.T) {
	state, trace := initializeInductorTest(t, ClassABSpeakerExampleGraph())
	beforeX, beforeCaps := append([]float64{}, state.x...), append([]float64{}, state.capV...)
	beforeStep, beforeCeiling := state.inductorStep, state.inductorCeiling
	before, err := json.Marshal(state.result)
	if err != nil {
		t.Fatal(err)
	}
	budget := newIntegrationBudget()
	budget.elapsed = func() time.Duration { return MaxIntegrationWallTime + time.Nanosecond }
	next, err := state.advanceBudget(trace, .0001, budget)
	var limit *integrationBudgetError
	after, marshalErr := json.Marshal(state.result)
	if next != nil || !errors.As(err, &limit) || limit.code != "integration-time-limit" || marshalErr != nil || string(before) != string(after) || !reflect.DeepEqual(beforeX, state.x) || !reflect.DeepEqual(beforeCaps, state.capV) || state.inductorStep != beforeStep || state.inductorCeiling != beforeCeiling {
		t.Fatal("deadline failed to preserve accepted state", next, err, marshalErr)
	}
}

func TestSpeakerPrivateStateAndReferencedSchemaGuards(t *testing.T) {
	state, _ := initializeInductorTest(t, speakerTestGraph())
	if len(state.result.Capacitors) != 0 || len(state.result.Inductors) != 0 {
		t.Fatal("private speaker storage exposed/double counted")
	}
	n := *state.net
	n.index = map[string]int{}
	for key, value := range state.net.index {
		n.index[key] = value
	}
	delete(n.index, speakerPrivate("SP1", "motion"))
	x := append([]float64{}, state.x...)
	before := append([]float64{}, x...)
	if err := n.restoreSpeakers(state.result, x); err == nil || !reflect.DeepEqual(before, x) {
		t.Fatal("missing private node overwrote another state")
	}
	p, _ := state.net.authored.Component("SP1")
	if finite(n.speakerResult(p, x).Energy) {
		t.Fatal("missing speaker state silently reported")
	}
	caps, err := detailedCapabilities("speaker.configure")
	if err != nil || len(caps) != 1 {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(SchemaRegistry()["urn:elab:speaker-config:1"], &cfg); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"coil_resistance_ohm", "coil_inductance_h", "resonance_hz", "motional_resistance_ohm", "motional_inductance_h"} {
		property := inputProperty(cfg, name)
		if property == nil || property["minimum"] == nil || property["maximum"] == nil || property["x-unit"] == nil {
			t.Fatal("referenced schema lacks bounds", name)
		}
	}
}

func TestSpeakerDoesNotOverrideTimerAnalysis(t *testing.T) {
	d, ok := findExample("555-tone")
	if !ok {
		t.Fatal("missing timer fixture")
	}
	g := d.graph.Clone()
	replaced := false
	for i, p := range g.Components {
		if p.Kind == "piezo" {
			v, err := MakeComponent(p.ID, "speaker", p.Position, "")
			if err != nil {
				t.Fatal(err)
			}
			v.Rotation = p.Rotation
			g.Components[i] = v
			replaced = true
		}
	}
	if !replaced {
		t.Fatal("missing sounder fixture")
	}
	for pass := 0; pass < 2; pass++ {
		c, _, err := g.Compile()
		if err != nil || c.Analysis != Timer555Analysis {
			t.Fatal("speaker overwrote latch analysis", c.Analysis, err)
		}
		for i, j := 0, len(g.Components)-1; i < j; i, j = i+1, j-1 {
			g.Components[i], g.Components[j] = g.Components[j], g.Components[i]
		}
	}
}
func TestSpeakerConstantTerminalVoltagePreviewHasNoFilterStartupStep(t *testing.T) {
	b := newExampleBuilder()
	b.part("V1", "dc-source", "1", -200, 0, 0)
	b.ground("GI", "V1.negative", -200, 100)
	b.part("SP1", "speaker", "", 100, 0, 0)
	b.wire("V1.positive", "SP1.positive")
	b.ground("GS", "SP1.negative", 100, 100)
	c, _, d := b.graph.Compile()
	if d != nil {
		t.Fatal(d)
	}
	p, _ := b.graph.Component("SP1")
	data, err := renderAudio(t.Context(), c, DefaultTransientSettings(), p, AudioRenderParams{ID: "SP1", Duration: .02, Gain: .1})
	if err != nil {
		t.Fatal(err)
	}
	for i := 44; i < len(data); i++ {
		if data[i] != 0 {
			t.Fatal("constant-voltage preview synthesized a filter step")
		}
	}
}

func TestSpeakerPreviewDoesNotNormalizeNumericalResidue(t *testing.T) {
	values := make([]float64, 240)
	pcm := make([]int16, len(values))
	for i := range values {
		values[i] = 1e-12 * math.Sin(float64(i))
	}
	fillSpeakerPCM(pcm, values, .25)
	for _, v := range pcm {
		if v != 0 {
			t.Fatal("numerical residue amplified")
		}
	}
	for i := range values {
		values[i] = .01 * math.Sin(float64(i))
	}
	fillSpeakerPCM(pcm, values, .1)
	peak := 0
	for _, v := range pcm {
		a := int(v)
		if a < 0 {
			a = -a
		}
		if a > peak {
			peak = a
		}
	}
	if peak < 1000 || peak > 3277 {
		t.Fatal("meaningful preview lost normalization", peak)
	}
}

func TestSpeakerMissingConfigurationRejectedBeforeLowering(t *testing.T) {
	g := speakerTestGraph()
	for i, p := range g.Components {
		if p.Kind == "speaker" {
			g.Components[i].Device = nil
		}
	}
	if _, _, d := g.Compile(); d == nil {
		t.Fatal("unconfigured speaker admitted to lowering")
	}
	var input map[string]any
	caps, err := detailedCapabilities("speaker.configure")
	if err != nil || len(caps) != 1 {
		t.Fatal("speaker capability count", len(caps), err)
	}
	if err := json.Unmarshal(caps[0].Input, &input); err != nil {
		t.Fatal(err)
	}
	if p := inputProperty(input, "parameters"); p == nil || p["$ref"] != "urn:elab:speaker-config:1" {
		t.Fatal("command does not reference the bounded speaker config", p)
	}
}
