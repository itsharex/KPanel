package panel

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/auth"
	"github.com/kejilion/kejilion-panel/internal/store"
)

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	directory := t.TempDir()
	webRoot := filepath.Join(directory, "web")
	dataDir := filepath.Join(directory, "data")
	if err := os.MkdirAll(webRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webRoot, "index.html"), []byte("<!doctype html><title>panel</title>"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := DefaultConfig()
	config.DataDir = dataDir
	config.StorePath = filepath.Join(dataDir, "state.json")
	config.BootstrapTokenPath = filepath.Join(dataDir, "bootstrap.token")
	config.AgentSocket = filepath.Join(directory, "run", "agent.sock")
	config.AgentTokenFile = filepath.Join(directory, "secrets", "agent.token")
	config.WebRoot = webRoot
	config.PublicURL = "http://panel.test"
	config.SecureCookie = false
	config.CookieName = "kejilion_session"
	config.SessionTTL = time.Hour
	config.SessionTTLText = "1h"

	storage, err := store.Open(config.StorePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = storage.Close() })
	hasher, err := auth.NewArgon2idHasher(auth.Argon2idParams{
		MemoryKiB: 8 * 1024, Iterations: 1, Threads: 1, SaltLength: 16, KeyLength: 32,
	})
	if err != nil {
		t.Fatal(err)
	}
	authService, err := auth.NewService(storage, hasher, auth.Config{
		BootstrapTokenPath: config.BootstrapTokenPath,
		SessionTTL:         time.Hour,
		LoginWindow:        time.Minute,
		MaxLoginFailures:   3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := authService.EnsureBootstrapToken(); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(
		config,
		authService,
		storage,
		NewAgentClient(config.AgentSocket, config.AgentTokenFile, config.MaxAgentBytes),
	)
	if err != nil {
		t.Fatal(err)
	}
	return server, config.BootstrapTokenPath
}

func TestAuthenticationHTTPFlow(t *testing.T) {
	server, tokenPath := newTestServer(t)

	status := performRequest(server, http.MethodGet, "/api/v1/auth/bootstrap", nil, nil)
	if status.Code != http.StatusOK || status.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("unexpected bootstrap status: %d headers=%v", status.Code, status.Header())
	}
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{
		"token": string(token), "username": "admin", "password": "a-strong-password",
	})
	bootstrap := performRequest(server, http.MethodPost, "/api/v1/auth/bootstrap", body, map[string]string{
		"Content-Type": "application/json",
		"Origin":       "http://panel.test",
	})
	if bootstrap.Code != http.StatusCreated {
		t.Fatalf("bootstrap failed: %d %s", bootstrap.Code, bootstrap.Body.String())
	}
	cookies := bootstrap.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected session and CSRF cookies, got %d", len(cookies))
	}
	var sessionCookie, csrfCookie *http.Cookie
	for _, cookie := range cookies {
		switch cookie.Name {
		case "kejilion_session":
			sessionCookie = cookie
		case "kejilion_csrf":
			csrfCookie = cookie
		}
	}
	if sessionCookie == nil || !sessionCookie.HttpOnly || csrfCookie == nil || csrfCookie.HttpOnly {
		t.Fatalf("unexpected cookie flags: %#v", cookies)
	}

	sessionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	sessionRequest.Host = "panel.test"
	sessionRequest.AddCookie(sessionCookie)
	sessionRequest.AddCookie(csrfCookie)
	sessionResponse := httptest.NewRecorder()
	server.ServeHTTP(sessionResponse, sessionRequest)
	if sessionResponse.Code != http.StatusOK {
		t.Fatalf("session failed: %d %s", sessionResponse.Code, sessionResponse.Body.String())
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRequest.Host = "panel.test"
	logoutRequest.Header.Set("Origin", "http://panel.test")
	logoutRequest.Header.Set("X-CSRF-Token", csrfCookie.Value)
	logoutRequest.AddCookie(sessionCookie)
	logoutRequest.AddCookie(csrfCookie)
	logoutResponse := httptest.NewRecorder()
	server.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusOK {
		t.Fatalf("logout failed: %d %s", logoutResponse.Code, logoutResponse.Body.String())
	}

	expiredRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	expiredRequest.Host = "panel.test"
	expiredRequest.AddCookie(sessionCookie)
	expiredResponse := httptest.NewRecorder()
	server.ServeHTTP(expiredResponse, expiredRequest)
	if expiredResponse.Code != http.StatusUnauthorized {
		t.Fatalf("logged-out session accepted: %d", expiredResponse.Code)
	}
}

func TestRejectsCrossOriginBootstrap(t *testing.T) {
	server, tokenPath := newTestServer(t)
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{
		"token": string(token), "username": "admin", "password": "a-strong-password",
	})
	response := performRequest(server, http.MethodPost, "/api/v1/auth/bootstrap", body, map[string]string{
		"Content-Type": "application/json",
		"Origin":       "https://evil.example",
	})
	if response.Code != http.StatusForbidden {
		t.Fatalf("cross-origin bootstrap returned %d", response.Code)
	}
}

