package main

import (
	"context"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func TestNativeInitializationReturnsUpdatingBeforePublication(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	var response workbench.Response
	s.app.Do(func() { response = s.invoke("example.flasher", workbench.ReplaceParams{}) })
	if response.Error != nil || response.Result.(workbench.Snapshot).Status != "updating" {
		t.Fatal("native initialization did not return immediate Updating state")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ready, err := s.service.Wait(ctx)
	if err != nil || ready.Mixed == nil || ready.Mixed.Stale || ready.Status != "running" {
		t.Fatal("native initializer did not publish fresh state", err)
	}
}
