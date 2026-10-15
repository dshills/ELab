package transport

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func TestStdioRetainsFailedShutdownUntilAcknowledgement(t *testing.T) {
	s := workbench.New()
	defer s.Close()
	directory := t.TempDir()
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(workbench.RecoveryConfigureParams{Directory: directory, Interval: 1})
	rev := s.Snapshot().CommandRevision
	r := s.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "configure", Key: "configure", Command: "recovery.configure", ExpectedCommandRevision: &rev, Params: raw}, "test")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatal("missing owned lease fixture", err)
	}
	lease := filepath.Join(directory, entries[0].Name())
	if err := os.Remove(lease); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(lease, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lease, "retain"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	in, input := io.Pipe()
	output, out := io.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer input.Close()
	defer output.Close()
	go func() { <-ctx.Done(); input.CloseWithError(ctx.Err()); output.CloseWithError(ctx.Err()) }()
	done := make(chan error, 1)
	go func() { done <- Stdio(ctx, s, in, out) }()
	encoder, decoder := json.NewEncoder(input), json.NewDecoder(output)
	ask := func(command string, p any, mutation bool, target any) {
		t.Helper()
		params, _ := json.Marshal(p)
		request := workbench.Request{Version: 1, ID: command, Command: command, Params: params}
		if mutation {
			rev = s.Snapshot().CommandRevision
			request.ExpectedCommandRevision, request.Key = &rev, command
		}
		if err := encoder.Encode(request); err != nil {
			t.Fatal(err)
		}
		var response workbench.Response
		if err := decoder.Decode(&response); err != nil || response.Error != nil {
			t.Fatal("stdio shutdown response", err, response.Error)
		}
		data, _ := json.Marshal(response.Result)
		if err := json.Unmarshal(data, target); err != nil {
			t.Fatal(err)
		}
	}
	var admitted workbench.Snapshot
	ask("app.close.start", workbench.ReplaceParams{}, true, &admitted)
	if !admitted.Closed || admitted.Shutdown == nil {
		t.Fatal("stdio lost shutdown admission")
	}
	var work workbench.ShutdownWork
	for work.Status != "failed" {
		ask("app.close.inspect", workbench.IDParams{ID: admitted.Shutdown.ID}, false, &work)
	}
	if work.ExitReady || work.Error == nil || !work.WorkersJoined {
		t.Fatal("stdio hid failed resource cleanup")
	}
	var acknowledged workbench.Snapshot
	ask("app.close.acknowledge", workbench.IDParams{ID: work.ID}, true, &acknowledged)
	if !acknowledged.ReadyToExit() || acknowledged.Shutdown.Status != "failed" {
		t.Fatal("acknowledgement lost failed terminal outcome")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("stdio did not exit after acknowledgement")
	}
}
