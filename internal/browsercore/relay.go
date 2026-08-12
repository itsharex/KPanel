package browsercore

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	HeaderTargetURL       = "X-KPanel-Browser-Target-URL"
	HeaderTargetMethod    = "X-KPanel-Browser-Target-Method"
	HeaderTargetHeaders   = "X-KPanel-Browser-Target-Headers"
	HeaderUpstreamStatus  = "X-KPanel-Browser-Upstream-Status"
	HeaderUpstreamHeaders = "X-KPanel-Browser-Upstream-Headers"
	HeaderMetadataCut     = "X-KPanel-Browser-Metadata-Truncated"

	defaultMaxRequestBytes   = int64(16 << 20)
	defaultMaxHeaderMetadata = 32 << 10
	defaultBodyIdleTimeout   = 30 * time.Second
)

var relayBufferPool = sync.Pool{New: func() any { return make([]byte, 32<<10) }}

type HeaderPair [2]string

type RelayConfig struct {
	AllowedOrigin   string
	RelayOrigin     string
	MaxRequestBytes int64
	BodyIdleTimeout time.Duration
}

type Relay struct {
	config  RelayConfig
	tokens  *TokenCodec
	policy  *TargetPolicy
	limiter *Limiter
	client  *http.Client
}

func NewRelay(config RelayConfig, tokens *TokenCodec, policy *TargetPolicy, limiter *Limiter) (*Relay, error) {
	if tokens == nil || policy == nil || limiter == nil {
		return nil, errors.New("browser relay dependencies are required")
	}
	allowedOrigin, err := NormalizeOrigin(config.AllowedOrigin)
	if err != nil {
		return nil, err
	}
	config.AllowedOrigin = allowedOrigin
	relayOrigin, err := NormalizeOrigin(config.RelayOrigin)
	if err != nil {
		return nil, err
	}
	if !SupportsServiceWorkerOrigin(allowedOrigin) || !SupportsServiceWorkerOrigin(relayOrigin) {
		return nil, errors.New("browser relay origins must support Service Worker secure contexts")
	}
	if relayOrigin == allowedOrigin {
		return nil, errors.New("browser relay must use an isolated origin")
	}
	config.RelayOrigin = relayOrigin
	if config.MaxRequestBytes <= 0 {
		config.MaxRequestBytes = defaultMaxRequestBytes
	}
	if config.MaxRequestBytes > 64<<20 {
		return nil, errors.New("browser relay request limit must not exceed 64 MiB")
	}
	if config.BodyIdleTimeout <= 0 {
		config.BodyIdleTimeout = defaultBodyIdleTimeout
	}
	if config.BodyIdleTimeout < 5*time.Second || config.BodyIdleTimeout > 5*time.Minute {
		return nil, errors.New("browser relay body idle timeout must be between 5 seconds and 5 minutes")
	}
	return &Relay{
		config:  config,
		tokens:  tokens,
		policy:  policy,
		limiter: limiter,
		client: &http.Client{
			Transport: NewSafeTransport(policy),
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (r *Relay) Handler() http.Handler {
	mux := http.NewServeMux()
	r.registerKernel(mux)
	mux.HandleFunc("GET /healthz", r.handleHealth)
	mux.HandleFunc("OPTIONS /v1/fetch", r.handlePreflight)
	mux.HandleFunc("POST /v1/fetch", r.handleFetch)
	return mux
}

func (r *Relay) CloseIdleConnections() {
	if transport, ok := r.client.Transport.(interface{ CloseIdleConnections() }); ok {
		transport.CloseIdleConnections()
	}
}

func (r *Relay) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"ok":true,"engine":"kpanel-browser-core","version":1}`)
}

func (r *Relay) handlePreflight(w http.ResponseWriter, request *http.Request) {
	if !r.allowOrigin(w, request) {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", strings.Join([]string{
		"Authorization", "Content-Type", HeaderTargetURL, HeaderTargetMethod, HeaderTargetHeaders,
	}, ", "))
	w.Header().Set("Access-Control-Max-Age", "600")
	w.WriteHeader(http.StatusNoContent)
}

func (r *Relay) handleFetch(w http.ResponseWriter, request *http.Request) {
	if !r.allowOrigin(w, request) {
		writeRelayProblem(w, http.StatusForbidden, "origin_not_allowed")
		return
	}
	claims, err := r.authenticate(request)
	if err != nil {
		writeRelayProblem(w, http.StatusUnauthorized, "invalid_session")
		return
	}
	release, ok := r.limiter.Acquire(claims.SessionID)
	if !ok {
		w.Header().Set("Retry-After", "1")
		writeRelayProblem(w, http.StatusTooManyRequests, "relay_busy")
		return
	}
	defer release()

	resolveContext, cancelResolve := context.WithTimeout(request.Context(), defaultConnectTimeout)
	target, err := r.policy.Resolve(resolveContext, request.Header.Get(HeaderTargetURL))
	cancelResolve()
	if err != nil {
		writeRelayProblem(w, http.StatusBadRequest, "invalid_target")
		return
	}
	method := strings.ToUpper(strings.TrimSpace(request.Header.Get(HeaderTargetMethod)))
	if method == "" {
		method = http.MethodGet
	}
	if !allowedTargetMethod(method) {
		writeRelayProblem(w, http.StatusBadRequest, "invalid_method")
		return
	}
	headers, err := decodeHeaderPairs(request.Header.Get(HeaderTargetHeaders))
	if err != nil {
		writeRelayProblem(w, http.StatusBadRequest, "invalid_headers")
		return
	}
	if request.ContentLength > r.config.MaxRequestBytes {
		writeRelayProblem(w, http.StatusRequestEntityTooLarge, "request_too_large")
		return
	}

	body := http.MaxBytesReader(w, request.Body, r.config.MaxRequestBytes)
	defer body.Close()
	var upstreamBody io.Reader = body
	if request.ContentLength == 0 {
		upstreamBody = http.NoBody
	}
	upstream, err := http.NewRequestWithContext(request.Context(), method, target.URL.String(), upstreamBody)
	if err != nil {
		writeRelayProblem(w, http.StatusBadRequest, "invalid_request")
		return
	}
	upstream.Header = headers
	upstream.Host = target.URL.Host
	upstream.ContentLength = request.ContentLength
	response, err := r.client.Do(upstream)
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeRelayProblem(w, http.StatusRequestEntityTooLarge, "request_too_large")
			return
		}
		writeRelayProblem(w, http.StatusBadGateway, "upstream_failed")
		return
	}
	response.Body = newIdleReadCloser(response.Body, r.config.BodyIdleTimeout)
	defer response.Body.Close()

	metadata, truncated := encodeResponseHeaders(response.Header)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set(HeaderUpstreamStatus, strconv.Itoa(response.StatusCode))
	w.Header().Set(HeaderUpstreamHeaders, metadata)
	if truncated {
		w.Header().Set(HeaderMetadataCut, "1")
	}
	w.Header().Set("Access-Control-Expose-Headers", strings.Join([]string{
		HeaderUpstreamStatus, HeaderUpstreamHeaders, HeaderMetadataCut,
	}, ", "))
	w.WriteHeader(http.StatusOK)
	buffer := relayBufferPool.Get().([]byte)
	defer relayBufferPool.Put(buffer)
	_ = copyRelayResponse(w, response.Body, buffer, r.config.BodyIdleTimeout)
}

func copyRelayResponse(w http.ResponseWriter, body io.Reader, buffer []byte, idleTimeout time.Duration) error {
	controller := http.NewResponseController(w)
	defer func() { _ = controller.SetWriteDeadline(time.Time{}) }()
	emptyReads := 0
	for {
		read, readErr := body.Read(buffer)
		if read > 0 {
			emptyReads = 0
			_ = controller.SetWriteDeadline(time.Now().Add(idleTimeout))
			written, writeErr := w.Write(buffer[:read])
			if writeErr != nil {
				return writeErr
			}
			if written != read {
				return io.ErrShortWrite
			}
		} else if readErr == nil {
			emptyReads++
			if emptyReads >= 100 {
				return io.ErrNoProgress
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			return readErr
		}
	}
}

func (r *Relay) authenticate(request *http.Request) (Claims, error) {
	value := request.Header.Get("Authorization")
	if !strings.HasPrefix(value, "Bearer ") || len(value) <= len("Bearer ") {
		return Claims{}, ErrInvalidToken
	}
	return r.tokens.Verify(strings.TrimPrefix(value, "Bearer "))
}

func (r *Relay) allowOrigin(w http.ResponseWriter, request *http.Request) bool {
	origin := request.Header.Get("Origin")
	if origin == "" || origin != r.config.RelayOrigin {
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Add("Vary", "Origin")
	return true
}

func NormalizeOrigin(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if strings.HasSuffix(value, "/") {
		value = strings.TrimSuffix(value, "/")
	}
	target, err := url.Parse(value)
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") ||
		target.User != nil || target.Path != "" || target.RawPath != "" || target.RawQuery != "" ||
		target.Fragment != "" || target.ForceQuery || target.Hostname() == "" ||
		strings.ContainsAny(target.Host, "\r\n\t ") {
		return "", errors.New("browser relay allowed origin must be an HTTP(S) origin")
	}
	port := target.Port()
	if port != "" {
		number, portErr := strconv.Atoi(port)
		if portErr != nil || number < 1 || number > 65535 {
			return "", errors.New("browser relay origin port is invalid")
		}
	}
	hostname, hostErr := normalizeHostname(target.Hostname())
	if hostErr != nil {
		return "", errors.New("browser relay origin host is invalid")
	}
	host := hostname
	if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	if port != "" && !((target.Scheme == "https" && port == "443") || (target.Scheme == "http" && port == "80")) {
		host = net.JoinHostPort(hostname, port)
	}
	return strings.ToLower(target.Scheme) + "://" + host, nil
}

func allowedTargetMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodOptions:
		return true
	default:
		return false
	}
}

func decodeHeaderPairs(encoded string) (http.Header, error) {
	headers := make(http.Header)
	if encoded == "" {
		return headers, nil
	}
	if len(encoded) > defaultMaxHeaderMetadata {
		return nil, errors.New("header metadata too large")
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(payload) > defaultMaxHeaderMetadata {
		return nil, errors.New("invalid header metadata")
	}
	var pairs []HeaderPair
	if json.Unmarshal(payload, &pairs) != nil || len(pairs) > 128 {
		return nil, errors.New("invalid header metadata")
	}
	for _, pair := range pairs {
		name := http.CanonicalHeaderKey(pair[0])
		if !validHeaderName(name) || !validHeaderValue(pair[1]) ||
			hopByHopHeader(name) || strings.EqualFold(name, "Host") || len(pair[1]) > 8<<10 {
			return nil, errors.New("invalid target header")
		}
		headers.Add(name, pair[1])
	}
	return headers, nil
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for index := 0; index < len(name); index++ {
		character := name[index]
		if !((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(character))) {
			return false
		}
	}
	return true
}

func validHeaderValue(value string) bool {
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character == '\t' {
			continue
		}
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

func encodeResponseHeaders(headers http.Header) (string, bool) {
	pairs := make([]HeaderPair, 0, len(headers))
	size := 0
	truncated := false
	for name, values := range headers {
		if hopByHopHeader(name) {
			continue
		}
		for _, value := range values {
			if len(pairs) >= 128 || size+len(name)+len(value) > defaultMaxHeaderMetadata {
				truncated = true
				continue
			}
			pairs = append(pairs, HeaderPair{name, value})
			size += len(name) + len(value)
		}
	}
	payload, _ := json.Marshal(pairs)
	return base64.RawURLEncoding.EncodeToString(payload), truncated
}

func hopByHopHeader(name string) bool {
	switch http.CanonicalHeaderKey(name) {
	case "Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "Te",
		"Trailer", "Transfer-Encoding", "Upgrade":
		return true
	default:
		return false
	}
}

func writeRelayProblem(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": status,
		"code":   code,
	})
}

type idleReadCloser struct {
	reader  io.ReadCloser
	timeout time.Duration
	mu      sync.Mutex
	timer   *time.Timer
	closed  bool
}

func newIdleReadCloser(reader io.ReadCloser, timeout time.Duration) io.ReadCloser {
	idle := &idleReadCloser{reader: reader, timeout: timeout}
	idle.mu.Lock()
	idle.timer = time.AfterFunc(timeout, func() { _ = idle.Close() })
	idle.mu.Unlock()
	return idle
}

func (r *idleReadCloser) Read(buffer []byte) (int, error) {
	n, err := r.reader.Read(buffer)
	r.mu.Lock()
	if !r.closed && err == nil {
		r.timer.Reset(r.timeout)
	}
	r.mu.Unlock()
	return n, err
}

func (r *idleReadCloser) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	r.timer.Stop()
	r.mu.Unlock()
	return r.reader.Close()
}
