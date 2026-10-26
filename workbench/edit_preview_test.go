package workbench

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestBatchPreviewPureExactDiffAndGuardedCommit(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	p := BatchParams{Name: "Slow the RC response", Edits: []BatchEdit{edit("component.set", ValueParams{ID: "R1", Value: "20k"}), edit("component.set", ValueParams{ID: "C1", Value: "2uF"})}}
	r := call(t, s, "edit.batch.preview", p, "preview")
	preview := r.Result.(*BatchPreview)
	if len(preview.Changes) != 2 || preview.Changes[0].ID != "C1" || preview.Changes[1].ID != "R1" || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("preview modified circuit or missed exact changed components", preview)
	}
	raw, _ := json.Marshal(p)
	r = s.Dispatch(context.Background(), Request{Version: 1, ID: "commit", Command: "edit.batch", Params: raw, ExpectedCommandRevision: &preview.CommandRevision, ExpectedDocumentGeneration: &preview.DocumentGeneration, Key: "commit"}, "test")
	if r.Error != nil || s.Snapshot().UndoName != p.Name {
		t.Fatal("shared atomic commit did not accept its preview", r.Error)
	}
	actual := schematicChanges(Graph{Components: before.Components, Wires: before.Wires}, s.graph, before.TransientSettings, s.transientSettings)
	if !reflect.DeepEqual(actual, preview.Changes) {
		t.Fatal("committed content differs from preview")
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	if !reflect.DeepEqual(s.Snapshot().Components, before.Components) {
		t.Fatal("one Undo failed to reverse the whole previewed change")
	}
}

func TestProposalIsSharedNonDestructiveAndRetiresStaleInputs(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	p := BatchParams{Name: "Tune resistance", Edits: []BatchEdit{edit("component.set", ValueParams{ID: "R1", Value: "20k"})}}
	before := s.Snapshot()
	call(t, s, "edit.proposal.set", p, "propose")
	proposed := s.Snapshot()
	if proposed.Proposal == nil || proposed.Proposal.Actor != "test" || proposed.Proposal.Stale || !reflect.DeepEqual(before.Components, proposed.Components) || before.Document != proposed.Document || !reflect.DeepEqual(before.Transient, proposed.Transient) || !reflect.DeepEqual(before.UndoEntry, proposed.UndoEntry) {
		t.Fatal("proposal changed circuit/dirty/physics/history")
	}
	oldID := proposed.Proposal.ID
	call(t, s, "view.pan", map[string]float64{"dx": 10, "dy": 20}, "pan")
	if s.Snapshot().Proposal.Stale {
		t.Fatal("view-only change invalidated a source-identical proposal")
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "30k"}, "user-edit")
	if !s.Snapshot().Proposal.Stale {
		t.Fatal("newer component edit did not retire preview")
	}
	stable := s.Snapshot()
	refuse(t, s, "edit.proposal.apply", IDParams{ID: oldID}, "conflict")
	if !reflect.DeepEqual(stable, s.Snapshot()) {
		t.Fatal("stale application overwrote newer user edit")
	}
	call(t, s, "edit.proposal.set", p, "refresh")
	fresh := s.Snapshot().Proposal
	if fresh.ID == oldID || fresh.Stale {
		t.Fatal("refresh did not create a new exact proposal identity")
	}
	refuse(t, s, "edit.proposal.dismiss", IDParams{ID: oldID}, "conflict")
	call(t, s, "edit.proposal.apply", IDParams{ID: fresh.ID}, "apply")
	if s.Snapshot().Proposal != nil || s.Snapshot().UndoName != p.Name || s.Snapshot().UndoEntry.Actor != "test" {
		t.Fatal("proposal not committed as one named edit")
	}
	part, _ := s.graph.Component("R1")
	if part.Value != 20000 {
		t.Fatal("apply did not use the newly reviewed batch")
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	part, _ = s.graph.Component("R1")
	if part.Value != 30000 {
		t.Fatal("Undo failed to restore the pre-apply user edit")
	}
}

