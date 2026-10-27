package workbench

import (
	"context"
	"encoding/json"
	"math"
	"math/cmplx"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestJFETIndependentSquareLawJacobianAndPolarity(t *testing.T) {
	p, _ := MakeComponent("J1", "njfet", Point{}, "")
	p.Device.JFET.Lambda = 0
	for _, v := range []struct{ vg, vd, want float64 }{{0, 5, .005}, {-1, 5, .00125}, {-2, 5, 0}, {-3, 5, 0}, {0, 1, .00375}} {
		i, _, _ := jfetChannel(p, v.vg, v.vd)
		if math.Abs(i-v.want) > 1e-14 {
			t.Fatal("independent square law", v, i)
		}
	}
	for _, v := range [][3]float64{{0, 5, 0}, {-1, 2, .2}, {-3, 4, 0}, {.6, .1, 0}, {0, -1, .5}} {
		i, j := jfetEvaluation(p, v)
		if math.Abs(i[0]+i[1]+i[2]) > 1e-12 {
			t.Fatal("terminal KCL", i)
		}
		power := 0.
		for k, current := range i {
			power += v[k] * current
		}
		if power < -1e-12 {
			t.Fatal("active channel heat", power)
		}
		for col := 0; col < 3; col++ {
			a, b := v, v
			eps := 1e-6
			a[col] += eps
			b[col] -= eps
			ia, _ := jfetEvaluation(p, a)
			ib, _ := jfetEvaluation(p, b)
			for row := 0; row < 3; row++ {
				want := (ia[row] - ib[row]) / (2 * eps)
				if math.Abs(j[row][col]-want) > 1e-7*math.Max(1, math.Abs(want)) {
					t.Fatal("JFET jacobian", v, row, col, j[row][col], want)
				}
			}
		}
		q, _ := MakeComponent("JP", "pjfet", Point{}, "")
		q.Device.JFET.Lambda = 0
		mirrored := [3]float64{-v[0], -v[1], -v[2]}
		ip, jp := jfetEvaluation(q, mirrored)
		for k := range i {
			if math.Abs(i[k]+ip[k]) > 1e-12 {
				t.Fatal("P symmetry", i, ip)
			}
			for l := range i {
				if math.Abs(j[k][l]-jp[k][l]) > 1e-12 {
					t.Fatal("P jacobian symmetry")
				}
			}
		}
		swapped := [3]float64{v[0], v[2], v[1]}
		is, _ := jfetEvaluation(p, swapped)
		if math.Abs(is[0]-i[0]) > 1e-12 || math.Abs(is[1]-i[2]) > 1e-12 || math.Abs(is[2]-i[1]) > 1e-12 {
			t.Fatal("drain/source reciprocity", i, is)
		}
	}
	i, _ := jfetEvaluation(p, [3]float64{.6, 1, 0})
	if i[0] < 1e-5 {
		t.Fatal("gate forward junction missing", i)
	}
}
func TestJFETSelfBiasAgainstIndependentQuadratic(t *testing.T) {
	for _, polarity := range []bool{false, true} {
		g := JFETCurrentExampleGraph(polarity)
		for k, p := range g.Components {
			if IsJFETKind(p.Kind) {
				g.Components[k].Device.JFET.Lambda = 0
			}
		}
		r, _, err := solveDCOperatingPoint(context.Background(), g, dcSourceLevels(g))
		if err != nil {
			t.Fatal(err)
		}
		if len(r.JFETs) != 1 {
			t.Fatal("missing JFET reading")
		}
		q := r.JFETs[0]
		a := .005 * 220 / 2
		u := 2 / (1 + math.Sqrt(1+4*a))
		want := .005 * u * u
		if polarity {
			want = -want
		}
		if math.Abs(q.DrainCurrent-want) > 1e-7 || math.Abs(q.GateCurrent+q.DrainCurrent-q.SourceCurrent) > 1e-12 || math.Abs(r.PowerResidual) > 1e-8 {
			t.Fatal("independent self bias", q, want, r.PowerResidual)
		}
	}
}
func jfetACFixture(t *testing.T) Graph {
	t.Helper()
	b := newExampleBuilder()
	b.part("VCC", "dc-source", "9", -300, -200, 0)
	b.ground("GV", "VCC.negative", -300, -30)
	b.part("VIN", "sine-source", ".01", -300, 230, 0)
	b.graph.Components[len(b.graph.Components)-1].Sine = &SineWaveform{Amplitude: .01, Offset: -1, Frequency: 10}
	b.ground("GI", "VIN.negative", -300, 400)
	b.part("J1", "njfet", "", 0, 0, 0)
	b.graph.Components[len(b.graph.Components)-1].Device.JFET.Lambda = 0
	b.wire("VIN.positive", "J1.gate")
	b.ground("GS", "J1.source", 100, 180)
	b.part("RD", "resistor", "1k", 0, -230, 0)
	b.wire("VCC.positive", "RD.1")
	b.wire("RD.2", "J1.drain")
	b.part("C1", "capacitor", "1u", 330, 10, 0)
	b.wire("J1.drain", "C1.1")
	b.ground("GC", "C1.2", 330, 180)
	return b.graph
}
func TestJFETACAgainstIndependentTransconductancePole(t *testing.T) {
	r := sweepForTest(t, jfetACFixture(t), acSettings("VIN", "v:J1.drain"))
	// At VGS=-1 V, IDSS=5mA/cutoff=-2V gives gm=2.5mS.
	// Reverse gate-drain junction adds only its declared 1G bleed.
	for _, p := range r.Points {
		want := complex(-.0025+1e-9, 0) / complex(.001+1e-9, 2*math.Pi*p.Input*1e-6)
		got := complex(p.AC[0].Real, p.AC[0].Imaginary)
		if cmplx.Abs(got-want) > 1e-8 {
			t.Fatal("independent JFET gain/pole", p.Input, got, want)
		}
	}
}
func TestJFETExamplesCaptureHistoryAndModelValidation(t *testing.T) {
	for _, id := range []string{"jfet-current", "pjfet-current", "jfet-preamp"} {
		t.Run(id, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: id}, "example")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			before := settled(t, s)
			if before.Mixed == nil || len(before.Mixed.JFETs) != 1 {
				t.Fatal("no accepted JFET", before.Diagnostics)
			}
			s.mu.Lock()
			capturedGraph := s.graph.Clone()
			s.mu.Unlock()
			p, _ := capturedGraph.Component("J1")
			c := *p.Device.JFET
			c.IDSS = .008
			call(t, s, "jfet.configure", JFETParams{ID: "J1", Parameters: c}, "configure")
			call(t, s, "edit.undo", struct{}{}, "undo")
			if !reflect.DeepEqual(before.Components, s.Snapshot().Components) {
				t.Fatal("history")
			}
			call(t, s, "simulation.restart", struct{}{}, "restart2")
			settled(t, s)
			duration, step := .025, .0001
			if id != "jfet-preamp" {
				duration = 0
			}
			_, op := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: s.Snapshot().Scope.Selected}, "capture")
			s.mu.Lock()
			record := s.operations[op.Operation.ID].record
			doc := VerificationDocument{Metadata: cloneVerificationMetadata(record.metadata), Rows: append([]VerificationRow{}, record.rows...)}
			s.mu.Unlock()
			raw, _ := json.Marshal(doc)
			if _, err := decodeRecording(raw); err != nil {
				t.Fatal("JFET capture", err)
			}
			path := filepath.Join(t.TempDir(), "jfet.elab")
			call(t, s, "project.save", SaveParams{Path: path}, "save")
			fresh := New()
			defer fresh.Close()
			call(t, fresh, "project.load", LoadParams{Path: path}, "reopen")
			after := settled(t, fresh)
			if !reflect.DeepEqual(before.Components, after.Components) || after.Document.RecordingCount != 1 {
				t.Fatal("JFET roundtrip")
			}
			if !strings.Contains(string(schematicSVG(capturedGraph, 1, 1)), "device_components") || !strings.Contains(schematicPartDescription(p), JFETModelID) {
				t.Fatal("JFET export model")
			}
			doc.Metadata.Mixed.JFETs[0].DrainCurrent += .01
			raw, _ = json.Marshal(doc)
			if _, err := decodeRecording(raw); err == nil {
				t.Fatal("forged JFET reading")
			}
			c.Cutoff = -c.Cutoff
			refuse(t, s, "jfet.configure", JFETParams{ID: "J1", Parameters: c}, "invalid-params")
		})
	}
}
func TestJFETCombinedFETCapacityAndStrictConfig(t *testing.T) {
	b := exampleBuilder{graph: JFETCurrentExampleGraph(false)}
	for k := 2; k <= 3; k++ {
		id := "J" + string(rune('0'+k))
		b.part(id, "njfet", "", float64(k)*200, 0, 0)
		for _, pin := range jfetPins {
			b.wire(id+"."+pin, "J1."+pin)
		}
	}
	_, _, d := b.graph.Compile()
	if d == nil || d.Code != "mixed-limit" {
		t.Fatal("combined FET limit", d)
	}
	p, _ := MakeComponent("J1", "njfet", Point{}, "")
	for _, c := range []JFETConfig{{JFETModelID, .005, 0, 0}, {JFETModelID, .005, 2, 0}, {JFETModelID, 0, -2, 0}, {JFETModelID, .005, -2, -1}, {"unknown", .005, -2, 0}} {
		if err := validateJFET(p.Kind, c); err == nil {
			t.Fatal("bad JFET accepted", c)
		}
	}
	for _, pin := range p.Terminals {
		if _, ok := TerminalPosition(p, pin); !ok {
			t.Fatal("missing JFET terminal", pin)
		}
	}
}

