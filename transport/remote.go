package transport

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/dshills/elab/workbench"
)

// RemoteClient pins one explicitly selected private endpoint. A new connection
// is required after its host restarts; it never falls back to a headless circuit.
// Independent calls use independent sockets and are safe to make concurrently.
type RemoteClient struct{ endpoint Endpoint }

// NewRemoteClient reads one bounded descriptor without opening or replacing a host.
func NewRemoteClient(descriptor string) (*RemoteClient, error) {
	f, err := os.Open(descriptor)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxRequestBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxRequestBytes {
		return nil, fmt.Errorf("endpoint descriptor exceeds 32 KiB")
	}
	var endpoint Endpoint
	if err = workbench.DecodeStrict(data, &endpoint); err != nil {
		return nil, err
	}
	if endpoint.Version != 1 || len(endpoint.Token) != 64 {
		return nil, fmt.Errorf("unsupported or invalid endpoint")
	}
	return &RemoteClient{endpoint: endpoint}, nil
}

// Send forwards exactly one JSON line without retrying or changing its guards.
// A transport failure does not establish whether a mutation was admitted.
func (r *RemoteClient) Send(ctx context.Context, line []byte) ([]byte, error) {
	if len(line) > MaxRequestBytes+1 || bytes.ContainsAny(line, "\r\n") {
		return nil, fmt.Errorf("expected one bounded JSON request line")
	}
	d := net.Dialer{Timeout: 3 * time.Second}
	c, err := d.DialContext(ctx, "unix", r.endpoint.Socket)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()
	if err = c.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return nil, err
	}
	_, err = fmt.Fprintf(c, "%s\n%s\n", r.endpoint.Token, line)
	if err != nil {
		return nil, err
	}
	reply, err := bufio.NewReader(io.LimitReader(c, MaxResponseBytes+1)).ReadBytes('\n')
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	return reply, nil
}
