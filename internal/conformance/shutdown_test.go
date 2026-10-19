package conformance

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
)

type shutdownClient struct {
	Client
	mode       string
	closed     bool
	closeCalls int
	closeError error
}

func (c *shutdownClient) Dispatch(ctx context.Context, request workbench.Request) (workbench.Response, error) {
	response, err := c.Client.Dispatch(ctx, request)
	if request.Command != "app.close" || err != nil {
		return response, err
	}
	c.closeCalls++
	state := response.Result.(workbench.Snapshot)
	switch c.mode {
	case "failed":
		state.Shutdown.Status, state.Shutdown.ExitReady = "failed", false
		state.Shutdown.RecoveryError = "fixture recovery failure"
	case "acknowledged-failure":
		state.Shutdown.Status = "failed"
	case "running":
		state.Shutdown.Status, state.Shutdown.WorkersJoined, state.Shutdown.ExitReady = "running", false, false
	case "missing":
		state.Shutdown = nil
	case "open":
		state.Closed = false
	case "transport":
		return response, errors.New("fixture lost close response")
	case "refused":
		response.Error = &workbench.Failure{Code: "canceled", Message: "fixture close admitted but wait canceled"}
	}
	response.Result = state
	return response, nil
}

func (c *shutdownClient) Close() error {
	c.closed = true
	return errors.Join(c.Client.Close(), c.closeError)
}

func TestElectricalShutdownRequiresCompletedCleanupAndRetainsResponse(t *testing.T) {
	for _, mode := range []string{"normal", "failed", "acknowledged-failure", "running", "missing", "open", "transport", "refused", "existing-evidence", "earlier-failure"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			var owned *shutdownClient
			factory := func(ctx context.Context, path string) (Client, error) {
				base, err := InProcess(ctx, path)
				owned = &shutdownClient{Client: base, mode: mode}
				return owned, err
			}
			failure := errors.New("fixture workflow failure")
			err := electricalSession(t.Context(), factory, dir, func(*session) error {
				if mode == "existing-evidence" {
					return os.WriteFile(filepath.Join(dir, "shutdown-response.json"), []byte("original evidence"), 0600)
				}
				if mode == "earlier-failure" {
					owned.closeError = errors.New("fixture process close failure")
					return failure
				}
				return nil
			})
			if (err == nil) != (mode == "normal") || !owned.closed || owned.closeCalls != 1 {
				t.Fatal("close outcome/ownership", err, owned.closed, owned.closeCalls)
			}
			data, readErr := os.ReadFile(filepath.Join(dir, "shutdown-response.json"))
			if readErr != nil {
				t.Fatal(readErr)
			}
			if mode == "existing-evidence" {
				if string(data) != "original evidence" || !strings.Contains(err.Error(), "record shutdown response") {
					t.Fatal("overwrote earlier response or lost write failure", string(data), err)
				}
				return
			}
			if mode == "earlier-failure" && (!errors.Is(err, failure) || !errors.Is(err, owned.closeError)) {
				t.Fatal("cleanup hid earlier or process failure", err)
			}
			var observed shutdownObservation
			if err := json.Unmarshal(data, &observed); err != nil {
				t.Fatal(err)
			}
			if observed.Started.IsZero() || observed.Finished.Before(observed.Started) || observed.Response.ID == "" || observed.Response.Version != 1 {
				t.Fatal("missing response identity/timing", observed)
			}
			wantError := mode != "normal" && mode != "earlier-failure"
			if (observed.Error != "") != wantError {
				t.Fatal("lost response failure", observed)
			}
			if mode == "failed" && !strings.Contains(string(data), "fixture recovery failure") {
				t.Fatal("lost cleanup failure detail", string(data))
			}
			if mode == "refused" && (observed.Response.Error == nil || observed.Response.Error.Code != "canceled") {
				t.Fatal("lost public refusal", observed)
			}
			if mode == "normal" {
				var state workbench.Snapshot
				encoded, _ := json.Marshal(observed.Response.Result)
				if err := json.Unmarshal(encoded, &state); err != nil || !state.ReadyToExit() || state.Shutdown.Status != "completed" {
					t.Fatal("missing successful terminal cleanup", state, err)
				}
			}
		})
	}
}
