package browsercore

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
		!strings.Contains(policy, "connect-src 'self'") ||
		!strings.Contains(policy, "worker-src 'self' blob:") ||
		!strings.Contains(policy, "'unsafe-eval'") ||
		!strings.Contains(policy, "'wasm-unsafe-eval'") {
		t.Fatalf("kernel CSP = %q", policy)
	}
	if strings.Contains(page.Body.String(), "signed-token") || strings.Contains(page.Body.String(), "secret") {
		t.Fatalf("kernel page exposes credentials: %q", page.Body.String())
	}
	if page.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("kernel page cache policy = %q", page.Header().Get("Cache-Control"))
	}
	if !strings.Contains(page.Body.String(), "/scramjet/scramjet.js") ||
		!strings.Contains(page.Body.String(), "/controller/controller.api.js") ||
		!strings.Contains(page.Body.String(), "allow-scripts allow-forms allow-modals allow-popups allow-downloads") {
		t.Fatalf("kernel page is missing the context-preserving runtime: %q", page.Body.String())
	}

	for _, asset := range []struct {
		path        string
		contentType string
		cache       string
	}{
		{path: "/kernel/kernel.js", contentType: "text/javascript; charset=utf-8", cache: "no-store"},
		{path: "/kernel/kernel.css", contentType: "text/css; charset=utf-8", cache: "no-store"},
		{path: "/kernel/runtime/v3/transport.mjs", contentType: "text/javascript; charset=utf-8", cache: "public, max-age=31536000, immutable"},
		{path: "/scramjet/scramjet.js", contentType: "text/javascript; charset=utf-8", cache: "public, max-age=31536000, immutable"},
		{path: "/scramjet/scramjet.wasm", contentType: "application/wasm", cache: "public, max-age=31536000, immutable"},
		{path: "/controller/controller.api.js", contentType: "text/javascript; charset=utf-8", cache: "public, max-age=31536000, immutable"},
		{path: "/controller/controller.inject.js", contentType: "text/javascript; charset=utf-8", cache: "public, max-age=31536000, immutable"},
		{path: "/controller/controller.sw.js", contentType: "text/javascript; charset=utf-8", cache: "public, max-age=31536000, immutable"},
	} {
		response := httptest.NewRecorder()
		relay.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, asset.path, nil))
		if response.Code != http.StatusOK || response.Header().Get("Content-Type") != asset.contentType || response.Body.Len() == 0 {
			t.Fatalf("asset %s = %d %q (%d bytes)", asset.path, response.Code, response.Header().Get("Content-Type"), response.Body.Len())
		}
		if response.Header().Get("Cache-Control") != asset.cache {
			t.Fatalf("asset %s cache policy = %q", asset.path, response.Header().Get("Cache-Control"))
		}
	}

	worker := httptest.NewRecorder()
	relay.Handler().ServeHTTP(worker, httptest.NewRequest(http.MethodGet, "/kernel/runtime/v3/sw.js", nil))
	if worker.Code != http.StatusOK || worker.Header().Get("Cache-Control") != "no-cache" ||
		worker.Header().Get("Service-Worker-Allowed") != "/" ||
		!strings.Contains(worker.Body.String(), "$scramjetController.shouldRoute") {
		t.Fatalf("service worker = %d %#v %q", worker.Code, worker.Header(), worker.Body.String())
	}
}

func TestKernelPreservesBrowserContextsInsteadOfSanitizingPages(t *testing.T) {
	script := string(kernelJS)
	for _, required := range []string{
		"new Controller",
		"new KPanelRelayTransport",
		"controller.createFrame(viewport)",
		"decodedFrameURL",
		"window.setInterval(syncFrameState, 500)",
		"frame.go(target)",
		"navigator.serviceWorker.register",
		"worker.scriptURL !== runtimeWorkerURL",
		"registration.active.state === 'activated'",
		"网页重写 Service Worker 激活超时",
		"网页重写控制器握手超时",
		"document.body?.childElementCount > 0",
		"transport.setToken(token)",
		"const navigationTimeoutMs = 45_000",
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("kernel is missing browser compatibility behavior %q", required)
		}
	}
	for _, forbidden := range []string{"allowedTags", "discardedTags", "sanitizeHTML", "viewport.srcdoc"} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("kernel still contains destructive sanitizer behavior %q", forbidden)
		}
	}

	transport := string(kernelTransport)
	for _, required := range []string{
		"export default class KPanelRelayTransport",
		"Authorization: `Bearer ${this.token}`",
		"body: response.body",
		"this.sessionChannel.postMessage({ type: 'session-expired' })",
		"blockedRequestHeaders",
	} {
		if !strings.Contains(transport, required) {
			t.Fatalf("kernel transport is missing relay behavior %q", required)
		}
	}
}

func TestPinnedV2RuntimeAssetsMatchReviewedDigests(t *testing.T) {
	for _, asset := range []struct {
		name   string
		bytes  []byte
		sha256 string
	}{
		{name: "scramjet.js", bytes: scramjetV2JS, sha256: "e116b8adbdae9e9d9bee6abd8370990faa2615796c2e8fc0b7b8942537c0d92e"},
		{name: "scramjet.wasm", bytes: scramjetV2WASM, sha256: "c8740c340a506686d9e46feb82894ab710cf57cd20564d001d0aef04310661e8"},
		{name: "controller.api.js", bytes: scramjetControllerAPI, sha256: "9bc38bc6dce704ebf71fd7c418151bf7192b0a6c10e15f1037c77e146c163c91"},
		{name: "controller.inject.js", bytes: scramjetControllerInject, sha256: "4744bb39bcfcdbe2baa43f26c760abc5c6248475da5086b2be0eefcd1724fb87"},
		{name: "controller.sw.js", bytes: scramjetControllerWorker, sha256: "805309725993d31a8e540c20c575dcc176a86e7fdfc6b9d69ccf4ee8f94c736d"},
	} {
		digest := sha256.Sum256(asset.bytes)
		if got := fmt.Sprintf("%x", digest); got != asset.sha256 {
			t.Fatalf("%s digest = %s, want %s", asset.name, got, asset.sha256)
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
	if _, err := NewRelay(RelayConfig{
		AllowedOrigin: "http://198.51.100.10:8080",
		RelayOrigin:   "http://198.51.100.10:8081",
	}, tokens, policy, NewLimiter(4, 2)); err == nil {
		t.Fatal("public HTTP origins without Service Worker support were accepted")
	}
	if _, err := NewRelay(RelayConfig{
		AllowedOrigin: "http://localhost:8080",
		RelayOrigin:   "http://127.0.0.1:8081",
	}, tokens, policy, NewLimiter(4, 2)); err != nil {
		t.Fatalf("loopback HTTP origins were rejected: %v", err)
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
