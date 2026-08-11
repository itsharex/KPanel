package panel

import (
	"net/http"
	"time"
)

const browserRelaySessionTTL = 10 * time.Minute

type browserRelaySessionResponse struct {
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
		RelayURL:  s.config.BrowserRelayURL,
		Token:     token,
		SessionID: claims.SessionID,
		ExpiresAt: time.Unix(claims.ExpiresAt, 0).UTC(),
	})
}
