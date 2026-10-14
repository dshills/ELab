package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func TestStdioAndSocketShareCommands(t *testing.T) {
	direct := workbench.NewExample()
	defer direct.Close()
	remote := workbench.NewExample()
	defer remote.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	a, _ := direct.Wait(ctx)
	b, _ := remote.Wait(ctx)
	if a.Revision != b.Revision {
		t.Fatal("different initial states")
	}
	path := filepath.Join(t.TempDir(), "endpoint.json")
	server, err := Listen(remote, path)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	input := `{"version":1,"id":"edit","command":"component.set","expected_revision":2,"key":"edit-r","params":{"id":"R1","value":"2k"}}
{"version":1,"id":"wait","command":"simulation.wait","params":{}}
`
	var local, external bytes.Buffer
	if err = Stdio(ctx, direct, strings.NewReader(input), &local); err != nil {
		t.Fatal(err)
	}
	if err = RemoteStdio(ctx, path, strings.NewReader(input), &external); err != nil {
		t.Fatal(err)
	}
	var catalog bytes.Buffer
	if err = RemoteStdio(ctx, path, strings.NewReader(`{"version":1,"id":"catalog","command":"capabilities.list","params":{}}`+"\n"), &catalog); err != nil {
		t.Fatal(err)
	}
	var discovery struct {
		Result struct{ Capabilities []workbench.Capability }
	}
	if err = json.Unmarshal(catalog.Bytes(), &discovery); err != nil || len(discovery.Result.Capabilities) != len(workbench.Catalog()) {
		t.Fatalf("catalog socket roundtrip failed: %v", err)
	}
	for _, output := range []string{local.String(), external.String()} {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		var result struct{ Result workbench.Snapshot }
		if err = json.Unmarshal([]byte(lines[1]), &result); err != nil {
			t.Fatal(err)
		}
		if result.Result.Circuit.Resistance != 2000 || result.Result.Result == nil || result.Result.Result.Stale {
			t.Fatal(output)
		}
	}
	stat, err := os.Stat(path)
	if err != nil || stat.Mode().Perm() != 0600 {
		t.Fatal("descriptor is not private")
	}
}
func TestSocketRefusesInvalidTokenAndExistingDescriptor(t *testing.T) {
	s := workbench.NewExample()
	defer s.Close()
	path := filepath.Join(t.TempDir(), "endpoint.json")
	server, err := Listen(s, path)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	if _, err = Listen(s, path); err == nil {
		t.Fatal("existing descriptor overwritten")
	}
	data, _ := os.ReadFile(path)
	var endpoint Endpoint
	json.Unmarshal(data, &endpoint)
	c, err := net.Dial("unix", endpoint.Socket)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(time.Second))
	c.Write([]byte(strings.Repeat("0", 64) + "\n{}\n"))
	buf := make([]byte, 64)
	n, _ := c.Read(buf)
	if n != 0 {
		t.Fatal("bad credential accepted")
	}
	server.Close()
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("descriptor cleanup failed")
	}
}

func TestEventObservationOverSocketDoesNotBlockControlAndStdioIdleParity(t *testing.T) {
	s := workbench.New()
	defer s.Close()
	if _, err := s.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	descriptor := filepath.Join(t.TempDir(), "endpoint.json")
	server, err := Listen(s, descriptor)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	revision := s.Snapshot().Revision
	raw, _ := json.Marshal(workbench.Request{Version: 1, ID: "observe", Command: "events.poll", Params: json.RawMessage(fmt.Sprintf(`{"after_revision":%d,"timeout_ms":1000}`, revision))})
	output := make(chan []byte, 1)
	failure := make(chan error, 1)
	go func() {
		var buffer bytes.Buffer
		err := RemoteStdio(context.Background(), descriptor, bytes.NewReader(append(raw, '\n')), &buffer)
		if err != nil {
			failure <- err
			return
		}
		output <- buffer.Bytes()
	}()
	deadline := time.Now().Add(time.Second)
	for {
		server.mu.Lock()
		connected := len(server.clients) > 0
		server.mu.Unlock()
		if connected {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("observer did not connect")
		}
		time.Sleep(time.Millisecond)
	}
	current := s.Snapshot()
	params, _ := json.Marshal(map[string]float64{"dx": 10, "dy": 0})
	revision = current.CommandRevision
	mutation := s.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "pan", Command: "view.pan", Params: params, ExpectedCommandRevision: &revision, Key: "pan"}, "maker")
	if mutation.Error != nil {
		t.Fatal(mutation.Error)
	}
	var event struct {
		Result workbench.EventPollResult
		Error  *workbench.Failure
	}
	select {
	case data := <-output:
		if err = json.Unmarshal(data, &event); err != nil {
			t.Fatal(err)
		}
	case err = <-failure:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("socket observer blocked control/wake")
	}
	if event.Error != nil || !event.Result.Changed || event.Result.Snapshot == nil || event.Result.Snapshot.LastActor != "maker" || event.Result.Snapshot.View.Center.X != current.View.Center.X+10 {
		t.Fatal("socket event loses shared attributed state")
	}
	idle := fmt.Sprintf("{\"version\":1,\"id\":\"idle\",\"command\":\"events.poll\",\"params\":{\"after_revision\":%d,\"timeout_ms\":0}}\n", event.Result.Revision)
	var local, remote bytes.Buffer
	if err = Stdio(context.Background(), s, strings.NewReader(idle), &local); err != nil {
		t.Fatal(err)
	}
	if err = RemoteStdio(context.Background(), descriptor, strings.NewReader(idle), &remote); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(local.Bytes(), remote.Bytes()) {
		t.Fatal("idle event response differs through stdio/socket")
	}
	event.Result = workbench.EventPollResult{}
	if err = json.Unmarshal(local.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	if event.Result.Changed || event.Result.Snapshot != nil {
		t.Fatal("idle transport observation invents snapshot/change")
	}
}
