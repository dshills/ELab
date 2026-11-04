package webagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func descriptor(t *testing.T, address string) string {
	t.Helper()
	e := Endpoint{Version: 1, Kind: Kind, URL: address, Instance: strings.Repeat("a", 48), Workspace: strings.Repeat("b", 48), Token: strings.Repeat("c", 48)}
	data, _ := json.Marshal(e)
	path := filepath.Join(t.TempDir(), "agent.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDescriptorRejectsRemoteAddressesAndNonprivateFiles(t *testing.T) {
	for _, address := range []string{"https://127.0.0.1:8000/", "http://example.org:8000/", "http://user@localhost:8000/", "http://localhost/", "http://localhost:8000/api/", "http://localhost:8000/?token=x"} {
		if client, err := New(descriptor(t, address)); err == nil {
			client.Close()
			t.Fatal("unsafe descriptor accepted", address)
		}
	}
	path := descriptor(t, "http://127.0.0.1:8000/")
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := New(path); err == nil {
		t.Fatal("public credential file accepted")
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := New(link); err == nil {
		t.Fatal("symlink credential accepted")
	}
}

func TestCredentialsNeverFollowRedirectsOrEnvironmentProxies(t *testing.T) {
	var leaked atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked.Add(1) }))
	defer other.Close()
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-ELab-Agent-Token") != strings.Repeat("c", 48) {
			t.Error("credential missing")
		}
		http.Redirect(w, r, other.URL, http.StatusFound)
	}))
	defer host.Close()
	t.Setenv("HTTP_PROXY", other.URL)
	t.Setenv("http_proxy", other.URL)
	client, err := New(descriptor(t, host.URL+"/"))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err = client.Send(context.Background(), []byte(`{"version":1}`)); err == nil {
		t.Fatal("redirect accepted")
	}
	if leaked.Load() != 0 {
		t.Fatal("credential request reached a proxy or redirect target")
	}
}

func TestNonJSONAndOversizedResponsesAreNotSuccessfulReplies(t *testing.T) {
	for _, body := range []string{"not JSON", `{"version":1} {"version":1}`, `{"pad":"` + strings.Repeat("a", 8*1024*1024) + `"}`} {
		host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}))
		client, err := New(descriptor(t, host.URL+"/"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = client.Send(t.Context(), []byte(`{}`)); err == nil {
			t.Fatal("invalid response accepted")
		}
		client.Close()
		host.Close()
	}
}
