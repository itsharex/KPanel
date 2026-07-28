package panel

import (
	"encoding/json"
	"net/http"
	"regexp"
)

var diagnosticCheckIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,47}$`)

func (s *Server) handleDiagnosticStart(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawPath != "" || r.URL.RawQuery != "" ||
		r.URL.Path != "/api/v1/diagnostic-jobs" {
		s.writeProblem(w, r, http.StatusNotFound, "route_not_found", "Route not found", "")
		return
	}
	if !s.checkOrigin(w, r) {
		return
	}
	_, session, ok := s.requireSession(w, r)
	if !ok || !s.checkCSRF(w, r, session) {
		return
	}
	var input struct {
		CheckID string `json:"checkId"`
	}
	if err := s.decodeJSON(w, r, &input); err != nil {
		return
	}
	if !diagnosticCheckIDPattern.MatchString(input.CheckID) {
		s.writeValidationProblem(w, r, "checkId", "checkId must be a fixed diagnostic selector")
		return
	}
	body, err := json.Marshal(input)
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, "request_encoding_failed", "Request encoding failed", "")
		return
	}
	change := map[string]any{"checkId": input.CheckID}
	if err := s.audit(
		r,
		session.User.ID,
		"diagnostic.run",
		"diagnostic",
		input.CheckID,
		"intent",
		change,
	); err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "audit_unavailable", "Audit storage unavailable", "")
		return
	}
	response, err := s.agent.Do(
		r.Context(),
		http.MethodPost,
		"/v1/diagnostic-jobs",
		"",
		requestID(r),
		body,
	)
	if err != nil {
		_ = s.audit(r, session.User.ID, "diagnostic.run", "diagnostic", input.CheckID, "failure", change)
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Agent unavailable", "")
		return
	}
	result := "failure"
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		result = "success"
	}
	_ = s.audit(r, session.User.ID, "diagnostic.run", "diagnostic", input.CheckID, result, change)
	s.writeAgentResponse(w, r, response)
}
