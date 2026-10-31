package workbench

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dshills/elab/internal/testutil"
)

func TestExtendedPublicCaptureSaveLoadAndCorruption(t *testing.T) {
	for _, id := range []string{"buck", "opamp-slew", "opamp-bandwidth", "shift-register", "555-tone", "light-alarm"} {
		t.Run(id, func(t *testing.T) {
			if id == "buck" && testutil.RunAtProductionSpeed(t) {
				return
			}
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: id}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			before := settled(t, s)
			if before.Document.Dirty || before.DeviceModel == nil || before.Mixed == nil {
				t.Fatal("template", before.Diagnostics)
			}
			d, _ := findExample(id)
			duration, step := d.settings.StepSeconds*2, d.settings.StepSeconds
			_, v := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: d.probes}, "verify")
			rec := s.operations[v.Operation.ID].record
			data, _ := json.Marshal(VerificationDocument{Metadata: rec.metadata, Rows: rec.rows})
			decoded, err := decodeRecording(data)
			if err != nil {
				t.Fatal("decode capture", err)
			}
			if recordingModelHashes(decoded.metadata)[ExtendedModelID] == "" {
				t.Fatal("missing model digest")
			}
			after := s.Snapshot()
			if !reflect.DeepEqual(before.Mixed, after.Mixed) {
				t.Fatal("offline solve changed live state")
			}
			call(t, s, "recording.keep.set", RecordingKeepParams{ID: v.Operation.ID, Keep: true}, "keep")
			path := filepath.Join(t.TempDir(), id+".elab")
			call(t, s, "project.save", SaveParams{Path: path}, "save")
			fresh := New()
			defer fresh.Close()
			call(t, fresh, "project.load", LoadParams{Path: path}, "open")
			r := settled(t, fresh)
			if r.DeviceModel == nil || r.Document.Dirty || r.Document.RecordingCount != 1 || !reflect.DeepEqual(before.Components, r.Components) {
				t.Fatal("save roundtrip", r.Document)
			}
			raw, _ := os.ReadFile(path)
			var obj map[string]any
			json.Unmarshal(raw, &obj)
			delete(obj, "device_model")
			bad, _ := json.Marshal(obj)
			if _, err := decodeDocument(bad); err == nil {
				t.Fatal("missing model accepted")
			}
			var capture map[string]any
			json.Unmarshal(data, &capture)
			m := capture["metadata"].(map[string]any)
			model := m["device_model"].(map[string]any)
			model["id"] = "forged"
			bad, _ = json.Marshal(capture)
			if _, err := decodeRecording(bad); err == nil {
				t.Fatal("forged model accepted")
			}
			snap := s.Snapshot()
			snap.DeviceModel.ID = "changed"
			for _, p := range snap.Components {
				if p.Device != nil && p.Device.PWM != nil {
					p.Device.PWM.Duty = .8
				}
			}
			if s.Snapshot().DeviceModel.ID != ExtendedModelID {
				t.Fatal("model alias")
			}
		})
	}
}

type fakeAudioPlayer struct {
	plays, stops int
	err          error
	data         []byte
}

