package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// Process exit and semantic shutdown are separate observations. In particular,
// app.close can return a successful envelope with failed cleanup/exit_ready=false.
type processExit struct {
	PID                  int       `json:"pid"`
	WaitStarted          time.Time `json:"wait_started_at"`
	WaitFinished         time.Time `json:"wait_finished_at"`
	WaitLimitSeconds     float64   `json:"wait_limit_s"`
	WaitDurationSeconds  float64   `json:"wait_duration_s"`
	GracefulWaitTimedOut bool      `json:"graceful_wait_timed_out"`
	ExitCode             *int      `json:"exit_code"`
	WaitError            string    `json:"wait_error,omitempty"`
	ContextError         string    `json:"context_error_before_cancel,omitempty"`
	CloseError           string    `json:"close_error,omitempty"`
}

func writeProcessExit(path string, observation processExit) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("record owned process exit: %w", err)
	}
	encodeErr := json.NewEncoder(file).Encode(observation)
	if err := errors.Join(encodeErr, file.Close()); err != nil {
		return fmt.Errorf("record owned process exit: %w", err)
	}
	return nil
}
