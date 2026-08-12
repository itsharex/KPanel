package panel

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/browsercore"
)

type panelReaderRoundTripFunc func(*http.Request) (*http.Response, error)

type panelReaderUnexpectedBody struct{ t *testing.T }

func (body panelReaderUnexpectedBody) Read([]byte) (int, error) {
	body.t.Fatal("redirect response body was read")
	return 0, io.EOF
}

func (panelReaderUnexpectedBody) Close() error { return nil }

func (function panelReaderRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestBrowserReaderAssetsEnforceAnOpaqueNoNetworkSandbox(t *testing.T) {
	server := &Server{}
	page := httptest.NewRecorder()
	server.handleBrowserReaderAsset(page, httptest.NewRequest(http.MethodGet, browserReaderAssetPrefix, nil))
	if page.Code != http.StatusOK {
		t.Fatalf("reader page = %d %q", page.Code, page.Body.String())
	}
	policy := page.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "sandbox allow-scripts") || !strings.Contains(policy, "connect-src 'none'") ||
		!strings.Contains(policy, "frame-ancestors 'self'") || strings.Contains(policy, "allow-same-origin") {
		t.Fatalf("reader CSP = %q", policy)
	}
	noncePattern := regexp.MustCompile(`script-src 'nonce-([^']+)'`)
	nonceMatch := noncePattern.FindStringSubmatch(policy)
	if len(nonceMatch) != 2 || !strings.Contains(policy, "style-src 'nonce-"+nonceMatch[1]+"'") {
		t.Fatalf("reader CSP nonce = %q", policy)
	}
	pageBody := page.Body.String()
	if !strings.Contains(pageBody, `<script nonce="`+nonceMatch[1]+`">`) ||
		!strings.Contains(pageBody, `<style nonce="`+nonceMatch[1]+`">`) ||
		!strings.Contains(pageBody, string(browserReaderJS)) || !strings.Contains(pageBody, string(browserReaderCSS)) {
		t.Fatal("reader page did not inline its nonce-authorized runtime assets")
	}
	for _, forbidden := range []string{"<script src=", `rel="stylesheet"`, browserReaderNonceMarker, browserReaderCSSMarker, browserReaderJSMarker} {
		if strings.Contains(pageBody, forbidden) {
			t.Fatalf("reader page retained forbidden external asset or template marker %q", forbidden)
		}
	}
	if page.Header().Get("X-Frame-Options") != "SAMEORIGIN" || page.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("reader headers = %#v", page.Header())
	}
	secondPage := httptest.NewRecorder()
	server.handleBrowserReaderAsset(secondPage, httptest.NewRequest(http.MethodGet, browserReaderAssetPrefix, nil))
	secondMatch := noncePattern.FindStringSubmatch(secondPage.Header().Get("Content-Security-Policy"))
	if len(secondMatch) != 2 || secondMatch[1] == nonceMatch[1] {
		t.Fatal("reader page CSP nonce was not regenerated per response")
	}

	script := httptest.NewRecorder()
	server.handleBrowserReaderAsset(script, httptest.NewRequest(http.MethodGet, browserReaderAssetPrefix+"reader.js", nil))
	if script.Code != http.StatusOK {
		t.Fatalf("reader script status = %d", script.Code)
	}
	source := script.Body.String()
	for _, forbidden := range []string{"fetch(", "Authorization", "document.cookie", "localStorage", ".innerHTML", ".srcdoc", "allow-same-origin"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("reader script contains forbidden capability %q", forbidden)
		}
	}
	for _, required := range []string{"event.ports", "port.postMessage", "discardedTags", "maxNodes", "maxImageTotalBytes", "data-kpanel-href"} {
		if !strings.Contains(source, required) {
			t.Fatalf("reader script is missing security control %q", required)
		}
	}
	if !strings.Contains(source, "event.source !== window.parent") || !strings.Contains(source, "event.ports.length !== 1") {
		t.Fatal("reader connection does not bind its private port to the parent window")
	}
}

