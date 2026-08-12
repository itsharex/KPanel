package panel

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/kejilion/kejilion-panel/internal/browsercore"
)

const (
	browserReaderAssetPrefix   = "/browser-reader/"
	browserReaderTokenHeader   = "X-KPanel-Browser-Token"
	browserReaderRequestBytes  = int64(4 << 10)
	browserReaderDocumentBytes = int64(8 << 20)
	browserReaderImageBytes    = int64(2 << 20)
	browserReaderMetadataBytes = 32 << 10
)

//go:embed browser_reader.html
var browserReaderHTML []byte

//go:embed browser_reader.js
var browserReaderJS []byte

//go:embed browser_reader.css
var browserReaderCSS []byte

type browserReaderFetchInput struct {
	URL  string `json:"url"`
	Kind string `json:"kind"`
}

func (s *Server) handleBrowserReaderAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		s.writeProblem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", "")
		return
	}
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		http.NotFound(w, r)
		return
	}
	var content []byte
	var contentType string
	switch r.URL.Path {
	case browserReaderAssetPrefix, browserReaderAssetPrefix + "index.html":
		content, contentType = browserReaderHTML, "text/html; charset=utf-8"
	case browserReaderAssetPrefix + "reader.js":
		content, contentType = browserReaderJS, "text/javascript; charset=utf-8"
	case browserReaderAssetPrefix + "reader.css":
		content, contentType = browserReaderCSS, "text/css; charset=utf-8"
	default:
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.Header().Set("Content-Security-Policy", "sandbox allow-scripts; default-src 'none'; base-uri 'none'; frame-ancestors 'self'; form-action 'none'; object-src 'none'; script-src 'self'; style-src 'self'; img-src blob: data:; connect-src 'none'")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=(), fullscreen=(), autoplay=()")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(content)
	}
}