func (p *fakeAudioPlayer) Play(b []byte) error {
	if p.err != nil {
		return p.err
	}
	p.plays++
	p.data = append([]byte(nil), b...)
	return nil
}
func (p *fakeAudioPlayer) Stop() { p.stops++ }
func waitAudio(t *testing.T, s *Service) *AudioPreview {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		a := s.Snapshot().Audio
		if a != nil && a.Status != "running" {
			return a
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("audio deadline")
	return nil
}
func TestAudioPublicNoAutoplayWAVPlaybackStopAndStale(t *testing.T) {
	s := New()
	defer s.Close()
	player := &fakeAudioPlayer{}
	s.audioPlayer = player
	call(t, s, "example.load", ExampleLoadParams{ID: "light-alarm"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "sensor.light.set", LightParams{ID: "LDR1", Illuminance: 1}, "dark")
	settled(t, s)
	before := s.Snapshot()
	call(t, s, "audio.render", AudioRenderParams{ID: "BZ1", Duration: .02, Gain: .1}, "render")
	a := waitAudio(t, s)
	if a.Status != "completed" || a.Playing || player.plays != 0 {
		t.Fatal("render autoplay/failed", a)
	}
	after := s.Snapshot()
	if !reflect.DeepEqual(before.Mixed, after.Mixed) || before.Document.Dirty != after.Document.Dirty {
		t.Fatal("audio changed live experiment")
	}
	id := a.ID
	call(t, s, "audio.play", IDParams{ID: id}, "play")
	call(t, s, "audio.stop", IDParams{ID: id}, "stop")
	call(t, s, "audio.play", IDParams{ID: id}, "replay")
	call(t, s, "audio.stop", IDParams{ID: id}, "stop2")
	if player.plays != 2 || len(player.data) != 524 {
		t.Fatal("player", player.plays, len(player.data))
	}
	for _, offset := range []int{44, len(player.data) - 2} {
		if binary.LittleEndian.Uint16(player.data[offset:offset+2]) != 0 {
			t.Fatal("missing edge fade")
		}
	}
	rms := 0.
	for j := 44; j < len(player.data); j += 2 {
		v := float64(int16(binary.LittleEndian.Uint16(player.data[j:j+2]))) / 32767
		rms += v * v
	}
	rms = math.Sqrt(rms / 240)
	if rms < .02 || rms > .1 {
		t.Fatal("powered buzzer audio", rms)
	}
	path := filepath.Join(t.TempDir(), "tone.wav")
	call(t, s, "export.start", ExportStartParams{Kind: "audio-wav", ID: id, Path: path}, "export")
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if s.Snapshot().ProjectWork.Status == "completed" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	data, err := os.ReadFile(path)
	if err != nil || !reflect.DeepEqual(data, player.data) {
		t.Fatal("WAV export", err)
	}
	call(t, s, "component.set", ValueParams{ID: "RDIV", Value: "20k"}, "edit")
	if s.Snapshot().Audio.Current {
		t.Fatal("stale audio current")
	}
	r := s.Snapshot().CommandRevision
	raw, _ := json.Marshal(IDParams{ID: id})
	out := s.Dispatch(context.Background(), Request{Version: 1, ID: "stale", Key: "stale", Command: "audio.play", Params: raw, ExpectedCommandRevision: &r}, "agent")
	if out.Error == nil || player.plays != 2 {
		t.Fatal("stale preview played")
	}
}

func TestExtendedAutomaticPlacementAndConfigurationBatch(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	for k, kind := range []string{"pwm-controller", "shift-register", "piezo", "active-buzzer"} {
		call(t, s, "component.place", PlaceParams{Kind: kind, Position: Point{X: float64(k) * 200}}, kind)
	}
	state := s.Snapshot()
	want := []string{"U1", "U2", "BZ1", "BZ2"}
	for k, p := range state.Components {
		if p.ID != want[k] {
			t.Fatal("automatic ID", p.ID, want[k])
		}
	}
	call(t, s, "pwm.configure", PWMParams{ID: "U1", Parameters: PWMConfig{100, .6}}, "pwm")
	call(t, s, "sound.configure", SoundParams{ID: "BZ1", Parameters: SoundConfig{30e-9, 2000}}, "sound")
	original := s.Snapshot()
	raw, _ := json.Marshal(DuplicateParams{ID: "BZ1", Position: &Point{X: 900}})
	rev := original.CommandRevision
	out := s.Dispatch(t.Context(), Request{Version: 1, ID: "duplicate", Key: "duplicate", Command: "component.duplicate", Params: raw, ExpectedCommandRevision: &rev}, "agent")
	if out.Error != nil {
		t.Fatal(out.Error)
	}
	graph := Graph{Components: s.Snapshot().Components}
	copy, ok := graph.Component("BZ3")
	if !ok || *copy.Device.Sound != (SoundConfig{30e-9, 2000}) {
		t.Fatal("duplicate lost sound parameters")
	}
	call(t, s, "component.place", PlaceParams{Kind: "opamp", ID: "U3", Position: Point{X: 1200}}, "amp")
	encode := func(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
	call(t, s, "edit.batch", BatchParams{Name: "Configure devices", Edits: []BatchEdit{
		{Command: "pwm.configure", Params: encode(PWMParams{ID: "U1", Parameters: PWMConfig{200, .4}})},
		{Command: "sound.configure", Params: encode(SoundParams{ID: "BZ1", Parameters: SoundConfig{40e-9, 1000}})},
		{Command: "opamp.dynamics.set", Params: encode(DynamicsParams{ID: "U3", Enabled: true, Parameters: OpAmpDynamics{2000, 3000}})},
	}}, "batch")
	graph = Graph{Components: s.Snapshot().Components}
	u, _ := graph.Component("U3")
	if u.Device == nil || u.Device.Dynamics.GBW != 2000 {
		t.Fatal("batch did not configure dynamics")
	}
	call(t, s, "edit.undo", struct{}{}, "undo-batch")
	graph = Graph{Components: s.Snapshot().Components}
	u, _ = graph.Component("U3")
	if u.Device != nil {
		t.Fatal("undo retained optional dynamics")
	}

	for _, kind := range []string{"pwm-controller", "shift-register", "piezo", "active-buzzer"} {
		part, _ := MakeComponent("PART", kind, Point{}, "")
		seen := map[string]bool{}
		for _, probe := range probeInventory(Graph{Components: []Component{part}}) {
			if seen[probe.ID] {
				t.Fatal("duplicate probe", probe.ID)
			}
			seen[probe.ID] = true
		}
	}
}

func TestShiftPublicSwitchBoundariesPreserveClockAndStorage(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "shift-register"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	settled(t, s)
	call(t, s, "simulation.step", StepParams{Duration: .25}, "fill")
	before := settled(t, s)
	want := before.Mixed.Shifts[0].Shifted
	if want == 0 {
		t.Fatal("no shifted data")
	}
	call(t, s, "switch.set", SwitchParams{ID: "SLATCH", Closed: true}, "latch")
	call(t, s, "simulation.step", StepParams{Duration: .0025}, "apply-latch")
	after := settled(t, s)
	if after.Mixed.Shifts[0].Latched != want || after.RunID != before.RunID || math.Abs(after.Mixed.Time-before.Mixed.Time-.0025) > 1e-12 {
		t.Fatal("latch boundary", after.Mixed.Shifts, before.Mixed.Time, after.Mixed.Time)
	}
	call(t, s, "switch.set", SwitchParams{ID: "SOE", Closed: true}, "disable")
	call(t, s, "simulation.step", StepParams{Duration: .0025}, "apply-disable")
	after = settled(t, s)
	if after.Mixed.Shifts[0].Enabled || after.Mixed.Shifts[0].Latched != want {
		t.Fatal("OE lost latched data")
	}
	call(t, s, "switch.set", SwitchParams{ID: "SCLEAR", Closed: true}, "clear")
	call(t, s, "simulation.step", StepParams{Duration: .0025}, "apply-clear")
	after = settled(t, s)
	if after.Mixed.Shifts[0].Shifted != 0 || after.Mixed.Shifts[0].Latched != want || after.RunID != before.RunID {
		t.Fatal("clear lost output storage", after.Mixed.Shifts)
	}
}

func TestAudioBackendFailurePreservesAcceptedPlayback(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "light-alarm"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	call(t, s, "audio.render", AudioRenderParams{ID: "BZ1", Start: .0001, Duration: .02001, Gain: .1}, "render")
	a := waitAudio(t, s)
	if a.Status != "completed" || a.Start != 1.0/audioRate || a.Duration != .02 {
		t.Fatal(a)
	}
	player := &fakeAudioPlayer{err: errors.New("isolated unavailable backend")}
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.audioPlayer = player
	s.audio.view.Playing = true
	s.audioPlaybackCancel = cancel
	s.mu.Unlock()
	before := s.Snapshot()
	raw, _ := json.Marshal(IDParams{ID: a.ID})
	rev := before.CommandRevision
	r := s.Dispatch(t.Context(), Request{Version: 1, ID: "play-fail", Key: "play-fail", Command: "audio.play", Params: raw, ExpectedCommandRevision: &rev}, "agent")
	if r.Error == nil || r.Error.Code != "audio-unavailable" || !reflect.DeepEqual(before, s.Snapshot()) || ctx.Err() != nil || player.stops != 0 {
		t.Fatal("failed replacement changed accepted playback", r.Error)
	}
}

func TestShiftIntermediateDataReportsPublicTerminal(t *testing.T) {
	b := newExampleBuilder()
	b.part("VCC", "dc-source", "5", -600, 0, 0)
	b.ground("GV", "VCC.negative", -600, 150)
	b.part("DATA", "dc-source", "2.5", -400, 0, 0)
	b.ground("GD", "DATA.negative", -400, 150)
	b.part("CLK", "finite-pulse-source", "5", -200, 0, 0)
	b.graph.Components[len(b.graph.Components)-1].Pulse = &PulseWaveform{HighVoltage: 5, HighSeconds: .001, LowSeconds: .001, RiseSeconds: .0001, FallSeconds: .0001}
	b.ground("GC", "CLK.negative", -200, 150)
	b.part("U1", "shift-register", "", 100, 0, 0)
	for _, pin := range []string{"vcc", "reset_n"} {
		b.wire("VCC.positive", "U1."+pin)
	}
	for _, pin := range []string{"gnd", "latch", "oe_n"} {
		b.wire("VCC.negative", "U1."+pin)
	}
	b.wire("DATA.positive", "U1.data")
	b.wire("CLK.positive", "U1.clk")
	c, status, d := b.graph.Compile()
	if status != "" || d != nil {
		t.Fatal(status, d)
	}
	k, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	err = k.advance(t.Context(), .0001)
	if err == nil || !strings.Contains(err.Error(), "U1: DATA") || strings.Contains(err.Error(), "$U1") {
		t.Fatal("private/absent diagnostic", err)
	}
}
