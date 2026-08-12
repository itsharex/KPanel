package panel

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/browsercore"
)

var cookieNamePattern = regexp.MustCompile(`^[!#$%&'*+\-.^_` + "`" + `|~0-9A-Za-z]+$`)

type Config struct {
	Listen                  string        `json:"listen"`
	DataDir                 string        `json:"dataDir"`
	StorePath               string        `json:"storePath"`
	BootstrapTokenPath      string        `json:"bootstrapTokenPath"`
	TOTPKeyPath             string        `json:"totpKeyPath"`
	AgentSocket             string        `json:"agentSocket"`
	AgentTokenFile          string        `json:"agentTokenFile"`
	WebRoot                 string        `json:"webRoot"`
	PublicURL               string        `json:"publicUrl"`
	BrowserMode             string        `json:"browserMode"`
	BrowserRelayURL         string        `json:"browserRelayUrl"`
	BrowserRelayInternalURL string        `json:"browserRelayInternalUrl"`
	BrowserRelaySecretFile  string        `json:"browserRelaySecretFile"`
	AllowIPHosts            bool          `json:"allowIpHosts"`
	SecureCookie            bool          `json:"secureCookie"`
	CookieName              string        `json:"cookieName"`
	SessionTTL              time.Duration `json:"-"`
	SessionTTLText          string        `json:"sessionTtl"`
	LoginWindow             time.Duration `json:"-"`
	LoginWindowText         string        `json:"loginWindow"`
	MaxLoginFailures        int           `json:"maxLoginFailures"`
	MaxRequestBytes         int64         `json:"maxRequestBytes"`
	MaxAgentBytes           int64         `json:"maxAgentBytes"`
	TrustedProxyCIDRs       []string      `json:"trustedProxyCidrs"`
	ClusterPrivateCIDRs     []string      `json:"clusterPrivateCidrs"`
}

func DefaultConfig() Config {
	return Config{
		Listen:            ":8080",
		DataDir:           "/var/lib/kejilion-panel",
		AgentSocket:       "/run/kejilion-panel/agent.sock",
		AgentTokenFile:    "/run/secrets/agent-token",
		WebRoot:           "/app/web",
		BrowserMode:       browsercore.RuntimeModeDisabled,
		SecureCookie:      true,
		SessionTTL:        12 * time.Hour,
		SessionTTLText:    "12h",
		LoginWindow:       15 * time.Minute,
		LoginWindowText:   "15m",
		MaxLoginFailures:  5,
		MaxRequestBytes:   1 << 20,
		MaxAgentBytes:     8 << 20,
		TrustedProxyCIDRs: []string{"127.0.0.0/8", "::1/128"},
	}
}

