package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func TestLiveSpectrumSharedStdioSocketAndRetry(t *testing.T) {
	s := workbench.New()
	defer s.Close()
	descriptor := filepath.Join(t.TempDir(), "endpoint.json")
	server, err := Listen(s, descriptor)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	serial := 0
	request := func(command string, p any, remote bool) (workbench.Response, []byte) {
		t.Helper()
		serial++
		raw, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		revision := s.Snapshot().CommandRevision
		req := workbench.Request{Version: 1, ID: fmt.Sprint(serial), Key: fmt.Sprint(serial), Command: command, Params: raw, ExpectedCommandRevision: &revision}
		data, err := json.Marshal(req)
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, '\n')
		var out bytes.Buffer
		if remote {
			err = RemoteStdio(t.Context(), descriptor, bytes.NewReader(data), &out)
		} else {
			err = Stdio(t.Context(), s, bytes.NewReader(data), &out)
		}
		if err != nil {
			t.Fatal(err)
		}
		var response workbench.Response
		if err = json.Unmarshal(out.Bytes(), &response); err != nil || response.Error != nil {
			t.Fatal(err, response.Error)
		}
		return response, data
	}
	request("example.load", workbench.ExampleLoadParams{ID: "rc"}, false)
	request("simulation.pause", struct{}{}, true)
	request("simulation.restart", struct{}{}, false)
	if _, err = s.Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
	request("scope.probes.set", workbench.ProbeParams{IDs: []string{"v:C1"}}, true)
	request("simulation.step", workbench.StepParams{Duration: .1}, false)
	before := s.Snapshot().Transient.Sample.Time
	_, admitted := request("scope.view.set", workbench.ScopeViewParams{View: "spectrum"}, true)
	deadline := time.Now().Add(3 * time.Second)
	for s.Snapshot().Spectrum.Live.Result == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	p := s.Snapshot().Spectrum.Live.Settings
	p.Hold = true
	request("spectrum.live.set", p, false)
	snapshot := s.Snapshot()
	if snapshot.Spectrum.Live.Result == nil || snapshot.Transient.Sample.Time != before {
		t.Fatal("shared live FFT changed acquisition or failed")
	}
	var retry bytes.Buffer
	if err = RemoteStdio(t.Context(), descriptor, bytes.NewReader(admitted), &retry); err != nil {
		t.Fatal(err)
	}
	if !s.Snapshot().Spectrum.Live.Settings.Hold {
		t.Fatal("old retry re-executed tab configuration")
	}
	data := []byte(`{"version":1,"id":"query","command":"project.inspect","params":{}}` + "\n")
	var a, b bytes.Buffer
	if err = Stdio(t.Context(), s, bytes.NewReader(data), &a); err != nil {
		t.Fatal(err)
	}
	if err = RemoteStdio(t.Context(), descriptor, bytes.NewReader(data), &b); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("held FFT differs across transports")
	}
	request("project.clear", map[string]bool{"discard_changes": true}, true)
	if s.Snapshot().Spectrum.Live != nil {
		t.Fatal("socket document replacement retained live state")
	}
}
