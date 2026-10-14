package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dshills/elab/transport"
	"github.com/dshills/elab/workbench"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Return through deferred cleanup before main chooses the process exit status.
func run() error {
	connect := flag.String("connect", "", "forward JSON commands to an existing endpoint descriptor")
	listen := flag.String("listen", "", "run a headless socket session; write a private endpoint descriptor here")
	appearanceFile := flag.String("appearance-file", "", "opt into persistent theme preferences using this file")
	recoveryDirectory := flag.String("recovery-directory", "", "opt into autosave using this private directory")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() { <-ctx.Done(); os.Stdin.Close() }()
	var err error
	if *connect != "" {
		err = transport.RemoteStdio(ctx, *connect, os.Stdin, os.Stdout)
	} else {
		s := workbench.New()
		defer s.Close()
		if *appearanceFile != "" {
			raw, _ := json.Marshal(workbench.AppearanceConfigureParams{Path: *appearanceFile})
			revision := s.Snapshot().CommandRevision
			response := s.Dispatch(ctx, workbench.Request{Version: 1, ID: "startup-appearance", Command: "appearance.configure", Params: raw, ExpectedCommandRevision: &revision, Key: "startup-appearance"}, "agent")
			if response.Error != nil {
				return fmt.Errorf("configure appearance: %s", response.Error.Message)
			}
		}
		if *recoveryDirectory != "" {
			raw, _ := json.Marshal(workbench.RecoveryConfigureParams{Directory: *recoveryDirectory, Enabled: true, Interval: 2})
			revision := s.Snapshot().Revision
			response := s.Dispatch(ctx, workbench.Request{Version: 1, ID: "startup-recovery", Command: "recovery.configure", Params: raw, ExpectedRevision: &revision, Key: "startup-recovery"}, "agent")
			if response.Error != nil {
				return fmt.Errorf("configure recovery: %s", response.Error.Message)
			}
		}
		if *listen != "" {
			server, e := transport.Listen(s, *listen)
			if e != nil {
				err = e
			} else {
				defer server.Close()
				fmt.Fprintln(os.Stderr, "ELab headless endpoint:", *listen)
				ticker := time.NewTicker(50 * time.Millisecond)
				defer ticker.Stop()
				for !s.Snapshot().Closed && ctx.Err() == nil {
					select {
					case <-ctx.Done():
					case <-ticker.C:
					}
				}
			}
		} else {
			err = transport.Stdio(ctx, s, os.Stdin, os.Stdout)
		}
	}
	if err != nil && ctx.Err() == nil {
		return err
	}
	return nil
}
