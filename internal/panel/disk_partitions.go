package panel

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

const diskPartitionAgentTimeout = 60 * time.Second

func (s *Server) handleDiskPartitionAction(w http.ResponseWriter, r *http.Request) {
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
	var input contract.DiskPartitionActionRequest
	if err := s.decodeJSON(w, r, &input); err != nil {
		return
	}
	if field, detail := contract.ValidateDiskPartitionAction(&input); field != "" {
		s.writeValidationProblem(w, r, field, detail)
		return
	}
	body, err := json.Marshal(input)
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, "request_encoding_failed", "Request encoding failed", "")
		return
	}
	action := "system.disk-partition." + input.Action
	targetID := opaqueIDPrefix(input.DeviceID)
	change := diskPartitionAuditChange(input)
	if err := s.audit(r, session.User.ID, action, "disk-partition", targetID, "intent", change); err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "audit_unavailable", "Audit storage unavailable", "")
		return
	}
	// Keep the bounded Agent submission alive after the browser disconnects;
	// the Agent persists before launching its independent systemd worker.
	agentContext, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), diskPartitionAgentTimeout)
	defer cancel()
	response, err := s.hostOps.Do(agentContext, http.MethodPost, "/v1/system/disk-partition-actions", "", requestID(r), body)
	if err != nil {
		_ = s.audit(r, session.User.ID, action, "disk-partition", targetID, "unknown", change)
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Agent unavailable", "")
		return
	}
	result := "failure"
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		result = "success"
	}
	_ = s.audit(r, session.User.ID, action, "disk-partition", targetID, result, change)
	s.writeAgentResponse(w, r, response)
}

func diskPartitionAuditChange(input contract.DiskPartitionActionRequest) map[string]any {
	change := map[string]any{
		"action":                    input.Action,
		"deviceIdPrefix":            opaqueIDPrefix(input.DeviceID),
		"resourceVersionPrefix":     opaqueIDPrefix(input.ExpectedResourceVersion),
		"mountPointProvided":        input.MountPoint != "",
		"persistProvided":           input.Persist != nil,
		"removePersistenceProvided": input.RemovePersistence != nil,
		"filesystemProvided":        input.Filesystem != "",
	}
	if input.MountPoint != "" {
		change["mountPointHash"], change["mountPointLength"] = auditValueMetadata(input.MountPoint)
	}
	if input.Persist != nil {
		change["persist"] = *input.Persist
	}
	if input.RemovePersistence != nil {
		change["removePersistence"] = *input.RemovePersistence
	}
	if input.Filesystem != "" {
		change["filesystem"] = input.Filesystem
	}
	return change
}

func opaqueIDPrefix(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return value
}
