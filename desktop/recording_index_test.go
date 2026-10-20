package main

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeRecordedIndexSharesStrictIntegerValidation(t *testing.T) {
	s, cleanup := recordedCursorRC(t)
	defer cleanup()
	for _, number := range []float32{.5, 1.5, 19.75} {
		t.Run(fmt.Sprint(number), func(t *testing.T) {
			before := s.service.Snapshot()
			raw, err := json.Marshal(map[string]any{"index": number})
			if err != nil {
				t.Fatal(err)
			}
			revision := before.CommandRevision
			response := s.service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "fraction", Key: fmt.Sprint("fraction-", number), Command: "recording.seek", Params: raw, ExpectedCommandRevision: &revision}, "agent.test")
			if response.Error == nil || response.Error.Code != "invalid-params" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("shared fractional-index request did not refuse without mutation", response.Error)
			}
			s.app.Do(func() {
				field, ok := findNode(s.view(), "recorded-index")
				if !ok || field.Change == nil {
					t.Fatal("recorded index control missing")
				}
				field.Change(ui.ChangeEvent{Number: number})
			})
			if !reflect.DeepEqual(before, s.service.Snapshot()) || s.errorMessage != response.Error.Message || s.requestHelp == nil {
				t.Fatal("native index truncated an invalid entry or lost shared refusal feedback", s.errorMessage)
			}
		})
	}
	for _, index := range []int{4, 20, 0} {
		before := s.service.Snapshot()
		uiAction(t, s, "recorded-index", &ui.ChangeEvent{Number: float32(index)})
		after := s.service.Snapshot()
		if after.Recording.Index != index || after.Recording.Time != float64(index)*.001 || after.CommandRevision != before.CommandRevision+1 || s.errorMessage != "" {
			t.Fatal("valid integer seek did not recover once at the exact stored row", after.Recording.Index, after.Recording.Time, s.errorMessage)
		}
		assertCursorMeasurementPreservesPhysics(t, before, after)
	}
}