func TestJFETTransientAgainstIndependentNonlinearODE(t *testing.T) {
	g := jfetACFixture(t)
	state, tr := initializeLossRC(t, g)
	ref, at, h := 0., 0., 1e-7
	// Independent explicit scalar square-law / 1uF drain-node equation.
	f := func(t, v float64) float64 {
		vg := -1 + .01*math.Sin(2*math.Pi*10*t)
		u := vg + 2
		i := .00125 * u * u
		if v < u {
			i = .0025 * (u*v - .5*v*v)
		}
		gateDrain := -1e-9 + (vg-v)*1e-9
		return ((9-v)/1000 - i + gateDrain) / 1e-6
	}
	for sample := 1; sample <= 200; sample++ {
		end := float64(sample) * .0001
		for at < end-h/2 {
			k1 := f(at, ref)
			k2 := f(at+h/2, ref+h*k1/2)
			k3 := f(at+h/2, ref+h*k2/2)
			k4 := f(at+h, ref+h*k3)
			ref += h * (k1 + 2*k2 + 2*k3 + k4) / 6
			at += h
		}
		var err error
		state, err = state.advance(tr, end)
		if err != nil {
			t.Fatal(err)
		}
		q := state.result.JFETs[0]
		if math.Abs(q.VDS-ref) > .0003 || math.Abs(state.result.PowerResidual) > 1e-8 || state.result.KCLResidual > 1e-9 {
			t.Fatal("independent transient", sample, q.VDS, ref, state.result.PowerResidual)
		}
	}
}

func TestJFETMissingConfigurationIsRejectedBeforeSolve(t *testing.T) {
	g := JFETCurrentExampleGraph(false)
	for k, p := range g.Components {
		if p.ID == "J1" {
			g.Components[k].Device = nil
		}
	}
	_, _, d := g.Compile()
	if d == nil || d.Code != "invalid-parameters" {
		t.Fatal("missing JFET parameters reached solve", d)
	}
}
