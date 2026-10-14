package workbench

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestDocumentGuardRefusesOldContextAfterBackgroundPublication(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	path := loadFixture(t)
	entered, release := blockProjectLoader(t, s, false)
	defer release()
	admitted := promptLiveCommand(t, s, "project.load.start", LoadParams{Path: path, DiscardChanges: true}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	// An old circuit can be rendered after admission but before publication.
	revision, generation := admitted.CommandRevision, admitted.Document.Generation
	release()
	awaitProjectStatus(t, s, "completed")
	before := settled(t, s)
	if before.CommandRevision != revision || before.Document.Generation == generation {
		t.Fatal("fixture does not isolate publication from command admission")
	}
	raw, _ := json.Marshal(ValueParams{ID: "R1", Value: "3k"})
	req := Request{Version: 1, ID: "guarded", Command: "component.set", Params: raw, Key: "guarded", ExpectedCommandRevision: &revision, ExpectedDocumentGeneration: &generation}
	r := s.Dispatch(context.Background(), req, "agent")
	if r.Error == nil || r.Error.Code != "conflict" || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("old rendered context edited a replacement document", r.Error)
	}
	// Refreshing the optional context guard permits the same intended action.
	generation = before.Document.Generation
	r = s.Dispatch(context.Background(), req, "agent")
	if r.Error != nil || settled(t, s).Circuit.Resistance != 3000 {
		t.Fatal("current document context was refused", r.Error)
	}
	// Retry guards are not intent: a successful key keeps its original response.
	generation = 0
	r = s.Dispatch(context.Background(), req, "agent")
	if r.Error != nil || !r.Duplicate {
		t.Fatal("context guard broke successful retry retention", r.Error)
	}
}

func TestDocumentGuardIsOptionalAndDoesNotReplaceRevisionGuard(t *testing.T) {
	s := NewExample()
	defer s.Close()
	state := settled(t, s)
	raw, _ := json.Marshal(ValueParams{ID: "R1", Value: "2k"})
	gen := state.Document.Generation
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "context-only", Key: "context-only", Command: "component.set", Params: raw, ExpectedDocumentGeneration: &gen}, "agent")
	if r.Error == nil || r.Error.Code != "invalid-request" {
		t.Fatal("document guard incorrectly replaced revision admission")
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "legacy")
	if settled(t, s).Circuit.Resistance != 2000 {
		t.Fatal("existing clients without document guard stopped working")
	}
}
