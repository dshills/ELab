package workbench

import (
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

// At either settled rail, the two matched conducting channels have equal
// voltage drop. Solve their scalar load line independently of the MNA stamps.
func bridgeLoadReference(resistance float64) (voltage, current float64) {
	lo, hi := 0., 2.5
	for range 64 {
		drop := (lo + hi) / 2
		channel := .1 * (3.5*drop - drop*drop/2) * (1 + .02*drop)
		if channel < (5-2*drop)/resistance {
			lo = drop
		} else {
			hi = drop
		}
	}
	voltage = 5 - lo - hi
	return voltage, voltage / resistance
}

func TestDiscreteHBridgeReversesRecordsAndReopens(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "discrete-hbridge"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	probes := []string{"v:DIR.positive", "v:RL", "i:RL"}
	initial := settled(t, s)
	if initial.Selected != "RL" || !reflect.DeepEqual(initial.Scope.Selected, probes) {
		t.Fatalf("example opened with selection %q, probes %v", initial.Selected, initial.Scope.Selected)
	}
	v, i := bridgeLoadReference(100)
	check := func(state Snapshot, sign float64) {
		t.Helper()
		if state.Mixed == nil || state.Mixed.Stale || len(state.Mixed.MOSFETs) != 4 || len(state.Mixed.Resistors) != 1 {
			t.Fatalf("missing four-device bridge result: %v", state.Diagnostics)
		}
		r := state.Mixed.Resistors[0]
		if math.Abs(r.Voltage-sign*v) > 2e-6 || math.Abs(r.Current-sign*i) > 2e-8 || state.Mixed.KCLResidual > 1e-8 || math.Abs(state.Mixed.PowerResidual) > 1e-7 {
			t.Fatalf("RL got %g V, %g A, want %g V, %g A; KCL %g, power residual %g", r.Voltage, r.Current, sign*v, sign*i, state.Mixed.KCLResidual, state.Mixed.PowerResidual)
		}
	}
	check(initial, -1)
	call(t, s, "simulation.step", StepParams{Duration: .021}, "reverse")
	check(s.Snapshot(), 1)

	duration, step := .045, .001
	_, captured := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: probes}, "record")
	read := func(owner *Service, id string) VerificationPage {
		t.Helper()
		page := call(t, owner, "operation.samples", VerificationPageParams{ID: id, Limit: 200}, "samples").Result.(VerificationPage)
		if len(page.Rows) == 0 || len(page.Rows) != page.Total {
			t.Fatal("incomplete recording", page.Total, len(page.Rows))
		}
		return page
	}
	record := read(s, captured.Operation.ID)
	signs := map[float64]bool{}
	for _, row := range record.Rows {
		if len(row.Values) != 3 {
			t.Fatal("missing recorded bridge probes", row)
		}
		sign := 1.
		if row.Values[0] == 5 {
			sign = -1
		} else if row.Values[0] != 0 {
			t.Fatal("unexpected direction input", row)
		}
		signs[sign] = true
		if math.Abs(row.Values[1]-sign*v) > 2e-6 || math.Abs(row.Values[2]-sign*i) > 2e-8 {
			t.Fatal("recorded bridge sample differs from the load line", row)
		}
	}
	if len(signs) != 2 {
		t.Fatal("recording missed a drive direction")
	}
	path := filepath.Join(t.TempDir(), "bridge.elab")
	call(t, s, "recording.keep.set", RecordingKeepParams{ID: captured.Operation.ID, Keep: true}, "keep")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "reopen")
	reopened := settled(t, fresh)
	check(reopened, -1)
	if !reopened.Paused || reopened.Document.Dirty || !reflect.DeepEqual(initial.Components, reopened.Components) || !reflect.DeepEqual(initial.Wires, reopened.Wires) || len(reopened.Operations) != 1 {
		t.Fatal("reopen lost the accepted bridge graph, recording, or paused clean state")
	}
	if !reflect.DeepEqual(read(fresh, reopened.Operations[0].ID).Rows, record.Rows) {
		t.Fatal("saved bridge recording changed on reopen")
	}
}

