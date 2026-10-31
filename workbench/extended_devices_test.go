package workbench

import (
	"context"
	"encoding/binary"
	"math"
	"testing"

	"github.com/dshills/elab/internal/testutil"
)

func extendedKernel(t *testing.T, id string) *simulationKernel {
	t.Helper()
	d, ok := findExample(id)
	if !ok {
		t.Fatal(id)
	}
	c, status, diagnostic := d.graph.Compile()
	if status != "" || diagnostic != nil {
		t.Fatalf("%s compile %s %+v", id, status, diagnostic)
	}
	k, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: d.settings, mode: "auto", run: 1, revision: 1})
	if err != nil {
		t.Fatalf("%s startup: %v", id, err)
	}
	return k
}
func advanceExtended(t *testing.T, k *simulationKernel, end, dt float64) {
	t.Helper()
	for at := k.trace.Sample.Time + dt; at <= end+dt*1e-7; at += dt {
		if err := k.advance(t.Context(), at); err != nil {
			t.Fatalf("advance %g: %v", at, err)
		}
	}
}
func TestExtendedExamplesElectrical(t *testing.T) {
	for _, id := range []string{"buck", "opamp-slew", "opamp-bandwidth", "shift-register", "555-tone", "light-alarm"} {
		t.Run(id, func(t *testing.T) {
			if id == "buck" && testutil.RunAtProductionSpeed(t) {
				return
			}
			k := extendedKernel(t, id)
			end := .002
			if id == "shift-register" {
				end = .05
			}
			advanceExtended(t, k, end, k.trace.Settings.StepSeconds)
			r := k.mixed.result
			if r.KCLResidual > 1e-7 || math.Abs(r.PowerResidual) > 1e-6 {
				t.Fatal("conservation", r.KCLResidual, r.PowerResidual)
			}
			if err := validateRecordedDynamics(k.compiled, k.compiled.MixedGraph, r); err != nil {
				t.Fatal(err)
			}
			if err := validateRecordedPWM(k.compiled, k.compiled.MixedGraph, r); err != nil {
				t.Fatal(err)
			}
			if err := validateRecordedShifts(k.compiled, k.compiled.MixedGraph, r); err != nil {
				t.Fatal(err)
			}
			if err := validateRecordedSounds(k.compiled, k.compiled.MixedGraph, r); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestPWMClockPassivityAndJacobian(t *testing.T) {
	for _, v := range [][3]float64{{3, 12, 0}, {-1, 2.5, -3}, {5, 0, 0}} {
		for _, q := range []float64{0, .2, .7, 1} {
			i, j := pwmCurrents(v, q)
			sum, power := 0., 0.
			for k, c := range i {
				sum += c
				power += (v[k] - v[2]) * c
			}
			if math.Abs(sum) > 1e-12 || power < 0 {
				t.Fatal(v, q, i)
			}
			for col := range v {
				a, b := v, v
				a[col] += 1e-7
				b[col] -= 1e-7
				ia, _ := pwmCurrents(a, q)
				ib, _ := pwmCurrents(b, q)
				for row := range v {
					want := (ia[row] - ib[row]) / 2e-7
					if math.Abs(want-j[row][col]) > 1e-7 {
						t.Fatal("PWM derivative", row, col, want, j[row][col])
					}
				}
			}
		}
	}
}
func TestOpAmpDynamicsSmallSignalAndSlew(t *testing.T) {
	k := extendedKernel(t, "opamp-bandwidth")
	input2, out2 := 0., 0.
	for j := 1; j <= 4000; j++ {
		if err := k.advance(t.Context(), float64(j)*1e-5); err != nil {
			t.Fatal(j, err)
		}
		if j > 3000 {
			r := k.mixed.result
			input := r.Nodes[k.compiled.TerminalNet["VIN.positive"]]
			output := r.OpAmps[0].OutputVoltage
			input2 += input * input
			out2 += output * output
		}
	}
	gain := math.Sqrt(out2 / input2)
	if math.Abs(gain-1/math.Sqrt2) > .015 {
		t.Fatal("1 kHz closed-loop corner", gain)
	}
	k = extendedKernel(t, "opamp-slew")
	for j := 1; j <= 1100; j++ {
		if err := k.advance(t.Context(), float64(j)*2e-5); err != nil {
			t.Fatal(j, err)
		}
		d := k.mixed.result.Dynamics[0]
		if math.Abs(d.Current/compensationC) > 1000.00001 {
			t.Fatal("slew exceeded", d)
		}
	}
}
func TestSoundWAVAndExplicitRender(t *testing.T) {
	k := extendedKernel(t, "light-alarm")
	p, _ := k.compiled.MixedGraph.Component("BZ1")
	data, err := renderAudio(context.Background(), k.compiled, k.trace.Settings, p, AudioRenderParams{ID: p.ID, Duration: .02, Gain: .1})
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 44+480 || string(data[:4]) != "RIFF" || binary.LittleEndian.Uint32(data[24:28]) != audioRate {
		t.Fatal("bad WAV")
	}
	silent := true
	for j := 44; j < len(data); j += 2 {
		if binary.LittleEndian.Uint16(data[j:j+2]) != 0 {
			silent = false
		}
	}
	if silent {
		t.Log("daylight alarm correctly silent")
	}
	// A powered active buzzer produces the configured tone without affecting its
	// modeled DC current. Playback is a separate command and is never called here.
	p.Device.Sound.Frequency = 1000
}

func TestBuckStartupRippleDutyAndLoad(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	k := extendedKernel(t, "buck")
	min, max, mean := math.Inf(1), math.Inf(-1), 0.
	for j := 1; j <= 10000; j++ {
		if err := k.advance(t.Context(), float64(j)*2e-5); err != nil {
			t.Fatal(j, err)
		}
		r := k.mixed.result
		if math.Abs(r.PowerResidual) > 2e-6 || r.KCLResidual > 1e-7 {
			t.Fatal("buck conservation", j, r.PowerResidual, r.KCLResidual)
		}
		if j > 9000 {
			v := r.Capacitors[0].Voltage
			min = math.Min(min, v)
			max = math.Max(max, v)
			mean += v / 1000
		}
	}
	t.Log("buck mean/ripple", mean, max-min)
	if mean < 3.5 || mean > 6 || max-min < .001 || max-min > .5 {
		t.Fatal("buck steady output", mean, min, max)
	}
	before := k.mixed.result.Capacitors[0].Voltage
	// A switch closure is solved through the same state-continuation path used
	// by the editor; it must retain winding current and capacitor voltage.
	net := *k.mixed.net
	net.c = k.mixed.net.c
	net.c.MixedGraph = net.c.MixedGraph.Clone()
	for j, p := range net.c.MixedGraph.Components {
		if p.ID == "SLOAD" {
			yes := true
			net.c.MixedGraph.Components[j].SwitchClosed = &yes
		}
	}
	state := *k.mixed
	state.net = &net
	k.mixed = &state
	advanceExtended(t, k, .3, 2e-5)
	after := k.mixed.result.Capacitors[0].Voltage
	if after >= before-.1 {
		t.Fatal("load does not lower open-loop output", before, after)
	}
	d, _ := findExample("buck")
	for j, p := range d.graph.Components {
		if p.ID == "PWM" {
			d.graph.Components[j].Device.PWM.Duty = .7
		}
	}
	c, _, diag := d.graph.Compile()
	if diag != nil {
		t.Fatal(diag)
	}
	high, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: d.settings, mode: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	advanceExtended(t, high, .2, 2e-5)
	if high.mixed.result.Capacitors[0].Voltage < mean+1 {
		t.Fatal("duty does not raise output")
	}
}
func TestShiftRegisterClockLatchClearAndEnable(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	k := extendedKernel(t, "shift-register")
	advanceExtended(t, k, 1.62, .0025)
	r := k.mixed.result.Shifts[0]
	if r.Shifted != 255 || r.Latched != 0 {
		t.Fatal("serial fill changed latch", r)
	}
	// Clock LATCH high using its actual switch; settle every hybrid guard.
	change := func(id string, closed bool) {
		net := *k.mixed.net
		net.c = k.mixed.net.c
		net.c.MixedGraph = net.c.MixedGraph.Clone()
		for j, p := range net.c.MixedGraph.Components {
			if p.ID == id {
				v := closed
				net.c.MixedGraph.Components[j].SwitchClosed = &v
			}
		}
		// A contact edit is an exact zero-time input boundary. Do not turn
		// an already changed switch into a fictitious future analog edge.
		state, err := net.solveContext(t.Context(), k.mixed.x, k.mixed.capV, 0, k.trace, k.trace.Sample.Time)
		if err != nil {
			t.Fatal(err)
		}
		k.mixed = state
		k.trace.Sample = mixedSample(k.compiled, state.result, mixedSourceVoltage(k.compiled, state.result))
	}
	change("SLATCH", true)
	r = k.mixed.result.Shifts[0]
	if r.Latched != 255 || !r.Enabled {
		t.Fatal("latch failed", r)
	}
	change("SOE", true)
	r = k.mixed.result.Shifts[0]
	if r.Enabled || r.Latched != 255 || math.Abs(r.PinCurrents[5]) > 1e-7 || r.Voltages[8] < 4.9 {
		t.Fatal("OE changes storage/serial", r)
	}
	change("SCLEAR", true)
	r = k.mixed.result.Shifts[0]
	if r.Shifted != 0 || r.Latched != 255 {
		t.Fatal("clear must preserve latch", r)
	}
	change("SOE", false)
	r = k.mixed.result.Shifts[0]
	if !r.Enabled || r.Voltages[0] < 1 {
		t.Fatal("re-enabled output lost latch", r)
	}
	if err := validateRecordedShifts(k.compiled, k.compiled.MixedGraph, k.mixed.result); err != nil {
		t.Fatal(err)
	}
}
func TestOpAmpDynamicsACCorner(t *testing.T) {
	g := OpAmpDynamicsExampleGraph(false)
	r := sweepForTest(t, g, acSettings("VIN", "v:U1"))
	for _, p := range r.Points {
		if p.Input > 950 && p.Input < 1050 {
			want := 1 / math.Sqrt(1+math.Pow(p.Input/1000, 2))
			got := math.Hypot(p.AC[0].Real, p.AC[0].Imaginary)
			if math.Abs(got-want) > .006 {
				t.Fatal("AC pole", got, want)
			}
		}
	}
}

func TestShiftCascadeAndSimultaneousLatch(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	b := newExampleBuilder()
	b.part("VCC", "dc-source", "5", -700, -400, 0)
	b.ground("GV", "VCC.negative", -700, -200)
	b.part("CLK", "finite-pulse-source", "5", -700, 100, 0)
	b.graph.Components[len(b.graph.Components)-1].Pulse = &PulseWaveform{HighVoltage: 5, LowVoltage: 0, HighSeconds: .01, LowSeconds: .01, RiseSeconds: .001, FallSeconds: .001}
	b.ground("GC", "CLK.negative", -700, 300)
	for j, id := range []string{"U1", "U2"} {
		b.part(id, "shift-register", "", float64(j)*400, 0, 0)
		b.wire("VCC.positive", id+".vcc")
		b.wire("VCC.negative", id+".gnd")
		b.wire("CLK.positive", id+".clk")
		b.wire("CLK.positive", id+".latch")
		b.wire("VCC.positive", id+".reset_n")
		b.wire("VCC.negative", id+".oe_n")
		if j == 0 {
			b.wire("VCC.positive", id+".data")
		} else {
			b.wire("U1.serial", id+".data")
		}
	}
	c, status, d := b.graph.Compile()
	if status != "" || d != nil {
		t.Fatal(status, d)
	}
	k, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	for edge := 1; edge <= 16; edge++ {
		advanceExtended(t, k, float64(edge-1)*.02+.005, .0005)
		a, b := k.mixed.result.Shifts[0], k.mixed.result.Shifts[1]
		wantFirst := uint8(255)
		if edge < 8 {
			wantFirst = uint8((1 << edge) - 1)
		}
		wantSecond := uint8(0)
		if edge > 8 {
			wantSecond = uint8((1 << (edge - 8)) - 1)
		}
		if a.Shifted != wantFirst || b.Shifted != wantSecond {
			t.Fatal("cascade order", edge, a.Shifted, b.Shifted, wantFirst, wantSecond)
		}
		wantLatch := uint8(0)
		if edge > 1 {
			wantLatch = uint8((1 << min(edge-1, 8)) - 1)
		}
		if a.Latched != wantLatch {
			t.Fatal("simultaneous latch must sample pre-edge", edge, a.Latched, wantLatch)
		}
		if err := validateRecordedShifts(c, c.MixedGraph, k.mixed.result); err != nil {
			t.Fatal("cascade recording", err)
		}
	}
}
func TestDynamicsCompensationAnalyticDerivative(t *testing.T) {
	p, _ := MakeComponent("U1", "opamp", Point{}, "")
	p.Device = &DeviceParameters{Dynamics: &OpAmpDynamics{1000, 2000}}
	for _, v := range []opAmpVoltages{{.001, 0, 1, 5, -5}, {.02, 0, 1, 5, 0}, {-1, 0, 1, 5, 0}} {
		for _, pole := range []float64{.01, 1, 4.99} {
			j, d, rail := poleCurrent(p, v, pole)
			if math.Abs(j) > compensationC*2000.00001 || rail != 3 && rail != 4 {
				t.Fatal(j, rail)
			}
			for col := range d {
				va, vb := v, v
				pa, pb := pole, pole
				h := 1e-7
				if col < 5 {
					va[col] += h
					vb[col] -= h
				} else {
					pa += h
					pb -= h
				}
				a, _, _ := poleCurrent(p, va, pa)
				b, _, _ := poleCurrent(p, vb, pb)
				want := (a - b) / (2 * h)
				if math.Abs(want-d[col]) > 1e-10 {
					t.Fatal("compensation derivative", col, want, d[col])
				}
			}
		}
	}
}
