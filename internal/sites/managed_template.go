package sites

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

const managedMarker = "# managed-by: kejilion-panel/v1"

var (
	ErrInvalidInput   = errors.New("invalid site request")
	ErrForbidden      = errors.New("site operation forbidden")
	ErrConflict       = errors.New("site resource conflict")
	ErrUnprocessable  = errors.New("site configuration is not processable")
	ErrUnavailable    = errors.New("site writer unavailable")
	ErrNeedsAttention = errors.New("site operation needs manual attention")
)

type SiteInput struct {
	PrimaryDomain           string   `json:"primaryDomain"`
	Aliases                 []string `json:"aliases,omitempty"`
	Type                    string   `json:"type"`
	Upstream                string   `json:"upstream,omitempty"`
	Enabled                 *bool    `json:"enabled,omitempty"`
	ExpectedResourceVersion string   `json:"expectedResourceVersion,omitempty"`
}

type managedSpec struct {
	Primary  string
	Aliases  []string
	Kind     contract.SiteKind
	Upstream string
}

func normalizeSiteInput(input SiteInput) (managedSpec, error) {
	if input.Enabled != nil && !*input.Enabled {
		return managedSpec{}, fmt.Errorf("%w: disabling sites is not supported", ErrUnprocessable)
	}
	primary, err := normalizeFQDN(input.PrimaryDomain)
	if err != nil {
		return managedSpec{}, err
	}
	if len(input.Aliases) > 20 {
		return managedSpec{}, fmt.Errorf("%w: at most 20 aliases are allowed", ErrUnprocessable)
	}
	seen := map[string]bool{primary: true}
	aliases := make([]string, 0, len(input.Aliases))
	for _, raw := range input.Aliases {
		alias, aliasErr := normalizeFQDN(raw)
		if aliasErr != nil {
			return managedSpec{}, aliasErr
		}
		if seen[alias] {
			return managedSpec{}, fmt.Errorf("%w: duplicate domain %q", ErrUnprocessable, alias)
		}
		seen[alias] = true
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)

	spec := managedSpec{Primary: primary, Aliases: aliases}
	switch input.Type {
	case "static":
		spec.Kind = contract.SiteStatic
		if strings.TrimSpace(input.Upstream) != "" {
			return managedSpec{}, fmt.Errorf("%w: static sites cannot define an upstream", ErrUnprocessable)
		}
	case "proxy":
		spec.Kind = contract.SiteReverseProxy
		spec.Upstream, err = normalizeUpstream(input.Upstream)
		if err != nil {
			return managedSpec{}, err
		}
	default:
		return managedSpec{}, fmt.Errorf("%w: type must be static or proxy", ErrInvalidInput)
	}
	return spec, nil
}

func normalizeFQDN(raw string) (string, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", fmt.Errorf("%w: invalid ASCII FQDN %q", ErrUnprocessable, raw)
	}
	value := strings.ToLower(raw)
	if len(value) > 253 || strings.HasSuffix(value, ".") || !strings.Contains(value, ".") ||
		strings.ContainsAny(value, `/*\:@[]`) || net.ParseIP(value) != nil {
		return "", fmt.Errorf("%w: invalid ASCII FQDN %q", ErrUnprocessable, raw)
	}
	labels := strings.Split(value, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", fmt.Errorf("%w: invalid ASCII FQDN %q", ErrUnprocessable, raw)
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
				return "", fmt.Errorf("%w: invalid ASCII FQDN %q", ErrUnprocessable, raw)
			}
		}
	}
	return value, nil
}

func normalizeUpstream(raw string) (string, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", fmt.Errorf("%w: upstream must be an http(s) origin", ErrUnprocessable)
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.User != nil || parsed.Opaque != "" || parsed.Path != "" || parsed.RawPath != "" ||
		parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return "", fmt.Errorf("%w: upstream must be an http(s) origin without credentials or a path", ErrUnprocessable)
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", fmt.Errorf("%w: upstream host is required", ErrUnprocessable)
	}
	if ip := net.ParseIP(host); ip != nil {
		if !ip.IsLoopback() && !isRFC1918(ip) {
			return "", fmt.Errorf("%w: upstream IP must be loopback or RFC1918", ErrUnprocessable)
		}
	} else if !validSingleDNSLabel(host) {
		return "", fmt.Errorf("%w: upstream host must be a single Docker DNS label", ErrUnprocessable)
	}
	port := parsed.Port()
	if port != "" {
		number, portErr := strconv.Atoi(port)
		if portErr != nil || number < 1 || number > 65535 {
			return "", fmt.Errorf("%w: upstream port must be between 1 and 65535", ErrUnprocessable)
		}
	}
	normalizedHost := host
	if strings.Contains(host, ":") {
		normalizedHost = "[" + host + "]"
	}
	if port != "" {
		normalizedHost += ":" + port
	}
	return parsed.Scheme + "://" + normalizedHost, nil
}

