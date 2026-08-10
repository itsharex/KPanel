package panel

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func (s *Server) handleAccountManagementAction(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
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
	var input contract.AccountManagementActionRequest
	if err := s.decodeJSON(w, r, &input); err != nil {
		return
	}
	if field, detail := contract.ValidateAccountManagementAction(&input); field != "" {
		s.writeValidationProblem(w, r, field, detail)
		return
	}
	body, err := json.Marshal(input)
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, "request_encoding_failed", "Request encoding failed", "")
		return
	}
	action := "system.accounts." + input.Action
	change := accountManagementAuditChange(input)
	if err := s.audit(r, session.User.ID, action, "system-account", input.Username, "intent", change); err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "audit_unavailable", "Audit storage unavailable", "")
		return
	}
	agentContext, cancelAgent := context.WithTimeout(context.WithoutCancel(r.Context()), systemResourceAgentTimeout)
	defer cancelAgent()
	response, err := s.hostOps.Do(
		agentContext, http.MethodPost, "/v1/system/account-actions", "", requestID(r), body,
	)
	if err != nil {
		_ = s.audit(r, session.User.ID, action, "system-account", input.Username, "unknown", change)
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Agent unavailable", "")
		return
	}
	result := "failure"
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		result = "success"
	}
	_ = s.audit(r, session.User.ID, action, "system-account", input.Username, result, change)
	s.writeAgentResponse(w, r, response)
}

func accountManagementAuditChange(input contract.AccountManagementActionRequest) map[string]any {
	change := map[string]any{"action": input.Action}
	if input.Username != "" {
		change["username"] = input.Username
	}
	if input.Role != "" {
		change["role"] = input.Role
	}
	if input.Credential != "" {
		change["credential"] = input.Credential
	}
	if input.KeyID != "" {
		change["keyId"] = input.KeyID
	}
	if input.PasswordAuthentication != nil {
		change["passwordAuthentication"] = *input.PasswordAuthentication
	}
	if input.RootLogin != "" {
		change["rootLogin"] = input.RootLogin
	}
	if input.RemoveHome != nil {
		change["removeHome"] = *input.RemoveHome
	}
	if input.Secret != "" {
		change["secretProvided"] = true
	}
	return change
}