func TestExcessBridgeFETRemainsSavable(t *testing.T) {
	b := exampleBuilder{graph: dualHBridgeExampleGraph()}
	b.part("M9", "nmos", "", 680, 145, 0)
	for _, pin := range []string{"gate", "drain", "source"} {
		b.wire("M9."+pin, "MNB."+pin)
	}
	s := newGraphService(SolveLED, b.graph)
	defer s.Close()
	refused := settled(t, s)
	if refused.Status != "unsupported" || len(refused.Diagnostics) == 0 || refused.Diagnostics[0].Code != "mixed-limit" {
		t.Fatal("expected a size refusal with the ninth FET", refused.Diagnostics)
	}
	path := filepath.Join(t.TempDir(), "nine-fets.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	loaded := settled(t, fresh)
	if loaded.Document.Dirty || !reflect.DeepEqual(loaded.Components, b.graph.Components) || !reflect.DeepEqual(loaded.Wires, b.graph.Wires) {
		t.Fatal("saving an over-limit circuit lost the ninth FET or its wiring")
	}
}

func TestEightFETBudgetRetainsTwoJFETLimit(t *testing.T) {
	b := exampleBuilder{graph: JFETCurrentExampleGraph(false)}
	addParallel := func(id, kind string) {
		b.part(id, kind, "", 420, float64(len(b.graph.Components))*20, 0)
		for _, pin := range jfetPins {
			b.wire(id+"."+pin, "J1."+pin)
		}
	}
	addParallel("J2", "njfet")
	for i := 1; i <= MaxNetworkMOSFETs-2; i++ {
		addParallel(fmt.Sprintf("M%d", i), "nmos")
	}
	s := newGraphService(SolveLED, b.graph)
	defer s.Close()
	state := settled(t, s)
	if state.Mixed == nil || state.Mixed.Stale || len(state.Mixed.JFETs) != 2 || len(state.Mixed.MOSFETs) != MaxNetworkMOSFETs-2 || state.Mixed.KCLResidual > 1e-8 {
		t.Fatal("JFETs and MOSFETs did not share the eight-device budget", state.Diagnostics)
	}
	addParallel("MX", "nmos")
	if _, _, d := b.graph.Compile(); d == nil || d.Code != "mixed-limit" {
		t.Fatal("ninth device in mixed FET network was not refused", d)
	}
}

func TestDualHBridgeSharedSupplyRecordingAndReopen(t *testing.T) {
	s := New()
	t.Cleanup(s.Close)
	call(t, s, "example.load", ExampleLoadParams{ID: "dual-hbridge"}, "load-dual")
	call(t, s, "simulation.pause", struct{}{}, "pause-dual")
	call(t, s, "simulation.restart", struct{}{}, "restart-dual")
	initial := settled(t, s)
	vA, iA := bridgeLoadReference(100)
	vB, iB := bridgeLoadReference(150)
	check := func(state Snapshot, a, b float64) {
		t.Helper()
		r := state.Mixed
		if r == nil || r.Stale || len(r.MOSFETs) != 8 || len(r.Resistors) != 2 || len(r.Sources) != 3 || r.KCLResidual > 1e-8 || math.Abs(r.PowerResidual) > 1e-7 {
			t.Fatalf("invalid dual bridge state: %v", state.Diagnostics)
		}
		for _, load := range r.Resistors {
			wantV, wantI := a*vA, a*iA
			if load.ID == "B_RL" {
				wantV, wantI = b*vB, b*iB
			}
			if math.Abs(load.Voltage-wantV) > 2e-6 || math.Abs(load.Current-wantI) > 2e-8 {
				t.Fatalf("%s got %g V, %g A, want %g V, %g A", load.ID, load.Voltage, load.Current, wantV, wantI)
			}
		}
		// The two inverters each draw 5 V across a 1 MΩ supply resistor.
		wantSupply := iA + iB + 2*5./1e6
		for _, source := range r.Sources {
			if source.ID == "VCC" && math.Abs(source.Current-wantSupply) > 2e-8 {
				t.Fatalf("shared supply current %g A, want %g A", source.Current, wantSupply)
			}
		}
	}
	check(initial, -1, 1)
	call(t, s, "simulation.step", StepParams{Duration: .022}, "first-direction-edge")
	check(s.Snapshot(), 1, -1)
	call(t, s, "simulation.step", StepParams{Duration: .011}, "second-direction-edge")
	check(s.Snapshot(), 1, 1)

	longDuration, longStep := .9, .001
	started := call(t, s, "simulation.verify", VerifyParams{Duration: &longDuration, Step: &longStep}, "long-dual").Result.(Snapshot)
	call(t, s, "operation.cancel", IDParams{ID: started.LastOperation}, "cancel-dual")
	canceled := awaitOperation(t, s, started.LastOperation)
	if canceled.Operation.Status != "canceled" || canceled.Metadata == nil || canceled.Metadata.SampleCount != 0 {
		t.Fatal("canceling the dual bridge run published partial results", canceled.Operation)
	}
	check(s.Snapshot(), 1, 1)

	duration, step := .045, .001
	_, captured := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: initial.Scope.Selected}, "record-dual")
	read := func(service *Service, id string) VerificationPage {
		t.Helper()
		return call(t, service, "operation.samples", VerificationPageParams{ID: id, Limit: 200}, "read-dual").Result.(VerificationPage)
	}
	rows := read(s, captured.Operation.ID).Rows
	if len(rows) < 46 || len(initial.Scope.Selected) != 6 {
		t.Fatal("dual bridge capture missed output rows or probes")
	}
	seen := map[[2]float64]bool{}
	for _, row := range rows {
		if len(row.Values) != 6 {
			t.Fatal("missing dual bridge probe values", row)
		}
		values := row.Values
		a, b := values[0], values[1]
		if a != 0 && a != 5 || b > 1e-8 && b < 5-1e-8 {
			continue // Only compare plateaus with the scalar on-channel load line.
		}
		if b < 2.5 {
			b = 0
		} else {
			b = 5
		}
		seen[[2]float64{a, b}] = true
		if math.Abs(values[2]-(1-.4*a)*vA) > 2e-6 || math.Abs(values[3]-(1-.4*b)*vB) > 2e-6 || math.Abs(values[4]-(1-.4*a)*iA) > 2e-8 || math.Abs(values[5]-(1-.4*b)*iB) > 2e-8 {
			t.Fatal("dual bridge plateau differs from the independent load lines", row)
		}
	}
	if len(seen) != 4 {
		t.Fatal("capture missed a combination of bridge directions", seen)
	}
	call(t, s, "recording.keep.set", RecordingKeepParams{ID: captured.Operation.ID, Keep: true}, "keep-dual")
	path := filepath.Join(t.TempDir(), "dual.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save-dual")
	fresh := New()
	t.Cleanup(fresh.Close)
	call(t, fresh, "project.load", LoadParams{Path: path}, "open-dual")
	reopened := settled(t, fresh)
	check(reopened, -1, 1)
	if !reopened.Paused || reopened.Document.Dirty || len(reopened.Operations) != 1 || !reflect.DeepEqual(initial.Components, reopened.Components) || !reflect.DeepEqual(initial.Wires, reopened.Wires) || !reflect.DeepEqual(rows, read(fresh, reopened.Operations[0].ID).Rows) {
		t.Fatal("dual bridge project or its retained capture changed on reopen")
	}
}
