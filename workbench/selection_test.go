package workbench

import (
	"reflect"
	"testing"
)

func selectionCircuit(t *testing.T) *Service {
	t.Helper()
	s := New()
	t.Cleanup(s.Close)
	call(t, s, "project.clear", map[string]any{"discard_changes": true}, "blank")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	for i, id := range []string{"R1", "R2", "R3"} {
		call(t, s, "component.place", PlaceParams{ID: id, Kind: "resistor", Position: Point{float64(i) * 10, 0}, Value: "1k"}, "place-"+id)
	}
	call(t, s, "wire.connect", ConnectParams{ID: "W1", From: "R1.2", To: "R2.1"}, "inside")
	call(t, s, "wire.connect", ConnectParams{ID: "W2", From: "R2.2", To: "R3.1"}, "outside")
	return s
}
func TestSelectionMoveValidatesFinalPositionsAndUndoesOnce(t *testing.T) {
	s := selectionCircuit(t)
	before := s.Snapshot()
	// A member may move into another member's former position in the same edit.
	call(t, s, "selection.move", SelectionOffset{[]string{"R1", "R2"}, Point{-10, 0}}, "move-pair")
	after := s.Snapshot()
	if after.ElectricalRevision != before.ElectricalRevision || after.CommandRevision != before.CommandRevision+1 || after.UndoName != "Move selection" || !reflect.DeepEqual(before.Wires, after.Wires) {
		t.Fatal("group move changed electrical state or added extra history")
	}
	call(t, s, "edit.undo", struct{}{}, "undo-pair")
	if !reflect.DeepEqual(before.Components, s.Snapshot().Components) {
		t.Fatal("one undo did not restore both positions")
	}
	call(t, s, "edit.redo", struct{}{}, "redo-pair")
	if !reflect.DeepEqual(after.Components, s.Snapshot().Components) {
		t.Fatal("redo did not restore the group")
	}
	before = s.Snapshot()
	used := len(s.usedIDs)
	refuse(t, s, "selection.move", SelectionOffset{[]string{"R1", "R2"}, Point{20, 0}}, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) || len(s.usedIDs) != used {
		t.Fatal("colliding move partially changed the graph")
	}
}
func TestSelectionClipboardIsDetachedAndCopiesOnlyInternalWires(t *testing.T) {
	s := selectionCircuit(t)
	before := s.Snapshot()
	call(t, s, "selection.copy", SelectionIDs{[]string{"R1", "R2"}}, "copy")
	copied := s.Snapshot()
	id := copied.SelectionClipboardID
	if id == 0 || copied.Document != before.Document || copied.UndoName != before.UndoName {
		t.Fatal("copy dirtied document/history")
	}
	call(t, s, "component.set", ValueParams{"R1", "2k"}, "change-original")
	call(t, s, "selection.paste", SelectionPaste{id, Point{100, 100}}, "paste")
	pasted := s.Snapshot()
	if len(pasted.Components) != 5 || len(pasted.Wires) != 3 || pasted.UndoName != "Paste selection" {
		t.Fatal("paste copied an external wire or lost an internal one")
	}
	originals := map[string]bool{"R1": true, "R2": true, "R3": true}
	newIDs := map[string]bool{}
	for _, p := range pasted.Components {
		if !originals[p.ID] {
			newIDs[p.ID] = true
			if p.Value != 1000 {
				t.Fatal("clipboard followed a later original edit")
			}
		}
	}
	for _, w := range pasted.Wires {
		if w.ID != "W1" && w.ID != "W2" {
			a, _, _ := s.graph.Terminal(w.From)
			b, _, _ := s.graph.Terminal(w.To)
			if !newIDs[a.ID] || !newIDs[b.ID] {
				t.Fatal("pasted wire crosses to original")
			}
		}
	}
	call(t, s, "edit.undo", struct{}{}, "undo-paste")
	if len(s.Snapshot().Components) != 3 {
		t.Fatal("paste not one undo action")
	}
	call(t, s, "selection.paste", SelectionPaste{id, Point{200, 100}}, "paste-again")
	for _, p := range s.Snapshot().Components {
		if newIDs[p.ID] {
			t.Fatal("Undo released a reserved component ID")
		}
	}
	call(t, s, "selection.copy", SelectionIDs{[]string{"R3"}}, "other-copy")
	before = s.Snapshot()
	refuse(t, s, "selection.paste", SelectionPaste{id, Point{300, 100}}, "conflict")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("clipboard conflict changed state")
	}
}
func TestSelectionDuplicateDeleteAndFailureAreAtomic(t *testing.T) {
	s := selectionCircuit(t)
	before := s.Snapshot()
	call(t, s, "selection.duplicate", SelectionOffset{[]string{"R1", "R2"}, Point{100, 100}}, "duplicate")
	after := s.Snapshot()
	if after.CommandRevision != before.CommandRevision+1 || after.UndoName != "Duplicate selection" || len(after.Wires) != 3 {
		t.Fatal("duplicate was not atomic")
	}
	call(t, s, "edit.undo", struct{}{}, "undo-duplicate")
	if !reflect.DeepEqual(s.Snapshot().Components, before.Components) {
		t.Fatal("duplicate undo failed")
	}
	call(t, s, "selection.delete", SelectionIDs{[]string{"R1", "R2"}}, "delete")
	after = s.Snapshot()
	if len(after.Components) != 1 || len(after.Wires) != 0 || after.UndoName != "Delete selection" {
		t.Fatal("group delete lost boundary semantics")
	}
	call(t, s, "edit.undo", struct{}{}, "undo-delete")
	if !reflect.DeepEqual(s.Snapshot().Wires, before.Wires) {
		t.Fatal("delete undo failed")
	}
	for _, p := range []SelectionOffset{{[]string{"R1", "missing"}, Point{100, 100}}, {[]string{"R1", "R1"}, Point{100, 100}}, {[]string{"R1"}, Point{20000, 0}}} {
		before = s.Snapshot()
		used := len(s.usedIDs)
		refuse(t, s, "selection.duplicate", p, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) || used != len(s.usedIDs) {
			t.Fatal("rejected duplicate changed state")
		}
	}
}
func TestSelectionBlockCopyRemapsAllMembersAndPorts(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "rc-block", DiscardChanges: true}, "example")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "selection.duplicate", SelectionOffset{[]string{"FILTER"}, Point{1000, 0}}, "copy-block")
	if err := s.graph.validateBlocks(); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range s.graph.Components {
		if p.Block != nil && p.ID != "FILTER" {
			found = true
			for _, id := range p.Block.Members {
				if id == "R1" || id == "C1" {
					t.Fatal("copy points to original member")
				}
			}
		}
	}
	if !found {
		t.Fatal("missing block copy")
	}
	before := s.Snapshot()
	refuse(t, s, "selection.move", SelectionOffset{[]string{"R1"}, Point{100, 0}}, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("hidden-member refusal changed state")
	}
}
func TestSelectionDuplicateRenamesVisibleNetsTogether(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "project.clear", map[string]any{"discard_changes": true}, "blank")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	for i, id := range []string{"L1", "L2"} {
		call(t, s, "component.place", PlaceParams{ID: id, Kind: "net-label", Position: Point{float64(i) * 100, 0}, Value: "BUS"}, id)
	}
	call(t, s, "selection.duplicate", SelectionOffset{[]string{"L1", "L2"}, Point{0, 100}}, "copy-nets")
	copied := ""
	for _, p := range s.graph.Components {
		if p.ID != "L1" && p.ID != "L2" {
			if p.Label == "BUS" {
				t.Fatal("copied net connects to original")
			}
			if copied != "" && p.Label != copied {
				t.Fatal("copied pair lost shared net")
			}
			copied = p.Label
		}
	}
}

func TestSelectionCapacityRefusalPreservesIDsAndHistory(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "project.clear", map[string]any{"discard_changes": true}, "blank")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "component.place", PlaceParams{ID: "R1", Kind: "resistor", Position: Point{}, Value: "1k"}, "one")
	for len(s.Snapshot().Components)*2 <= MaxComponents {
		ids := []string{}
		for _, p := range s.Snapshot().Components {
			ids = append(ids, p.ID)
		}
		call(t, s, "selection.duplicate", SelectionOffset{ids, Point{float64(len(ids)) * 10, 0}}, "grow-"+s.Snapshot().Selected)
	}
	before := s.Snapshot()
	used := len(s.usedIDs)
	ids := []string{}
	for _, p := range before.Components {
		ids = append(ids, p.ID)
	}
	refuse(t, s, "selection.duplicate", SelectionOffset{ids, Point{5000, 0}}, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) || len(s.usedIDs) != used {
		t.Fatal("capacity refusal changed document, history or reserved IDs")
	}
}
