package workbench

import (
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
	b := exampleBuilder{graph: discreteHBridgeExampleGraph()}
	b.part("M5", "nmos", "", 680, 145, 0)
	for _, pin := range []string{"gate", "drain", "source"} {
		b.wire("M5."+pin, "MNB."+pin)
	}
	s := newGraphService(SolveLED, b.graph)
	defer s.Close()
	refused := settled(t, s)
	if refused.Status != "unsupported" || len(refused.Diagnostics) == 0 || refused.Diagnostics[0].Code != "mixed-limit" {
		t.Fatal("expected a size refusal with the fifth FET", refused.Diagnostics)
	}
	path := filepath.Join(t.TempDir(), "five-fets.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	loaded := settled(t, fresh)
	if loaded.Document.Dirty || !reflect.DeepEqual(loaded.Components, b.graph.Components) || !reflect.DeepEqual(loaded.Wires, b.graph.Wires) {
		t.Fatal("saving an over-limit circuit lost the fifth FET or its wiring")
	}
}

func TestFourFETBudgetRetainsTwoJFETLimit(t *testing.T) {
	b := exampleBuilder{graph: JFETCurrentExampleGraph(false)}
	addParallel := func(id, kind string) {
		b.part(id, kind, "", 420, float64(len(b.graph.Components))*20, 0)
		for _, pin := range jfetPins {
			b.wire(id+"."+pin, "J1."+pin)
		}
	}
	addParallel("J2", "njfet")
	addParallel("M1", "nmos")
	addParallel("M2", "nmos")
	s := newGraphService(SolveLED, b.graph)
	defer s.Close()
	state := settled(t, s)
	if state.Mixed == nil || state.Mixed.Stale || len(state.Mixed.JFETs) != 2 || len(state.Mixed.MOSFETs) != 2 || state.Mixed.KCLResidual > 1e-8 {
		t.Fatal("two JFETs and two MOSFETs did not share the four-device budget", state.Diagnostics)
	}
	addParallel("M3", "nmos")
	if _, _, d := b.graph.Compile(); d == nil || d.Code != "mixed-limit" {
		t.Fatal("fifth device in mixed FET network was not refused", d)
	}
}
