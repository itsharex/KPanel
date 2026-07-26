package panel

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

const (
	maxSiteAliases        = 20
	maxSiteDomainLength   = 253
	maxSiteUpstreamLength = 2048
	maxSiteUpstreams      = 8
)

var siteIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

type optionalString struct {
	Value string
	Set   bool
}

func (value *optionalString) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("null is not allowed")
	}
	var decoded string
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	value.Value = decoded
	value.Set = true
	return nil
}

type optionalBool struct {
	Value bool
	Set   bool
}

func (value *optionalBool) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("null is not allowed")
	}
	var decoded bool
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	value.Value = decoded
	value.Set = true
	return nil
}

type optionalStrings struct {
	Value []string
	Set   bool
}

func (value *optionalStrings) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("null is not allowed")
	}
	var decoded []string
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	value.Value = decoded
	value.Set = true
	return nil
}

type optionalInt struct {
	Value int
	Set   bool
}

func (value *optionalInt) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("null is not allowed")
	}
	var decoded int
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	value.Value = decoded
	value.Set = true
	return nil
}

type siteWriteInput struct {
	PrimaryDomain           optionalString  `json:"primaryDomain"`
	Aliases                 optionalStrings `json:"aliases"`
	Type                    optionalString  `json:"type"`
	Upstream                optionalString  `json:"upstream"`
	Upstreams               optionalStrings `json:"upstreams"`
	RedirectTarget          optionalString  `json:"redirectTarget"`
	RedirectCode            optionalInt     `json:"redirectCode"`
	PHPVersion              optionalString  `json:"phpVersion"`
	Enabled                 optionalBool    `json:"enabled"`
	ExpectedResourceVersion optionalString  `json:"expectedResourceVersion"`
}

type siteAgentPayload struct {
	PrimaryDomain           *string   `json:"primaryDomain,omitempty"`
	Aliases                 *[]string `json:"aliases,omitempty"`
	Type                    *string   `json:"type,omitempty"`
	Upstream                *string   `json:"upstream,omitempty"`
	Upstreams               *[]string `json:"upstreams,omitempty"`
	RedirectTarget          *string   `json:"redirectTarget,omitempty"`
	RedirectCode            *int      `json:"redirectCode,omitempty"`
	PHPVersion              *string   `json:"phpVersion,omitempty"`
	Enabled                 *bool     `json:"enabled,omitempty"`
	ExpectedResourceVersion *string   `json:"expectedResourceVersion,omitempty"`
}

func (s *Server) handleSiteCreate(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/sites" || r.URL.RawPath != "" || r.URL.RawQuery != "" {
		s.writeProblem(w, r, http.StatusNotFound, "route_not_found", "Route not found", "")
		return
	}
	s.handleSiteWrite(w, r, http.MethodPost, "/v1/sites", "", true)
}

func (s *Server) handleSiteUpdate(w http.ResponseWriter, r *http.Request) {
	agentPath, siteID, ok := allowedSiteUpdatePath(r.URL.Path)
	if !ok || r.URL.RawPath != "" || r.URL.RawQuery != "" {
		s.writeProblem(w, r, http.StatusNotFound, "route_not_found", "Route not found", "")
		return
	}
	s.handleSiteWrite(w, r, http.MethodPatch, agentPath, siteID, false)
}

func (s *Server) handleSiteWrite(
	w http.ResponseWriter,
	r *http.Request,
	method string,
	agentPath string,
	siteID string,
	create bool,
) {
	if !s.checkOrigin(w, r) {
		return
	}
	_, session, ok := s.requireSession(w, r)
	if !ok || !s.checkCSRF(w, r, session) {
		return
	}

	var input siteWriteInput
	if err := s.decodeJSON(w, r, &input); err != nil {
		return
	}
	if field, detail := validateSiteWriteInput(&input, create); field != "" {
		s.writeValidationProblem(w, r, field, detail)
		return
	}
	payload := input.agentPayload()
	body, err := json.Marshal(payload)
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, "request_encoding_failed", "Request encoding failed", "")
		return
	}

	action := "site.update"
	targetID := siteID
	if create {
		action = "site.create"
		targetID = input.PrimaryDomain.Value
	}
	change := map[string]any{
		"kind":            input.Type.Value,
		"domain":          input.PrimaryDomain.Value,
		"resourceVersion": input.ExpectedResourceVersion.Value,
	}
	if err := s.audit(r, session.User.ID, action, "site", targetID, "intent", change); err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "audit_unavailable", "Audit storage unavailable", "")
		return
	}

	response, err := s.agent.Do(r.Context(), method, agentPath, "", requestID(r), body)
	if err != nil {
		_ = s.audit(r, session.User.ID, action, "site", targetID, "failure", change)
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Agent unavailable", "")
		return
	}
	result := "failure"
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		result = "success"
	}
	_ = s.audit(r, session.User.ID, action, "site", targetID, result, change)
	s.writeAgentResponse(w, response)
}

