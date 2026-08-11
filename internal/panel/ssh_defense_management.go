package panel

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func (s *Server) handleSSHDefenseAction(w http.ResponseWriter, r *http.Request) {
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
	var input contract.SSHDefenseActionRequest
	if err := s.decodeJSON(w, r, &input); err != nil {
		return
	}
	if field, detail := contract.ValidateSSHDefenseAction(&input); field != "" {
		s.writeValidationProblem(w, r, field, detail)
		return
	}
	body, err := json.Marshal(input)
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, "request_encoding_failed", "Request encoding failed", "")
		return
	}
	action := "system.ssh-defense." + input.Action
	change := sshDefenseAuditChange(input)
	if err := s.audit(r, session.User.ID, action, "ssh-defense", input.Address, "intent", change); err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "audit_unavailable", "Audit storage unavailable", "")
		return
	}
	agentContext, cancelAgent := context.WithTimeout(context.WithoutCancel(r.Context()), systemResourceAgentTimeout)
	defer cancelAgent()
	response, err := s.hostOps.Do(agentContext, http.MethodPost, "/v1/system/ssh-defense/actions", "", requestID(r), body)
	if err != nil {
		_ = s.audit(r, session.User.ID, action, "ssh-defense", input.Address, "unknown", change)
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Agent unavailable", "")
		return
	}
	result := "failure"
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		result = "success"
	}
	_ = s.audit(r, session.User.ID, action, "ssh-defense", input.Address, result, change)
	s.writeAgentResponse(w, r, response)
}

func sshDefenseAuditChange(input contract.SSHDefenseActionRequest) map[string]any {
	change := map[string]any{"action": input.Action}
	if input.Profile != "" {
		change["profile"] = input.Profile
	}
	if input.Address != "" {
		change["address"] = input.Address
	}
	return change
}