func TestBrowserReaderFetchKeepsTokenInPanelAndUsesFixedRelay(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	authenticated, err := server.auth.Authenticate(sessionCookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	codec, err := browsercore.NewTokenCodec([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	readerToken, _, err := codec.IssueScoped(authenticated.User.ID, browsercore.TokenScopeReader, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := browsercore.EncodeHeaderPairs([]browsercore.HeaderPair{
		{"Content-Type", "text/html; charset=utf-8"},
		{"Location", "https://example.com/next"},
	})
	if err != nil {
		t.Fatal(err)
	}
	relayCalls := 0
	server.config.BrowserMode = browsercore.RuntimeModeReader
	server.config.BrowserRelayURL = "http://198.51.100.10:8081"
	server.config.BrowserRelayInternalURL = "http://browser-relay:8090"
	server.browserTokens = codec
	server.browserRelayClient = &http.Client{Transport: panelReaderRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		relayCalls++
		if request.URL.String() != "http://browser-relay:8090/v1/fetch" ||
			request.Header.Get("Origin") != server.config.BrowserRelayURL ||
			request.Header.Get("Authorization") != "Bearer "+readerToken ||
			request.Header.Get(browsercore.HeaderTargetURL) != "https://example.com/path" ||
			request.Header.Get(browsercore.HeaderTargetMethod) != http.MethodGet || request.ContentLength != 0 {
			t.Fatalf("internal relay request = %#v", request)
		}
		encoded := request.Header.Get(browsercore.HeaderTargetHeaders)
		payload, decodeErr := base64.RawURLEncoding.DecodeString(encoded)
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		var pairs []browsercore.HeaderPair
		if json.Unmarshal(payload, &pairs) != nil {
			t.Fatalf("target headers = %s", payload)
		}
		for _, pair := range pairs {
			if strings.EqualFold(pair[0], "Cookie") || strings.EqualFold(pair[0], "Authorization") ||
				strings.EqualFold(pair[0], "Origin") || strings.EqualFold(pair[0], "Referer") {
				t.Fatalf("unsafe target header = %#v", pair)
			}
		}
		headers := make(http.Header)
		headers.Set(browsercore.HeaderUpstreamStatus, "200")
		headers.Set(browsercore.HeaderUpstreamHeaders, metadata)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     headers,
			Body:       io.NopCloser(strings.NewReader("<main>safe reader body</main>")),
		}, nil
	})}

	body, _ := json.Marshal(browserReaderFetchInput{URL: "https://example.com/path", Kind: "document"})
	request := newAuthenticatedSiteRequest(
		sessionCookie, csrfCookie, http.MethodPost, "/api/v1/browser/reader/fetch", body, true,
	)
	request.Header.Set(browserReaderTokenHeader, readerToken)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "<main>safe reader body</main>" || relayCalls != 1 {
		t.Fatalf("reader response = %d %q, calls=%d", response.Code, response.Body.String(), relayCalls)
	}
	if response.Header().Get(browsercore.HeaderUpstreamHeaders) != metadata || response.Header().Get("Set-Cookie") != "" {
		t.Fatalf("reader response headers = %#v", response.Header())
	}

	betaToken, _, err := codec.Issue(authenticated.User.ID, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	wrongScope := newAuthenticatedSiteRequest(
		sessionCookie, csrfCookie, http.MethodPost, "/api/v1/browser/reader/fetch", body, true,
	)
	wrongScope.Header.Set(browserReaderTokenHeader, betaToken)
	wrongScopeResponse := httptest.NewRecorder()
	server.ServeHTTP(wrongScopeResponse, wrongScope)
	if wrongScopeResponse.Code != http.StatusUnauthorized || relayCalls != 1 {
		t.Fatalf("wrong scope response = %d %q, calls=%d", wrongScopeResponse.Code, wrongScopeResponse.Body.String(), relayCalls)
	}
}

func TestBrowserReaderFetchRejectsOversizedImageResponse(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	authenticated, err := server.auth.Authenticate(sessionCookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	codec, _ := browsercore.NewTokenCodec([]byte("0123456789abcdef0123456789abcdef"))
	readerToken, _, _ := codec.IssueScoped(authenticated.User.ID, browsercore.TokenScopeReader, 10*time.Minute)
	metadata, _ := browsercore.EncodeHeaderPairs([]browsercore.HeaderPair{{"Content-Type", "image/png"}})
	server.config.BrowserMode = browsercore.RuntimeModeReader
	server.config.BrowserRelayURL = "http://198.51.100.10:8081"
	server.config.BrowserRelayInternalURL = "http://browser-relay:8090"
	server.browserTokens = codec
	server.browserRelayClient = &http.Client{Transport: panelReaderRoundTripFunc(func(*http.Request) (*http.Response, error) {
		headers := make(http.Header)
		headers.Set(browsercore.HeaderUpstreamStatus, "200")
		headers.Set(browsercore.HeaderUpstreamHeaders, metadata)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     headers,
			Body:       io.NopCloser(bytes.NewReader(make([]byte, browserReaderImageBytes+1))),
		}, nil
	})}
	for _, kind := range []string{"image", "document"} {
		t.Run(kind, func(t *testing.T) {
			body, _ := json.Marshal(browserReaderFetchInput{URL: "https://example.com/image.png", Kind: kind})
			request := newAuthenticatedSiteRequest(
				sessionCookie, csrfCookie, http.MethodPost, "/api/v1/browser/reader/fetch", body, true,
			)
			request.Header.Set(browserReaderTokenHeader, readerToken)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != http.StatusRequestEntityTooLarge ||
				!strings.Contains(response.Body.String(), "browser_reader_response_too_large") {
				t.Fatalf("oversized reader response = %d %q", response.Code, response.Body.String())
			}
		})
	}
}

func TestBrowserReaderFetchDoesNotBufferRedirectBodies(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	authenticated, err := server.auth.Authenticate(sessionCookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	codec, _ := browsercore.NewTokenCodec([]byte("0123456789abcdef0123456789abcdef"))
	readerToken, _, _ := codec.IssueScoped(authenticated.User.ID, browsercore.TokenScopeReader, 10*time.Minute)
	metadata, _ := browsercore.EncodeHeaderPairs([]browsercore.HeaderPair{{"Location", "https://example.com/next"}})
	server.config.BrowserMode = browsercore.RuntimeModeReader
	server.config.BrowserRelayURL = "http://198.51.100.10:8081"
	server.config.BrowserRelayInternalURL = "http://browser-relay:8090"
	server.browserTokens = codec
	server.browserRelayClient = &http.Client{Transport: panelReaderRoundTripFunc(func(*http.Request) (*http.Response, error) {
		headers := make(http.Header)
		headers.Set(browsercore.HeaderUpstreamStatus, "302")
		headers.Set(browsercore.HeaderUpstreamHeaders, metadata)
		return &http.Response{StatusCode: http.StatusOK, Header: headers, Body: panelReaderUnexpectedBody{t: t}}, nil
	})}
	body, _ := json.Marshal(browserReaderFetchInput{URL: "https://example.com/", Kind: "document"})
	request := newAuthenticatedSiteRequest(sessionCookie, csrfCookie, http.MethodPost,
		"/api/v1/browser/reader/fetch", body, true)
	request.Header.Set(browserReaderTokenHeader, readerToken)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.Len() != 0 ||
		response.Header().Get(browsercore.HeaderUpstreamStatus) != "302" {
		t.Fatalf("redirect response = %d %q %#v", response.Code, response.Body.String(), response.Header())
	}
}
