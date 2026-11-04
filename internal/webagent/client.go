// Package webagent connects an MCP bridge to one explicitly enabled browser workspace.
package webagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dshills/elab/transport"
	"github.com/dshills/elab/workbench"
)

const Kind = "elab-web-workspace"

type Endpoint struct {
	Version   int    `json:"version"`
	Kind      string `json:"kind"`
	URL       string `json:"url"`
	Instance  string `json:"instance"`
	Workspace string `json:"workspace"`
	Token     string `json:"token"`
}

var identifier = regexp.MustCompile(`^[a-f0-9]{48}$`)

func (e Endpoint) Validate() error {
	u, err := url.Parse(e.URL)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.Path != "/" || u.RawPath != "" {
		return fmt.Errorf("invalid local browser workspace address")
	}
	if host := u.Hostname(); host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return fmt.Errorf("browser workspace must use loopback")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 || e.Version != 1 || e.Kind != Kind || !identifier.MatchString(e.Instance) || !identifier.MatchString(e.Workspace) || !identifier.MatchString(e.Token) {
		return fmt.Errorf("invalid browser workspace descriptor")
	}
	return nil
}

type Client struct {
	endpoint Endpoint
	http     *http.Client
}

// New pins the selected workspace/host/credential for this bridge's lifetime.
// Revocation or host restart requires an explicit reconnect; there is no fallback.
func New(path string) (*Client, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("workspace descriptor must be a private regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, fmt.Errorf("workspace descriptor changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(f, 16385))
	if err != nil || len(data) > 16384 {
		return nil, fmt.Errorf("workspace descriptor exceeds its read bound")
	}
	var e Endpoint
	if err = workbench.DecodeStrict(data, &e); err != nil {
		return nil, err
	}
	if err = e.Validate(); err != nil {
		return nil, err
	}
	return &Client{endpoint: e, http: &http.Client{
		Transport: &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 3 * time.Second}).DialContext},
		Timeout:   12 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}, nil
}

func (c *Client) Close() { c.http.CloseIdleConnections() }

func (c *Client) Send(ctx context.Context, line []byte) ([]byte, error) {
	if len(line) > transport.MaxRequestBytes {
		return nil, fmt.Errorf("expected one bounded ELab request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint.URL+"api/agent/command", bytes.NewReader(line))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ELab-Agent-Token", c.endpoint.Token)
	req.Header.Set("X-ELab-Instance", c.endpoint.Instance)
	req.Header.Set("X-ELab-Workspace", c.endpoint.Workspace)
	response, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "application/json") {
		return nil, fmt.Errorf("workspace connection refused (HTTP %d); reconnect explicitly in ELab", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, transport.MaxResponseBytes+1))
	if err != nil || len(data) > transport.MaxResponseBytes || !json.Valid(data) {
		return nil, fmt.Errorf("workspace returned an invalid or oversized response; mutation outcome may be unknown")
	}
	return data, nil
}
