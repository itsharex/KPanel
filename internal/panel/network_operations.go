package panel

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func (s *Server) handleTrafficShutdownAction(w http.ResponseWriter, r *http.Request) {
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
	var input contract.TrafficShutdownActionRequest
	if err := s.decodeJSON(w, r, &input); err != nil {
		return
	}
	if field, detail := contract.ValidateTrafficShutdownAction(&input); field != "" {
		s.writeValidationProblem(w, r, field, detail)
		return
	}
	body, err := json.Marshal(input)
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, "request_encoding_failed", "Request encoding failed", "")
		return
	}
	action := "system.traffic-shutdown." + input.Action
	change := map[string]any{"action": input.Action}
	if input.Action == "enable" {
		change["rxThresholdGiB"] = *input.RXThresholdGiB
		change["txThresholdGiB"] = *input.TXThresholdGiB
		change["resetDay"] = *input.ResetDay
	}
	if err := s.audit(r, session.User.ID, action, "network-operation", "traffic-shutdown", "intent", change); err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "audit_unavailable", "Audit storage unavailable", "")
		return
	}
	agentContext, cancelAgent := context.WithTimeout(context.WithoutCancel(r.Context()), systemResourceAgentTimeout)
	defer cancelAgent()
	response, err := s.hostOps.Do(
		agentContext, http.MethodPost, "/v1/system/traffic-shutdown/actions", "", requestID(r), body,
	)
	if err != nil {
		_ = s.audit(r, session.User.ID, action, "network-operation", "traffic-shutdown", "failure", change)
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Agent unavailable", "")
		return
	}
	result := "failure"
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		result = "success"
	}
	_ = s.audit(r, session.User.ID, action, "network-operation", "traffic-shutdown", result, change)
	s.writeAgentResponse(w, r, response)
}