// LoadConfig reads an optional JSON file first and then applies environment
// variables. Empty environment values do not override file/default values.
func LoadConfig(path string) (Config, error) {
	config := DefaultConfig()
	if path == "" {
		path = strings.TrimSpace(os.Getenv("KEJILION_PANEL_CONFIG"))
	}
	if path != "" {
		content, err := os.ReadFile(path)
		if err != nil {
			return Config{}, fmt.Errorf("read config file: %w", err)
		}
		if len(content) > 1<<20 {
			return Config{}, errors.New("config file exceeds 1 MiB")
		}
		decoder := json.NewDecoder(bytes.NewReader(content))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&config); err != nil {
			return Config{}, fmt.Errorf("decode config file: %w", err)
		}
	}

	applyStringEnv("KEJILION_PANEL_LISTEN", &config.Listen)
	applyStringEnv("KEJILION_PANEL_DATA_DIR", &config.DataDir)
	applyStringEnv("KEJILION_PANEL_STORE_PATH", &config.StorePath)
	applyStringEnv("KEJILION_PANEL_BOOTSTRAP_TOKEN_FILE", &config.BootstrapTokenPath)
	applyStringEnv("KEJILION_PANEL_TOTP_KEY_FILE", &config.TOTPKeyPath)
	applyStringEnv("KEJILION_PANEL_AGENT_SOCKET", &config.AgentSocket)
	applyStringEnv("KEJILION_PANEL_AGENT_TOKEN_FILE", &config.AgentTokenFile)
	applyStringEnv("KEJILION_PANEL_WEB_ROOT", &config.WebRoot)
	applyStringEnv("KEJILION_PANEL_PUBLIC_URL", &config.PublicURL)
	applyStringEnv("KEJILION_PANEL_BROWSER_MODE", &config.BrowserMode)
	applyStringEnv("KEJILION_PANEL_BROWSER_RELAY_URL", &config.BrowserRelayURL)
	applyStringEnv("KEJILION_PANEL_BROWSER_RELAY_INTERNAL_URL", &config.BrowserRelayInternalURL)
	applyStringEnv("KEJILION_PANEL_BROWSER_RELAY_SECRET_FILE", &config.BrowserRelaySecretFile)
	applyStringEnv("KEJILION_PANEL_COOKIE_NAME", &config.CookieName)
	applyStringEnv("KEJILION_PANEL_SESSION_TTL", &config.SessionTTLText)
	applyStringEnv("KEJILION_PANEL_LOGIN_WINDOW", &config.LoginWindowText)
	if value := strings.TrimSpace(os.Getenv("KEJILION_PANEL_TRUSTED_PROXY_CIDRS")); value != "" {
		config.TrustedProxyCIDRs = splitCommaSeparated(value)
	}
	if value := strings.TrimSpace(os.Getenv("KEJILION_PANEL_CLUSTER_PRIVATE_CIDRS")); value != "" {
		config.ClusterPrivateCIDRs = splitCommaSeparated(value)
	}

	if value := strings.TrimSpace(os.Getenv("KEJILION_PANEL_SECURE_COOKIE")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse KEJILION_PANEL_SECURE_COOKIE: %w", err)
		}
		config.SecureCookie = parsed
	}
	if value := strings.TrimSpace(os.Getenv("KEJILION_PANEL_ALLOW_IP_HOSTS")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse KEJILION_PANEL_ALLOW_IP_HOSTS: %w", err)
		}
		config.AllowIPHosts = parsed
	}
	if value := strings.TrimSpace(os.Getenv("KEJILION_PANEL_MAX_LOGIN_FAILURES")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse KEJILION_PANEL_MAX_LOGIN_FAILURES: %w", err)
		}
		config.MaxLoginFailures = parsed
	}

	var err error
	config.SessionTTL, err = time.ParseDuration(config.SessionTTLText)
	if err != nil {
		return Config{}, fmt.Errorf("parse sessionTtl: %w", err)
	}
	config.LoginWindow, err = time.ParseDuration(config.LoginWindowText)
	if err != nil {
		return Config{}, fmt.Errorf("parse loginWindow: %w", err)
	}

	config.DataDir = filepath.Clean(config.DataDir)
	if config.StorePath == "" {
		config.StorePath = filepath.Join(config.DataDir, "panel-state.json")
	}
	if config.BootstrapTokenPath == "" {
		config.BootstrapTokenPath = filepath.Join(config.DataDir, "bootstrap.token")
	}
	if config.TOTPKeyPath == "" {
		config.TOTPKeyPath = filepath.Join(config.DataDir, "totp-encryption.key")
	}
	if config.CookieName == "" {
		if config.SecureCookie {
			config.CookieName = "__Host-kejilion_session"
		} else {
			config.CookieName = "kejilion_session"
		}
	}
	config.PublicURL = strings.TrimRight(strings.TrimSpace(config.PublicURL), "/")
	config.BrowserMode = strings.ToLower(strings.TrimSpace(config.BrowserMode))
	config.BrowserRelayURL = strings.TrimRight(strings.TrimSpace(config.BrowserRelayURL), "/")
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	if config.BrowserRelayURL != "" {
		config.BrowserRelayURL, _ = browsercore.NormalizeOrigin(config.BrowserRelayURL)
	}
	return config, nil
}

