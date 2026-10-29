package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dshills/elab/internal/agentmcp"
	"github.com/dshills/elab/transport"
	"github.com/dshills/elab/workbench"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Return through deferred cleanup before main chooses the process exit status.
func run() error {
	opts, parseErr := parseOptions(os.Args[1:], os.Stderr)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}
	if parseErr != nil {
		return parseErr
	}
	if opts.config {
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		return writeMCPConfig(os.Stdout, executable, opts)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() { <-ctx.Done(); os.Stdin.Close() }()
	var err error
	if opts.connect != "" {
		if opts.mcp {
			client, e := transport.NewRemoteClient(opts.connect)
			if e != nil {
				return e
			}
			err = agentmcp.NewServer(client.Send).Run(ctx, &mcp.StdioTransport{MaxLineLength: agentmcp.MaxFrameBytes})
		} else {
			err = transport.RemoteStdio(ctx, opts.connect, os.Stdin, os.Stdout)
		}
	} else {
		s := workbench.New()
		defer s.Close()
		if opts.appearance != "" {
			raw, _ := json.Marshal(workbench.AppearanceConfigureParams{Path: opts.appearance})
			revision := s.Snapshot().CommandRevision
			response := s.Dispatch(ctx, workbench.Request{Version: 1, ID: "startup-appearance", Command: "appearance.configure", Params: raw, ExpectedCommandRevision: &revision, Key: "startup-appearance"}, "agent")
			if response.Error != nil {
				return fmt.Errorf("configure appearance: %s", response.Error.Message)
			}
		}
		if opts.recovery != "" {
			raw, _ := json.Marshal(workbench.RecoveryConfigureParams{Directory: opts.recovery, Enabled: true, Interval: 2})
			revision := s.Snapshot().Revision
			response := s.Dispatch(ctx, workbench.Request{Version: 1, ID: "startup-recovery", Command: "recovery.configure", Params: raw, ExpectedRevision: &revision, Key: "startup-recovery"}, "agent")
			if response.Error != nil {
				return fmt.Errorf("configure recovery: %s", response.Error.Message)
			}
		}
		if opts.listen != "" {
			server, e := transport.Listen(s, opts.listen)
			if e != nil {
				err = e
			} else {
				defer server.Close()
				fmt.Fprintln(os.Stderr, "ELab headless endpoint:", opts.listen)
				ticker := time.NewTicker(50 * time.Millisecond)
				defer ticker.Stop()
				for !s.ReadyToExit() && ctx.Err() == nil {
					select {
					case <-ctx.Done():
					case <-ticker.C:
					}
				}
			}
		} else if opts.mcp {
			err = agentmcp.NewServer(agentmcp.Local(s)).Run(ctx, &mcp.StdioTransport{MaxLineLength: agentmcp.MaxFrameBytes})
		} else {
			err = transport.Stdio(ctx, s, os.Stdin, os.Stdout)
		}
	}
	if err != nil && ctx.Err() == nil {
		return err
	}
	return nil
}
