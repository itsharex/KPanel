package panel

import (
	"encoding/json"
	"net"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

var panelHostnamePattern = regexp.MustCompile(
	`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*$`,
)

func (s *Server) handleSystemAction(w http.ResponseWriter, r *http.Request) {
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
	var input contract.SystemActionRequest
	if err := s.decodeJSON(w, r, &input); err != nil {
		return
	}
	if field, detail := validateSystemAction(&input); field != "" {
		s.writeValidationProblem(w, r, field, detail)
		return
	}
	body, err := json.Marshal(input)
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, "request_encoding_failed", "Request encoding failed", "")
		return
	}
	action := "system." + input.Action
	change := systemActionAuditChange(input)
	if err := s.audit(r, session.User.ID, action, "system", input.Action, "intent", change); err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "audit_unavailable", "Audit storage unavailable", "")
		return
	}
	response, err := s.hostOps.Do(r.Context(), http.MethodPost, "/v1/system/actions", "", requestID(r), body)
	if err != nil {
		_ = s.audit(r, session.User.ID, action, "system", input.Action, "failure", change)
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Agent unavailable", "")
		return
	}
	result := "failure"
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		result = "success"
	}
	_ = s.audit(r, session.User.ID, action, "system", input.Action, result, change)
	s.writeAgentResponse(w, r, response)
}

func validateSystemAction(input *contract.SystemActionRequest) (string, string) {
	if input.Action == "" {
		return "action", "action is required"
	}
	switch input.Action {
	case "hostname":
		input.Hostname = strings.ToLower(strings.TrimSpace(input.Hostname))
		if len(input.Hostname) > 253 || !panelHostnamePattern.MatchString(input.Hostname) {
			return "hostname", "hostname is invalid"
		}
	case "ssh-port":
		if input.Port == 0 {
			return "port", "port must be between 1 and 65535"
		}
	case "ssh-defense":
		if input.Enabled == nil {
			return "enabled", "enabled is required"
		}
	case "dns":
		if len(input.Servers) < 1 {
			return "servers", "one to four DNS servers are required"
		}
		for index, server := range input.Servers {
			server = strings.TrimSpace(server)
			if net.ParseIP(server) == nil {
				return "servers", "every DNS server must be an IP address"
			}
			input.Servers[index] = server
		}
	case "timezone":
		input.Timezone = strings.TrimSpace(input.Timezone)
		if input.Timezone == "" || len(input.Timezone) > 128 ||
			strings.Contains(input.Timezone, "..") || strings.ContainsAny(input.Timezone, "\x00\r\n") {
			return "timezone", "timezone is invalid"
		}
	case "process-signal":
		if input.PID <= 0 || input.StartTimeTicks == 0 ||
			(input.Signal != "term" && input.Signal != "kill") {
			return "process", "pid, startTimeTicks and a valid signal are required"
		}
		allowed := contract.SystemActionRequest{
			Action: input.Action, PID: input.PID,
			StartTimeTicks: input.StartTimeTicks, Signal: input.Signal,
		}
		if !reflect.DeepEqual(*input, allowed) {
			return "request", "only process identity and signal are allowed"
		}
	case "swap":
		if input.SwapSizeMiB < 0 {
			return "swapSizeMiB", "swapSizeMiB must be zero or a positive integer"
		}
	case "mirror":
		switch input.MirrorPreset {
		case "cn-default", "cn-edu", "abroad", "smart":
		default:
			return "mirrorPreset", "mirrorPreset must be cn-default, cn-edu, abroad, or smart"
		}
	case "ip-preference":
		if input.Preference != "ipv4" && input.Preference != "system_default" {
			return "preference", "preference must be ipv4 or system_default"
		}
	case "kernel-tuning":
		switch input.Profile {
		case "high", "balanced", "web", "stream", "game", "off":
		default:
			return "profile", "profile must be high, balanced, web, stream, game, or off"
		}
	case "bbr":
		if input.Enabled == nil {
			return "enabled", "enabled is required"
		}
	case "bbrv3":
		if input.MaintenancePolicy != "install" && input.MaintenancePolicy != "update" &&
			input.MaintenancePolicy != "uninstall" {
			return "maintenancePolicy", "maintenancePolicy must be install, update, or uninstall"
		}
		allowed := contract.SystemActionRequest{
			Action:            input.Action,
			MaintenancePolicy: input.MaintenancePolicy,
		}
		if !reflect.DeepEqual(*input, allowed) {
			return "request", "only action and maintenancePolicy are allowed for bbrv3"
		}
	case "update":
		if input.MaintenancePolicy != "full" {
			return "maintenancePolicy", "maintenancePolicy must be full"
		}
	case "cleanup":
		if input.MaintenancePolicy != "cache" && input.MaintenancePolicy != "standard" {
			return "maintenancePolicy", "maintenancePolicy must be cache or standard"
		}
	case "reboot":
		if input.Hostname != "" || input.Port != 0 || len(input.Servers) != 0 ||
			input.Timezone != "" || input.SwapSizeMiB != 0 || input.MirrorPreset != "" ||
			input.Preference != "" || input.Profile != "" || input.MaintenancePolicy != "" ||
			input.Enabled != nil {
			return "request", "only action is allowed for reboot"
		}
	default:
		return "action", "unsupported system action"
	}
	return "", ""
}

func systemActionAuditChange(input contract.SystemActionRequest) map[string]any {
	change := map[string]any{"action": input.Action}
	switch input.Action {
	case "hostname":
		change["hostname"] = input.Hostname
	case "ssh-port":
		change["port"] = input.Port
	case "ssh-defense":
		change["enabled"] = input.Enabled != nil && *input.Enabled
	case "dns":
		change["servers"] = input.Servers
	case "timezone":
		change["timezone"] = input.Timezone
	case "process-signal":
		change["pid"] = input.PID
		change["startTimeTicks"] = input.StartTimeTicks
		change["signal"] = input.Signal
	case "swap":
		change["swapSizeMiB"] = input.SwapSizeMiB
	case "mirror":
		change["mirrorPreset"] = input.MirrorPreset
	case "ip-preference":
		change["preference"] = input.Preference
	case "kernel-tuning":
		change["profile"] = input.Profile
	case "bbr":
		change["enabled"] = input.Enabled != nil && *input.Enabled
	case "update", "cleanup", "bbrv3":
		change["maintenancePolicy"] = input.MaintenancePolicy
	case "reboot":
		change["requested"] = true
	}
	return change
}
