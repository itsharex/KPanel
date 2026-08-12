package panel

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/browsercore"
)

const browserRelaySessionTTL = 10 * time.Minute

type browserRelaySessionResponse struct {
	Mode      string    `json:"mode"`
	RelayURL  string    `json:"relayUrl"`
	Token     string    `json:"token"`
	SessionID string    `json:"sessionId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (s *Server) handleBrowserSessionCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		s.writeProblem(w, r, http.StatusBadRequest, "invalid_browser_session_request", "Invalid browser session request", "")
		return
	}
	if !s.checkOrigin(w, r) {
		return
	}
	_, session, ok := s.requireSession(w, r)
	if !ok || !s.checkCSRF(w, r, session) {
		return
	}
	if !browsercore.RuntimeModeEnabled(s.config.BrowserMode) {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "browser_beta_disabled", "Browser Beta is disabled", "Enable the explicit browser beta mode to use this experimental runtime.")
		return
	}
	if code := s.validateBrowserSessionOrigin(r); code != "" {
		if code == "browser_origin_mismatch" {
			s.writeProblem(w, r, http.StatusMisdirectedRequest, code, "Browser origin does not match the configured public URL", "Use the configured Panel HTTPS origin to start Browser Beta.")
		} else {
			s.writeProblem(w, r, http.StatusServiceUnavailable, code, "Browser Beta requires a secure context", "Use HTTPS for both Panel and Browser Relay; HTTP is supported only on loopback origins.")
		}
		return
	}
	if s.browserTokens == nil || s.config.BrowserRelayURL == "" {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "browser_core_unavailable", "Browser core is not configured", "")
		return
	}
	token, claims, err := s.browserTokens.Issue(session.User.ID, browserRelaySessionTTL)
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, "browser_session_failed", "Browser session could not be created", "")
		return
	}
	s.writeJSON(w, http.StatusCreated, browserRelaySessionResponse{
		Mode:      "beta",
		RelayURL:  s.config.BrowserRelayURL,
		Token:     token,
		SessionID: claims.SessionID,
		ExpiresAt: time.Unix(claims.ExpiresAt, 0).UTC(),
	})
}

func (s *Server) validateBrowserSessionOrigin(r *http.Request) string {
	configured, err := browsercore.NormalizeOrigin(s.config.PublicURL)
	if err != nil || !browsercore.SupportsServiceWorkerOrigin(configured) {
		return "browser_secure_context_required"
	}
	if actual, ok := s.requestHTTPSOrigin(r); ok {
		actual, err = browsercore.NormalizeOrigin(actual)
		if err != nil || actual != configured {
			return "browser_origin_mismatch"
		}
		return ""
	}

	actual, err := browsercore.NormalizeOrigin("http://" + strings.TrimSpace(r.Host))
	if err != nil || !browsercore.SupportsServiceWorkerOrigin(actual) {
		return "browser_secure_context_required"
	}
	parsedConfigured, err := url.Parse(configured)
	if err != nil || parsedConfigured.Scheme != "http" || actual != configured {
		return "browser_origin_mismatch"
	}
	requestOrigin, err := browsercore.NormalizeOrigin(r.Header.Get("Origin"))
	if err != nil || requestOrigin != configured {
		return "browser_origin_mismatch"
	}
	return ""
}