func validateSiteWriteInput(input *siteWriteInput, create bool) (field, detail string) {
	if create {
		if input.ExpectedResourceVersion.Set {
			return "expectedResourceVersion", "expectedResourceVersion is not allowed when creating a site"
		}
		if !input.PrimaryDomain.Set || input.PrimaryDomain.Value == "" {
			return "primaryDomain", "primaryDomain is required"
		}
		if !input.Type.Set || input.Type.Value == "" {
			return "type", "type is required"
		}
	} else {
		if !input.ExpectedResourceVersion.Set ||
			!resourceVersionPattern.MatchString(input.ExpectedResourceVersion.Value) {
			return "expectedResourceVersion", "a valid expectedResourceVersion is required"
		}
		if !input.PrimaryDomain.Set || input.PrimaryDomain.Value == "" {
			return "primaryDomain", "primaryDomain is required"
		}
		if !input.Type.Set || input.Type.Value == "" {
			return "type", "type is required"
		}
	}

	if input.PrimaryDomain.Set {
		normalized, valid := normalizePanelSiteDomain(input.PrimaryDomain.Value)
		if !valid {
			return "primaryDomain", "primaryDomain must be a valid ASCII domain"
		}
		input.PrimaryDomain.Value = normalized
	}
	if input.Aliases.Set {
		if len(input.Aliases.Value) > maxSiteAliases {
			return "aliases", "at most 20 aliases are allowed"
		}
		seen := make(map[string]struct{}, len(input.Aliases.Value)+1)
		if input.PrimaryDomain.Set {
			seen[input.PrimaryDomain.Value] = struct{}{}
		}
		for index, alias := range input.Aliases.Value {
			normalized, valid := normalizePanelSiteDomain(alias)
			if !valid {
				return "aliases", "each alias must be a valid ASCII domain"
			}
			if _, duplicate := seen[normalized]; duplicate {
				return "aliases", "aliases must not contain duplicate domains"
			}
			seen[normalized] = struct{}{}
			input.Aliases.Value[index] = normalized
		}
	}
	if input.Type.Set && !validPanelSiteType(input.Type.Value) {
		return "type", "type must be static, php, proxy, proxy_domain, load_balance, or redirect"
	}
	if input.Upstream.Set {
		if len(input.Upstream.Value) > maxSiteUpstreamLength || hasControlCharacter(input.Upstream.Value) {
			return "upstream", "upstream is too long or contains control characters"
		}
		if input.Upstream.Value != "" && !validPanelSiteUpstream(input.Upstream.Value) {
			return "upstream", "upstream must be an http(s) origin without credentials, query, fragment, or path"
		}
	}
	if input.Upstreams.Set {
		if len(input.Upstreams.Value) > maxSiteUpstreams {
			return "upstreams", "at most 8 upstreams are allowed"
		}
		seen := make(map[string]struct{}, len(input.Upstreams.Value))
		for _, upstream := range input.Upstreams.Value {
			if len(upstream) > maxSiteUpstreamLength || hasControlCharacter(upstream) ||
				!validPanelSiteUpstream(upstream) {
				return "upstreams", "each upstream must be an http(s) origin without credentials, query, fragment, or path"
			}
			if _, duplicate := seen[upstream]; duplicate {
				return "upstreams", "upstreams must not contain duplicates"
			}
			seen[upstream] = struct{}{}
		}
	}
	if input.RedirectTarget.Set {
		if len(input.RedirectTarget.Value) > maxSiteUpstreamLength ||
			hasControlCharacter(input.RedirectTarget.Value) ||
			(input.RedirectTarget.Value != "" && !validPanelSiteUpstream(input.RedirectTarget.Value)) {
			return "redirectTarget", "redirectTarget must be an http(s) origin without credentials, query, fragment, or path"
		}
	}
	if input.RedirectCode.Set && input.RedirectCode.Value != 301 && input.RedirectCode.Value != 302 &&
		input.RedirectCode.Value != 307 && input.RedirectCode.Value != 308 {
		return "redirectCode", "redirectCode must be 301, 302, 307, or 308"
	}
	if input.PHPVersion.Set && input.PHPVersion.Value != "" &&
		input.PHPVersion.Value != "latest" && input.PHPVersion.Value != "7.4" {
		return "phpVersion", "phpVersion must be latest or 7.4"
	}

	switch input.Type.Value {
	case "static":
		if input.Upstream.Value != "" || len(input.Upstreams.Value) != 0 ||
			input.RedirectTarget.Value != "" || input.RedirectCode.Set || input.PHPVersion.Value != "" {
			return "type", "static sites cannot define runtime, upstream, or redirect settings"
		}
	case "php":
		if input.Upstream.Value != "" || len(input.Upstreams.Value) != 0 ||
			input.RedirectTarget.Value != "" || input.RedirectCode.Set {
			return "type", "php sites cannot define upstream or redirect settings"
		}
	case "proxy", "proxy_domain":
		if !input.Upstream.Set || input.Upstream.Value == "" {
			return "upstream", "upstream is required for proxy sites"
		}
		if input.Type.Value == "proxy_domain" && !validPanelDomainOrigin(input.Upstream.Value) {
			return "upstream", "domain proxy upstream must use an ASCII domain"
		}
		if len(input.Upstreams.Value) != 0 || input.RedirectTarget.Value != "" ||
			input.RedirectCode.Set || input.PHPVersion.Value != "" {
			return "type", "proxy sites cannot define load balancing, redirect, or PHP settings"
		}
	case "load_balance":
		if !input.Upstreams.Set || len(input.Upstreams.Value) < 2 {
			return "upstreams", "2 to 8 upstreams are required for load balancing"
		}
		if input.Upstream.Value != "" || input.RedirectTarget.Value != "" ||
			input.RedirectCode.Set || input.PHPVersion.Value != "" {
			return "type", "load balancing sites cannot define single upstream, redirect, or PHP settings"
		}
		for _, upstream := range input.Upstreams.Value {
			if !validPanelHTTPOrigin(upstream) {
				return "upstreams", "load balancing upstreams must use http"
			}
		}
	case "redirect":
		if !input.RedirectTarget.Set || input.RedirectTarget.Value == "" {
			return "redirectTarget", "redirectTarget is required for redirect sites"
		}
		if !validPanelDomainOrigin(input.RedirectTarget.Value) {
			return "redirectTarget", "redirectTarget must use an ASCII domain"
		}
		if input.Upstream.Value != "" || len(input.Upstreams.Value) != 0 ||
			input.PHPVersion.Value != "" {
			return "type", "redirect sites cannot define upstream or PHP settings"
		}
	}
	return "", ""
}

