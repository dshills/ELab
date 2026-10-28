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

func TestSpectrumMeasurementsStdioSocketAndRetry(t *testing.T) {
	s := workbench.New()
	defer s.Close()
	descriptor := filepath.Join(t.TempDir(), "endpoint.json")
	server, err := Listen(s, descriptor)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	serial := 0
	send := func(command string, p any, remote bool) []byte {
		t.Helper()
		serial++
		raw, _ := json.Marshal(p)
		rev := s.Snapshot().CommandRevision
		req := workbench.Request{Version: 1, ID: fmt.Sprint(serial), Key: fmt.Sprint(serial), Command: command, Params: raw, ExpectedCommandRevision: &rev}
		data, _ := json.Marshal(req)
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
			t.Fatal(command, err, response.Error)
		}
		return data
	}
	send("example.load", workbench.ExampleLoadParams{ID: "rc"}, false)
	send("simulation.pause", struct{}{}, true)
	send("simulation.restart", struct{}{}, false)
	if _, err = s.Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
	send("scope.probes.set", workbench.ProbeParams{IDs: []string{"v:C1"}}, true)
	send("simulation.step", workbench.StepParams{Duration: .1}, false)
	send("scope.view.set", workbench.ScopeViewParams{View: "spectrum"}, true)
	deadline := time.Now().Add(3 * time.Second)
	for s.Snapshot().Spectrum.Live.Result == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if s.Snapshot().Spectrum.Live.Result == nil {
		t.Fatal("no live frame")
	}
	cfg := s.Snapshot().Spectrum.Live.Settings
	cfg.Hold = true
	send("spectrum.live.set", cfg, true)
	before := s.Snapshot().Transient.Sample.Time
	admitted := send("spectrum.cursor.set", map[string]any{"source": "live", "cursor": "a", "frequency_hz": "100 Hz"}, true)
	send("spectrum.cursor.set", workbench.SpectrumCursorParams{Source: "live", Cursor: "b", Peak: 1}, false)
	send("spectrum.markers.set", workbench.ScopeVisibilityParams{Visible: false}, true)
	measurement := s.Snapshot().Spectrum.Live.Measurements
	if measurement.A == nil || measurement.B == nil || measurement.DeltaHz == nil || measurement.Visible || before != s.Snapshot().Transient.Sample.Time {
		t.Fatal("shared measurement commands")
	}
	send("spectrum.cursor.set", workbench.SpectrumCursorParams{Source: "live", Cursor: "a", Clear: true}, false)
	var retry bytes.Buffer
	if err = RemoteStdio(t.Context(), descriptor, bytes.NewReader(admitted), &retry); err != nil {
		t.Fatal(err)
	}
	if s.Snapshot().Spectrum.Live.Measurements.A != nil {
		t.Fatal("retry reapplied retired placement")
	}
	query := []byte(`{"version":1,"id":"inspect","command":"project.inspect","params":{}}` + "\n")
	var a, b bytes.Buffer
	if err = Stdio(t.Context(), s, bytes.NewReader(query), &a); err != nil {
		t.Fatal(err)
	}
	if err = RemoteStdio(t.Context(), descriptor, bytes.NewReader(query), &b); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("measurement snapshots differ between transports")
	}
	send("project.clear", map[string]bool{"discard_changes": true}, true)
	if !s.Snapshot().Spectrum.Measurements.Visible || s.Snapshot().Spectrum.Live != nil {
		t.Fatal("document retained measurements")
	}
}
