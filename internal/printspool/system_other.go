//go:build !darwin || !cgo

package printspool

import "context"

func (systemBackend) Destinations(context.Context) ([]Destination, error) { return nil, ErrUnsupported }
func (systemBackend) Submit(context.Context, string, string, string, Options) (Receipt, error) {
	return Receipt{}, ErrUnsupported
}
func (systemBackend) Inspect(context.Context, int) (Receipt, error) { return Receipt{}, ErrUnsupported }
func (systemBackend) Cancel(context.Context, int) error             { return ErrUnsupported }
