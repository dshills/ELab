package workbench

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func awaitSweep(t *testing.T, s *Service, status string) Snapshot {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		state := s.Snapshot()
		if state.DCSweep != nil {
			for _, op := range state.DCSweep.Operations {
				if op.ID == state.DCSweep.Selected && op.Status == status {
					return state
				}
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("sweep did not reach %s: %+v", status, s.Snapshot().DCSweep)
	return Snapshot{}
}
func sweepSettingsForLED() SweepSettings {
	return SweepSettings{ComponentID: "V1", Parameter: "voltage", Start: "0 V", Stop: "5 V", Points: 11, ProbeIDs: []string{"i:D1", "v:D1"}}
}
func TestSharedDCSweepPreservesLiveCircuitOwnsResultsAndSaves(t *testing.T) {
	s := NewExample()
	defer s.Close()
	before := settled(t, s)
	cfg := sweepSettingsForLED()
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "start")
	after := awaitSweep(t, s, "completed")
	state := after.DCSweep
	if after.Document.SchematicDirty != before.Document.SchematicDirty || !after.Document.RecordingsDirty || after.Document.RecordingCount != 1 || after.Document.FormatVersion != 5 || !reflect.DeepEqual(after.Components, before.Components) || after.RunID != before.RunID || after.ElectricalRevision != before.ElectricalRevision || !reflect.DeepEqual(after.Result, before.Result) || !reflect.DeepEqual(after.UndoEntry, before.UndoEntry) {
		t.Fatal("sweep mutated live source or failed recording semantics")
	}
	if len(state.Result.Points) != 11 || !state.Operations[0].SourceUnchanged || state.Operations[0].Actor != "test" {
		t.Fatal("sweep capture incomplete", state)
	}
	id := state.Selected
	inspection := call(t, s, "sweep.inspect", IDParams{ID: id}, "inspect").Result.(SweepInspection)
	inspection.Metadata.Source.Graph.Components[0].Value = 12
	inspection.Result.Points[0].Values[0] = 1
	original := call(t, s, "sweep.inspect", IDParams{ID: id}, "again").Result.(SweepInspection)
	if original.Metadata.Source.Graph.Components[0].Value == 12 || original.Result.Points[0].Values[0] == 1 {
		t.Fatal("query aliases job data")
	}
	path := filepath.Join(t.TempDir(), "sweep.elab")
	call(t, s, "sweep.cursor.set", SweepCursorParams{Index: ptrSweepIndex(7)}, "cursor")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	document, err := decodeDocument(data)
	if err != nil || document.Version != 5 || len(document.Sweeps.Entries) != 1 {
		t.Fatal("version-5 sweep save", err)
	}
	loaded := New()
	defer loaded.Close()
	call(t, loaded, "project.load", LoadParams{Path: path}, "open")
	got := settled(t, loaded)
	if got.Document.Dirty || got.Document.RecordingCount != 1 || got.DCSweep == nil || got.DCSweep.Cursor == nil || *got.DCSweep.Cursor != 7 || !reflect.DeepEqual(got.DCSweep.Result, state.Result) {
		t.Fatal("saved curve was not restored", got.Document)
	}
	call(t, loaded, "sweep.keep.set", RecordingKeepParams{ID: got.DCSweep.Selected, Keep: false}, "exclude")
	if loaded.Snapshot().Document.FormatVersion != 4 || !loaded.Snapshot().Document.RecordingsDirty {
		t.Fatal("excluding last sweep did not preserve version-4 writing")
	}
}
func ptrSweepIndex(v int) *int { return &v }
func TestBlockedDCSweepAllowsEditingAndCancellation(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	entered, release := make(chan sweepPlan, 1), make(chan struct{})
	s.sweepCompute = func(ctx context.Context, plan sweepPlan, progress func(int)) (*SweepResult, error) {
		entered <- plan
		select {
		case <-release:
			return computeDCSweep(ctx, plan, progress)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	cfg := sweepSettingsForLED()
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "start")
	plan := <-entered
	id := s.Snapshot().DCSweep.Selected
	refuse(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "busy")
	promptLiveCommand(t, s, "component.move", MoveParams{ID: "R1", Position: Point{X: 500, Y: 300}})
	promptLiveCommand(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"})
	if s.Snapshot().DCSweep.Operations[0].SourceUnchanged {
		t.Fatal("later source edit not reflected")
	}
	for _, p := range plan.graph.Components {
		if p.ID == "R1" && p.Value != 1000 {
			t.Fatal("capture consumed later edit")
		}
	}
	call(t, s, "sweep.cancel", IDParams{ID: id}, "cancel")
	after := awaitSweep(t, s, "canceled")
	if after.DCSweep.Result != nil || after.Document.RecordingCount != 0 {
		t.Fatal("canceled sweep leaked partial/kept data")
	}
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "again")
	<-entered
	done := make(chan struct{})
	go func() { s.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("close did not join canceled sweep worker")
	}
}
func TestSweepFailureAndTamperedStorage(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	s.sweepCompute = func(context.Context, sweepPlan, func(int)) (*SweepResult, error) {
		return nil, errors.New("controlled solver failure")
	}
	cfg := sweepSettingsForLED()
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "start")
	after := awaitSweep(t, s, "failed")
	if after.DCSweep.Result != nil || after.Document.RecordingCount != 0 || after.DCSweep.Operations[0].Error.Code != "sweep-failed" {
		t.Fatal("failed sweep published data")
	}
	plan, err := newSweepPlan(ExampleGraph(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	result, err := computeDCSweep(context.Background(), plan, func(int) {})
	if err != nil {
		t.Fatal(err)
	}
	record := SweepRecord{Metadata: SweepMetadata{Format: "elab.dc-sweep", Version: 1, Engine: DCSweepEngineID, OperationID: "fixture", Source: dcCircuitSnapshot(plan.graph, DefaultTransientSettings()), Settings: plan.settings, Sources: plan.sources}, Result: *result}
	raw, _ := json.Marshal(record)
	if _, err := validateSweepRecord(raw); err != nil {
		t.Fatal("own record rejected", err)
	}
	record.Result.Points[0].Input = math.Nextafter(0, 1)
	raw, _ = json.Marshal(record)
	if _, err := validateSweepRecord(raw); err == nil {
		t.Fatal("accepted corrupted axis")
	}
}

func TestSweepExportCapturesOriginalDataAndCancellationPreservesFile(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	cfg := sweepSettingsForLED()
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "start")
	state := awaitSweep(t, s, "completed")
	id := state.DCSweep.Selected
	directory := t.TempDir()
	path := filepath.Join(directory, "curve.json")
	entered, release := blockExportPreparation(t, s)
	defer release()
	call(t, s, "export.start", ExportStartParams{Kind: "sweep-json", ID: id, Path: path}, "export")
	<-entered
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "edit")
	release()
	awaitProjectStatus(t, s, "completed")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	record, err := validateSweepRecord(data)
	if err != nil || !reflect.DeepEqual(record.Result, *state.DCSweep.Result) {
		t.Fatal("export substituted current editor data", err)
	}
	s.mu.Lock()
	s.projectExport = prepareProjectExport
	s.mu.Unlock()
	refuse(t, s, "sweep.export", VerificationExportParams{ID: id, Path: path, Format: "json"}, "file-exists")
	// A second held export cannot replace the existing file after cancellation.
	entered, release2 := blockExportPreparation(t, s)
	defer release2()
	call(t, s, "export.start", ExportStartParams{Kind: "sweep-json", ID: id, Path: path, Overwrite: true}, "cancel-export")
	<-entered
	work := s.Snapshot().ProjectWork
	call(t, s, "project.work.cancel", IDParams{ID: work.ID}, "cancel")
	release2()
	awaitProjectStatus(t, s, "canceled")
	current, err := os.ReadFile(path)
	if err != nil || !reflect.DeepEqual(data, current) {
		t.Fatal("canceled export replaced destination", err)
	}
}
func TestSweepRecordBoundsCursorGuardAndVersionRefusal(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	cfg := sweepSettingsForLED()
	for i := 0; i < MaxDCSweepOperations; i++ {
		call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, fmt.Sprintf("sweep-%d", i))
		awaitSweep(t, s, "completed")
	}
	refuse(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "record-limit")
	first := s.Snapshot().DCSweep.Operations[0].ID
	call(t, s, "sweep.keep.set", RecordingKeepParams{ID: first, Keep: false}, "exclude")
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "replacement")
	state := awaitSweep(t, s, "completed")
	refuse(t, s, "sweep.inspect", IDParams{ID: first}, "not-found")
	refuse(t, s, "sweep.cursor.set", SweepCursorParams{OperationID: first, Index: ptrSweepIndex(0)}, "conflict")
	if len(state.DCSweep.Operations) != 8 {
		t.Fatal("sweep retention grew beyond bound")
	}
	raw, _ := json.Marshal(s.documentLockedForTest())
	var object map[string]any
	_ = json.Unmarshal(raw, &object)
	object["version"] = float64(4)
	bad, _ := json.Marshal(object)
	if _, err := decodeDocument(bad); err == nil {
		t.Fatal("version 4 accepted sweep payload")
	}
	// A plot range that cannot distinguish all requested values is rejected.
	cfg.Start = "5"
	cfg.Stop = strconv.FormatFloat(math.Nextafter(5, 6), 'g', -1, 64)
	cfg.Points = 3
	if _, err := newSweepPlan(ExampleGraph(), cfg); err == nil {
		t.Fatal("duplicate representable axis points accepted")
	}
}
func (s *Service) documentLockedForTest() Document {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.documentLocked()
}
func TestPendingProjectOpenRefusesLaterSweepAdmission(t *testing.T) {
	path := loadFixture(t)
	s := NewExample()
	defer s.Close()
	settled(t, s)
	entered, release := blockProjectLoader(t, s, false)
	defer release()
	call(t, s, "project.load.start", LoadParams{Path: path, DiscardChanges: true}, "open")
	<-entered
	cfg := sweepSettingsForLED()
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "later-sweep")
	awaitSweep(t, s, "completed")
	release()
	awaitProjectStatus(t, s, "superseded")
	if s.Snapshot().DCSweep.Result == nil {
		t.Fatal("late open erased new curve")
	}
}

