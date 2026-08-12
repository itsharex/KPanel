package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/browsercore"
)

func TestReadSecretRequiresBoundedStrongFile(t *testing.T) {
	directory := t.TempDir()
	validPath := filepath.Join(directory, "valid")
	if err := os.WriteFile(validPath, []byte("0123456789abcdef0123456789abcdef\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secret, err := browsercore.LoadSecretFile(validPath)
	if err != nil || string(secret) != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("valid secret = %q, %v", secret, err)
	}

	shortPath := filepath.Join(directory, "short")
	if err := os.WriteFile(shortPath, []byte("short"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := browsercore.LoadSecretFile(shortPath); err == nil {
		t.Fatal("short secret accepted")
	}
	if _, err := browsercore.LoadSecretFile(""); err == nil {
		t.Fatal("missing path accepted")
	}
}

func TestRelayHealthcheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/healthz" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	if err := relayHealthcheck(server.URL + "/healthz"); err != nil {
		t.Fatal(err)
	}
}

func TestRelayHealthcheckRejectsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	if err := relayHealthcheck(server.URL); err == nil {
		t.Fatal("failed healthcheck accepted")
	}
}

func TestRelayHealthcheckRequiresExpectedRuntimeMode(t *testing.T) {
	t.Setenv("KEJILION_BROWSER_RELAY_EXPECT_MODE", browsercore.RuntimeModeReader)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"mode":"reader"}`))
	}))
	t.Cleanup(server.Close)
	if err := relayHealthcheck(server.URL); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KEJILION_BROWSER_RELAY_EXPECT_MODE", browsercore.RuntimeModeBeta)
	if err := relayHealthcheck(server.URL); err == nil {
		t.Fatal("relay mode mismatch was accepted")
	}
}

func TestDisabledRuntimeHandlerKeepsHealthAndRejectsBrowserTraffic(t *testing.T) {
	handler := disabledRuntimeHandler()
	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d", health.Code)
	}
	if !strings.Contains(health.Body.String(), `"mode":"disabled"`) {
		t.Fatalf("disabled health payload = %q", health.Body.String())
	}
	head := httptest.NewRecorder()
	handler.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/healthz", nil))
	if head.Code != http.StatusOK {
		t.Fatalf("HEAD health status = %d", head.Code)
	}
	post := httptest.NewRecorder()
	handler.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/healthz", nil))
	if post.Code != http.StatusMethodNotAllowed || post.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("POST health status = %d, headers = %v", post.Code, post.Header())
	}

	request := httptest.NewRecorder()
	handler.ServeHTTP(request, httptest.NewRequest(http.MethodGet, "/kpanel-browser/kernel.html", nil))
	if request.Code != http.StatusServiceUnavailable || request.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("browser status = %d, headers = %v", request.Code, request.Header())
	}
}

func TestRelayRejectsUnsupportedRuntimeModeBeforeListening(t *testing.T) {
	if err := runArgs([]string{"-mode=alpha"}); err == nil {
		t.Fatal("unsupported runtime mode accepted")
	}
}