func TestProposalReservesNoIDsButRetiresAfterAllocationAndReplacement(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	batch := BatchParams{Name: "Add RTEST", Edits: []BatchEdit{edit("component.place", PlaceParams{ID: "RTEST", Kind: "resistor", Position: Point{500, 500}})}}
	call(t, s, "edit.proposal.set", batch, "propose")
	if s.usedIDs["RTEST"] {
		t.Fatal("preview reserved an ID")
	}
	// Reserving another ID changes automatic allocation even when final graph
	// is identical. The old preview must not silently change its predicted IDs.
	call(t, s, "edit.batch", BatchParams{Name: "Temporary", Edits: []BatchEdit{edit("component.place", PlaceParams{ID: "Temporary", Kind: "resistor", Position: Point{600, 600}}), edit("object.delete", IDParams{ID: "Temporary"})}}, "temporary")
	if !s.Snapshot().Proposal.Stale {
		t.Fatal("lifetime allocation change did not retire the preview")
	}
	id := s.Snapshot().Proposal.ID
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "new-document")
	if s.Snapshot().Proposal != nil || s.proposal != nil {
		t.Fatal("document replacement retained proposal")
	}
	refuse(t, s, "edit.proposal.apply", IDParams{ID: id}, "conflict")
}

func TestExampleExperimentDifferencesTrackEditsAndUndo(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	if len(s.Snapshot().Example.Changes) != 0 {
		t.Fatal("original template advertises changes")
	}
	call(t, s, "component.set", ValueParams{ID: "C1", Value: "2uF"}, "experiment")
	state := s.Snapshot()
	if !state.Example.Modified || len(state.Example.Changes) != 1 || state.Example.Changes[0].ID != "C1" || state.Example.Info.Try == "" || state.Example.Info.Expected == "" {
		t.Fatal("experiment lost the suggested observation or exact diff")
	}
	copy := cloneSnapshot(state)
	copy.Example.Changes[0].Before[0] = '!'
	if !json.Valid(state.Example.Changes[0].Before) {
		t.Fatal("caller corrupted template comparison")
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	if s.Snapshot().Example.Modified || len(s.Snapshot().Example.Changes) != 0 {
		t.Fatal("Undo failed to restore the original experiment")
	}
}

func TestProposalRetryCopiesAndNestedParameterDetails(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	batch := BatchParams{Name: "Capacitor startup", Edits: []BatchEdit{edit("capacitor.initial.set", ValueParams{ID: "C1", Value: "1 V"})}}
	raw, _ := json.Marshal(batch)
	revision := s.Snapshot().CommandRevision
	req := Request{Version: 1, ID: "stage", Command: "edit.proposal.set", Params: raw, ExpectedCommandRevision: &revision, Key: "stage-key"}
	response := s.Dispatch(context.Background(), req, "test")
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	returned := response.Result.(Snapshot).Proposal
	if len(returned.Preview.Changes) != 1 || !strings.Contains(strings.Join(returned.Preview.Changes[0].Details, " "), "initial voltage") {
		t.Fatal("non-value parameter missing from human-readable review", returned)
	}
	returned.Batch.Edits[0].Params[0] = '!'
	returned.Preview.Changes[0].Details[0] = "caller edit"
	replayed := s.Dispatch(context.Background(), req, "test")
	owned := replayed.Result.(Snapshot).Proposal
	if !replayed.Duplicate || !json.Valid(owned.Batch.Edits[0].Params) || owned.Preview.Changes[0].Details[0] == "caller edit" {
		t.Fatal("retry cache shared mutable proposal state")
	}
	call(t, s, "edit.proposal.apply", IDParams{ID: owned.ID}, "apply")
	again := s.Dispatch(context.Background(), req, "test")
	if !again.Duplicate || s.Snapshot().Proposal != nil {
		t.Fatal("retry replay recreated an already applied proposal")
	}
}
