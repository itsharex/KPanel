package sites

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

const (
	managedMarkerV1 = "# managed-by: kejilion-panel/v1"
	managedMarker   = "# managed-by: kejilion-panel/v2"
)

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
	Recipe                  string   `json:"recipe,omitempty"`
	Upstream                string   `json:"upstream,omitempty"`
	Upstreams               []string `json:"upstreams,omitempty"`
	RedirectTarget          string   `json:"redirectTarget,omitempty"`
	RedirectCode            int      `json:"redirectCode,omitempty"`
	PHPVersion              string   `json:"phpVersion,omitempty"`
	Enabled                 *bool    `json:"enabled,omitempty"`
	ExpectedResourceVersion string   `json:"expectedResourceVersion,omitempty"`
}

type managedSpec struct {
	Primary        string
	Aliases        []string
	Kind           contract.SiteKind
	Upstream       string
	Upstreams      []string
	RedirectCode   int
	RedirectTarget string
	PHPVersion     string
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
		err = rejectUnusedSiteFields(input, "static")
	case "php":
		spec.Kind = contract.SitePHP
		spec.PHPVersion = input.PHPVersion
		if spec.PHPVersion == "" {
			spec.PHPVersion = "latest"
		}
		if spec.PHPVersion != "latest" && spec.PHPVersion != "7.4" {
			return managedSpec{}, fmt.Errorf("%w: phpVersion must be latest or 7.4", ErrUnprocessable)
		}
		err = rejectUnusedSiteFields(input, "php")
	case "proxy":
		spec.Kind = contract.SiteReverseProxy
		spec.Upstream, err = normalizeUpstream(input.Upstream, upstreamPrivate)
		if err == nil {
			err = rejectUnusedSiteFields(input, "proxy")
		}
	case "proxy_domain":
		spec.Kind = contract.SiteDomainProxy
		spec.Upstream, err = normalizeUpstream(input.Upstream, upstreamDomain)
		if err == nil {
			err = rejectUnusedSiteFields(input, "proxy_domain")
		}
	case "load_balance":
		spec.Kind = contract.SiteLoadBalance
		spec.Upstreams, err = normalizeUpstreams(input.Upstreams)
		if err == nil {
			err = rejectUnusedSiteFields(input, "load_balance")
		}
	case "redirect":
		spec.Kind = contract.SiteRedirect
		spec.RedirectTarget, err = normalizeUpstream(input.RedirectTarget, upstreamDomain)
		if err != nil {
			break
		}
		spec.RedirectCode = input.RedirectCode
		if spec.RedirectCode == 0 {
			spec.RedirectCode = 301
		}
		if spec.RedirectCode != 301 && spec.RedirectCode != 302 &&
			spec.RedirectCode != 307 && spec.RedirectCode != 308 {
			return managedSpec{}, fmt.Errorf("%w: redirectCode must be 301, 302, 307, or 308", ErrUnprocessable)
		}
		err = rejectUnusedSiteFields(input, "redirect")
	default:
		return managedSpec{}, fmt.Errorf(
			"%w: type must be static, php, proxy, proxy_domain, load_balance, or redirect",
			ErrInvalidInput,
		)
	}
	if err != nil {
		return managedSpec{}, err
	}
	return spec, nil
}

