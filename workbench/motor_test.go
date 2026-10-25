package workbench

import (
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func motorGraph(locked bool, voltage string) Graph {
	b := newExampleBuilder()
	b.part("V1", "dc-source", voltage, -200, 0, 0)
	b.ground("GND", "V1.negative", -200, 180)
	b.part("M1", "dc-motor", "", 100, 0, 0)
	b.graph.Components[len(b.graph.Components)-1].Power.Motor.Locked = locked
	b.wire("V1.positive", "M1.positive")
	b.wire("M1.negative", "GND.reference")
	return b.graph
}
func TestMotorSteadyStateAndLockedRotorRLReference(t *testing.T) {
	for _, locked := range []bool{false, true} {
		g := motorGraph(locked, "12")
		k := batteryKernel(t, g)
		trace := k.trace
		for j := 1; j <= 200; j++ {
			next, updated, err := advanceMixed(k.mixed, trace, float64(j)*.001)
			if err != nil {
				t.Fatal(locked, j, err)
			}
			k.mixed, trace = next, updated
		}
		m := k.mixed.result.Motors[0]
		if locked {
			want := 1.5 * (1 - math.Exp(-.2*8/.01))
			if math.Abs(m.Current-want) > 1e-7 || math.Abs(m.Omega) > 1e-12 {
				t.Fatal("locked RL law", m, want)
			}
		} else {
			want := .02 * 12 / (8*1e-5 + .02*.02)
			if math.Abs(m.Omega-want) > .01 || math.Abs(m.Current-(12-.02*want)/8) > 1e-4 {
				t.Fatal("steady speed/EMF law", m, want)
			}
		}
		if math.Abs(k.mixed.result.PowerResidual) > 1e-7 || k.mixed.result.KCLResidual > 1e-9 {
			t.Fatal("motor conservation", k.mixed.result)
		}
		if len(k.mixed.result.Inductors) != 0 || len(k.mixed.result.Capacitors) != 0 || len(k.mixed.result.Resistors) != 0 || len(k.mixed.result.Sources) != 1 {
			t.Fatal("private motor parts leaked")
		}
		if err := validateRecordedMotors(k.compiled, g, k.mixed.result); err != nil {
			t.Fatal(err)
		}
	}
}
func TestMotorDCOperatingPointAndLoad(t *testing.T) {
	g := motorGraph(false, "12")
	for _, load := range []float64{0, 1e-4, .01} {
		g.Components[2].Power.Motor.Load = load
		r, _, err := solveDCOperatingPoint(t.Context(), g, nil)
		if err != nil {
			t.Fatal(err)
		}
		m := r.Motors[0]
		want := .02 * 12 / (8*(1e-5+load) + .02*.02)
		if math.Abs(m.Omega-want) > 1e-7 || math.Abs(r.PowerResidual) > 1e-7 {
			t.Fatal("load line", m, want, r.PowerResidual)
		}
	}
}

func TestMotorPWM555DrivesShaftAndFlyback(t *testing.T) {
	speeds := []float64{}
	for _, position := range []float64{.05, .5, .95} {
		g := MotorExampleGraph(true)
		for i, p := range g.Components {
			if p.ID == "P1" {
				v := position
				g.Components[i].WiperPosition = &v
			}
		}
		k := batteryKernel(t, g)
		trace := k.trace
		high, low, flyback := false, false, false
		for j := 1; j <= 400; j++ {
			next, updated, err := advanceMixed(k.mixed, trace, float64(j)*.0001)
			if err != nil {
				t.Fatal("PWM", position, j, err)
			}
			k.mixed, trace = next, updated
			r := next.result
			high = high || r.Timers555[0].OutputVoltage > 3
			low = low || r.Timers555[0].OutputVoltage < 1
			flyback = flyback || r.Diodes[0].Current > .01
			if math.Abs(r.PowerResidual) > 1e-6 || r.KCLResidual > 1e-9 {
				t.Fatal("PWM conservation", position, j, r.PowerResidual, r.KCLResidual)
			}
		}
		if !high || !low || !flyback || k.mixed.result.Motors[0].RPM < 500 {
			t.Fatal("PWM did not drive motor/flyback", position, high, low, flyback, k.mixed.result.Motors)
		}
		speeds = append(speeds, k.mixed.result.Motors[0].RPM)
	}
	if speeds[2] < speeds[0]*1.05 {
		t.Fatal("PWM control failed to change shaft speed", speeds)
	}
}

func TestMotorLiveLoadPreservesStorageAndCapture(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "motor-dc"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	settled(t, s)
	call(t, s, "simulation.step", StepParams{Duration: .05}, "spin")
	before := settled(t, s)
	call(t, s, "motor.load.set", MotorLoadParams{ID: "MTR1", Load: .0001}, "drag")
	after := settled(t, s)
	if after.InputWork.MotorLoad == nil || after.InputWork.Status != "completed" || after.RunID != before.RunID || after.Mixed.Time != before.Mixed.Time || after.Mixed.Motors[0].Omega != before.Mixed.Motors[0].Omega || after.Mixed.Motors[0].Current != before.Mixed.Motors[0].Current {
		t.Fatal("load consumed speed/current/time", after.InputWork, after.Mixed.Motors)
	}
	call(t, s, "simulation.step", StepParams{Duration: .05}, "slow")
	slow := settled(t, s)
	if slow.Mixed.Motors[0].RPM >= after.Mixed.Motors[0].RPM*.8 {
		t.Fatal("load failed to slow motor", slow.Mixed.Motors)
	}
	duration, step := .01, .0005
	_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:MTR1", "i:MTR1"}}, "capture")
	record := s.operations[view.Operation.ID].record
	raw, _ := json.Marshal(VerificationDocument{Metadata: record.metadata, Rows: record.rows})
	if _, err := decodeRecording(raw); err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	json.Unmarshal(raw, &object)
	object["metadata"].(map[string]any)["mixed"].(map[string]any)["motors"].([]any)[0].(map[string]any)["rpm"] = 42
	bad, _ := json.Marshal(object)
	if _, err := decodeRecording(bad); err == nil {
		t.Fatal("forged motor RPM")
	}
}

func TestMotorHighSpeedDoesNotUseElectricalVoltageRangeForRPM(t *testing.T) {
	b := newExampleBuilder()
	b.part("VP", "dc-source", "12", -200, 0, 0)
	b.part("VN", "dc-source", "12", -200, 200, 0)
	b.ground("GND", "VP.negative", -400, 160)
	b.wire("VN.positive", "GND.reference")
	b.part("M1", "dc-motor", "", 100, 0, 0)
	m := b.graph.Components[len(b.graph.Components)-1].Power.Motor
	m.Constant = .001
	m.Friction = 1e-7
	m.Resistance = .1
	m.Inductance = 1
	m.Inertia = 1e-7
	b.wire("VP.positive", "M1.positive")
	b.wire("VN.negative", "M1.negative")
	k := batteryKernel(t, b.graph)
	trace := k.trace
	peak := 0.
	for j := 1; j <= 110; j++ {
		next, updated, err := advanceMixed(k.mixed, trace, float64(j)*.01)
		if err != nil {
			t.Fatal(j, err)
		}
		k.mixed, trace = next, updated
		peak = math.Max(peak, next.result.Motors[0].Omega)
		for _, v := range next.result.Nodes {
			if math.Abs(v) > 30 {
				t.Fatal("electrical voltage exceeded original limit")
			}
		}
	}
	if peak < 30000 {
		t.Fatal("fixture did not exercise high angular speed", peak)
	}
}

func TestMotorStagedFailureCancellationAndReturnToCommitted(t *testing.T) {
	for _, mode := range []string{"failed", "canceled", "superseded"} {
		t.Run(mode, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: "motor-dc"}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			settled(t, s)
			call(t, s, "simulation.step", StepParams{Duration: .02}, "spin")
			before := settled(t, s)
			entered, release := blockInputSolver(t, s, mode == "failed")
			defer release()
			promptLiveCommand(t, s, "motor.load.set", MotorLoadParams{"MTR1", .0001})
			awaitLiveSampleBlock(t, entered)
			pending := s.Snapshot()
			if !reflect.DeepEqual(pending.Components, before.Components) || !reflect.DeepEqual(pending.Mixed, before.Mixed) {
				t.Fatal("pending load leaked")
			}
			if mode == "canceled" {
				promptLiveCommand(t, s, "simulation.work.cancel", IDParams{ID: pending.InputWork.ID})
			}
			if mode == "superseded" {
				promptLiveCommand(t, s, "motor.load.set", MotorLoadParams{"MTR1", 0})
			}
			release()
			after := awaitInputStatus(t, s, mode)
			if !reflect.DeepEqual(after.Components, before.Components) || !reflect.DeepEqual(after.Mixed, before.Mixed) || !reflect.DeepEqual(after.UndoEntry, before.UndoEntry) {
				t.Fatal("unpublished work altered accepted state")
			}
		})
	}
}
func TestMotorConfigurationPersistenceUndoAndStrictFields(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "motor-dc"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	cfg := motorDefault()
	cfg.Locked = true
	cfg.Load = .0001
	call(t, s, "motor.configure", MotorParams{"MTR1", cfg}, "configure")
	settled(t, s)
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	settled(t, s)
	p, _ := s.graph.Component("MTR1")
	if p.Power.Motor.Locked {
		t.Fatal("undo motor")
	}
	call(t, s, "edit.redo", HistoryParams{}, "redo")
	settled(t, s)
	path := filepath.Join(t.TempDir(), "motor.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	settled(t, s)
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	a := settled(t, fresh)
	p, _ = fresh.graph.Component("MTR1")
	if *p.Power.Motor != cfg || a.Mixed.Motors[0].Omega != 0 {
		t.Fatal("saved motor configuration")
	}
	a.Components[2].Power.Motor.Load = 0
	if fresh.Snapshot().Components[2].Power.Motor.Load != cfg.Load {
		t.Fatal("configuration alias")
	}
	raw, _ := json.Marshal(cfg)
	var obj map[string]any
	json.Unmarshal(raw, &obj)
	delete(obj, "locked_rotor")
	refuse(t, s, "motor.configure", map[string]any{"id": "MTR1", "parameters": obj}, "invalid-params")
}
