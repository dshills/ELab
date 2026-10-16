package workbench

import (
	"math"
	"reflect"
	"testing"
)

func inductorPotGraph(t *testing.T) Graph {
	t.Helper()
	g := potGraph(t, "capacitor")
	coil, _ := MakeComponent("L1", "inductor", Point{X: 800}, "1 H")
	g.Components = append(g.Components, coil)
	for k := range g.Wires {
		if g.Wires[k].To == "C1.1" {
			g.Wires[k].To = "L1.1"
		}
	}
	g.Wires = append(g.Wires, Wire{ID: "WL", From: "L1.2", To: "C1.1", Connected: true})
	return g
}

func TestInductorSourceAndWiperEventsPreserveBothStorageStates(t *testing.T) {
	for _, wiper := range []bool{false, true} {
		g := RLCExampleGraph()
		if wiper {
			g = inductorPotGraph(t)
		}
		s := newGraphService(SolveLED, g)
		call(t, s, "simulation.pause", struct{}{}, "pause")
		settled(t, s)
		call(t, s, "simulation.step", StepParams{Duration: .001}, "charge")
		before := s.Snapshot()
		if wiper {
			call(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .8}, "wiper")
		} else {
			call(t, s, "simulation.input.set", InputParams{Mode: "low"}, "power-off")
			if !reflect.DeepEqual(before.Mixed, s.Snapshot().Mixed) || s.Snapshot().Transient.PendingMode != "low" {
				t.Fatal("paused source input did not retain accepted state until Step")
			}
			call(t, s, "simulation.step", StepParams{Duration: .001}, "apply-input")
			after := s.Snapshot()
			found := false
			s.mu.Lock()
			samples := cloneTransient(s.transient).Samples
			s.mu.Unlock()
			for _, sample := range samples {
				if sample.Time == before.Transient.Sample.Time && sample.Input == 0 {
					found = true
					values := sample.frame.values
					if math.Abs(values["i:L1"]-before.Mixed.Inductors[0].Current) > 1e-12 || math.Abs(values["v:C1.1"]-values["v:C1.2"]-before.Mixed.Capacitors[0].Voltage) > 1e-12 {
						t.Fatal("source edge changed current or charge at accepted time", sample)
					}
				}
			}
			if !found || after.RunID != before.RunID || after.Transient.PendingMode != "" || after.Mixed.Stale {
				t.Fatal("missing same-time source event or continuation", after.Diagnostics)
			}
			s.Close()
			continue
		}
		after := awaitInputStatus(t, s, "completed")
		if after.RunID != before.RunID || after.Transient.Sample.Time != before.Transient.Sample.Time || math.Abs(after.Mixed.Inductors[0].Current-before.Mixed.Inductors[0].Current) > 1e-12 || math.Abs(after.Mixed.Capacitors[0].Voltage-before.Mixed.Capacitors[0].Voltage) > 1e-12 || after.Mixed.KCLResidual > 1e-9 {
			t.Fatal("same-time actuation changed stored current/charge/time", wiper, before.Mixed, after.Mixed)
		}
		call(t, s, "simulation.step", StepParams{Duration: .001}, "after-event")
		if s.Snapshot().Mixed.Stale {
			t.Fatal("actuated circuit did not continue", wiper, s.Snapshot().Diagnostics)
		}
		s.Close()
	}
}

func TestInductorSwitchEdgePreservesAcceptedCurrentAndCapacitorVoltage(t *testing.T) {
	g := RLCExampleGraph()
	part, _ := MakeComponent("S1", "switch", Point{X: 800}, "")
	closed := true
	part.SwitchClosed = &closed
	g.Components = append(g.Components, part)
	g.Wires = append(g.Wires, Wire{ID: "WS1", From: "S1.1", To: "R1.1", Connected: true}, Wire{ID: "WS2", From: "S1.2", To: "R1.2", Connected: true})
	s := newGraphService(SolveLED, g)
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	call(t, s, "simulation.step", StepParams{Duration: .001}, "charge")
	s.mu.Lock()
	state, trace := cloneMixedState(s.mixedState), cloneTransient(s.transient)
	s.mu.Unlock()
	left, _, err := advanceMixed(state, trace, .002)
	if err != nil {
		t.Fatal(err)
	}
	call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: false}, "open")
	call(t, s, "simulation.step", StepParams{Duration: .001}, "edge")
	after := s.Snapshot()
	if math.Abs(after.Mixed.Inductors[0].Current-left.result.Inductors[0].Current) > 1e-12 || math.Abs(after.Mixed.Capacitors[0].Voltage-left.result.Capacitors[0].Voltage) > 1e-12 || after.Transient.Sample.Time != .002 || after.SwitchStates[0].Pending {
		t.Fatal("switch edge reset memory or missed boundary", after.Diagnostics)
	}
}

func TestInductorCanceledAndSupersededWorkCannotPublishStorageHistory(t *testing.T) {
	for _, command := range []string{"simulation.work.cancel", "simulation.restart", "inductor.initial.set"} {
		t.Run(command, func(t *testing.T) {
			s := newGraphService(SolveLED, RLCExampleGraph())
			defer s.Close()
			call(t, s, "simulation.pause", struct{}{}, "pause")
			settled(t, s)
			call(t, s, "simulation.step", StepParams{Duration: .001}, "charge")
			entered, release := blockFirstLiveSample(t, s, true)
			defer release()
			call(t, s, "simulation.step.start", StepParams{Duration: .01}, "start")
			awaitLiveSampleBlock(t, entered)
			var params any = struct{}{}
			if command == "simulation.work.cancel" {
				params = IDParams{ID: s.Snapshot().StepWork.ID}
			}
			if command == "inductor.initial.set" {
				params = ValueParams{ID: "L1", Value: "10 mA"}
			}
			promptLiveCommand(t, s, command, params)
			before := awaitTestInitialization(t, s)
			release()
			s.liveWorkers.Wait()
			after := s.Snapshot()
			if after.StepWork.Status != "canceled" && after.StepWork.Status != "superseded" || !reflect.DeepEqual(before.Mixed, after.Mixed) || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Components, after.Components) {
				t.Fatal("obsolete numerical work changed L/C history", after.StepWork)
			}
		})
	}
}
