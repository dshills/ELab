package transport

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestCapacityNoiseSocketMatchesDirectSnapshot(t *testing.T) {
	f, err := os.Open("../workbench/testdata/noise-capacity.elab.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	data, err := io.ReadAll(z)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "noise.elab")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	s := workbench.New()
	defer s.Close()
	state, err := s.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	descriptor := filepath.Join(dir, "endpoint.json")
	server, err := Listen(s, descriptor)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	params, _ := json.Marshal(map[string]string{"path": path})
	rev := state.CommandRevision
	req := workbench.Request{Version: 1, ID: "load", Command: "project.load", Params: params, ExpectedCommandRevision: &rev, Key: "load"}
	raw, _ := json.Marshal(req)
	var output bytes.Buffer
	if err = RemoteStdio(context.Background(), descriptor, bytes.NewReader(append(raw, '\n')), &output); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result workbench.Snapshot
		Error  *workbench.Failure
	}
	if err = json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	if output.Len() <= 1024*1024 || output.Len() > MaxResponseBytes {
		t.Fatal("fixture no longer tests transport boundary", output.Len())
	}
	direct, _ := json.Marshal(s.Snapshot().DCSweep.Result)
	remote, _ := json.Marshal(response.Result.DCSweep.Result)
	if !bytes.Equal(direct, remote) {
		t.Fatal("socket truncated or changed noise results")
	}
}
