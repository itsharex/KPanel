package agent

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/dockerx"
	"github.com/kejilion/kejilion-panel/internal/sites"
	"github.com/kejilion/kejilion-panel/internal/systeminfo"
)

func TestBearerRequired(t *testing.T) {
	server := testServer(t)
	request := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("without token status = %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("with token status = %d body=%s", response.Code, response.Body.String())
	}
}

func TestSitesPageEndpoint(t *testing.T) {
	server := testServer(t)
	request := httptest.NewRequest(http.MethodGet, "/v1/sites", nil)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"items"`) {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
}

func testServer(t *testing.T) *Server {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"conf.d", "html", "certs"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	socket := fakeDockerSocket(t)
	docker := dockerx.New(socket, root, t.TempDir())
	server, err := NewServer(Config{
		Token: []byte(strings.Repeat("x", 32)), Version: "test", ProtocolVersion: "test",
		WebRoot: root, System: systeminfo.NewCollector(),
		Sites: sites.NewDiscoverer(root), Docker: docker,
	})
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func fakeDockerSocket(t *testing.T) string {
	t.Helper()
	if os.PathSeparator == '\\' {
		// Windows' standard library may not expose Unix sockets. Health is allowed
		// to report degraded, which is sufficient for HTTP middleware tests.
		return filepath.Join(t.TempDir(), "missing.sock")
	}
	path := filepath.Join(t.TempDir(), "docker.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_ping" {
				_, _ = w.Write([]byte("OK"))
				return
			}
			http.NotFound(w, r)
		})}
		_ = server.Serve(listener)
	}()
	return path
}

var _ = context.Background
