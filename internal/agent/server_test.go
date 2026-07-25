package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
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

func TestSiteWriteErrorStatusMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"invalid request", sites.ErrInvalidInput, http.StatusBadRequest, "invalid_site_request"},
		{"read only", sites.ErrForbidden, http.StatusForbidden, "site_read_only"},
		{"conflict", sites.ErrConflict, http.StatusConflict, "resource_conflict"},
		{"validation", sites.ErrUnprocessable, http.StatusUnprocessableEntity, "site_validation_failed"},
		{"needs attention", sites.ErrNeedsAttention, http.StatusServiceUnavailable, "site_needs_attention"},
		{"unavailable", sites.ErrUnavailable, http.StatusServiceUnavailable, "sites_unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			(&Server{}).writeSiteError(
				response,
				"request-id",
				fmt.Errorf("wrapped: %w", test.err),
			)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			var problem contract.Problem
			if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
				t.Fatal(err)
			}
			if problem.Code != test.wantCode || problem.Status != test.wantStatus {
				t.Fatalf("problem = %#v, want code=%q status=%d", problem, test.wantCode, test.wantStatus)
			}
			if problem.Retryable != (test.wantStatus >= 500) {
				t.Fatalf("retryable = %v for status %d", problem.Retryable, test.wantStatus)
			}
		})
	}
}

func TestSiteWriteRoutesRejectMalformedRequests(t *testing.T) {
	server := testServer(t)
	token := "Bearer " + strings.Repeat("x", 32)
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{
			name: "unknown create field", method: http.MethodPost, path: "/v1/sites",
			body:       `{"primaryDomain":"example.com","type":"static","unknown":true}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "create query", method: http.MethodPost, path: "/v1/sites?mode=unsafe",
			body:       `{"primaryDomain":"example.com","type":"static"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "encoded update path", method: http.MethodPatch,
			path: "/v1/sites/" + strings.Repeat("%61", 32), body: `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "update query", method: http.MethodPatch,
			path: "/v1/sites/" + strings.Repeat("a", 32) + "?force=true", body: `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid update id", method: http.MethodPatch,
			path: "/v1/sites/" + strings.Repeat("A", 32), body: `{}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name: "unsupported collection method", method: http.MethodDelete,
			path: "/v1/sites", wantStatus: http.StatusMethodNotAllowed,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			request.Header.Set("Authorization", token)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func TestExternalContainerLogsAreForbidden(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("Unix Socket integration test")
	}
	server := testServer(t)
	id := strings.Repeat("a", 64)
	request := httptest.NewRequest(http.MethodGet, "/v1/docker/containers/"+id+"/logs", nil)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("external logs status = %d body=%s", response.Code, response.Body.String())
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
	docker.ConfigureDaemonAccess("", true)
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
			switch {
			case r.URL.Path == "/_ping":
				_, _ = w.Write([]byte("OK"))
				return
			case strings.HasSuffix(r.URL.Path, "/json"):
				_, _ = w.Write([]byte(`{"Id":"` + strings.Repeat("a", 64) + `","Config":{"Labels":{}},"State":{"Status":"running"}}`))
				return
			}
			http.NotFound(w, r)
		})}
		_ = server.Serve(listener)
	}()
	return path
}

var _ = context.Background
