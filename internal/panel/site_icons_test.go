package panel

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestSiteIconProxyRequiresSessionAndReturnsPrivateCacheHeaders(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	id := strings.Repeat("a", 32)
	body := append([]byte("\x89PNG\r\n\x1a\n"), []byte("panel-icon")...)
	agent := &stubAgent{response: AgentResponse{
		StatusCode: http.StatusOK, ContentType: "image/png", Body: body,
	}}
	server.agent = agent

	unauthenticated := performRequest(
		server,
		http.MethodGet,
		"/api/v1/sites/"+id+"/icon",
		nil,
		nil,
	)
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", unauthenticated.Code)
	}
	if calls := agent.snapshotCalls(); len(calls) != 0 {
		t.Fatalf("Agent called before authentication: %#v", calls)
	}

	response := authenticatedRequest(
		server,
		http.MethodGet,
		"/api/v1/sites/"+id+"/icon",
		nil,
		sessionCookie,
		csrfCookie,
		nil,
	)
	if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), body) {
		t.Fatalf("icon response = %d %q", response.Code, response.Body.Bytes())
	}
	sum := sha256.Sum256(body)
	expectedETag := `"` + hex.EncodeToString(sum[:]) + `"`
	if response.Header().Get("Content-Type") != "image/png" ||
		response.Header().Get("Cache-Control") != siteIconBrowserCache ||
		response.Header().Get("ETag") != expectedETag ||
		response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("icon headers = %#v", response.Header())
	}
	calls := agent.snapshotCalls()
	if len(calls) != 1 || calls[0].method != http.MethodGet ||
		calls[0].path != "/v1/sites/"+id+"/icon" || calls[0].rawQuery != "" {
		t.Fatalf("Agent calls = %#v", calls)
	}

	notModified := authenticatedRequest(
		server,
		http.MethodGet,
		"/api/v1/sites/"+id+"/icon",
		nil,
		sessionCookie,
		csrfCookie,
		map[string]string{"If-None-Match": "W/" + expectedETag},
	)
	if notModified.Code != http.StatusNotModified || notModified.Body.Len() != 0 ||
		notModified.Header().Get("ETag") != expectedETag ||
		notModified.Header().Get("Cache-Control") != siteIconBrowserCache {
		t.Fatalf("conditional response = %d headers=%#v body=%q", notModified.Code, notModified.Header(), notModified.Body.Bytes())
	}
}

func TestSiteIconProxyRejectsMalformedPathsBeforeAgent(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	id := strings.Repeat("b", 32)
	agent := &stubAgent{}
	server.agent = agent
	for name, target := range map[string]string{
		"uppercase":     "/api/v1/sites/" + strings.Repeat("B", 32) + "/icon",
		"short":         "/api/v1/sites/" + strings.Repeat("b", 31) + "/icon",
		"extra segment": "/api/v1/sites/" + id + "/extra/icon",
		"query":         "/api/v1/sites/" + id + "/icon?refresh=true",
	} {
		t.Run(name, func(t *testing.T) {
			response := authenticatedRequest(
				server,
				http.MethodGet,
				target,
				nil,
				sessionCookie,
				csrfCookie,
				nil,
			)
			expected := http.StatusNotFound
			if name == "query" {
				expected = http.StatusBadRequest
			}
			if response.Code != expected {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, expected, response.Body.String())
			}
		})
	}
	if calls := agent.snapshotCalls(); len(calls) != 0 {
		t.Fatalf("Agent called for malformed icon routes: %#v", calls)
	}
}

