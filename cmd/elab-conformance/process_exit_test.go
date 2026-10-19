package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Only subprocesses explicitly created by the tests enter this fixture.
func TestOwnedProcessExitFixture(t *testing.T) {
	mode := os.Getenv("ELAB_CONFORMANCE_EXIT_FIXTURE")
	if mode == "" {
		return
	}
	fmt.Println("owned-process-ready")
	if mode == "timeout" {
		time.Sleep(time.Hour)
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
	if mode == "exit-error" {
		os.Exit(17)
	}
	os.Exit(0)
}

func ownedExitFixture(t *testing.T, mode string) *processClient {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, executable, "-test.run=^TestOwnedProcessExitFixture$")
	command.Env = append(os.Environ(), "ELAB_CONFORMANCE_EXIT_FIXTURE="+mode)
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	log, err := os.OpenFile(filepath.Join(t.TempDir(), "process.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	command.Stderr = log
	if err = command.Start(); err != nil {
		log.Close()
		t.Fatal(err)
	}
	// Synchronize before starting the short timeout; process launch speed is not
	// the behavior under test, including under the race detector.
	if line, readErr := bufio.NewReader(stdout).ReadString('\n'); readErr != nil || line != "owned-process-ready\n" {
		cancel()
		command.Wait()
		log.Close()
		t.Fatal("fixture readiness", line, readErr)
	}
	client := &processClient{command: command, stdin: stdin, context: ctx, cancel: cancel, log: log, done: make(chan error, 1)}
	go func() { client.done <- command.Wait() }()
	return client
}

func TestOwnedProcessExitEvidence(t *testing.T) {
	for _, mode := range []string{"normal", "exit-error", "timeout", "caller-canceled", "existing-evidence"} {
		t.Run(mode, func(t *testing.T) {
			fixtureMode := mode
			if mode == "caller-canceled" {
				fixtureMode = "timeout"
			}
			client := ownedExitFixture(t, fixtureMode)
			path := filepath.Join(filepath.Dir(client.log.Name()), "process-exit.json")
			if mode == "existing-evidence" {
				if err := os.WriteFile(path, []byte("original evidence"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "caller-canceled" {
				client.cancel()
			}
			limit := 5 * time.Second
			if mode == "timeout" {
				limit = 30 * time.Millisecond
			}
			err := client.closeWithin(limit)
			if (err == nil) != (mode == "normal") || client.command.ProcessState == nil {
				t.Fatal("exit outcome", mode, err, client.command.ProcessState)
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if mode == "existing-evidence" {
				if string(data) != "original evidence" || !strings.Contains(err.Error(), "record owned process exit") {
					t.Fatal("overwrote earlier evidence or lost write failure", string(data), err)
				}
				return
			}
			var observed processExit
			if err := json.Unmarshal(data, &observed); err != nil {
				t.Fatal(err)
			}
			if observed.PID != client.command.Process.Pid || observed.ExitCode == nil || *observed.ExitCode != client.command.ProcessState.ExitCode() || observed.WaitStarted.IsZero() || observed.WaitFinished.Before(observed.WaitStarted) || observed.WaitDurationSeconds < 0 || observed.WaitLimitSeconds != limit.Seconds() {
				t.Fatal("missing process identity/timing/outcome", observed)
			}
			if observed.GracefulWaitTimedOut != (mode == "timeout") || (observed.WaitError == "") != (mode == "normal") || (observed.CloseError == "") != (mode == "normal") {
				t.Fatal("wrong failure classification", observed)
			}
			if (observed.ContextError != "") != (mode == "caller-canceled") {
				t.Fatal("caller cancellation confused with harness timeout", observed)
			}
			if mode == "exit-error" && *observed.ExitCode != 17 {
				t.Fatal("lost nonzero exit", observed)
			}
			info, err := os.Stat(path)
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatal("exit evidence permissions", info, err)
			}
		})
	}
}
