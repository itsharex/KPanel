package sites

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

const maxConfigBytes = 2 << 20

var (
	directivePattern = regexp.MustCompile(`(?m)^[\t ]*([A-Za-z_]+)[\t ]+([^;{}]+);`)
	upstreamPattern  = regexp.MustCompile(`(?s)(?:^|\s)upstream[\t ]+([A-Za-z0-9_-]+)[\t ]*\{(.*?)\}`)
	domainPattern    = regexp.MustCompile(`^(?:\*\.)?(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)*[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)
)

type Discoverer struct {
	WebRoot string
	Now     func() time.Time
}

func NewDiscoverer(webRoot string) *Discoverer {
	return &Discoverer{WebRoot: webRoot, Now: time.Now}
}

func (d *Discoverer) Discover() ([]contract.SiteSummary, error) {
	if d.WebRoot == "" {
		d.WebRoot = "/home/web"
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	now := d.Now().UTC()
	confRoot := filepath.Join(d.WebRoot, "conf.d")
	entries, err := os.ReadDir(confRoot)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read site configs: %w", err)
	}

	var result []contract.SiteSummary
	represented := make(map[string]bool)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 ||
			filepath.Ext(name) != ".conf" || name == "default.conf" || name == "map.conf" {
			continue
		}
		path := filepath.Join(confRoot, name)
		site, parseErr := d.fromConfig(path, now)
		if parseErr != nil {
			site = d.unreadableConfig(path, parseErr, now)
		}
		for _, domain := range site.Domains {
			represented[strings.TrimPrefix(domain, "*.")] = true
		}
		if site.PrimaryDomain != "" {
			represented[strings.TrimPrefix(site.PrimaryDomain, "*.")] = true
		}
		result = append(result, site)
	}
	domainCounts := make(map[string]int)
	for _, site := range result {
		for _, domain := range site.Domains {
			domainCounts[domain]++
		}
	}
	for i := range result {
		for _, domain := range result[i].Domains {
			if domainCounts[domain] > 1 {
				result[i].Consistency = contract.ConsistencyAmbiguous
				result[i].AllowedActions = []string{}
				result[i].Warnings = uniqueStrings(append(result[i].Warnings, "同一域名出现在多个配置文件中，无法唯一归属"))
				break
			}
		}
	}

	result = append(result, d.orphanHTML(represented, now)...)
	result = append(result, d.orphanCertificates(represented, now)...)
	sort.Slice(result, func(i, j int) bool {
		return result[i].PrimaryDomain < result[j].PrimaryDomain
	})
	return result, nil
}

func (d *Discoverer) fromConfig(path string, now time.Time) (contract.SiteSummary, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return contract.SiteSummary{}, err
	}
	if !info.Mode().IsRegular() {
		return contract.SiteSummary{}, errors.New("config is not a regular file")
	}
	if info.Size() > maxConfigBytes {
		return contract.SiteSummary{}, fmt.Errorf("config exceeds %d bytes", maxConfigBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return contract.SiteSummary{}, err
	}
	clean := stripComments(string(data))
	directives := parseDirectives(clean)
	domains := validDomains(strings.Fields(strings.Join(directives["server_name"], " ")))
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	primary := choosePrimary(domains, stem)
	if primary == "" {
		primary = stem
	}

	kind, target, documentRoot, warnings := d.classify(clean, directives)
	consistency := contract.ConsistencyInSync
	health := "discovered"
	if len(domains) == 0 {
		warnings = append(warnings, "未能从 server_name 确认有效域名")
		kind = contract.SiteUnknown
	}
	if kind == contract.SiteUnknown {
		consistency = contract.ConsistencyReadOnly
		health = "unknown"
		warnings = append(warnings, "配置结构无法安全归类，保持只读")
	}
	tls, certArtifact, certBytes := d.discoverTLS(directives, primary)
	configHash := hashBytes(data)
	artifacts := []contract.Artifact{{Kind: "nginx_config", Path: path, Hash: configHash}}
	if documentRoot != "" {
		artifacts = append(artifacts, contract.Artifact{Kind: "document_root", Path: documentRoot})
		if stat, statErr := os.Stat(documentRoot); statErr != nil || !stat.IsDir() {
			warnings = append(warnings, "站点目录不存在或不可访问")
			health = "degraded"
		}
	}
	if certArtifact != nil {
		artifacts = append(artifacts, *certArtifact)
	}
	versionInput := append([]byte("config:"+configHash+"\n"), certBytes...)
	versionInput = append(versionInput, []byte("\nkind:"+string(kind)+"\ntarget:"+target+"\nroot:"+documentRoot)...)
	return contract.SiteSummary{
		ID:              stableID("site", primary, path),
		PrimaryDomain:   primary,
		Domains:         domains,
		Kind:            kind,
		Enabled:         true,
		Health:          health,
		TLS:             tls,
		Target:          target,
		DocumentRoot:    documentRoot,
		Origin:          contract.OriginDiscovered,
		Consistency:     consistency,
		ResourceVersion: hashBytes(versionInput),
		AllowedActions:  []string{},
		Artifacts:       artifacts,
		Warnings:        uniqueStrings(warnings),
		ReconciledAt:    now,
	}, nil
}

func (d *Discoverer) classify(clean string, directives map[string][]string) (contract.SiteKind, string, string, []string) {
	var warnings []string
	roots := uniqueStrings(directives["root"])
	proxies := uniqueStrings(directives["proxy_pass"])
	fastCGI := directives["fastcgi_pass"]
	returns := directives["return"]
	if len(roots) > 1 {
		warnings = append(warnings, "检测到多个 document root")
	}
	documentRoot := ""
	if len(roots) == 1 {
		documentRoot = d.hostDocumentRoot(strings.TrimSpace(roots[0]))
	}
	if len(proxies) > 0 && len(fastCGI) > 0 {
		return contract.SiteUnknown, "", documentRoot, append(warnings, "同时检测到反向代理与 FastCGI")
	}
	if len(proxies) > 0 {
		resolved := resolveProxyTargets(clean, proxies)
		if len(resolved) == 0 {
			return contract.SiteUnknown, "", documentRoot, append(warnings, "无法解析反向代理上游")
		}
		return contract.SiteReverseProxy, strings.Join(resolved, ", "), documentRoot, warnings
	}
	if len(fastCGI) > 0 && documentRoot != "" {
		return contract.SitePHP, "", documentRoot, warnings
	}
	for _, value := range returns {
		fields := strings.Fields(value)
		if len(fields) >= 2 && (fields[0] == "301" || fields[0] == "302" || fields[0] == "307" || fields[0] == "308") {
			return contract.SiteRedirect, sanitizeTarget(fields[1]), documentRoot, warnings
		}
	}
	if documentRoot != "" {
		return contract.SiteStatic, "", documentRoot, warnings
	}
	return contract.SiteUnknown, "", "", warnings
}

func (d *Discoverer) hostDocumentRoot(value string) string {
	value = strings.Trim(value, `"'`)
	value = filepath.ToSlash(value)
	const containerRoot = "/var/www/"
	if strings.HasPrefix(value, containerRoot) {
		relative := strings.TrimPrefix(value, containerRoot)
		return filepath.Clean(filepath.Join(d.WebRoot, filepath.FromSlash(relative)))
	}
	if strings.HasPrefix(value, filepath.ToSlash(filepath.Clean(d.WebRoot))+"/") {
		return filepath.Clean(filepath.FromSlash(value))
	}
	return value
}

func (d *Discoverer) discoverTLS(directives map[string][]string, primary string) (contract.TLSStatus, *contract.Artifact, []byte) {
	tls := contract.TLSStatus{Status: "disabled"}
	certificates := directives["ssl_certificate"]
	if len(certificates) == 0 {
		return tls, nil, nil
	}
	tls.Enabled = true
	tls.Status = "missing"
	certPath := strings.Trim(certificates[0], `"'`)
	if strings.HasPrefix(certPath, "/etc/nginx/certs/") {
		certPath = filepath.Join(d.WebRoot, "certs", filepath.Base(certPath))
	} else if !filepath.IsAbs(certPath) && primary != "" {
		certPath = filepath.Join(d.WebRoot, "certs", primary+"_cert.pem")
	}
	info, err := os.Lstat(certPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > 1<<20 {
		return tls, &contract.Artifact{Kind: "certificate", Path: certPath}, nil
	}
	data, err := os.ReadFile(certPath)
	if err != nil {
		return tls, &contract.Artifact{Kind: "certificate", Path: certPath}, nil
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		tls.Status = "invalid"
		return tls, &contract.Artifact{Kind: "certificate", Path: certPath, Hash: hashBytes(data)}, data
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		tls.Status = "invalid"
		return tls, &contract.Artifact{Kind: "certificate", Path: certPath, Hash: hashBytes(data)}, data
	}
	expires := cert.NotAfter.UTC()
	tls.ExpiresAt = &expires
	tls.Source = "filesystem"
	switch {
	case expires.Before(d.Now()):
		tls.Status = "expired"
	case expires.Before(d.Now().Add(30 * 24 * time.Hour)):
		tls.Status = "expiring"
	default:
		tls.Status = "valid"
	}
	return tls, &contract.Artifact{Kind: "certificate", Path: certPath, Hash: hashBytes(data)}, data
}

func (d *Discoverer) unreadableConfig(path string, cause error, now time.Time) contract.SiteSummary {
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return contract.SiteSummary{
		ID:              stableID("site", stem, path),
		PrimaryDomain:   stem,
		Domains:         []string{},
		Kind:            contract.SiteUnknown,
		Enabled:         true,
		Health:          "unreadable",
		TLS:             contract.TLSStatus{Status: "unknown"},
		Origin:          contract.OriginDiscovered,
		Consistency:     contract.ConsistencyReadOnly,
		ResourceVersion: stableID("version", path, cause.Error()),
		AllowedActions:  []string{},
		Artifacts:       []contract.Artifact{{Kind: "nginx_config", Path: path}},
		Warnings:        []string{"配置无法安全读取：" + cause.Error()},
		ReconciledAt:    now,
	}
}

func (d *Discoverer) orphanHTML(represented map[string]bool, now time.Time) []contract.SiteSummary {
	root := filepath.Join(d.WebRoot, "html")
	entries, _ := os.ReadDir(root)
	var result []contract.SiteSummary
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || represented[name] || !validDomain(name) {
			continue
		}
		path := filepath.Join(root, name)
		result = append(result, orphanSite(name, "orphan_html", path, now))
		represented[name] = true
	}
	return result
}

func (d *Discoverer) orphanCertificates(represented map[string]bool, now time.Time) []contract.SiteSummary {
	root := filepath.Join(d.WebRoot, "certs")
	entries, _ := os.ReadDir(root)
	var result []contract.SiteSummary
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(name, "_cert.pem") {
			continue
		}
		domain := strings.TrimSuffix(name, "_cert.pem")
		if represented[domain] || !validDomain(domain) {
			continue
		}
		result = append(result, orphanSite(domain, "orphan_certificate", filepath.Join(root, name), now))
		represented[domain] = true
	}
	return result
}

