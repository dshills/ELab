package workbench

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestTimer555InitialExamples(t *testing.T) {
	for _, g := range []Graph{Timer555FlasherExampleGraph(), Timer555OscillatorExampleGraph(), Timer555OneShotExampleGraph()} {
		c, status, d := g.Compile()
		if d != nil {
			t.Fatal(status, d)
		}
		k, err := newSimulationKernel(context.Background(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
		if err != nil {
			t.Fatal(err)
		}
		if len(k.mixed.result.Timers555) != 1 || !k.mixed.result.Timers555[0].High {
			t.Fatal("555 startup", k.mixed.result)
		}
		for step := 1; step <= 900; step++ {
			if err = k.advance(context.Background(), float64(step)*.001); err != nil {
				t.Fatal(step, err)
			}
		}
		if k.mixed.result.KCLResidual > 1e-9 || math.Abs(k.mixed.result.PowerResidual) > 1e-8 {
			t.Fatal("timer conservation", k.mixed.result)
		}
		events := []InputEvent{}
		for _, event := range k.trace.Events {
			if strings.HasPrefix(event.Mode, "timer555:") {
				events = append(events, event)
			}
		}
		t.Logf("%s: %+v", c.Analysis, events)
		if len(events) == 0 {
			t.Fatal("timer never switched")
		}
	}
}
func TestTimer555LatchPriorityAndHold(t *testing.T) {
	base := [8]float64{0, 2, 0, 5, 10. / 3, 2, 0, 5}
	for _, test := range []struct {
		name                      string
		prev                      bool
		trigger, threshold, reset float64
		want                      bool
	}{{"hold high", true, 2, 2, 5, true}, {"hold low", false, 2, 2, 5, false}, {"trigger", false, 0, 2, 5, true}, {"threshold", true, 2, 4, 5, false}, {"trigger priority", false, 0, 4, 5, true}, {"reset priority", true, 0, 4, 0, false}} {
		t.Run(test.name, func(t *testing.T) {
			v := base
			v[1], v[5], v[3] = test.trigger, test.threshold, test.reset
			got, err := timer555Decision(v, timer555Memory{High: test.prev, Powered: true})
			if err != nil || got.High != test.want || !got.Powered {
				t.Fatal(got, err)
			}
		})
	}
	base[7] = 0
	got, err := timer555Decision(base, timer555Memory{High: true, Powered: true})
	if err != nil || got != (timer555Memory{}) {
		t.Fatal("unpowered latch", got, err)
	}
	base[7] = 13
	if _, err = timer555Decision(base, timer555Memory{}); err == nil {
		t.Fatal("overvoltage not refused")
	}
}
func TestTimer555TerminalJacobianAndPower(t *testing.T) {
	for _, memory := range []timer555Memory{{}, {Powered: true}, {Powered: true, High: true}} {
		v := [8]float64{-1, 1, 2, 4, 2.333, 1.5, 2.5, 4}
		e := timer555Currents(v, memory)
		sum, power := 0., 0.
		for k, i := range e.Current {
			sum += i
			power += (v[k] - v[0]) * i
		}
		if math.Abs(sum) > 1e-12 || power < 0 {
			t.Fatal("555 current/power", sum, power)
		}
		for column := range v {
			lo, hi := v, v
			lo[column] -= 1e-6
			hi[column] += 1e-6
			a, b := timer555Currents(lo, memory), timer555Currents(hi, memory)
			for row := range v {
				finiteDifference := (b.Current[row] - a.Current[row]) / 2e-6
				if math.Abs(e.Jacobian[row][column]-finiteDifference) > 1e-8 {
					t.Fatal(fmt.Sprint(memory), row, column, e.Jacobian[row][column], finiteDifference)
				}
			}
		}
	}
}

// Eliminate the discharge junction by hand. These fixed resistances, leakage
// and thresholds are independent of the MNA stamp and its latch evaluator.
func timer555PhaseReference(rb float64, high bool) (target, tau float64) {
	ga, gb, gd := 1./10000, 1/rb, 1e-9
	if !high {
		gd = .1
	}
	g := gb*(ga+gd)/(ga+gb+gd) + 2e-9
	target = (5 * ga * gb / (ga + gb + gd)) / g
	tau = 10e-6 / g
	return
}
func timer555AstableReference(rb float64) (first, high, low float64) {
	up, tu := timer555PhaseReference(rb, true)
	down, td := timer555PhaseReference(rb, false)
	first = tu * math.Log(up/(up-10./3))
	high = tu * math.Log((up-5./3)/(up-10./3))
	low = td * math.Log((10./3-down)/(5./3-down))
	return
}
func timer555Kernel(t *testing.T, g Graph) *simulationKernel {
	t.Helper()
	c, status, d := g.Compile()
	if d != nil {
		t.Fatal(status, d)
	}
	k, err := newSimulationKernel(context.Background(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	return k
}
func TestTimer555RepeatedCyclesAgainstIndependentRCReference(t *testing.T) {
	for _, step := range []float64{.001, .005} {
		k := timer555Kernel(t, Timer555FlasherExampleGraph())
		for i := 1; i <= int(2.5/step); i++ {
			if err := k.advance(context.Background(), float64(i)*step); err != nil {
				t.Fatal(i, err)
			}
			r := k.mixed.result
			if r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
				t.Fatal("555 conservation", i, r)
			}
			if r.Timers555[0].High && r.LEDs[0].Current < .002 || !r.Timers555[0].High && r.LEDs[0].Current > 1e-9 {
				t.Fatal("LED disagrees with latch")
			}
		}
		events := []InputEvent{}
		for _, event := range k.trace.Events {
			if strings.HasPrefix(event.Mode, "timer555:") {
				events = append(events, event)
			}
		}
		first, high, low := timer555AstableReference(47000)
		at := first
		maxError := 0.
		if len(events) < 6 {
			t.Fatal("insufficient complete cycles", events)
		}
		for i, event := range events {
			want := "low"
			if i%2 == 1 {
				want = "high"
			}
			if event.Mode != "timer555:U1:"+want {
				t.Fatal("555 event order", event)
			}
			e := math.Abs(event.Time - at)
			maxError = math.Max(maxError, e)
			if e > 1e-5 {
				t.Fatalf("step %g edge %d got %.12g expected %.12g", step, i, event.Time, at)
			}
			if i%2 == 0 {
				at += low
			} else {
				at += high
			}
		}
		t.Logf("step %.3g s, maximum event reference error %.9g s; steady period %.9g s", step, maxError, high+low)
	}
}
func TestTimer555OneShotWidthAndResetPriorityInCircuit(t *testing.T) {
	g := Timer555OneShotExampleGraph()
	k := timer555Kernel(t, g)
	for i := 1; i <= 800; i++ {
		if err := k.advance(context.Background(), float64(i)*.001); err != nil {
			t.Fatal(i, err)
		}
	}
	got := 0.
	for _, event := range k.trace.Events {
		if event.Mode == "timer555:U1:low" {
			got = event.Time
			break
		}
	}
	conductance := 1./47000 + 2e-9
	target := (5. / 47000) / conductance
	tau := 10e-6 / conductance
	want := tau * math.Log(target/(target-10./3))
	if got == 0 || math.Abs(got-want) > 1e-5 || k.mixed.result.Timers555[0].High {
		t.Fatal("one-shot reference", got, want)
	}
	t.Logf("one-shot %.12g s, independent RC %.12g s", got, want)
	// Assert reset with a closed SPST while trigger is low. Reset wins in the
	// actual network, not only in the standalone decision-table test.
	for i := range g.Wires {
		if g.Wires[i].To == "U1.reset" {
			g.Wires[i].From = "S1.1"
		}
	}
	sw, _ := MakeComponent("S1", "switch", Point{300, -300}, "")
	closed := true
	sw.SwitchClosed = &closed
	pull, _ := MakeComponent("RR", "resistor", Point{100, -300}, "10k")
	g.Components = append(g.Components, sw, pull)
	g.Wires = append(g.Wires, Wire{ID: "WS1", From: "S1.1", To: "RR.2", Connected: true}, Wire{ID: "WS2", From: "S1.2", To: "GNDU.reference", Connected: true}, Wire{ID: "WR1", From: "RR.1", To: "U1.vcc", Connected: true})
	reset := timer555Kernel(t, g)
	if reset.mixed.result.Timers555[0].High || reset.mixed.result.LEDs[0].Current > 1e-9 {
		t.Fatal("reset did not override low trigger")
	}
}
func TestTimer555LiveWiperPreservesMemoryChargeAndTime(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "555-oscillator"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	call(t, s, "simulation.step", StepParams{Duration: .3}, "charge")
	before := s.Snapshot()
	if !before.Mixed.Timers555[0].High || before.Mixed.Capacitors[0].Voltage < 5./3 {
		t.Fatal("wiper fixture not in hold band")
	}
	call(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .8}, "adjust")
	after := settled(t, s)
	if after.RunID != before.RunID || after.Transient.Sample.Time != before.Transient.Sample.Time || math.Abs(after.Mixed.Capacitors[0].Voltage-before.Mixed.Capacitors[0].Voltage) > 1e-10 || !after.Mixed.Timers555[0].High {
		t.Fatal("wiper lost charge, time or high latch", after.Diagnostics)
	}
	call(t, s, "simulation.step", StepParams{Duration: .3}, "faster")
	if s.Snapshot().Mixed.Timers555[0].High {
		t.Fatal("shorter timing resistance did not advance next edge")
	}
}
func TestTimer555FailedStepLeavesAcceptedStateAndTraceUntouched(t *testing.T) {
	k := timer555Kernel(t, Timer555FlasherExampleGraph())
	for i := 1; i <= 620; i++ {
		if err := k.advance(context.Background(), float64(i)*.001); err != nil {
			t.Fatal(err)
		}
	}
	original := cloneMixed(k.mixed.result)
	memory := append([]timer555Memory{}, k.mixed.net.timerMemory...)
	trace := cloneTransient(k.trace)
	budget := newIntegrationBudget()
	budget.attempts = MaxIntegrationAttempts - 1
	if _, _, err := advanceMixedBudget(k.mixed, k.trace, .65, budget); err == nil {
		t.Fatal("exhausted event step succeeded")
	}
	if !reflect.DeepEqual(original, cloneMixed(k.mixed.result)) || !reflect.DeepEqual(memory, k.mixed.net.timerMemory) || !reflect.DeepEqual(trace, cloneTransient(k.trace)) {
		t.Fatal("failed step leaked latch, charge, time or traces")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := k.mixed.net.solveContext(ctx, k.mixed.x, k.mixed.capV, 0, k.trace, k.mixed.result.Time); !errors.Is(err, context.Canceled) {
		t.Fatal("555 initialization ignores cancellation", err)
	}
}

func TestTimer555InductorLoadKeepsCurrentAtOutputEdge(t *testing.T) {
	g := Timer555FlasherExampleGraph()
	// The control bypass is optional in this noise-free model. Leave it open
	// here so the existing conservative LC ceiling is set by the timing cap.
	parts := []Component{}
	for _, p := range g.Components {
		if p.ID != "CC" && p.ID != "GNDC" {
			parts = append(parts, p)
		}
	}
	g.Components = parts
	wires := []Wire{}
	for _, w := range g.Wires {
		if !strings.HasPrefix(w.From, "CC.") && !strings.HasPrefix(w.To, "CC.") {
			wires = append(wires, w)
		}
	}
	g.Wires = wires
	coil, _ := MakeComponent("L1", "inductor", Point{750, 180}, "1 H")
	load, _ := MakeComponent("RL", "resistor", Point{750, 0}, "1k")
	g.Components = append(g.Components, coil, load)
	g.Wires = append(g.Wires, Wire{ID: "WL1", From: "U1.out", To: "RL.1", Connected: true}, Wire{ID: "WL2", From: "RL.2", To: "L1.1", Connected: true}, Wire{ID: "WL3", From: "L1.2", To: "GNDL.reference", Connected: true})
	k := timer555Kernel(t, g)
	for i := 1; i <= 130; i++ {
		if err := k.advance(context.Background(), float64(i)*.005); err != nil {
			t.Fatal(i, err)
		}
	}
	edge := false
	for i := 1; i < len(k.trace.Samples); i++ {
		before, after := k.trace.Samples[i-1], k.trace.Samples[i]
		if before.Time == after.Time && before.State == "high" && after.State == "low" {
			edge = true
			for _, id := range []string{"i:L1", "v:C1"} {
				a, b := before.frame.values[id], after.frame.values[id]
				if math.Abs(a-b) > 1e-9 {
					t.Fatal("timer edge jumped storage", id, a, b)
				}
			}
		}
	}
	if !edge || k.mixed.result.Timers555[0].High || k.mixed.result.KCLResidual > 1e-9 || math.Abs(k.mixed.result.PowerResidual) > 1e-8 {
		t.Fatal("555 coil/output edge", k.mixed.result)
	}
}

func TestTimer555ExternalControlChangesTripLevels(t *testing.T) {
	g := Timer555FlasherExampleGraph()
	parts := []Component{}
	for _, p := range g.Components {
		if p.ID != "CC" && p.ID != "GNDC" {
			parts = append(parts, p)
		}
	}
	g.Components = parts
	wires := []Wire{}
	for _, w := range g.Wires {
		if !strings.HasPrefix(w.From, "CC.") && !strings.HasPrefix(w.To, "CC.") {
			wires = append(wires, w)
		}
	}
	g.Wires = wires
	control, _ := MakeComponent("VC", "dc-source", Point{800, 0}, "2.4")
	g.Components = append(g.Components, control)
	g.Wires = append(g.Wires, Wire{ID: "WC1", From: "VC.positive", To: "U1.control", Connected: true}, Wire{ID: "WC2", From: "VC.negative", To: "GNDU.reference", Connected: true})
	k := timer555Kernel(t, g)
	if math.Abs(k.mixed.result.Timers555[0].TriggerLevel-1.2) > 1e-10 || math.Abs(k.mixed.result.Timers555[0].ThresholdLevel-2.4) > 1e-10 {
		t.Fatal("external control ignored")
	}
	for i := 1; i <= 500; i++ {
		if err := k.advance(context.Background(), float64(i)*.001); err != nil {
			t.Fatal(err)
		}
	}
	target, tau := timer555PhaseReference(47000, true)
	want := tau * math.Log(target/(target-2.4))
	got := 0.
	for _, e := range k.trace.Events {
		if e.Mode == "timer555:U1:low" {
			got = e.Time
			break
		}
	}
	if got == 0 || math.Abs(got-want) > 1e-5 {
		t.Fatal("external control event reference", got, want)
	}
}
func TestTimer555LiveSwitchPreservesHighLatchInHoldBand(t *testing.T) {
	g := Timer555FlasherExampleGraph()
	sw, _ := MakeComponent("S1", "switch", Point{260, 0}, "")
	g.Components = append(g.Components, sw)
	for i := range g.Wires {
		if g.Wires[i].From == "U1.out" && g.Wires[i].To == "RLED.1" {
			g.Wires[i].From = "S1.2"
		}
	}
	g.Wires = append(g.Wires, Wire{ID: "WS", From: "U1.out", To: "S1.1", Connected: true})
	s := newGraphService(SolveLED, g)
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	call(t, s, "simulation.step", StepParams{Duration: .3}, "charge")
	before := s.Snapshot()
	if !before.Mixed.Timers555[0].High || before.Mixed.Capacitors[0].Voltage < 5./3 {
		t.Fatal("switch fixture hold band")
	}
	call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: true}, "close")
	call(t, s, "simulation.step", StepParams{Duration: .001}, "apply")
	after := s.Snapshot()
	if !after.Mixed.Timers555[0].High || after.Mixed.LEDs[0].Current < .002 || after.Mixed.Capacitors[0].Voltage < before.Mixed.Capacitors[0].Voltage || after.RunID != before.RunID {
		t.Fatal("switch replacement lost latch or charge", after.Diagnostics)
	}
}

func TestTimer555TwoIndependentLatches(t *testing.T) {
	g := Timer555FlasherExampleGraph()
	second := g.Clone()
	for i := range second.Components {
		p := &second.Components[i]
		p.ID = "B" + p.ID
		p.Position.X += 1600
		if p.ID == "BC1" {
			p.Value = 20e-6
		}
	}
	for i := range second.Wires {
		w := &second.Wires[i]
		w.ID = "B" + w.ID
		w.From = "B" + w.From
		w.To = "B" + w.To
	}
	g.Components = append(g.Components, second.Components...)
	g.Wires = append(g.Wires, second.Wires...)
	k := timer555Kernel(t, g)
	for i := 1; i <= 260; i++ {
		if err := k.advance(context.Background(), float64(i)*.005); err != nil {
			t.Fatal(i, err)
		}
	}
	if len(k.mixed.result.Timers555) != 2 || !k.mixed.result.Timers555[0].High || k.mixed.result.Timers555[1].High {
		t.Fatal("independent timing memories coupled", k.mixed.result.Timers555)
	}
	if k.mixed.result.KCLResidual > 1e-9 || math.Abs(k.mixed.result.PowerResidual) > 1e-8 {
		t.Fatal("two-timer conservation")
	}
}
