package main

import (
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMCPConfigIsPortableJSONWithoutReadingCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not created.endpoint.json")
	for _, args := range [][]string{{"--mcp-config"}, {"--mcp-config", "--connect", path}} {
		o, err := parseOptions(args, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		exe := filepath.Join(t.TempDir(), "agent with spaces")
		if err = writeMCPConfig(&out, exe, o); err != nil {
			t.Fatal(err)
		}
		var result struct {
			Servers map[string]struct {
				Command string   `json:"command"`
				Args    []string `json:"args"`
			} `json:"mcpServers"`
		}
		if json.Unmarshal(out.Bytes(), &result) != nil {
			t.Fatal("invalid JSON")
		}
		server := result.Servers["elab"]
		want := []string{"--mcp"}
		if len(args) > 1 {
			want = append(want, "--connect", path)
		}
		if server.Command != exe || !reflect.DeepEqual(server.Args, want) {
			t.Fatal("configuration changed paths/arguments", server)
		}
	}
}

func TestAgentModesRefuseAmbiguousOrUnusedFlags(t *testing.T) {
	for _, args := range [][]string{{"--mcp", "--connect", ""}, {"--listen", ""}, {"--mcp", "--listen", "new"}, {"--listen", "new", "--connect", "old"}, {"--mcp-config", "--listen", "new"}, {"--connect", "old", "--recovery-directory", "other"}, {"unexpected"}} {
		if _, err := parseOptions(args, io.Discard); err == nil {
			t.Error("accepted ambiguous mode", args)
		}
	}
	for _, args := range [][]string{nil, {"--mcp"}, {"--mcp", "--connect", "private"}, {"--listen", "private"}, {"--connect", "private"}} {
		if _, err := parseOptions(args, io.Discard); err != nil {
			t.Error(args, err)
		}
	}
}
