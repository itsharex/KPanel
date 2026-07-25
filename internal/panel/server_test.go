package panel

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/auth"
	"github.com/kejilion/kejilion-panel/internal/store"
)

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	directory := t.TempDir()
	webRoot := filepath.Join(directory, "web")
	if err := os.MkdirAll(webRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webRoot, "index.html"), []byte("<!doctype html><title>panel</title>"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := DefaultConfig()
	config.DataDir = directory
	config.StorePath = filepath.Join(directory, "state.json")
	config.BootstrapTokenPath = filepath.Join(directory, "bootstrap.token")
	config.AgentSocket = filepath.Join(directory, "agent.sock")
	config.AgentTokenFile = filepath.Join(directory, "agent.token")
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
	sessionRequest.AddCookie(sessionCookie)
	sessionRequest.AddCookie(csrfCookie)
	sessionResponse := httptest.NewRecorder()
	server.ServeHTTP(sessionResponse, sessionRequest)
	if sessionResponse.Code != http.StatusOK {
		t.Fatalf("session failed: %d %s", sessionResponse.Code, sessionResponse.Body.String())
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
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

func TestSPAFallback(t *testing.T) {
	server, _ := newTestServer(t)
	response := performRequest(server, http.MethodGet, "/sites/example", nil, nil)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte("<title>panel</title>")) {
		t.Fatalf("SPA fallback failed: %d %s", response.Code, response.Body.String())
	}
}

func performRequest(handler http.Handler, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
