//go:build darwin && cgo

package printspool

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// Exercise libcups against an owned Unix-socket IPP fixture. No system queues
// are modified and no physical printer receives these requests.
func TestCUPSBackendAgainstIsolatedIPPSpooler(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "elab-cups-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "cups.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var operations []uint16
	var capturedPDF []byte
	var capturedOptions map[string]string
	server := &http.Server{ReadHeaderTimeout: time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
		if err != nil || len(body) < 8 {
			http.Error(w, "bad request", 400)
			return
		}
		op := binary.BigEndian.Uint16(body[2:4])
		attrs, payload := ippTestAttributes(body[8:])
		mu.Lock()
		operations = append(operations, op)
		if op == 5 || op == 2 {
			capturedOptions = attrs
		}
		if op == 6 || op == 2 {
			capturedPDF = append([]byte{}, payload...)
		}
		mu.Unlock()
		response := append([]byte{1, 1, 0, 0}, body[4:8]...)
		response = append(response, 1)
		response = ippTestString(response, 0x47, "attributes-charset", "utf-8")
		response = ippTestString(response, 0x48, "attributes-natural-language", "en")
		switch op {
		case 0x4001, 0x4002:
			response = append(response, 4)
			response = ippTestString(response, 0x42, "printer-name", "lab")
			response = ippTestString(response, 0x41, "printer-info", "Fixture printer")
			response = ippTestString(response, 0x45, "printer-uri-supported", "ipp://localhost/printers/lab")
			response = ippTestString(response, 0x22, "printer-is-accepting-jobs", string([]byte{1}))
			response = ippTestInteger(response, 0x23, "printer-state", 3)
		case 2, 5, 6, 9:
			response = append(response, 2)
			response = ippTestInteger(response, 0x21, "job-id", 77)
			response = ippTestString(response, 0x45, "job-uri", "ipp://localhost/jobs/77")
			state := 3
			if op == 9 {
				state = 9
			}
			response = ippTestInteger(response, 0x23, "job-state", uint32(state))
		case 8, 0x4005:
		case 11:
			response = append(response, 4)
			response = ippTestString(response, 0x42, "printer-name", "lab")
			response = ippTestString(response, 0x45, "printer-uri-supported", "ipp://localhost/printers/lab")
			response = ippTestString(response, 0x22, "printer-is-accepting-jobs", string([]byte{1}))
			response = ippTestInteger(response, 0x23, "printer-state", 3)
			response = ippTestInteger(response, 0x23, "printer-type", 0)
			response = ippTestString(response, 0x49, "document-format-supported", "application/pdf")
		default:
			t.Errorf("unexpected IPP operation %#x", op)
		}
		response = append(response, 3)
		w.Header().Set("Content-Type", "application/ipp")
		_, _ = w.Write(response)
	})}
	go func() { _ = server.Serve(listener) }()
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	backend := systemBackend{socket: socket}
	dests, err := backend.Destinations(ctx)
	found := false
	for _, dest := range dests {
		found = found || dest.ID == "lab" && dest.Accepting && dest.Name == "Fixture printer"
	}
	// libcups can merge remembered user destinations into a server response.
	// Only the explicit fixture destination is ever used for submission here.
	if err != nil || !found {
		t.Fatal("discovery", dests, err)
	}
	pdf := []byte("%PDF-1.4\nfixture\n%%EOF\n")
	path := filepath.Join(dir, "page.pdf")
	if err := os.WriteFile(path, pdf, 0600); err != nil {
		t.Fatal(err)
	}
	receipt, err := backend.Submit(ctx, "lab", path, "ELab fixture", Options{Paper: "a4", Orientation: "landscape", Copies: 2})
	if err != nil || receipt.ID != 77 {
		mu.Lock()
		t.Log("fixture IPP operations", operations)
		mu.Unlock()
		t.Fatal("submission", receipt, err)
	}
	view, err := backend.Inspect(ctx, receipt.ID)
	if err != nil || view.State != "completed" {
		t.Fatal("inspection", view, err)
	}
	if err := backend.Cancel(ctx, receipt.ID); err != nil {
		t.Fatal("cancel", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if !bytes.Equal(capturedPDF, pdf) {
		t.Fatalf("libcups PDF body differs: %q", capturedPDF)
	}
	for key, want := range map[string]string{"media": "a4", "copies": string([]byte{0, 0, 0, 2}), "orientation-requested": string([]byte{0, 0, 0, 4}), "print-scaling": "none", "sides": "one-sided"} {
		if capturedOptions[key] != want {
			t.Errorf("job option %s: %q != %q (operations %v)", key, capturedOptions[key], want, operations)
		}
	}
}

func ippTestString(b []byte, tag byte, name, value string) []byte {
	b = append(b, tag, byte(len(name)>>8), byte(len(name)))
	b = append(b, name...)
	b = append(b, byte(len(value)>>8), byte(len(value)))
	return append(b, value...)
}
func ippTestInteger(b []byte, tag byte, name string, value uint32) []byte {
	return ippTestString(b, tag, name, string([]byte{byte(value >> 24), byte(value >> 16), byte(value >> 8), byte(value)}))
}
func ippTestAttributes(b []byte) (map[string]string, []byte) {
	attrs := map[string]string{}
	for len(b) > 0 {
		tag := b[0]
		b = b[1:]
		if tag == 3 {
			return attrs, b
		}
		if tag < 0x10 {
			continue
		}
		if len(b) < 2 {
			return attrs, nil
		}
		n := int(binary.BigEndian.Uint16(b))
		b = b[2:]
		if len(b) < n+2 {
			return attrs, nil
		}
		name := string(b[:n])
		b = b[n:]
		v := int(binary.BigEndian.Uint16(b))
		b = b[2:]
		if len(b) < v {
			return attrs, nil
		}
		attrs[name] = string(b[:v])
		b = b[v:]
	}
	return attrs, nil
}