func (c Config) Validate() error {
	mode, err := browsercore.NormalizeRuntimeMode(c.BrowserMode)
	if err != nil {
		return err
	}
	switch {
	case strings.TrimSpace(c.Listen) == "":
		return errors.New("listen address is required")
	case strings.TrimSpace(c.DataDir) == "" || !filepath.IsAbs(c.DataDir):
		return errors.New("dataDir must be absolute")
	case strings.TrimSpace(c.StorePath) == "" || !filepath.IsAbs(c.StorePath):
		return errors.New("storePath must be absolute")
	case strings.TrimSpace(c.BootstrapTokenPath) == "" || !filepath.IsAbs(c.BootstrapTokenPath):
		return errors.New("bootstrapTokenPath must be absolute")
	case strings.TrimSpace(c.TOTPKeyPath) != "" && !filepath.IsAbs(c.TOTPKeyPath):
		return errors.New("totpKeyPath must be absolute")
	case strings.TrimSpace(c.AgentSocket) == "" || !filepath.IsAbs(c.AgentSocket):
		return errors.New("agentSocket must be absolute")
	case strings.TrimSpace(c.AgentTokenFile) == "" || !filepath.IsAbs(c.AgentTokenFile):
		return errors.New("agentTokenFile must be absolute")
	case strings.TrimSpace(c.WebRoot) == "" || !filepath.IsAbs(c.WebRoot):
		return errors.New("webRoot must be absolute")
	case strings.TrimSpace(c.BrowserRelaySecretFile) != "" && !filepath.IsAbs(c.BrowserRelaySecretFile):
		return errors.New("browserRelaySecretFile must be absolute")
	case c.SessionTTL < 5*time.Minute || c.SessionTTL > 7*24*time.Hour:
		return errors.New("sessionTtl must be between 5 minutes and 7 days")
	case c.LoginWindow < time.Minute || c.LoginWindow > 24*time.Hour:
		return errors.New("loginWindow must be between 1 minute and 24 hours")
	case c.MaxLoginFailures < 1 || c.MaxLoginFailures > 100:
		return errors.New("maxLoginFailures must be between 1 and 100")
	case c.MaxRequestBytes < 1024 || c.MaxRequestBytes > 16<<20:
		return errors.New("maxRequestBytes must be between 1 KiB and 16 MiB")
	case c.MaxAgentBytes < 1024 || c.MaxAgentBytes > 64<<20:
		return errors.New("maxAgentBytes must be between 1 KiB and 64 MiB")
	}
	protectedPaths := map[string]string{
		"dataDir":            c.DataDir,
		"storePath":          c.StorePath,
		"bootstrapTokenPath": c.BootstrapTokenPath,
		"agentTokenFile":     c.AgentTokenFile,
	}
	if c.TOTPKeyPath != "" {
		protectedPaths["totpKeyPath"] = c.TOTPKeyPath
	}
	if c.BrowserRelaySecretFile != "" {
		protectedPaths["browserRelaySecretFile"] = c.BrowserRelaySecretFile
	}
	for label, protectedPath := range protectedPaths {
		if pathsOverlap(c.WebRoot, protectedPath) {
			return fmt.Errorf("webRoot must not overlap %s", label)
		}
	}
	if samePath(c.StorePath, c.BootstrapTokenPath) ||
		(c.TOTPKeyPath != "" && samePath(c.StorePath, c.TOTPKeyPath)) ||
		samePath(c.StorePath, c.AgentTokenFile) ||
		(c.TOTPKeyPath != "" && samePath(c.BootstrapTokenPath, c.TOTPKeyPath)) ||
		samePath(c.BootstrapTokenPath, c.AgentTokenFile) ||
		(c.TOTPKeyPath != "" && samePath(c.TOTPKeyPath, c.AgentTokenFile)) ||
		(c.BrowserRelaySecretFile != "" && (samePath(c.BrowserRelaySecretFile, c.StorePath) ||
			samePath(c.BrowserRelaySecretFile, c.BootstrapTokenPath) ||
			samePath(c.BrowserRelaySecretFile, c.AgentTokenFile) ||
			(c.TOTPKeyPath != "" && samePath(c.BrowserRelaySecretFile, c.TOTPKeyPath)))) {
		return errors.New("store and secret paths must be distinct")
	}
	if (c.BrowserRelayURL == "") != (c.BrowserRelaySecretFile == "") {
		return errors.New("browserRelayUrl and browserRelaySecretFile must be configured together")
	}
	if browsercore.RuntimeModeUsesRelay(c.BrowserMode) && c.BrowserRelayURL == "" {
		return errors.New("browser reader and beta modes require browserRelayUrl and browserRelaySecretFile")
	}
	if mode == browsercore.RuntimeModeReader && c.BrowserRelayInternalURL == "" {
		return errors.New("browser reader mode requires browserRelayInternalUrl")
	}
	if !cookieNamePattern.MatchString(c.CookieName) {
		return errors.New("cookieName is invalid")
	}
	if c.SecureCookie && !strings.HasPrefix(c.CookieName, "__Host-") {
		return errors.New("secure cookie name must use the __Host- prefix")
	}
	if !c.SecureCookie && strings.HasPrefix(c.CookieName, "__Host-") {
		return errors.New("__Host- cookies require secureCookie=true")
	}
	csrfCookieName := "kejilion_csrf"
	if c.SecureCookie {
		csrfCookieName = "__Host-kejilion_csrf"
	}
	if c.CookieName == csrfCookieName {
		return errors.New("session and CSRF cookie names must be distinct")
	}
	for _, cidr := range c.TrustedProxyCIDRs {
		if strings.TrimSpace(cidr) == "" {
			return errors.New("trustedProxyCidrs must not contain empty entries")
		}
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("invalid trusted proxy CIDR %q", cidr)
		}
	}
	if len(c.ClusterPrivateCIDRs) > 32 {
		return errors.New("clusterPrivateCidrs must contain at most 32 entries")
	}
	for _, cidr := range c.ClusterPrivateCIDRs {
		if strings.TrimSpace(cidr) == "" {
			return errors.New("clusterPrivateCidrs must not contain empty entries")
		}
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("invalid cluster private CIDR %q", cidr)
		}
	}
	if c.PublicURL != "" {
		parsed, err := url.Parse(c.PublicURL)
		if err != nil || parsed.Host == "" || parsed.Hostname() == "" ||
			(parsed.Scheme != "https" && parsed.Scheme != "http") {
			return errors.New("publicUrl must be an absolute HTTP(S) URL")
		}
		if parsed.User != nil || parsed.Opaque != "" || parsed.RawQuery != "" ||
			parsed.Fragment != "" || parsed.Path != "" || parsed.RawPath != "" || parsed.ForceQuery {
			return errors.New("publicUrl must be an origin without userinfo, path, query, or fragment")
		}
		if strings.ContainsAny(parsed.Host, "\r\n\t ") {
			return errors.New("publicUrl contains invalid host characters")
		}
		if port := parsed.Port(); port != "" {
			number, err := strconv.Atoi(port)
			if err != nil || number < 1 || number > 65535 {
				return errors.New("publicUrl port is invalid")
			}
		}
		if c.SecureCookie && parsed.Scheme != "https" {
			return errors.New("secure cookies require an HTTPS publicUrl")
		}
	}
	if c.BrowserRelayURL != "" {
		relayOrigin, err := browsercore.NormalizeOrigin(c.BrowserRelayURL)
		if err != nil {
			return errors.New("browserRelayUrl must be an HTTP(S) origin")
		}
		if c.PublicURL != "" {
			publicOrigin, publicErr := browsercore.NormalizeOrigin(c.PublicURL)
			if publicErr == nil && relayOrigin == publicOrigin {
				return errors.New("browserRelayUrl must use an isolated origin")
			}
		}
	}
	if c.BrowserRelayInternalURL != "" {
		if _, err := browsercore.NormalizeOrigin(c.BrowserRelayInternalURL); err != nil {
			return errors.New("browserRelayInternalUrl must be an HTTP(S) origin")
		}
	}
	if browsercore.RuntimeModeEnabled(c.BrowserMode) {
		if c.PublicURL == "" {
			return errors.New("browser beta mode requires publicUrl")
		}
		if !browsercore.SupportsServiceWorkerOrigin(c.PublicURL) ||
			!browsercore.SupportsServiceWorkerOrigin(c.BrowserRelayURL) {
			return errors.New("browser beta origins must support Service Worker secure contexts")
		}
		publicOrigin, _ := url.Parse(c.PublicURL)
		if publicOrigin.Scheme == "https" && !c.SecureCookie {
			return errors.New("browser beta HTTPS publicUrl requires secureCookie=true")
		}
	}
	return nil
}

func applyStringEnv(name string, target *string) {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		*target = value
	}
}

func splitCommaSeparated(value string) []string {
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, strings.TrimSpace(item))
	}
	return result
}

func samePath(left, right string) bool {
	relative, err := filepath.Rel(filepath.Clean(left), filepath.Clean(right))
	return err == nil && relative == "."
}

func pathsOverlap(left, right string) bool {
	return pathWithin(left, right) || pathWithin(right, left)
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil {
		return false
	}
	return relative == "." ||
		(relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}
