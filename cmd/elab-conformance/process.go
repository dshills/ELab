package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/dshills/elab/internal/conformance"
	"github.com/dshills/elab/transport"
	"github.com/dshills/elab/workbench"
)

type processClient struct {
	command  *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	encoder  *json.Encoder
	decoder  *json.Decoder
	endpoint string
	done     chan error
	log      *os.File
	cancel   context.CancelFunc
	context  context.Context
}

func processFactory(binary, mode string) conformance.Factory {
	return func(ctx context.Context, dir string) (_ conformance.Client, err error) {
		if err = os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
		processCtx, cancel := context.WithCancel(ctx)
		client := &processClient{cancel: cancel, context: processCtx, done: make(chan error, 1)}
		client.log, err = os.OpenFile(filepath.Join(dir, "process.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			cancel()
			return nil, err
		}
		args := []string{}
		if mode != "stdio" {
			client.endpoint = filepath.Join(dir, "endpoint.json")
			flag := "--listen"
			if mode == "native" {
				flag = "--agent-endpoint"
			}
			args = []string{flag, client.endpoint}
			if mode == "native" {
				args = append(args, "--no-appearance", "--recovery-directory", filepath.Join(dir, "recovery"))
			}
		}
		client.command = exec.CommandContext(processCtx, binary, args...)
		client.command.Stderr = client.log
		var stdoutWriter *os.File
		if mode == "stdio" {
			client.stdin, err = client.command.StdinPipe()
			if err == nil {
				var stdout *os.File
				stdout, stdoutWriter, err = os.Pipe()
				if err == nil {
					// Own the reader so concurrent Wait cannot close it before
					// the final app.close response is decoded.
					client.stdout = stdout
					client.command.Stdout = stdoutWriter
					client.encoder = json.NewEncoder(client.stdin)
					client.decoder = json.NewDecoder(stdout)
				}
			}
		} else {
			client.command.Stdout = client.log
		}
		if err != nil {
			cancel()
			if client.stdin != nil {
				client.stdin.Close()
			}
			client.log.Close()
			return nil, err
		}
		if err = client.command.Start(); err != nil {
			cancel()
			if client.stdin != nil {
				client.stdin.Close()
			}
			if client.stdout != nil {
				client.stdout.Close()
			}
			if stdoutWriter != nil {
				stdoutWriter.Close()
			}
			client.log.Close()
			return nil, err
		}
		if stdoutWriter != nil {
			stdoutWriter.Close()
		}
		go func() { client.done <- client.command.Wait() }()
		if mode != "stdio" {
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				if _, err = os.Stat(client.endpoint); err == nil {
					return client, nil
				} else if !os.IsNotExist(err) {
					break
				}
				select {
				case <-ctx.Done():
					err = ctx.Err()
				case stopped := <-client.done:
					client.done <- stopped
					err = fmt.Errorf("owned process ended before endpoint: %v", stopped)
				case <-ticker.C:
					continue
				}
				break
			}
			return nil, errors.Join(err, client.Close())
		}
		return client, nil
	}
}
func (c *processClient) Dispatch(ctx context.Context, r workbench.Request) (response workbench.Response, err error) {
	if err = ctx.Err(); err != nil {
		return response, err
	}
	if c.endpoint != "" {
		data, e := json.Marshal(r)
		if e != nil {
			return response, e
		}
		var output bytes.Buffer
		if err = transport.RemoteStdio(ctx, c.endpoint, bytes.NewReader(append(data, '\n')), &output); err != nil {
			return response, err
		}
		err = json.Unmarshal(output.Bytes(), &response)
	} else {
		if err = c.encoder.Encode(r); err == nil {
			err = c.decoder.Decode(&response)
		}
	}
	return response, err
}
func (c *processClient) Close() error {
	return c.closeWithin(5 * time.Second)
}

func (c *processClient) closeWithin(limit time.Duration) error {
	started := time.Now()
	observation := processExit{PID: c.command.Process.Pid, WaitStarted: started.UTC(), WaitLimitSeconds: limit.Seconds()}
	if c.stdin != nil {
		c.stdin.Close()
	}
	timer := time.NewTimer(limit)
	defer timer.Stop()
	var err error
	var contextErr error
	select {
	case err = <-c.done:
		contextErr = c.context.Err()
	case <-timer.C:
		observation.GracefulWaitTimedOut = true
		contextErr = c.context.Err()
		c.cancel()
		err = <-c.done
	}
	observation.WaitFinished = time.Now().UTC()
	observation.WaitDurationSeconds = time.Since(started).Seconds()
	if err != nil {
		observation.WaitError = err.Error()
	}
	if c.command.ProcessState != nil {
		code := c.command.ProcessState.ExitCode()
		observation.ExitCode = &code
	}
	// Read the process context before our final cleanup cancellation. A caller
	// deadline and the harness's own timeout are different exit circumstances.
	if contextErr != nil {
		observation.ContextError = contextErr.Error()
	}
	if observation.GracefulWaitTimedOut {
		err = errors.Join(fmt.Errorf("owned session PID %d did not close gracefully within %s", observation.PID, limit), err)
	}
	c.cancel()
	if c.stdout != nil {
		err = errors.Join(err, c.stdout.Close())
	}
	err = errors.Join(err, c.log.Close())
	if err != nil {
		observation.CloseError = err.Error()
	}
	return errors.Join(err, writeProcessExit(filepath.Join(filepath.Dir(c.log.Name()), "process-exit.json"), observation))
}
