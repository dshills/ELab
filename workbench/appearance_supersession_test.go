package workbench

import (
	"context"
	"testing"
)

func TestAppearanceWorkCancellationPreservesEarlierSupersession(t *testing.T) {
	path := appearancePath(t, "light")
	s := New()
	entered, release := make(chan struct{}), make(chan struct{})
	done := appearanceRelease(t, s, release)
	s.appearanceRead = func(ctx context.Context, path string) (appearanceLoaded, error) {
		close(entered)
		<-release
		return readAppearance(ctx, path)
	}
	first := appearanceCommand(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "first").Result.(Snapshot).AppearanceWork
	<-entered
	latest := appearanceCommand(t, s, "appearance.theme.set", map[string]string{"theme": "dark"}, "latest").Result.(Snapshot).AppearanceWork
	appearanceCommand(t, s, "appearance.work.cancel", IDParams{ID: latest.ID}, "cancel-latest")
	if old := inspectAppearanceWork(t, s, first.ID); !old.Superseded || old.Status != "canceling" {
		t.Fatalf("canceling the newest request erased earlier supersession: %+v", old)
	}
	if work := inspectAppearanceWork(t, s, latest.ID); work.Status != "canceled" || work.Superseded || work.Published {
		t.Fatalf("queued cancellation outcome: %+v", work)
	}
	done()
	if old := awaitAppearanceWork(t, s, first.ID, "superseded"); !old.Superseded || old.Published {
		t.Fatalf("superseded read outcome changed: %+v", old)
	}
	if s.Snapshot().Theme != "dark" || s.Snapshot().Appearance.Status != "disabled" || savedAppearance(t, path) != "light" {
		t.Fatal("queued cancellation changed the theme or saved file")
	}
}
