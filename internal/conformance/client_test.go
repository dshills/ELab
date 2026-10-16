package conformance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
)

type interferingClient struct {
	Client
	before func(workbench.Request) error
}

func (c *interferingClient) Dispatch(ctx context.Context, r workbench.Request) (workbench.Response, error) {
	if c.before != nil {
		if err := c.before(r); err != nil {
			return workbench.Response{}, err
		}
	}
	return c.Client.Dispatch(ctx, r)
}

func TestAdmissionRetriesOnlyRefusedPresentationRaces(t *testing.T) {
	for _, mode := range []string{"layout", "electrical", "document", "invalid", "transport", "continuous"} {
		t.Run(mode, func(t *testing.T) {
			base, err := InProcess(t.Context(), t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer base.Close()
			attempts := 0
			client := &interferingClient{Client: base}
			client.before = func(r workbench.Request) error {
				if r.ExpectedCommandRevision == nil {
					return nil
				}
				attempts++
				if mode == "transport" {
					return errors.New("transport disconnected")
				}
				if mode == "invalid" || (attempts > 1 && mode != "continuous") {
					return nil
				}
				command, params := "view.viewport.set", `{"width_px":600,"height_px":300}`
				if mode == "electrical" {
					command, params = "component.place", `{"id":"R99","kind":"resistor","position":{"x":0,"y":0}}`
				} else if mode == "document" {
					command, params = "example.load", `{"id":"split-supply","discard_changes":true}`
				} else if mode == "continuous" {
					command, params = "view.pan", `{"dx":1,"dy":0}`
				}
				state := base.(*localClient).service.Snapshot()
				identity := fmt.Sprintf("interference-%d", attempts)
				response, err := base.Dispatch(t.Context(), workbench.Request{Version: 1, ID: identity, Key: identity, ExpectedCommandRevision: &state.CommandRevision, Command: command, Params: []byte(params)})
				if err != nil {
					return err
				}
				if response.Error != nil {
					t.Fatalf("interference setup: %+v", response.Error)
				}
				return nil
			}
			s := &session{client: client, ctx: t.Context(), directory: t.TempDir()}
			params := map[string]any{"id": "R1", "kind": "resistor", "position": workbench.Point{}}
			if mode == "invalid" {
				params["kind"] = "missing-device"
			}
			err = s.ask("component.place", params, true, nil)
			switch mode {
			case "layout":
				if err != nil || attempts != 2 || len(s.calls) != 1 || len(s.commands) != 1 || len(base.(*localClient).service.Snapshot().Components) != 1 {
					t.Fatal("layout retry changed logical evidence or applied twice", err, attempts, s.calls)
				}
			case "electrical", "document":
				if err == nil || !strings.Contains(err.Error(), "state changed") || attempts != 1 {
					t.Fatal("unexpected state replacement was retried", err, attempts)
				}
			case "continuous":
				if err == nil || !strings.Contains(err.Error(), "eight attempts") || attempts != 8 {
					t.Fatal("retry budget not enforced", err, attempts)
				}
			default:
				if err == nil || attempts != 1 {
					t.Fatal("non-conflict failure was retried", err, attempts)
				}
			}
		})
	}
}
