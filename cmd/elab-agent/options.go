package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"
)

type options struct {
	connect, listen, appearance, recovery string
	mcp, config                           bool
}

func parseOptions(args []string, output io.Writer) (options, error) {
	var o options
	flags := flag.NewFlagSet("elab-agent", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&o.connect, "connect", "", "use only this existing private endpoint descriptor")
	flags.StringVar(&o.listen, "listen", "", "run a headless socket host with a new private descriptor")
	flags.StringVar(&o.appearance, "appearance-file", "", "opt into persistent theme preferences using this file")
	flags.StringVar(&o.recovery, "recovery-directory", "", "opt into autosave using this private directory")
	flags.BoolVar(&o.mcp, "mcp", false, "serve standard local MCP over stdin/stdout (headless, or with --connect)")
	flags.BoolVar(&o.config, "mcp-config", false, "print client JSON for this executable; add --connect for a desktop host")
	if err := flags.Parse(args); err != nil {
		return o, err
	}
	var emptyPath error
	flags.Visit(func(f *flag.Flag) {
		if (f.Name == "connect" || f.Name == "listen") && f.Value.String() == "" {
			emptyPath = fmt.Errorf("--%s requires a nonempty descriptor path", f.Name)
		}
	})
	if emptyPath != nil {
		return o, emptyPath
	}
	if flags.NArg() != 0 {
		return o, fmt.Errorf("unexpected positional arguments")
	}
	if o.listen != "" && (o.connect != "" || o.mcp || o.config) {
		return o, fmt.Errorf("--listen cannot be combined with --connect, --mcp or --mcp-config")
	}
	if o.connect != "" && (o.appearance != "" || o.recovery != "") {
		return o, fmt.Errorf("persistent settings belong to the host; do not combine them with --connect")
	}
	return o, nil
}

// Configuration is printed, never installed into a client or credential store.
func writeMCPConfig(out io.Writer, executable string, o options) error {
	command, err := filepath.Abs(executable)
	if err != nil {
		return err
	}
	args := []string{"--mcp"}
	for _, p := range []struct{ flag, value string }{{"--connect", o.connect}, {"--appearance-file", o.appearance}, {"--recovery-directory", o.recovery}} {
		if p.value != "" {
			absolute, err := filepath.Abs(p.value)
			if err != nil {
				return err
			}
			args = append(args, p.flag, absolute)
		}
	}
	return json.NewEncoder(out).Encode(map[string]any{"mcpServers": map[string]any{"elab": map[string]any{"command": command, "args": args}}})
}
