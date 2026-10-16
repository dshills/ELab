// Package transport adapts the shared workbench contract to bounded JSON lines.
package transport

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dshills/elab/workbench"
)

const MaxRequestBytes = 32 * 1024
const MaxResponseBytes = 1024 * 1024

type Endpoint struct {
	Version int    `json:"version"`
	Socket  string `json:"socket"`
	Token   string `json:"token"`
}

func response(ctx context.Context, s *workbench.Service, line []byte, actor string) workbench.Response {
	var request workbench.Request
	if len(line) > MaxRequestBytes {
		return workbench.Response{Version: 1, Error: &workbench.Failure{Code: "request-too-large", Message: "request exceeds 32 KiB"}}
	}
	if err := workbench.DecodeStrict(line, &request); err != nil {
		return workbench.Response{Version: 1, Error: &workbench.Failure{Code: "invalid-json", Message: err.Error()}}
	}
	return s.Dispatch(ctx, request, actor)
}

// Stdio serves one session until EOF or app.close. stdout contains only JSON.
func Stdio(ctx context.Context, s *workbench.Service, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 4096), MaxRequestBytes+1)
	encoder := json.NewEncoder(out)
	for scanner.Scan() {
		if err := encoder.Encode(response(ctx, s, scanner.Bytes(), "agent.stdio")); err != nil {
			return err
		}
		if s.Snapshot().ReadyToExit() {
			return nil
		}
	}
	return scanner.Err()
}

type Server struct {
	listener              net.Listener
	cancel                context.CancelFunc
	service               *workbench.Service
	replies               sync.RWMutex
	directory, descriptor string
	descriptorInfo        os.FileInfo
	once                  sync.Once
	wg                    sync.WaitGroup
	mu                    sync.Mutex
	clients               map[net.Conn]bool
}

// Listen writes a new private descriptor, never replacing an existing file.
// The socket lives in an owned short /tmp directory for macOS sockaddr limits.
func Listen(s *workbench.Service, descriptor string) (*Server, error) {
	dir, err := os.MkdirTemp("/tmp", "elab-api-")
	if err != nil {
		return nil, err
	}
	if err = os.Chmod(dir, 0700); err != nil {
		os.Remove(dir)
		return nil, err
	}
	path := filepath.Join(dir, "api.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		os.Remove(dir)
		return nil, err
	}
	cleanup := func() { listener.Close(); os.Remove(path); os.Remove(dir) }
	if err = os.Chmod(path, 0600); err != nil {
		cleanup()
		return nil, err
	}
	var secret [32]byte
	if _, err = rand.Read(secret[:]); err != nil {
		cleanup()
		return nil, err
	}
	endpoint := Endpoint{1, path, hex.EncodeToString(secret[:])}
	file, err := os.OpenFile(descriptor, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		cleanup()
		return nil, err
	}
	info, statErr := file.Stat()
	err = json.NewEncoder(file).Encode(endpoint)
	err = errors.Join(err, statErr)
	err = errors.Join(err, file.Close())
	if err != nil {
		os.Remove(descriptor)
		cleanup()
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	server := &Server{listener: listener, cancel: cancel, service: s, directory: dir, descriptor: descriptor, descriptorInfo: info, clients: map[net.Conn]bool{}}
	slots := make(chan struct{}, 8)
	server.wg.Add(1)
	go func() {
		defer server.wg.Done()
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			select {
			case slots <- struct{}{}:
			default:
				connection.Close()
				continue
			}
			server.mu.Lock()
			if ctx.Err() != nil {
				server.mu.Unlock()
				connection.Close()
				<-slots
				return
			}
			server.clients[connection] = true
			server.mu.Unlock()
			server.wg.Add(1)
			go func(c net.Conn) {
				defer server.wg.Done()
				defer func() { c.Close(); server.mu.Lock(); delete(server.clients, c); server.mu.Unlock(); <-slots }()
				server.serveConnection(ctx, s, endpoint.Token, c)
			}(connection)
		}
	}()
	return server, nil
}

func (server *Server) serveConnection(ctx context.Context, s *workbench.Service, token string, c net.Conn) {
	c.SetDeadline(time.Now().Add(10 * time.Second))
	reader := bufio.NewReader(io.LimitReader(c, MaxRequestBytes+100))
	credential, err := reader.ReadString('\n')
	if err != nil || subtle.ConstantTimeCompare([]byte(credential), []byte(token+"\n")) != 1 {
		return
	}
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return
	}
	// Domain shutdown can become exit-ready while its admitting request is
	// still serializing/writing the final response. Retain that response
	// through graceful endpoint retirement, within the existing deadlines.
	server.replies.RLock()
	defer server.replies.RUnlock()
	if ctx.Err() != nil {
		return
	}
	bounded, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	result := response(bounded, s, line, "agent.socket")
	encoded, err := json.Marshal(result)
	if err == nil && len(encoded) <= MaxResponseBytes {
		c.Write(append(encoded, '\n'))
	} else {
		json.NewEncoder(c).Encode(workbench.Response{Version: 1, ID: result.ID, Revision: result.Revision, Error: &workbench.Failure{Code: "response-too-large", Message: "response exceeds 1 MiB"}})
	}
}

func (s *Server) Close() {
	s.once.Do(func() {
		s.listener.Close()
		graceful := s.service.Snapshot().ReadyToExit()
		if graceful {
			s.replies.Lock()
		}
		s.cancel()
		s.mu.Lock()
		for c := range s.clients {
			c.Close()
		}
		s.mu.Unlock()
		if graceful {
			s.replies.Unlock()
		}
		s.wg.Wait()
		os.Remove(filepath.Join(s.directory, "api.sock"))
		os.Remove(s.directory)
		if info, err := os.Stat(s.descriptor); err == nil && os.SameFile(info, s.descriptorInfo) {
			os.Remove(s.descriptor)
		}
	})
}

// RemoteStdio forwards lines to a running UI or headless server. Inspect again
// after transport failure, or retry with the original mutation key.
func RemoteStdio(ctx context.Context, descriptor string, in io.Reader, out io.Writer) error {
	data, err := os.ReadFile(descriptor)
	if err != nil {
		return err
	}
	var endpoint Endpoint
	if err = workbench.DecodeStrict(data, &endpoint); err != nil {
		return err
	}
	if endpoint.Version != 1 || len(endpoint.Token) != 64 {
		return fmt.Errorf("unsupported or invalid endpoint")
	}
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 4096), MaxRequestBytes+1)
	for scanner.Scan() {
		d := net.Dialer{Timeout: 3 * time.Second}
		c, err := d.DialContext(ctx, "unix", endpoint.Socket)
		if err != nil {
			return err
		}
		c.SetDeadline(time.Now().Add(10 * time.Second))
		_, err = fmt.Fprintf(c, "%s\n%s\n", endpoint.Token, scanner.Bytes())
		if err == nil {
			var line []byte
			line, err = bufio.NewReader(io.LimitReader(c, MaxResponseBytes+1)).ReadBytes('\n')
			if err == nil {
				_, err = out.Write(line)
			}
		}
		c.Close()
		if err != nil {
			return err
		}
	}
	return scanner.Err()
}