func TestRejectsMissingOriginAndUnexpectedHost(t *testing.T) {
	server, tokenPath := newTestServer(t)
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{
		"token": string(token), "username": "admin", "password": "a-strong-password",
	})

	missingOrigin := performRequest(server, http.MethodPost, "/api/v1/auth/bootstrap", body, map[string]string{
		"Content-Type": "application/json",
	})
	if missingOrigin.Code != http.StatusForbidden {
		t.Fatalf("missing Origin returned %d", missingOrigin.Code)
	}

	unexpectedHost := performRequest(server, http.MethodGet, "/api/v1/auth/bootstrap", nil, map[string]string{
		"Host": "attacker.test",
	})
	if unexpectedHost.Code != http.StatusMisdirectedRequest {
		t.Fatalf("unexpected Host returned %d", unexpectedHost.Code)
	}
}

func TestSPAFallback(t *testing.T) {
	server, _ := newTestServer(t)
	response := performRequest(server, http.MethodGet, "/sites/example", nil, nil)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte("<title>panel</title>")) {
		t.Fatalf("SPA fallback failed: %d %s", response.Code, response.Body.String())
	}
}

func TestStaticFilesRejectSymbolicLinks(t *testing.T) {
	server, _ := newTestServer(t)
	outside := filepath.Join(filepath.Dir(server.config.WebRoot), "outside")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("must-not-leak"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(server.config.WebRoot, "linked")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	response := performRequest(server, http.MethodGet, "/linked/secret.txt", nil, nil)
	if response.Code != http.StatusNotFound || strings.Contains(response.Body.String(), "must-not-leak") {
		t.Fatalf("symbolic-link target was served: %d %q", response.Code, response.Body.String())
	}
}

func TestRemoteIPTrustsOnlyConfiguredProxies(t *testing.T) {
	server, _ := newTestServer(t)

	trusted := httptest.NewRequest(http.MethodGet, "/", nil)
	trusted.RemoteAddr = "127.0.0.1:12345"
	trusted.Header.Set("X-Real-IP", "198.51.100.25")
	if got := server.remoteIP(trusted); got != "198.51.100.25" {
		t.Fatalf("trusted proxy address was ignored: %q", got)
	}

	untrusted := httptest.NewRequest(http.MethodGet, "/", nil)
	untrusted.RemoteAddr = "192.0.2.10:12345"
	untrusted.Header.Set("X-Real-IP", "198.51.100.25")
	if got := server.remoteIP(untrusted); got != "192.0.2.10" {
		t.Fatalf("untrusted proxy spoofed client address: %q", got)
	}
}

func TestDockerActionFailsClosedWhenIntentAuditCannotPersist(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("cannot remove an open lock file on Windows")
	}
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)

	dataDir := server.config.DataDir
	if err := os.RemoveAll(dataDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dataDir, []byte("block store directory recreation"), 0o600); err != nil {
		t.Fatal(err)
	}

	containerID := strings.Repeat("a", 64)
	body, err := json.Marshal(map[string]string{
		"resourceVersion": "sha256:" + strings.Repeat("b", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/docker/containers/"+containerID+"/restart",
		bytes.NewReader(body),
	)
	request.Host = "panel.test"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://panel.test")
	request.Header.Set("X-CSRF-Token", csrfCookie.Value)
	request.AddCookie(sessionCookie)
	request.AddCookie(csrfCookie)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable ||
		!strings.Contains(response.Body.String(), "audit_unavailable") {
		t.Fatalf("Docker action did not fail closed: %d %s", response.Code, response.Body.String())
	}
}

func TestAllowedDockerActionPath(t *testing.T) {
	t.Parallel()
	id := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	path, gotID, action, ok := allowedDockerActionPath("/api/v1/docker/containers/" + id + "/restart")
	if !ok || gotID != id || action != "restart" || path != "/v1/docker/containers/"+id+"/restart" {
		t.Fatalf("unexpected action mapping: path=%q id=%q action=%q ok=%v", path, gotID, action, ok)
	}
	for _, invalid := range []string{
		"/api/v1/docker/containers/not-an-id/restart",
		"/api/v1/docker/containers/" + id + "/delete",
		"/api/v1/docker/containers/" + id + "/restart/extra",
	} {
		if _, _, _, ok := allowedDockerActionPath(invalid); ok {
			t.Errorf("accepted unsafe action path %q", invalid)
		}
	}
}

func bootstrapCookies(t *testing.T, server *Server, tokenPath string) (*http.Cookie, *http.Cookie) {
	t.Helper()
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]string{
		"token": string(token), "username": "admin", "password": "a-strong-password",
	})
	if err != nil {
		t.Fatal(err)
	}
	response := performRequest(server, http.MethodPost, "/api/v1/auth/bootstrap", body, map[string]string{
		"Content-Type": "application/json",
		"Origin":       "http://panel.test",
	})
	if response.Code != http.StatusCreated {
		t.Fatalf("bootstrap failed: %d %s", response.Code, response.Body.String())
	}
	var sessionCookie, csrfCookie *http.Cookie
	for _, cookie := range response.Result().Cookies() {
		switch cookie.Name {
		case "kejilion_session":
			sessionCookie = cookie
		case "kejilion_csrf":
			csrfCookie = cookie
		}
	}
	if sessionCookie == nil || csrfCookie == nil {
		t.Fatalf("authentication cookies missing: %#v", response.Result().Cookies())
	}
	return sessionCookie, csrfCookie
}

func performRequest(handler http.Handler, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Host = "panel.test"
	for name, value := range headers {
		if name == "Host" {
			request.Host = value
		} else {
			request.Header.Set(name, value)
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
