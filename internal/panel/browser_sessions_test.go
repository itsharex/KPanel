package panel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	server.config.BrowserRelayURL = "https://browser-relay.example.com"
	server.browserTokens = codec

	response := authenticatedSiteRequest(
		server, sessionCookie, csrfCookie, http.MethodPost, "/api/v1/browser/sessions", nil, true,
	)
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
	if payload.RelayURL != server.config.BrowserRelayURL || payload.SessionID != claims.SessionID ||
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
}
