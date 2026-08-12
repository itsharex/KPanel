package panel

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/browsercore"
)

func TestBrowserSessionCreateIssuesRelayScopedToken(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	codec, err := browsercore.NewTokenCodec([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	server.config.PublicURL = "http://localhost:8080"
	server.config.BrowserRelayURL = "https://browser-relay.example.com"
	server.config.BrowserMode = browsercore.RuntimeModeBeta
	server.browserTokens = codec

	request := newAuthenticatedSiteRequest(
		sessionCookie, csrfCookie, http.MethodPost, "/api/v1/browser/sessions", nil, true,
	)
	request.Host = "localhost:8080"
	request.Header.Set("Origin", "http://localhost:8080")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache control = %q", response.Header().Get("Cache-Control"))
	}
	var payload browserRelaySessionResponse
	if json.Unmarshal(response.Body.Bytes(), &payload) != nil {
		t.Fatalf("invalid response: %s", response.Body.String())
	}
	claims, err := codec.Verify(payload.Token)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Mode != browsercore.RuntimeModeBeta || payload.RelayURL != server.config.BrowserRelayURL || payload.SessionID != claims.SessionID ||
		claims.Subject == "" || payload.ExpiresAt.Unix() != claims.ExpiresAt ||
		!payload.ExpiresAt.Equal(payload.ExpiresAt.UTC()) {
		t.Fatalf("response = %#v, claims = %#v", payload, claims)
	}
}

func TestBrowserSessionCreateRequiresAuthOriginCSRFAndConfiguration(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)

	unauthenticated := httptest.NewRequest(http.MethodPost, "/api/v1/browser/sessions", nil)
	unauthenticated.Host = "panel.test"
	unauthenticated.Header.Set("Origin", "http://panel.test")
	unauthenticatedResponse := httptest.NewRecorder()
	server.ServeHTTP(unauthenticatedResponse, unauthenticated)
	if unauthenticatedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", unauthenticatedResponse.Code)
	}

	withoutCSRF := authenticatedSiteRequest(
		server, sessionCookie, csrfCookie, http.MethodPost, "/api/v1/browser/sessions", nil, false,
	)
	if withoutCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF status = %d", withoutCSRF.Code)
	}

	unconfigured := authenticatedSiteRequest(
		server, sessionCookie, csrfCookie, http.MethodPost, "/api/v1/browser/sessions", nil, true,
	)
	if unconfigured.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured status = %d, body = %s", unconfigured.Code, unconfigured.Body.String())
	}
	if !strings.Contains(unconfigured.Body.String(), "browser_disabled") {
		t.Fatalf("disabled beta response = %s", unconfigured.Body.String())
	}
}