func rejectUnusedSiteFields(input SiteInput, kind string) error {
	if input.Recipe != "" {
		return fmt.Errorf("%w: %s sites cannot define a recipe", ErrUnprocessable, kind)
	}
	if kind != "proxy" && kind != "proxy_domain" && strings.TrimSpace(input.Upstream) != "" {
		return fmt.Errorf("%w: %s sites cannot define upstream", ErrUnprocessable, kind)
	}
	if kind != "load_balance" && len(input.Upstreams) != 0 {
		return fmt.Errorf("%w: %s sites cannot define upstreams", ErrUnprocessable, kind)
	}
	if kind != "redirect" && (strings.TrimSpace(input.RedirectTarget) != "" || input.RedirectCode != 0) {
		return fmt.Errorf("%w: %s sites cannot define a redirect", ErrUnprocessable, kind)
	}
	if kind != "php" && strings.TrimSpace(input.PHPVersion) != "" {
		return fmt.Errorf("%w: %s sites cannot define phpVersion", ErrUnprocessable, kind)
	}
	return nil
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

type upstreamPolicy int

const (
	upstreamPrivate upstreamPolicy = iota
	upstreamDomain
	upstreamBalanced
)

func normalizeUpstream(raw string, policy upstreamPolicy) (string, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", fmt.Errorf("%w: upstream must be an http(s) origin", ErrUnprocessable)
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.User != nil || parsed.Opaque != "" || (parsed.Path != "" && parsed.Path != "/") || parsed.RawPath != "" ||
		parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return "", fmt.Errorf("%w: upstream must be an http(s) origin without credentials or a path", ErrUnprocessable)
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", fmt.Errorf("%w: upstream host is required", ErrUnprocessable)
	}
	ip := net.ParseIP(host)
	switch policy {
	case upstreamPrivate:
		if ip != nil {
			if !ip.IsLoopback() && !ip.IsPrivate() {
				return "", fmt.Errorf("%w: upstream IP must be loopback or private", ErrUnprocessable)
			}
		} else if !validSingleDNSLabel(host) {
			return "", fmt.Errorf("%w: upstream host must be a single Docker DNS label", ErrUnprocessable)
		}
	case upstreamDomain:
		if ip != nil {
			return "", fmt.Errorf("%w: domain upstream must use an ASCII FQDN", ErrUnprocessable)
		}
		if normalized, domainErr := normalizeFQDN(host); domainErr != nil || normalized != host {
			return "", fmt.Errorf("%w: domain upstream must use an ASCII FQDN", ErrUnprocessable)
		}
	case upstreamBalanced:
		if ip != nil {
			if !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsGlobalUnicast() {
				return "", fmt.Errorf("%w: load balancing target IP is not routable", ErrUnprocessable)
			}
		} else if strings.Contains(host, ".") {
			if normalized, domainErr := normalizeFQDN(host); domainErr != nil || normalized != host {
				return "", fmt.Errorf("%w: invalid load balancing target", ErrUnprocessable)
			}
		} else if !validSingleDNSLabel(host) {
			return "", fmt.Errorf("%w: invalid load balancing target", ErrUnprocessable)
		}
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

func normalizeUpstreams(values []string) ([]string, error) {
	if len(values) < 2 || len(values) > 8 {
		return nil, fmt.Errorf("%w: load balancing requires 2 to 8 upstreams", ErrUnprocessable)
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		normalized, err := normalizeUpstream(value, upstreamBalanced)
		if err != nil {
			return nil, err
		}
		parsed, _ := url.Parse(normalized)
		if parsed.Scheme != "http" {
			return nil, fmt.Errorf("%w: load balancing upstreams must use http", ErrUnprocessable)
		}
		if seen[normalized] {
			return nil, fmt.Errorf("%w: duplicate load balancing upstream", ErrUnprocessable)
		}
		seen[normalized] = true
		result = append(result, normalized)
	}
	sort.Strings(result)
	return result, nil
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
	body.WriteString("\n# This file is generated by KPanel. Manual edits make it read-only.\n")
	if spec.Kind == contract.SiteLoadBalance {
		name := managedUpstreamName(spec.Primary)
		body.WriteString("upstream ")
		body.WriteString(name)
		body.WriteString(" {\n")
		body.WriteString("    hash $remote_addr consistent;\n")
		for _, upstream := range spec.Upstreams {
			parsed, _ := url.Parse(upstream)
			target := parsed.Host
			if parsed.Port() == "" {
				if parsed.Scheme == "https" {
					target += ":443"
				} else {
					target += ":80"
				}
			}
			body.WriteString("    server ")
			body.WriteString(target)
			body.WriteString(";\n")
		}
		body.WriteString("    keepalive 64;\n")
		body.WriteString("}\n\n")
	}
	body.WriteString("server {\n")
	body.WriteString("    listen 80;\n")
	body.WriteString("    listen [::]:80;\n")
	body.WriteString("    server_name ")
	body.WriteString(strings.Join(serverNames, " "))
	body.WriteString(";\n\n")
	body.WriteString("    location ^~ /.well-known/acme-challenge/ {\n")
	body.WriteString("        default_type \"text/plain\";\n")
	body.WriteString("        root /var/www/letsencrypt;\n")
	body.WriteString("    }\n\n")
	switch spec.Kind {
	case contract.SiteStatic:
		body.WriteString("    root /var/www/html/")
		body.WriteString(spec.Primary)
		body.WriteString(";\n")
		body.WriteString("    index index.html;\n\n")
		body.WriteString("    location / {\n")
		body.WriteString("        try_files $uri $uri/ =404;\n")
		body.WriteString("    }\n")
		body.WriteString("\n    client_max_body_size 50m;\n")
	case contract.SitePHP:
		body.WriteString("    root /var/www/html/")
		body.WriteString(spec.Primary)
		body.WriteString(";\n")
		body.WriteString("    index index.php index.html;\n\n")
		body.WriteString("    location / {\n")
		body.WriteString("        try_files $uri $uri/ /index.php?$args;\n")
		body.WriteString("    }\n\n")
		body.WriteString("    location ~ \\.php$ {\n")
		body.WriteString("        try_files $uri =404;\n")
		body.WriteString("        include fastcgi_params;\n")
		body.WriteString("        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;\n")
		body.WriteString("        fastcgi_pass unix:/run/")
		if spec.PHPVersion == "7.4" {
			body.WriteString("php74")
		} else {
			body.WriteString("php")
		}
		body.WriteString("/php-fpm.sock;\n")
		body.WriteString("    }\n\n")
		body.WriteString("    location ~ /\\. { deny all; }\n")
		body.WriteString("    client_max_body_size 50m;\n")
	case contract.SiteWordPress:
		// Keep the same paths and Nginx behavior as kejilion.sh's
		// wordpress.com.conf while retaining a canonical KPanel marker.
		body.WriteString("    listen 443 ssl;\n")
		body.WriteString("    listen [::]:443 ssl;\n")
		body.WriteString("    listen 443 quic;\n")
		body.WriteString("    listen [::]:443 quic;\n\n")
		body.WriteString("    ssl_certificate /etc/nginx/certs/")
		body.WriteString(spec.Primary)
		body.WriteString("_cert.pem;\n")
		body.WriteString("    ssl_certificate_key /etc/nginx/certs/")
		body.WriteString(spec.Primary)
		body.WriteString("_key.pem;\n\n")
		body.WriteString("    if ($scheme = http) {\n")
		body.WriteString("        return 301 https://$host$request_uri;\n")
		body.WriteString("    }\n\n")
		body.WriteString("    root /var/www/html/")
		body.WriteString(spec.Primary)
		body.WriteString("/wordpress;\n")
		body.WriteString("    index index.php;\n\n")
		body.WriteString("    location / {\n")
		body.WriteString("        try_files $uri $uri/ /index.php?$args;\n")
		body.WriteString("    }\n\n")
		body.WriteString("    location ~ \\.php$ {\n")
		body.WriteString("        fastcgi_pass unix:/run/php/php-fpm.sock;\n")
		body.WriteString("        fastcgi_index index.php;\n")
		body.WriteString("        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;\n")
		body.WriteString("        include fastcgi_params;\n")
		body.WriteString("        add_header Alt-Svc 'h3=\":443\"; ma=86400';\n")
		body.WriteString("    }\n\n")
		body.WriteString("    location ~* \\.(js|css|png|jpg|jpeg|gif|ico|bmp|swf|eot|svg|ttf|woff|woff2|webp)$ {\n")
		body.WriteString("        aio threads;\n")
		body.WriteString("        add_header Cache-Control \"public, max-age=2592000\";\n")
		body.WriteString("        add_header Alt-Svc 'h3=\":443\"; ma=86400';\n")
		body.WriteString("        log_not_found off;\n")
		body.WriteString("        access_log off;\n")
		body.WriteString("    }\n\n")
		body.WriteString("    location ~ /\\. { deny all; }\n")
		body.WriteString("    client_max_body_size 50m;\n")
	case contract.SiteRedirect:
		body.WriteString("    location / {\n")
		body.WriteString("        return ")
		body.WriteString(strconv.Itoa(spec.RedirectCode))
		body.WriteString(" ")
		body.WriteString(spec.RedirectTarget)
		body.WriteString("$request_uri;\n")
		body.WriteString("    }\n")
	case contract.SiteReverseProxy, contract.SiteDomainProxy, contract.SiteLoadBalance:
		target := spec.Upstream
		if spec.Kind == contract.SiteLoadBalance {
			parsed, _ := url.Parse(spec.Upstreams[0])
			target = parsed.Scheme + "://" + managedUpstreamName(spec.Primary)
		}
		body.WriteString("    location / {\n")
		body.WriteString("        proxy_pass ")
		body.WriteString(target)
		body.WriteString(";\n")
		body.WriteString("        proxy_http_version 1.1;\n")
		if spec.Kind == contract.SiteDomainProxy {
			parsed, _ := url.Parse(spec.Upstream)
			body.WriteString("        proxy_set_header Host ")
			body.WriteString(parsed.Host)
			body.WriteString(";\n")
		} else {
			body.WriteString("        proxy_set_header Host $host;\n")
		}
		body.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
		body.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
		body.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
		body.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
		body.WriteString("        proxy_set_header Connection \"upgrade\";\n")
		if spec.Kind == contract.SiteDomainProxy {
			parsed, _ := url.Parse(spec.Upstream)
			body.WriteString("        proxy_ssl_server_name on;\n")
			body.WriteString("        proxy_ssl_name ")
			body.WriteString(parsed.Hostname())
			body.WriteString(";\n")
		}
		body.WriteString("    }\n")
	}
	body.WriteString("}\n")
	return []byte(body.String())
}

// renderWordPressBootstrapConfig temporarily serves only ACME challenges while
// the final certificate is issued. It never proxies PHP or exposes the staged
// WordPress files.
func renderWordPressBootstrapConfig(primary string) []byte {
	return []byte(
		"# managed-by: kejilion-panel/wordpress-bootstrap\n" +
			"server {\n" +
			"    listen 80;\n" +
			"    listen [::]:80;\n" +
			"    server_name " + primary + ";\n\n" +
			"    location ^~ /.well-known/acme-challenge/ {\n" +
			"        default_type \"text/plain\";\n" +
			"        root /var/www/letsencrypt;\n" +
			"    }\n\n" +
			"    location / { return 503; }\n" +
			"}\n",
	)
}

func renderManagedConfigV1(spec managedSpec) []byte {
	serverNames := append([]string{spec.Primary}, spec.Aliases...)
	var body strings.Builder
	body.WriteString(managedMarkerV1)
	body.WriteString("\n# This file is generated by KPanel. Manual edits make it read-only.\n")
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

func managedUpstreamName(primary string) string {
	sum := sha256.Sum256([]byte(primary))
	return "kp_" + hex.EncodeToString(sum[:6])
}

func renderDefaultIndex(domain string) []byte {
	return []byte("<!doctype html>\n<html lang=\"zh-CN\">\n<head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>" +
		domain + "</title></head>\n<body><main><h1>网站已创建</h1><p>" + domain +
		" 已由 KPanel 安全创建。</p></main></body>\n</html>\n")
}

func renderDefaultPHP(domain string) []byte {
	return []byte("<?php\nheader('Content-Type: text/html; charset=utf-8');\n?><!doctype html>\n" +
		"<html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" " +
		"content=\"width=device-width,initial-scale=1\"><title>" + domain + "</title></head>" +
		"<body><main><h1>PHP 网站已创建</h1><p>" + domain +
		" 已由 KPanel 安全创建。</p></main></body></html>\n")
}

func siteNeedsDocumentRoot(kind contract.SiteKind) bool {
	return kind == contract.SiteStatic || kind == contract.SitePHP || kind == contract.SiteWordPress
}

func managedDefaultDocument(spec managedSpec) (string, []byte) {
	if spec.Kind == contract.SitePHP {
		return "index.php", renderDefaultPHP(spec.Primary)
	}
	return "index.html", renderDefaultIndex(spec.Primary)
}

func (d *Discoverer) markManagedSite(site *contract.SiteSummary, data []byte, configPath string) {
	v1 := bytes.HasPrefix(data, []byte(managedMarkerV1+"\n"))
	v2 := bytes.HasPrefix(data, []byte(managedMarker+"\n"))
	if !v1 && !v2 {
		return
	}
	site.Origin = contract.OriginWeb
	spec, err := managedSpecFromSummary(*site)
	expectedPath := ""
	if err == nil {
		expectedPath = filepath.Join(d.WebRoot, "conf.d", spec.Primary+".conf")
	}
	expectedConfig := renderManagedConfig(spec)
	if v1 {
		if spec.Kind != contract.SiteStatic && spec.Kind != contract.SiteReverseProxy {
			err = errors.New("v1 marker used by an unsupported site type")
		} else {
			expectedConfig = renderManagedConfigV1(spec)
		}
	}
	if err != nil || filepath.Clean(configPath) != filepath.Clean(expectedPath) ||
		!bytes.Equal(data, expectedConfig) {
		site.Consistency = contract.ConsistencyDrifted
		site.AllowedActions = []string{}
		site.Warnings = uniqueStrings(append(site.Warnings,
			"Panel managed marker exists but the canonical template has drifted; the site is read-only.",
		))
		return
	}
	if spec.Kind == contract.SiteStatic || spec.Kind == contract.SitePHP || spec.Kind == contract.SiteWordPress {
		expectedRoot := filepath.Join(d.WebRoot, "html", spec.Primary)
		if spec.Kind == contract.SiteWordPress {
			expectedRoot = filepath.Join(expectedRoot, "wordpress")
		}
		if filepath.Clean(site.DocumentRoot) != filepath.Clean(expectedRoot) {
			site.Consistency = contract.ConsistencyDrifted
			site.AllowedActions = []string{}
			return
		}
	}
	site.Consistency = contract.ConsistencyInSync
	if spec.Kind == contract.SiteWordPress {
		site.AllowedActions = []string{"delete"}
	} else {
		site.AllowedActions = []string{"update", "delete"}
	}
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
	case contract.SitePHP:
		switch site.Target {
		case "php":
			spec.PHPVersion = "latest"
		case "php74":
			spec.PHPVersion = "7.4"
		default:
			return managedSpec{}, errors.New("PHP managed site has an unknown runtime")
		}
	case contract.SiteWordPress:
		if site.Target != "php" {
			return managedSpec{}, errors.New("WordPress managed site has an unknown runtime")
		}
		spec.PHPVersion = "latest"
	case contract.SiteReverseProxy:
		spec.Upstream, err = normalizeUpstream(site.Target, upstreamPrivate)
		if err != nil || spec.Upstream != site.Target {
			return managedSpec{}, errors.New("proxy managed site has a non-canonical upstream")
		}
	case contract.SiteDomainProxy:
		spec.Upstream, err = normalizeUpstream(site.Target, upstreamDomain)
		if err != nil || spec.Upstream != site.Target {
			return managedSpec{}, errors.New("domain proxy managed site has a non-canonical upstream")
		}
	case contract.SiteLoadBalance:
		values := strings.Split(site.Target, ",")
		for index := range values {
			values[index] = strings.TrimSpace(values[index])
		}
		spec.Upstreams, err = normalizeUpstreams(values)
		if err != nil || strings.Join(spec.Upstreams, ", ") != site.Target {
			return managedSpec{}, errors.New("load balancing site has non-canonical upstreams")
		}
	case contract.SiteRedirect:
		code, target, ok := strings.Cut(site.Target, " ")
		spec.RedirectCode, err = strconv.Atoi(code)
		if !ok || err != nil {
			return managedSpec{}, errors.New("redirect managed site has an invalid target")
		}
		spec.RedirectTarget, err = normalizeUpstream(target, upstreamDomain)
		if err != nil || fmt.Sprintf("%d %s", spec.RedirectCode, spec.RedirectTarget) != site.Target {
			return managedSpec{}, errors.New("redirect managed site has a non-canonical target")
		}
	default:
		return managedSpec{}, errors.New("unsupported managed site kind")
	}
	return spec, nil
}