func orphanSite(domain, kind, path string, now time.Time) contract.SiteSummary {
	return contract.SiteSummary{
		ID:              stableID("site", domain, path),
		PrimaryDomain:   domain,
		Domains:         []string{domain},
		Kind:            contract.SiteUnknown,
		Enabled:         false,
		Health:          "orphaned",
		TLS:             contract.TLSStatus{Status: "unknown"},
		Origin:          contract.OriginDiscovered,
		Consistency:     contract.ConsistencyReadOnly,
		ResourceVersion: stableID("version", kind, path),
		AllowedActions:  []string{},
		Artifacts:       []contract.Artifact{{Kind: kind, Path: path}},
		Warnings:        []string{"存在孤立产物但未找到可关联的 Nginx 站点配置"},
		ReconciledAt:    now,
	}
}

func parseDirectives(clean string) map[string][]string {
	result := make(map[string][]string)
	for _, match := range directivePattern.FindAllStringSubmatch(clean, -1) {
		result[strings.ToLower(match[1])] = append(result[strings.ToLower(match[1])], strings.TrimSpace(match[2]))
	}
	return result
}

func stripComments(input string) string {
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		var quote rune
		escaped := false
		for pos, r := range line {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if quote != 0 {
				if r == quote {
					quote = 0
				}
				continue
			}
			if r == '\'' || r == '"' {
				quote = r
				continue
			}
			if r == '#' {
				lines[i] = line[:pos]
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}

func resolveProxyTargets(clean string, proxies []string) []string {
	upstreams := make(map[string][]string)
	for _, match := range upstreamPattern.FindAllStringSubmatch(clean, -1) {
		for _, directive := range directivePattern.FindAllStringSubmatch(match[2], -1) {
			if strings.EqualFold(directive[1], "server") {
				target := strings.Fields(directive[2])
				if len(target) > 0 {
					upstreams[match[1]] = append(upstreams[match[1]], sanitizeTarget(target[0]))
				}
			}
		}
	}
	var result []string
	for _, raw := range proxies {
		target := sanitizeTarget(strings.TrimSpace(raw))
		name := strings.TrimPrefix(strings.TrimPrefix(target, "http://"), "https://")
		if resolved := upstreams[name]; len(resolved) > 0 {
			result = append(result, resolved...)
		} else if target != "" && !strings.ContainsAny(target, "$ \t\r\n") {
			result = append(result, target)
		}
	}
	return uniqueStrings(result)
}

func sanitizeTarget(raw string) string {
	raw = strings.TrimSpace(strings.Trim(raw, `"'`))
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" {
		return raw
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func validDomains(values []string) []string {
	var result []string
	for _, value := range values {
		value = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(value), "."))
		if validDomain(value) {
			result = append(result, value)
		}
	}
	return uniqueStrings(result)
}

func validDomain(value string) bool {
	plain := strings.TrimPrefix(value, "*.")
	return net.ParseIP(plain) != nil || (len(plain) <= 253 && domainPattern.MatchString(value))
}

func choosePrimary(domains []string, fallback string) string {
	for _, domain := range domains {
		if !strings.HasPrefix(domain, "*.") {
			return domain
		}
	}
	if len(domains) > 0 {
		return domains[0]
	}
	if validDomain(fallback) {
		return fallback
	}
	return ""
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func stableID(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:16])
}
