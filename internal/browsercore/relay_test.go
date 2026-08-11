package browsercore

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type discardResponseWriter struct {
	header http.Header
	status int
	bytes  int
}

type shortResponseWriter struct{ header http.Header }

func (w *shortResponseWriter) Header() http.Header { return w.header }

func (*shortResponseWriter) WriteHeader(int) {}

func (*shortResponseWriter) Write(payload []byte) (int, error) { return len(payload) - 1, nil }

func (w *discardResponseWriter) Header() http.Header { return w.header }

func (w *discardResponseWriter) WriteHeader(status int) { w.status = status }

func (w *discardResponseWriter) Write(payload []byte) (int, error) {
	w.bytes += len(payload)
	return len(payload), nil
}

func newTestRelay(t testing.TB, roundTripper http.RoundTripper) (*Relay, string) {
	t.Helper()
	tokens, err := NewTokenCodec([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := tokens.Issue("admin", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	policy := NewTargetPolicy(staticResolver{
		"example.com": {netip.MustParseAddr("93.184.216.34")},
	})
	relay, err := NewRelay(RelayConfig{
		AllowedOrigin: "https://panel.example.com",
		RelayOrigin:   "https://relay.example.com",
	}, tokens, policy, NewLimiter(4, 2))
	if err != nil {
		t.Fatal(err)
	}
	if roundTripper != nil {
		relay.client.Transport = roundTripper
	}
	return relay, token
}

func relayRequest(t testing.TB, token string, body io.Reader) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "https://relay.example.com/v1/fetch", body)
	request.Header.Set("Origin", "https://relay.example.com")
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set(HeaderTargetURL, "https://example.com/page")
	request.Header.Set(HeaderTargetMethod, http.MethodGet)
	return request
}

func TestRelayStreamsResponseAndKeepsUpstreamMetadataOutOfBrowserHeaders(t *testing.T) {
	var observed *http.Request
	relay, token := newTestRelay(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		observed = request
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header: http.Header{
				"Content-Type":    {"text/html; charset=utf-8"},
				"Set-Cookie":      {"session=target-secret; Secure"},
				"X-Frame-Options": {"DENY"},
			},
			Body: io.NopCloser(strings.NewReader("<h1>ok</h1>")),
		}, nil
	}))
	pairs, _ := json.Marshal([]HeaderPair{{"Accept-Language", "zh-CN"}, {"Cookie", "target=1"}})
	request := relayRequest(t, token, nil)
	request.Header.Set(HeaderTargetHeaders, base64.RawURLEncoding.EncodeToString(pairs))
	recorder := httptest.NewRecorder()
	relay.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || recorder.Body.String() != "<h1>ok</h1>" {
		t.Fatalf("response = %d %q", recorder.Code, recorder.Body.String())
	}
	if observed == nil || observed.URL.String() != "https://example.com/page" ||
		observed.Header.Get("Cookie") != "target=1" {
		t.Fatalf("upstream request = %#v", observed)
	}
	if recorder.Header().Get("Set-Cookie") != "" || recorder.Header().Get("X-Frame-Options") != "" {
		t.Fatalf("upstream headers escaped metadata: %#v", recorder.Header())
	}
	if recorder.Header().Get(HeaderUpstreamStatus) != "201" {
		t.Fatalf("upstream status metadata = %q", recorder.Header().Get(HeaderUpstreamStatus))
	}
	encoded := recorder.Header().Get(HeaderUpstreamHeaders)
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	var responsePairs []HeaderPair
	if json.Unmarshal(payload, &responsePairs) != nil || len(responsePairs) != 3 {
		t.Fatalf("response metadata = %s", payload)
	}
}

func TestRelayRejectsOriginTokenTargetAndHopByHopHeaders(t *testing.T) {
	relay, token := newTestRelay(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("upstream must not be called")
		return nil, nil
	}))

	tests := []struct {
		name   string
		mutate func(*http.Request)
		status int
	}{
		{name: "origin", mutate: func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") }, status: http.StatusForbidden},
		{name: "panel origin", mutate: func(r *http.Request) { r.Header.Set("Origin", "https://panel.example.com") }, status: http.StatusForbidden},
		{name: "token", mutate: func(r *http.Request) { r.Header.Set("Authorization", "Bearer broken") }, status: http.StatusUnauthorized},
		{name: "private target", mutate: func(r *http.Request) { r.Header.Set(HeaderTargetURL, "http://127.0.0.1/") }, status: http.StatusBadRequest},
		{name: "method", mutate: func(r *http.Request) { r.Header.Set(HeaderTargetMethod, http.MethodConnect) }, status: http.StatusBadRequest},
		{name: "host header", mutate: func(r *http.Request) {
			pairs, _ := json.Marshal([]HeaderPair{{"Host", "evil.example"}})
			r.Header.Set(HeaderTargetHeaders, base64.RawURLEncoding.EncodeToString(pairs))
		}, status: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := relayRequest(t, token, nil)
			test.mutate(request)
			recorder := httptest.NewRecorder()
			relay.Handler().ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d", recorder.Code, test.status)
			}
		})
	}
}