func isRFC1918(ip net.IP) bool {
	ip = ip.To4()
	if ip == nil {
		return false
	}
	return ip[0] == 10 ||
		(ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) ||
		(ip[0] == 192 && ip[1] == 168)
}

func validSingleDNSLabel(value string) bool {
	if len(value) == 0 || len(value) > 63 || value[0] == '-' || value[len(value)-1] == '-' {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
			return false
		}
	}
	return true
}

func renderManagedConfig(spec managedSpec) []byte {
	serverNames := append([]string{spec.Primary}, spec.Aliases...)
	var body strings.Builder
	body.WriteString(managedMarker)
	body.WriteString("\n# This file is generated by Kejilion Panel. Manual edits make it read-only.\n")
	body.WriteString("server {\n")
	body.WriteString("    listen 80;\n")
	body.WriteString("    listen [::]:80;\n")
	body.WriteString("    server_name ")
	body.WriteString(strings.Join(serverNames, " "))
	body.WriteString(";\n\n")
	if spec.Kind == contract.SiteStatic {
		body.WriteString("    root /var/www/html/")
		body.WriteString(spec.Primary)
		body.WriteString(";\n")
		body.WriteString("    index index.html;\n\n")
		body.WriteString("    location / {\n")
		body.WriteString("        try_files $uri $uri/ =404;\n")
		body.WriteString("    }\n")
	} else {
		body.WriteString("    location / {\n")
		body.WriteString("        proxy_pass ")
		body.WriteString(spec.Upstream)
		body.WriteString(";\n")
		body.WriteString("        proxy_http_version 1.1;\n")
		body.WriteString("        proxy_set_header Host $host;\n")
		body.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
		body.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
		body.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
		body.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
		body.WriteString("        proxy_set_header Connection \"upgrade\";\n")
		body.WriteString("    }\n")
	}
	body.WriteString("}\n")
	return []byte(body.String())
}

func renderDefaultIndex(domain string) []byte {
	return []byte("<!doctype html>\n<html lang=\"zh-CN\">\n<head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>" +
		domain + "</title></head>\n<body><main><h1>网站已创建</h1><p>" + domain +
		" 已由 Kejilion Panel 安全创建。</p></main></body>\n</html>\n")
}

func (d *Discoverer) markManagedSite(site *contract.SiteSummary, data []byte, configPath string) {
	if !bytes.HasPrefix(data, []byte(managedMarker+"\n")) {
		return
	}
	site.Origin = contract.OriginWeb
	spec, err := managedSpecFromSummary(*site)
	expectedPath := ""
	if err == nil {
		expectedPath = filepath.Join(d.WebRoot, "conf.d", spec.Primary+".conf")
	}
	if err != nil || filepath.Clean(configPath) != filepath.Clean(expectedPath) ||
		!bytes.Equal(data, renderManagedConfig(spec)) {
		site.Consistency = contract.ConsistencyDrifted
		site.AllowedActions = []string{}
		site.Warnings = uniqueStrings(append(site.Warnings,
			"Panel managed marker exists but the canonical template has drifted; the site is read-only.",
		))
		return
	}
	if spec.Kind == contract.SiteStatic {
		expectedRoot := filepath.Join(d.WebRoot, "html", spec.Primary)
		if filepath.Clean(site.DocumentRoot) != filepath.Clean(expectedRoot) {
			site.Consistency = contract.ConsistencyDrifted
			site.AllowedActions = []string{}
			return
		}
	}
	site.Consistency = contract.ConsistencyInSync
	site.AllowedActions = []string{"update"}
}

func managedSpecFromSummary(site contract.SiteSummary) (managedSpec, error) {
	primary, err := normalizeFQDN(site.PrimaryDomain)
	if err != nil {
		return managedSpec{}, err
	}
	spec := managedSpec{Primary: primary, Kind: site.Kind}
	for _, domain := range site.Domains {
		normalized, domainErr := normalizeFQDN(domain)
		if domainErr != nil {
			return managedSpec{}, domainErr
		}
		if normalized != primary {
			spec.Aliases = append(spec.Aliases, normalized)
		}
	}
	sort.Strings(spec.Aliases)
	switch site.Kind {
	case contract.SiteStatic:
		if site.Target != "" {
			return managedSpec{}, errors.New("static managed site has a target")
		}
	case contract.SiteReverseProxy:
		spec.Upstream, err = normalizeUpstream(site.Target)
		if err != nil || spec.Upstream != site.Target {
			return managedSpec{}, errors.New("proxy managed site has a non-canonical upstream")
		}
	default:
		return managedSpec{}, errors.New("unsupported managed site kind")
	}
	return spec, nil
}
