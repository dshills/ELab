package workbench

import (
	"context"
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPhotodiodeLawAgainstIndependentReference(t *testing.T) {
	p, err := MakeComponent("PD", "photodiode", Point{}, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []float64{-10, -1, 0, .1, .3} {
		for _, light := range []float64{0, 10, 100} {
			p.Device.Photo.Light = light
			i, g, _, ph := photodiodeLaw(p, v)
			vt := 1.380649e-23 * 300.15 / 1.602176634e-19
			want := p.Device.Photo.DarkCurrent*math.Expm1(v/(1.5*vt)) + v*1e-9 - .5*light*1e-6
			if math.Abs(i-want) > 1e-12 {
				t.Fatal("independent photodiode law", v, light, i, want)
			}
			h := 1e-7
			ip, _, _, _ := photodiodeLaw(p, v+h)
			im, _, _, _ := photodiodeLaw(p, v-h)
			if math.Abs(g-(ip-im)/(2*h)) > 1e-8 {
				t.Fatal("photodiode Jacobian")
			}
			if ph != .5*light*1e-6 {
				t.Fatal("responsivity")
			}
		}
	}
}
func TestPhotodiodeDCAndTransimpedance(t *testing.T) {
	for _, tc := range []struct {
		id    string
		graph Graph
		probe string
		want  float64
	}{{"load", PhotodiodeLoadExampleGraph(), "v:RL", .5009499050095}, {"tia", PhotodiodeTIAExampleGraph(), "v:U1", 1}} {
		r, _, err := solveDCOperatingPoint(context.Background(), tc.graph, dcSourceLevels(tc.graph))
		if err != nil {
			t.Fatal(tc.id, err)
		}
		if len(r.Photodiodes) != 1 {
			t.Fatal("photodiode result missing")
		}
		pd := r.Photodiodes[0]
		if tc.id == "load" && math.Abs(pd.Current+pd.Photocurrent+pd.Parameters.DarkCurrent-pd.Voltage*1e-9) > 1e-12 {
			t.Fatal("reverse junction current", pd)
		}
		c, _, d := tc.graph.Compile()
		if d != nil {
			t.Fatal(d)
		}
		sample := mixedSample(c, r, 0)
		probe := Probe{}
		for _, p := range probeInventory(tc.graph) {
			if p.ID == tc.probe {
				probe = p
			}
		}
		got, ok := frameValue(sample, probe)
		if !ok || math.Abs(got-tc.want) > .002 {
			t.Fatal("independent optical conversion", tc.id, got, tc.want, ok)
		}
	}
}
func TestPhotodiodeLiveLightPreservesChargeAndRun(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "photodiode-load"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := settled(t, s)
	call(t, s, "simulation.step", StepParams{Duration: .005}, "charge")
	before = settled(t, s)
	pd := before.Mixed.Photodiodes[0]
	call(t, s, "photodiode.light.set", OpticalLightParams{ID: "PD1", Light: 20}, "light")
	after := settled(t, s)
	if after.InputWork == nil || after.InputWork.Status != "completed" || after.InputWork.Command != "photodiode.light.set" || after.RunID != before.RunID || after.Mixed.Time != before.Mixed.Time || math.Abs(after.Mixed.Photodiodes[0].Voltage-pd.Voltage) > 1e-10 || after.Mixed.Photodiodes[0].Parameters.Light != 20 {
		t.Fatal("optical input atomicity", after.Diagnostics, after.InputWork, after.Mixed)
	}
	call(t, s, "simulation.step", StepParams{Duration: .001}, "response")
	after = settled(t, s)
	target := (1e5*(.5*20e-6+1e-9) - 5) / (1 + 1e5/1e9)
	want := target + (pd.Voltage-target)*math.Exp(-.001/((1/(1/1e5+1/1e9))*10e-9))
	if math.Abs(after.Mixed.Photodiodes[0].Voltage-want) > 2e-5 {
		t.Fatal("independent optical RC response", after.Mixed.Photodiodes[0].Voltage, want)
	}
	if math.Abs(after.Mixed.Photodiodes[0].Voltage-pd.Voltage) < .1 {
		t.Fatal("light response did not evolve")
	}
	if after.Mixed.KCLResidual > 1e-8 || math.Abs(after.Mixed.PowerResidual) > 1e-7 {
		t.Fatal("electrical/optical power", after.Mixed)
	}
}

func TestPhotodiodeCapturePersistenceAndOpticalSweep(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "photodiode-tia"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := settled(t, s)
	duration, step := .0001, .000001
	_, op := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:U1", "i:PD1"}}, "verify")
	s.mu.Lock()
	rec := s.operations[op.Operation.ID].record
	doc := VerificationDocument{Metadata: cloneVerificationMetadata(rec.metadata), Rows: append([]VerificationRow{}, rec.rows...)}
	s.mu.Unlock()
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeRecording(data); err != nil {
		t.Fatal("photodiode capture", err)
	}
	original := cloneVerificationMetadata(doc.Metadata)
	for _, mutate := range []func(*PhotodiodeResult){func(p *PhotodiodeResult) { p.Photocurrent += 1e-6 }, func(p *PhotodiodeResult) { p.Energy += 1e-9 }, func(p *PhotodiodeResult) { p.StorageCurrent += 1e-6 }, func(p *PhotodiodeResult) { p.Parameters.Light++ }} {
		doc.Metadata = cloneVerificationMetadata(original)
		mutate(&doc.Metadata.Mixed.Photodiodes[0])
		data, err = json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeRecording(data); err == nil {
			t.Fatal("forged photodiode reading accepted")
		}
	}
	path := filepath.Join(t.TempDir(), "photo.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	settled(t, fresh)
	if !reflect.DeepEqual(before.Components, fresh.Snapshot().Components) || fresh.Snapshot().Document.RecordingCount != 1 {
		t.Fatal("photodiode project round trip")
	}
	cfg := SweepSettings{ComponentID: "PD1", Parameter: "light", Start: "0", Stop: "40", Points: 5, ProbeIDs: []string{"v:U1"}}
	r := sweepForTest(t, PhotodiodeTIAExampleGraph(), cfg)
	for _, p := range r.Points {
		want := p.Input * .5 * 1e-6 * 1e5
		if math.Abs(p.Values[0]-want) > .002 {
			t.Fatal("independent TIA optical sweep", p, want)
		}
	}
	photoCfg := DefaultPhotodiodeConfig()
	call(t, s, "photodiode.configure", PhotodiodeParams{ID: "PD1", Parameters: photoCfg}, "configure")
	call(t, s, "edit.undo", struct{}{}, "undo")
	if !reflect.DeepEqual(before.Components, s.Snapshot().Components) {
		t.Fatal("photodiode history")
	}
}
func TestPhotodiodeStorageCountsAndNoise(t *testing.T) {
	g := PhotodiodeLoadExampleGraph()
	u := powerResources(g)
	if u.caps != 1 || u.parts != 1 {
		t.Fatal("photodiode storage not budgeted", u)
	}
	r := sweepForTest(t, g, noiseSettings("V1", "v:PD1"))
	found := false
	for _, c := range r.Noise.Contributors {
		if c.ID == "PD1:optical-and-dark-shot" {
			found = true
			pd := r.Noise.Bias.Photodiodes[0]
			want := 2 * noiseCharge * (pd.Photocurrent + math.Abs(pd.JunctionCurrent))
			if math.Abs(c.Density*c.Density/want-1) > 1e-12 {
				t.Fatal("photodiode shot noise", c, want)
			}
		}
	}
	if !found {
		t.Fatal("optical shot source absent")
	}
}

func TestPhotodiodeDrawingAndSVGMetadata(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "blank")
	call(t, s, "component.place", PlaceParams{Kind: "photodiode", Position: Point{}}, "place")
	p, ok := s.graph.Component("PD1")
	if !ok || p.Device.Photo == nil {
		t.Fatal("automatic detector placement")
	}
	preview, err := PreviewExample("photodiode-load")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.SVG, "photodiode") || !strings.Contains(preview.SVG, "responsivity_a_per_w") {
		t.Fatal("SVG detector provenance missing")
	}
}

func TestPhotodiodeGraphWorkerAndSnapshotOwnership(t *testing.T) {
	g := PhotodiodeLoadExampleGraph()
	cloned := g.Clone()
	cloned.Components[2].Device.Photo.Light = 99
	if g.Components[2].Device.Photo.Light != 10 {
		t.Fatal("photodiode config graph alias")
	}
	c, _, d := g.Compile()
	if d != nil {
		t.Fatal(d)
	}
	n := newMixedNetwork(c)
	if !n.nonlinear {
		t.Fatal("photodiode nonlinear flag absent")
	}
	r := sweepForTest(t, g, SweepSettings{ComponentID: "PD1", Parameter: "light", Start: "0", Stop: "50", Points: 3, ProbeIDs: []string{"i:PD1"}})
	if len(r.Points) != 3 || g.Components[2].Device.Photo.Light != 10 {
		t.Fatal("light worker mutated source")
	}
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "photodiode-load"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := settled(t, s)
	call(t, s, "photodiode.light.set", OpticalLightParams{ID: "PD1", Light: 30}, "light")
	settled(t, s)
	old, ok := (Graph{Components: before.Components}).Component("PD1")
	if !ok || old.Device == nil || old.Device.Photo == nil {
		t.Fatal("old detector snapshot absent")
	}
	if old.Device.Photo.Light != 10 {
		t.Fatal("old snapshot changed")
	}
	call(t, s, "edit.undo", struct{}{}, "undo")
	p, ok := (Graph{Components: s.Snapshot().Components}).Component("PD1")
	if !ok || p.Device == nil || p.Device.Photo == nil {
		t.Fatal("undo detector absent")
	}
	if p.Device.Photo.Light != 10 {
		t.Fatal("light history aliased")
	}
}
