package transport

import (
	"bytes"
	"context"
	"encoding/json"
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
