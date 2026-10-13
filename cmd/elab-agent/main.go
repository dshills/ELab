package main

import (
	"context"
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
	connect := flag.String("connect", "", "forward JSON commands to an existing endpoint descriptor")
	listen := flag.String("listen", "", "run a headless socket session; write a private endpoint descriptor here")
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