func TestBrowserSessionCreateSupportsReaderOnPublicHTTPAndProxyHTTPS(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	codec, err := browsercore.NewTokenCodec([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	server.config.BrowserMode = browsercore.RuntimeModeReader
	server.config.BrowserRelayURL = "http://198.51.100.10:8081"
	server.config.BrowserRelayInternalURL = "http://browser-relay:8090"
	server.config.AllowIPHosts = true
	server.browserTokens = codec
	server.browserRelayClient = &http.Client{}

	for name, configure := range map[string]func(*http.Request){
		"public HTTP IP": func(request *http.Request) {
			request.Host = "198.51.100.10:8080"
			request.Header.Set("Origin", "http://198.51.100.10:8080")
		},
		"trusted proxy HTTPS": func(request *http.Request) {
			request.RemoteAddr = "127.0.0.1:12345"
			request.Host = "panel.example.com"
			request.Header.Set("Origin", "https://panel.example.com")
			request.Header.Set("X-Forwarded-Proto", "https")
		},
	} {
		t.Run(name, func(t *testing.T) {
			request := newAuthenticatedSiteRequest(
				sessionCookie, csrfCookie, http.MethodPost, "/api/v1/browser/sessions", nil, true,
			)
			configure(request)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != http.StatusCreated {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var payload browserRelaySessionResponse
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			claims, err := codec.Verify(payload.Token)
			if err != nil || claims.Scope != browsercore.TokenScopeReader || payload.Mode != browsercore.RuntimeModeReader ||
				payload.RelayURL != request.Header.Get("Origin") {
				t.Fatalf("reader response = %#v, claims = %#v, err = %v", payload, claims, err)
			}
		})
	}
}

func TestBrowserSessionCreateRequiresConfiguredSecureBrowserOrigin(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	codec, err := browsercore.NewTokenCodec([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	server.config.BrowserMode = browsercore.RuntimeModeBeta
	server.config.BrowserRelayURL = "https://browser-relay.example.com"
	server.browserTokens = codec

	t.Run("trusted HTTPS proxy", func(t *testing.T) {
		server.config.PublicURL = "https://panel.example.com"
		request := newAuthenticatedSiteRequest(
			sessionCookie, csrfCookie, http.MethodPost, "/api/v1/browser/sessions", nil, true,
		)
		request.RemoteAddr = "127.0.0.1:12345"
		request.Host = "panel.example.com"
		request.Header.Set("Origin", "https://panel.example.com")
		request.Header.Set("X-Forwarded-Proto", "https")
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusCreated {
			t.Fatalf("trusted HTTPS proxy status = %d, body = %s", response.Code, response.Body.String())
		}
	})

	t.Run("trusted HTTPS proxy origin mismatch", func(t *testing.T) {
		server.config.PublicURL = "https://panel.example.com"
		request := newAuthenticatedSiteRequest(
			sessionCookie, csrfCookie, http.MethodPost, "/api/v1/browser/sessions", nil, true,
		)
		request.RemoteAddr = "127.0.0.1:12345"
		request.Host = "other.example.com"
		request.Header.Set("Origin", "https://other.example.com")
		request.Header.Set("X-Forwarded-Proto", "https")
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusMisdirectedRequest ||
			!strings.Contains(response.Body.String(), "browser_origin_mismatch") {
			t.Fatalf("proxy origin mismatch status = %d, body = %s", response.Code, response.Body.String())
		}
	})

	t.Run("direct TLS", func(t *testing.T) {
		server.config.PublicURL = "https://panel.example.com"
		request := newAuthenticatedSiteRequest(
			sessionCookie, csrfCookie, http.MethodPost, "/api/v1/browser/sessions", nil, true,
		)
		request.Host = "panel.example.com"
		request.Header.Set("Origin", "https://panel.example.com")
		request.TLS = &tls.ConnectionState{}
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusCreated {
			t.Fatalf("direct TLS status = %d, body = %s", response.Code, response.Body.String())
		}
	})

	t.Run("public HTTP IP", func(t *testing.T) {
		server.config.PublicURL = "https://panel.example.com"
		server.config.AllowIPHosts = true
		request := newAuthenticatedSiteRequest(
			sessionCookie, csrfCookie, http.MethodPost, "/api/v1/browser/sessions", nil, true,
		)
		request.Host = "198.51.100.10:8080"
		request.Header.Set("Origin", "http://198.51.100.10:8080")
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable ||
			!strings.Contains(response.Body.String(), "browser_secure_context_required") {
			t.Fatalf("public HTTP IP status = %d, body = %s", response.Code, response.Body.String())
		}
	})

	t.Run("localhost HTTP", func(t *testing.T) {
		server.config.PublicURL = "http://localhost:8080"
		server.config.BrowserRelayURL = "http://127.0.0.1:8081"
		server.config.AllowIPHosts = false
		request := newAuthenticatedSiteRequest(
			sessionCookie, csrfCookie, http.MethodPost, "/api/v1/browser/sessions", nil, true,
		)
		request.Host = "localhost:8080"
		request.Header.Set("Origin", "http://localhost:8080")
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusCreated {
			t.Fatalf("localhost HTTP status = %d, body = %s", response.Code, response.Body.String())
		}
	})
}
