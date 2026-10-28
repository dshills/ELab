package transport

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestAnalysisPagesSocketAndStdioParity(t *testing.T) {
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
	file := filepath.Join(dir, "capacity.elab")
	if err = os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	s := workbench.New()
	defer s.Close()
	before, err := s.Wait(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(workbench.LoadParams{Path: file})
	rev := before.CommandRevision
	r := s.Dispatch(t.Context(), workbench.Request{Version: 1, ID: "load", Command: "project.load", Params: raw, ExpectedCommandRevision: &rev, Key: "load"}, "test")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	before, err = s.Wait(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !before.Paused {
		t.Fatal("fixture must be paused before ownership comparisons")
	}
	descriptor := filepath.Join(dir, "endpoint.json")
	server, err := Listen(s, descriptor)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	for _, q := range []struct {
		command string
		params  any
	}{{"analysis.list", struct{}{}}, {"analysis.page", workbench.AnalysisPageParams{Kind: "sweep", ID: before.DCSweep.Selected, DocumentGeneration: before.Document.Generation, Offset: 50, Limit: 50}}} {
		params, err := json.Marshal(q.params)
		if err != nil {
			t.Fatal(err)
		}
		request, err := json.Marshal(workbench.Request{Version: 1, ID: q.command, Command: q.command, Params: params})
		if err != nil {
			t.Fatal(err)
		}
		request = append(request, '\n')
		var a, b bytes.Buffer
		if err = Stdio(t.Context(), s, bytes.NewReader(request), &a); err != nil {
			t.Fatal(err)
		}
		if err = RemoteStdio(t.Context(), descriptor, bytes.NewReader(request), &b); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a.Bytes(), b.Bytes()) || b.Len() > 512*1024 {
			t.Fatal("page parity/envelope", a.Len(), b.Len())
		}
		var response workbench.Response
		if err = json.Unmarshal(b.Bytes(), &response); err != nil || response.Error != nil {
			t.Fatal("page failed", err, response.Error)
		}
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("queries altered application state")
	}
}