func (s *Server) handleBrowserReaderFetch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	mode, _ := browsercore.NormalizeRuntimeMode(s.config.BrowserMode)
	if mode != browsercore.RuntimeModeReader || s.browserTokens == nil || s.browserRelayClient == nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "browser_reader_unavailable", "Browser reader is unavailable", "")
		return
	}
	if r.URL.RawPath != "" || r.URL.RawQuery != "" || !s.checkOrigin(w, r) {
		return
	}
	_, session, ok := s.requireSession(w, r)
	if !ok || !s.checkCSRF(w, r, session) {
		return
	}
	readerToken := r.Header.Get(browserReaderTokenHeader)
	claims, err := s.browserTokens.Verify(readerToken)
	if err != nil || claims.Scope != browsercore.TokenScopeReader || claims.Subject != session.User.ID {
		s.writeProblem(w, r, http.StatusUnauthorized, "browser_session_expired", "Browser session is invalid or expired", "")
		return
	}
	var input browserReaderFetchInput
	if err := decodeLimitedJSON(w, r, browserReaderRequestBytes, &input); err != nil {
		return
	}
	input.URL = strings.TrimSpace(input.URL)
	if len(input.URL) == 0 || len(input.URL) > 2_048 || (input.Kind != "document" && input.Kind != "image") {
		s.writeProblem(w, r, http.StatusBadRequest, "invalid_browser_reader_request", "Invalid browser reader request", "")
		return
	}
	headerPairs := []browsercore.HeaderPair{
		{"Accept-Language", safeReaderHeaderValue(r.Header.Get("Accept-Language"), 256, "zh-CN,zh;q=0.9,en;q=0.7")},
		{"User-Agent", safeReaderHeaderValue(r.UserAgent(), 512, "KPanel-Reader/1.0")},
	}
	if input.Kind == "image" {
		headerPairs = append(headerPairs, browsercore.HeaderPair{"Accept", "image/avif,image/webp,image/png,image/jpeg,image/gif;q=0.9,*/*;q=0.1"})
	} else {
		headerPairs = append(headerPairs, browsercore.HeaderPair{"Accept", "text/html,application/xhtml+xml,text/plain,application/json;q=0.9,image/avif,image/webp,image/png,image/jpeg,image/gif;q=0.8,*/*;q=0.1"})
	}
	encodedHeaders, err := browsercore.EncodeHeaderPairs(headerPairs)
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, "browser_reader_failed", "Browser reader request failed", "")
		return
	}
	internalOrigin, err := browsercore.NormalizeOrigin(s.config.BrowserRelayInternalURL)
	if err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "browser_reader_unavailable", "Browser reader is unavailable", "")
		return
	}
	relayRequest, err := http.NewRequestWithContext(r.Context(), http.MethodPost, internalOrigin+"/v1/fetch", http.NoBody)
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, "browser_reader_failed", "Browser reader request failed", "")
		return
	}
	relayRequest.Header.Set("Authorization", "Bearer "+readerToken)
	relayRequest.Header.Set("Origin", strings.TrimRight(s.config.BrowserRelayURL, "/"))
	relayRequest.Header.Set(browsercore.HeaderTargetURL, input.URL)
	relayRequest.Header.Set(browsercore.HeaderTargetMethod, http.MethodGet)
	relayRequest.Header.Set(browsercore.HeaderTargetHeaders, encodedHeaders)
	response, err := s.browserRelayClient.Do(relayRequest)
	if err != nil {
		s.writeProblem(w, r, http.StatusBadGateway, "browser_reader_relay_unavailable", "Browser reader relay is unavailable", "")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		status := http.StatusBadGateway
		code := "browser_reader_relay_failed"
		if response.StatusCode == http.StatusUnauthorized {
			status, code = http.StatusUnauthorized, "browser_session_expired"
		} else if response.StatusCode == http.StatusTooManyRequests {
			status, code = http.StatusTooManyRequests, "browser_reader_busy"
		} else if response.StatusCode == http.StatusRequestEntityTooLarge {
			status, code = http.StatusRequestEntityTooLarge, "browser_reader_response_too_large"
		} else if response.StatusCode == http.StatusBadRequest {
			status, code = http.StatusBadRequest, "browser_target_rejected"
		}
		s.writeProblem(w, r, status, code, "Browser reader request failed", "")
		return
	}
	upstreamStatus, err := strconv.Atoi(response.Header.Get(browsercore.HeaderUpstreamStatus))
	metadata := response.Header.Get(browsercore.HeaderUpstreamHeaders)
	if err != nil || upstreamStatus < 100 || upstreamStatus > 599 || len(metadata) > browserReaderMetadataBytes {
		s.writeProblem(w, r, http.StatusBadGateway, "browser_reader_invalid_response", "Browser reader relay returned an invalid response", "")
		return
	}
	if upstreamStatus >= 300 && upstreamStatus < 400 {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set(browsercore.HeaderUpstreamStatus, strconv.Itoa(upstreamStatus))
		w.Header().Set(browsercore.HeaderUpstreamHeaders, metadata)
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusOK)
		return
	}
	limit := browserReaderDocumentBytes
	if input.Kind == "image" || browserReaderMetadataIsImage(metadata) {
		limit = browserReaderImageBytes
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		s.writeProblem(w, r, http.StatusBadGateway, "browser_reader_incomplete_response", "Browser reader response was incomplete", "")
		return
	}
	readerError := response.Trailer.Get(browsercore.HeaderReaderError)
	if readerError == "" {
		readerError = response.Header.Get(browsercore.HeaderReaderError)
	}
	if readerError != "" {
		if readerError == "response_too_large" {
			s.writeProblem(w, r, http.StatusRequestEntityTooLarge, "browser_reader_response_too_large", "Browser reader response exceeds the safety limit", "")
		} else {
			s.writeProblem(w, r, http.StatusBadGateway, "browser_reader_incomplete_response", "Browser reader response was incomplete", "")
		}
		return
	}
	if int64(len(content)) > limit {
		s.writeProblem(w, r, http.StatusRequestEntityTooLarge, "browser_reader_response_too_large", "Browser reader response exceeds the safety limit", "")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set(browsercore.HeaderUpstreamStatus, strconv.Itoa(upstreamStatus))
	w.Header().Set(browsercore.HeaderUpstreamHeaders, metadata)
	if response.Header.Get(browsercore.HeaderMetadataCut) == "1" {
		w.Header().Set(browsercore.HeaderMetadataCut, "1")
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func browserReaderMetadataIsImage(encoded string) bool {
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return false
	}
	var pairs []browsercore.HeaderPair
	if json.Unmarshal(payload, &pairs) != nil {
		return false
	}
	for _, pair := range pairs {
		if strings.EqualFold(pair[0], "Content-Type") {
			value := strings.ToLower(strings.TrimSpace(strings.SplitN(pair[1], ";", 2)[0]))
			switch value {
			case "image/avif", "image/gif", "image/jpeg", "image/png", "image/webp":
				return true
			}
		}
	}
	return false
}

func safeReaderHeaderValue(value string, limit int, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > limit || strings.ContainsAny(value, "\x00\r\n") {
		return fallback
	}
	return value
}