func TestSiteIconProxyPreservesAgentFailuresAndRejectsUnsafeResponses(t *testing.T) {
	id := strings.Repeat("c", 32)
	tests := []struct {
		name         string
		agent        *stubAgent
		expectedCode int
		expectedBody string
	}{
		{
			name: "Agent not found",
			agent: &stubAgent{response: AgentResponse{
				StatusCode:  http.StatusNotFound,
				ContentType: "application/problem+json",
				Body:        []byte(`{"code":"site_icon_not_found","status":404}`),
			}},
			expectedCode: http.StatusNotFound,
			expectedBody: "site_icon_not_found",
		},
		{
			name:         "transport failure",
			agent:        &stubAgent{err: errors.New("socket unavailable")},
			expectedCode: http.StatusServiceUnavailable,
			expectedBody: "agent_unavailable",
		},
		{
			name: "active media",
			agent: &stubAgent{response: AgentResponse{
				StatusCode:  http.StatusOK,
				ContentType: "image/svg+xml",
				Body:        []byte(`<svg><script>alert(1)</script></svg>`),
			}},
			expectedCode: http.StatusBadGateway,
			expectedBody: "invalid_site_icon",
		},
		{
			name: "forged MIME",
			agent: &stubAgent{response: AgentResponse{
				StatusCode:  http.StatusOK,
				ContentType: "image/png",
				Body:        []byte("<!doctype html>"),
			}},
			expectedCode: http.StatusBadGateway,
			expectedBody: "invalid_site_icon",
		},
		{
			name: "oversized media",
			agent: &stubAgent{response: AgentResponse{
				StatusCode:  http.StatusOK,
				ContentType: "image/png",
				Body: append(
					[]byte("\x89PNG\r\n\x1a\n"),
					bytes.Repeat([]byte("x"), 256<<10)...,
				),
			}},
			expectedCode: http.StatusBadGateway,
			expectedBody: "invalid_site_icon",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, tokenPath := newTestServer(t)
			sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
			server.agent = test.agent
			response := authenticatedRequest(
				server,
				http.MethodGet,
				"/api/v1/sites/"+id+"/icon",
				nil,
				sessionCookie,
				csrfCookie,
				nil,
			)
			if response.Code != test.expectedCode ||
				!strings.Contains(response.Body.String(), test.expectedBody) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			expectedCache := "no-store"
			if test.expectedCode == http.StatusNotFound {
				expectedCache = siteIconMissingBrowserCache
			}
			if response.Header().Get("Cache-Control") != expectedCache {
				t.Fatalf("failure cache policy = %q, want %q", response.Header().Get("Cache-Control"), expectedCache)
			}
		})
	}
}

func TestAllowedSiteIconPathIsExact(t *testing.T) {
	id := strings.Repeat("d", 32)
	if path, gotID, ok := allowedSiteIconPath("/api/v1/sites/" + id + "/icon"); !ok ||
		path != "/v1/sites/"+id+"/icon" || gotID != id {
		t.Fatalf("valid path = %q, %q, %v", path, gotID, ok)
	}
	for _, invalid := range []string{
		"/api/v1/sites/" + id,
		"/api/v1/sites/" + id + "/icon/",
		"/api/v1/sites/" + id + "/extra/icon",
		"/api/v1/sites/" + strings.Repeat("D", 32) + "/icon",
	} {
		if _, _, ok := allowedSiteIconPath(invalid); ok {
			t.Fatalf("invalid path allowed: %s", invalid)
		}
	}
}

func TestAllowedSiteAppearancePathIsExact(t *testing.T) {
	id := strings.Repeat("e", 32)
	publicPath := "/api/v1/sites/" + id + "/appearance"
	if path, ok := allowedAgentPath(publicPath); !ok || path != "/v1/sites/"+id+"/appearance" {
		t.Fatalf("allowedAgentPath(%q) = %q, %v", publicPath, path, ok)
	}
	for _, invalid := range []string{
		"/api/v1/sites/" + strings.Repeat("E", 32) + "/appearance",
		"/api/v1/sites/" + id + "/appearance/",
		"/api/v1/sites/" + id + "/icon/appearance",
	} {
		if path, ok := allowedAgentPath(invalid); ok {
			t.Fatalf("allowedAgentPath(%q) unexpectedly allowed %q", invalid, path)
		}
	}
}
