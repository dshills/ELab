package conformance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dshills/elab/transport"
	"github.com/dshills/elab/workbench"
)

// A full-size circuit must keep its production wall deadline. Build its owned
// solver without instrumentation; the controller and report checks still run
// under race, alongside the core's controlled-clock numerical race tests.
func ordinaryAgentFactory(t *testing.T) Factory {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "elab-agent")
	build := exec.CommandContext(t.Context(), "go", "build", "-race=false", "-o", binary, "../../cmd/elab-agent")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build ordinary agent: %v\n%s", err, output)
	}
	return func(ctx context.Context, dir string) (Client, error) {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
		log, err := os.OpenFile(filepath.Join(dir, "process.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		child, cancel := context.WithCancel(ctx)
		endpoint := filepath.Join(dir, "endpoint.json")
		command := exec.CommandContext(child, binary, "--listen", endpoint)
		command.Stdout, command.Stderr = log, log
		if err := command.Start(); err != nil {
			cancel()
			return nil, errors.Join(err, log.Close())
		}
		done := make(chan error, 1)
		go func() { done <- command.Wait() }()
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for {
			if _, err := os.Stat(endpoint); err == nil {
				return &ordinaryAgentClient{endpoint, cancel, done, log}, nil
			} else if !os.IsNotExist(err) {
				cancel()
				return nil, errors.Join(err, <-done, log.Close())
			}
			select {
			case err := <-done:
				cancel()
				return nil, errors.Join(fmt.Errorf("ordinary agent stopped before endpoint: %v", err), log.Close())
			case <-ctx.Done():
				cancel()
				return nil, errors.Join(ctx.Err(), <-done, log.Close())
			case <-ticker.C:
			}
		}
	}
}

type ordinaryAgentClient struct {
	endpoint string
	cancel   context.CancelFunc
	done     chan error
	log      *os.File
}

func (c *ordinaryAgentClient) Dispatch(ctx context.Context, request workbench.Request) (response workbench.Response, err error) {
	raw, err := json.Marshal(request)
	if err != nil {
		return response, err
	}
	var result bytes.Buffer
	if err := transport.RemoteStdio(ctx, c.endpoint, bytes.NewReader(append(raw, '\n')), &result); err != nil {
		return response, err
	}
	err = json.Unmarshal(result.Bytes(), &response)
	return response, err
}
func (c *ordinaryAgentClient) Close() error {
	var err error
	select {
	case err = <-c.done:
	case <-time.After(5 * time.Second):
		c.cancel()
		err = errors.Join(fmt.Errorf("ordinary agent did not close gracefully"), <-c.done)
	}
	c.cancel()
	return errors.Join(err, c.log.Close())
}