func TestSweepRecoveryRestoresKeptCurveAndConfiguredState(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	cfg := sweepSettingsForLED()
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "start")
	before := awaitSweep(t, s, "completed")
	call(t, s, "sweep.cursor.set", SweepCursorParams{Index: ptrSweepIndex(4)}, "cursor")
	configureRecovery(t, s, t.TempDir(), false)
	call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
	choice := recoveryChoice(t, s)
	if s.Snapshot().Recovery.Candidates[0].Recordings != 1 {
		t.Fatal("recovery omitted curve count")
	}
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "clear")
	choice.DiscardChanges = true
	call(t, s, "recovery.restore", choice, "restore")
	after := settled(t, s)
	if after.DCSweep == nil || !reflect.DeepEqual(after.DCSweep.Result, before.DCSweep.Result) || !reflect.DeepEqual(after.DCSweep.Settings, cfg) || after.DCSweep.Cursor == nil || *after.DCSweep.Cursor != 4 {
		t.Fatal("recovery lost captured curve or settings")
	}
}
func TestDCSweepCapacitorFloatingNodeFailsWithoutPartialCurve(t *testing.T) {
	g := EmptyGraph()
	for _, p := range []PlaceParams{{ID: "Supply", Kind: "dc-source", Position: Point{X: -200}}, {ID: "Return", Kind: "ground", Position: Point{X: 200}}, {ID: "CapA", Kind: "capacitor", Position: Point{Y: -100}}, {ID: "CapB", Kind: "capacitor", Position: Point{Y: 100}}, {ID: "Load", Kind: "resistor", Position: Point{X: 100, Y: -100}}, {ID: "Feed", Kind: "resistor", Position: Point{X: -100, Y: -100}}} {
		c, err := MakeComponent(p.ID, p.Kind, p.Position, "")
		if err != nil {
			t.Fatal(err)
		}
		g.Components = append(g.Components, c)
	}
	pairs := [][2]string{{"Supply.positive", "Feed.1"}, {"Feed.2", "CapA.1"}, {"CapA.2", "CapB.1"}, {"CapB.2", "Return.reference"}, {"Supply.negative", "Return.reference"}, {"Load.1", "Supply.positive"}, {"Load.2", "Return.reference"}}
	for i, p := range pairs {
		g.Wires = append(g.Wires, Wire{ID: fmtWire(i), From: p[0], To: p[1], Connected: true})
	}
	plan, err := newSweepPlan(g, SweepSettings{ComponentID: "Supply", Parameter: "voltage", Start: "0", Stop: "5", Points: 3, ProbeIDs: []string{"v:CapA"}})
	if err != nil {
		t.Fatal("transient-connected fixture invalid", err)
	}
	result, err := computeDCSweep(context.Background(), plan, func(int) {})
	if err == nil || result != nil {
		t.Fatal("DC floating capacitor junction fabricated a curve")
	}
	for i := range g.Components {
		if g.Components[i].ID == "Supply" {
			g.Components[i].Value = 0
		}
	}
	plan, err = newSweepPlan(g, SweepSettings{ComponentID: "Load", Parameter: "resistance", Start: "100", Stop: "1k", Points: 3, ProbeIDs: []string{"v:CapA"}})
	if err != nil {
		t.Fatal(err)
	}
	result, err = computeDCSweep(context.Background(), plan, func(int) {})
	if err == nil || result != nil {
		t.Fatal("zero-residual floating island was accepted")
	}

}