func TestRelayCapsStreamingRequestBody(t *testing.T) {
	relay, token := newTestRelay(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		_, err := io.ReadAll(request.Body)
		return nil, err
	}))
	relay.config.MaxRequestBytes = 4
	request := relayRequest(t, token, bytes.NewBufferString("12345"))
	request.ContentLength = -1
	recorder := httptest.NewRecorder()
	relay.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestRelayPreflightIsNarrowAndHealthDoesNotExposeSecrets(t *testing.T) {
	relay, _ := newTestRelay(t, nil)
	preflight := httptest.NewRequest(http.MethodOptions, "https://relay.example.com/v1/fetch", nil)
	preflight.Header.Set("Origin", "https://relay.example.com")
	response := httptest.NewRecorder()
	relay.Handler().ServeHTTP(response, preflight)
	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != "https://relay.example.com" {
		t.Fatalf("preflight = %d %#v", response.Code, response.Header())
	}

	health := httptest.NewRecorder()
	relay.Handler().ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK || strings.Contains(health.Body.String(), "secret") {
		t.Fatalf("health response = %d %q", health.Code, health.Body.String())
	}
}

func TestKernelAssetsAreBoundToThePanelOrigin(t *testing.T) {
	relay, _ := newTestRelay(t, nil)

	page := httptest.NewRecorder()
	relay.Handler().ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/kernel/", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), `data-panel-origin="https://panel.example.com"`) {
		t.Fatalf("kernel page = %d %q", page.Code, page.Body.String())
	}
	policy := page.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "frame-ancestors https://panel.example.com") ||
		!strings.Contains(policy, "connect-src 'self'") {
		t.Fatalf("kernel CSP = %q", policy)
	}
	if strings.Contains(page.Body.String(), "signed-token") || strings.Contains(page.Body.String(), "secret") {
		t.Fatalf("kernel page exposes credentials: %q", page.Body.String())
	}

	for _, asset := range []struct {
		path        string
		contentType string
	}{
		{path: "/kernel/kernel.js", contentType: "text/javascript; charset=utf-8"},
		{path: "/kernel/kernel.css", contentType: "text/css; charset=utf-8"},
	} {
		response := httptest.NewRecorder()
		relay.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, asset.path, nil))
		if response.Code != http.StatusOK || response.Header().Get("Content-Type") != asset.contentType || response.Body.Len() == 0 {
			t.Fatalf("asset %s = %d %q (%d bytes)", asset.path, response.Code, response.Header().Get("Content-Type"), response.Body.Len())
		}
	}
}

func TestRelayCanonicalizesOriginsBeforeIsolationChecks(t *testing.T) {
	tokens, err := NewTokenCodec([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	policy := NewTargetPolicy(staticResolver{"example.com": {netip.MustParseAddr("93.184.216.34")}})
	relay, err := NewRelay(RelayConfig{
		AllowedOrigin: "https://PANEL.example.com:443/",
		RelayOrigin:   "https://RELAY.example.com:443/",
	}, tokens, policy, NewLimiter(4, 2))
	if err != nil {
		t.Fatal(err)
	}
	if relay.config.AllowedOrigin != "https://panel.example.com" || relay.config.RelayOrigin != "https://relay.example.com" {
		t.Fatalf("origins = %q, %q", relay.config.AllowedOrigin, relay.config.RelayOrigin)
	}
	if _, err := NewRelay(RelayConfig{
		AllowedOrigin: "https://panel.example.com",
		RelayOrigin:   "https://PANEL.example.com:443/",
	}, tokens, policy, NewLimiter(4, 2)); err == nil {
		t.Fatal("equivalent panel and relay origins were accepted")
	}
	if _, err := NewRelay(RelayConfig{
		AllowedOrigin: "https://panel.example.com;script-src",
		RelayOrigin:   "https://relay.example.com",
	}, tokens, policy, NewLimiter(4, 2)); err == nil {
		t.Fatal("origin with a CSP delimiter was accepted")
	}
}

func TestIdleReadCloserClosesAStalledBody(t *testing.T) {
	reader, writer := io.Pipe()
	idle := newIdleReadCloser(reader, 10*time.Millisecond)
	done := make(chan error, 1)
	go func() {
		buffer := make([]byte, 1)
		_, err := idle.Read(buffer)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("stalled read returned without an error")
		}
	case <-time.After(time.Second):
		t.Fatal("stalled read did not close")
	}
	_ = writer.Close()
}

func TestCopyRelayResponseRejectsShortWrites(t *testing.T) {
	writer := &shortResponseWriter{header: make(http.Header)}
	err := copyRelayResponse(writer, strings.NewReader("payload"), make([]byte, 32), time.Second)
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("copy error = %v", err)
	}
}

func BenchmarkLimiterAcquireRelease(b *testing.B) {
	limiter := NewLimiter(24, 6)
	for b.Loop() {
		release, ok := limiter.Acquire("session")
		if !ok {
			b.Fatal("acquire rejected")
		}
		release()
	}
}

func BenchmarkRelayStream64KiB(b *testing.B) {
	payload := bytes.Repeat([]byte("kpanel-browser-core\n"), 3277)[:64<<10]
	relay, token := newTestRelay(b, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"text/html; charset=utf-8"}},
			Body:       io.NopCloser(bytes.NewReader(payload)),
		}, nil
	}))
	handler := relay.Handler()
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()
	for b.Loop() {
		request := relayRequest(b, token, nil)
		response := &discardResponseWriter{header: make(http.Header)}
		handler.ServeHTTP(response, request)
		if response.status != http.StatusOK || response.bytes != len(payload) {
			b.Fatalf("response = %d (%d bytes)", response.status, response.bytes)
		}
	}
}
