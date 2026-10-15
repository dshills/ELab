// Package printspool connects only to the local system spooler. It never selects
// arbitrary network servers or cancels jobs outside the IDs supplied by ELab.
package printspool

import (
	"context"
	"errors"
)

var ErrUnsupported = errors.New("printing requires the macOS cgo build")
var ErrUncertain = errors.New("submission acknowledgment missing; check the system print queue before retrying")
var ErrNotFound = errors.New("print job no longer retained by the spooler")

type Destination struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Default   bool   `json:"default"`
	Accepting bool   `json:"accepting_jobs"`
	State     string `json:"state"`
}

type Options struct {
	Paper, Orientation string
	Copies             int
}

type Receipt struct {
	ID    int
	State string
}

type Backend interface {
	Destinations(context.Context) ([]Destination, error)
	Submit(context.Context, string, string, string, Options) (Receipt, error)
	Inspect(context.Context, int) (Receipt, error)
	Cancel(context.Context, int) error
}

type systemBackend struct{ socket string }

func System() Backend { return systemBackend{} }

func Terminal(state string) bool {
	return state == "completed" || state == "canceled" || state == "aborted" || state == "failed"
}
