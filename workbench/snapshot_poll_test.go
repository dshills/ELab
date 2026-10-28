package workbench

import "testing"

func TestSnapshotIfChangedSkipsIdleAndPublishesDetachedState(t *testing.T) {
	s := New()
	defer s.Close()
	before := settled(t, s)
	if _, changed := s.SnapshotIfChanged(before.Revision); changed {
		t.Fatal("unchanged state rebuilt")
	}
	call(t, s, "example.load", ExampleLoadParams{ID: "rc"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	next, changed := s.SnapshotIfChanged(before.Revision)
	if !changed || len(next.Components) == 0 {
		t.Fatal("publication missed")
	}
	revision := next.Revision
	next.Components[0].ID = "corrupt"
	if s.Snapshot().Components[0].ID == "corrupt" {
		t.Fatal("poll exposed service ownership")
	}
	if _, changed := s.SnapshotIfChanged(revision); changed {
		t.Fatal("poll changed revision")
	}
	if a := testing.AllocsPerRun(100, func() { s.SnapshotIfChanged(revision) }); a != 0 {
		t.Fatal("idle poll allocates", a)
	}
	s.Close()
	closed, changed := s.SnapshotIfChanged(revision)
	if !changed || !closed.Closed {
		t.Fatal("shutdown publication missed")
	}
}
func BenchmarkSnapshotPolling(b *testing.B) {
	s := NewExample()
	defer s.Close()
	if _, err := s.Wait(b.Context()); err != nil {
		b.Fatal(err)
	}
	revision := s.Snapshot().Revision
	b.Run("full", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			s.Snapshot()
		}
	})
	b.Run("unchanged", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, changed := s.SnapshotIfChanged(revision); changed {
				b.Fatal("idle fixture changed")
			}
		}
	})
}