func (input siteWriteInput) agentPayload() siteAgentPayload {
	var payload siteAgentPayload
	if input.PrimaryDomain.Set {
		payload.PrimaryDomain = &input.PrimaryDomain.Value
	}
	if input.Aliases.Set {
		payload.Aliases = &input.Aliases.Value
	}
	if input.Type.Set {
		payload.Type = &input.Type.Value
	}
	if input.Upstream.Set {
		payload.Upstream = &input.Upstream.Value
	}
	if input.Upstreams.Set {
		payload.Upstreams = &input.Upstreams.Value
	}
	if input.RedirectTarget.Set {
		payload.RedirectTarget = &input.RedirectTarget.Value
	}
	if input.RedirectCode.Set {
		payload.RedirectCode = &input.RedirectCode.Value
	}
	if input.PHPVersion.Set {
		payload.PHPVersion = &input.PHPVersion.Value
	}
	if input.Enabled.Set {
		payload.Enabled = &input.Enabled.Value
	}
	if input.ExpectedResourceVersion.Set {
		payload.ExpectedResourceVersion = &input.ExpectedResourceVersion.Value
	}
	return payload
}

func validPanelSiteType(value string) bool {
	switch value {
	case "static", "php", "proxy", "proxy_domain", "load_balance", "redirect":
		return true
	default:
		return false
	}
}

func allowedSiteUpdatePath(publicPath string) (agentPath, siteID string, allowed bool) {
	const prefix = "/api/v1/sites/"
	siteID = strings.TrimPrefix(publicPath, prefix)
	if siteID == publicPath || !siteIDPattern.MatchString(siteID) {
		return "", "", false
	}
	return "/v1/sites/" + siteID, siteID, true
}

func normalizePanelSiteDomain(value string) (string, bool) {
	if value == "" || strings.TrimSpace(value) != value || len(value) > maxSiteDomainLength ||
		strings.HasSuffix(value, ".") || !strings.Contains(value, ".") || net.ParseIP(value) != nil {
		return "", false
	}
	value = strings.ToLower(value)
	for _, label := range strings.Split(value, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", false
		}
		for _, character := range label {
			if (character < 'a' || character > 'z') &&
				(character < '0' || character > '9') && character != '-' {
				return "", false
			}
		}
	}
	return value, true
}

func validPanelSiteUpstream(value string) bool {
	if strings.TrimSpace(value) != value {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.User != nil || parsed.Hostname() == "" || parsed.Opaque != "" ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawPath != "" ||
		parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return false
	}
	return true
}

func validPanelHTTPOrigin(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "http" && validPanelSiteUpstream(value)
}

func validPanelDomainOrigin(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || !validPanelSiteUpstream(value) {
		return false
	}
	normalized, valid := normalizePanelSiteDomain(parsed.Hostname())
	return valid && normalized == strings.ToLower(parsed.Hostname())
}

func hasControlCharacter(value string) bool {
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return true
		}
	}
	return false
}
