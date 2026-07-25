package panel

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Listen             string        `json:"listen"`
	DataDir            string        `json:"dataDir"`
	StorePath          string        `json:"storePath"`
	BootstrapTokenPath string        `json:"bootstrapTokenPath"`
	AgentSocket        string        `json:"agentSocket"`
	AgentTokenFile     string        `json:"agentTokenFile"`
	WebRoot            string        `json:"webRoot"`
	PublicURL          string        `json:"publicUrl"`
	SecureCookie       bool          `json:"secureCookie"`
	CookieName         string        `json:"cookieName"`
	SessionTTL         time.Duration `json:"-"`
	SessionTTLText     string        `json:"sessionTtl"`
	LoginWindow        time.Duration `json:"-"`
	LoginWindowText    string        `json:"loginWindow"`
	MaxLoginFailures   int           `json:"maxLoginFailures"`
	MaxRequestBytes    int64         `json:"maxRequestBytes"`
	MaxAgentBytes      int64         `json:"maxAgentBytes"`
}

func DefaultConfig() Config {
	return Config{
		Listen:           ":8080",
		DataDir:          "/var/lib/kejilion-panel",
		AgentSocket:      "/run/kejilion-panel/agent.sock",
		AgentTokenFile:   "/run/secrets/agent-token",
		WebRoot:          "/app/web",
		SecureCookie:     true,
		SessionTTL:       12 * time.Hour,
		SessionTTLText:   "12h",
		LoginWindow:      15 * time.Minute,
		LoginWindowText:  "15m",
		MaxLoginFailures: 5,
		MaxRequestBytes:  1 << 20,
		MaxAgentBytes:    8 << 20,
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
	applyStringEnv("KEJILION_PANEL_AGENT_SOCKET", &config.AgentSocket)
	applyStringEnv("KEJILION_PANEL_AGENT_TOKEN_FILE", &config.AgentTokenFile)
	applyStringEnv("KEJILION_PANEL_WEB_ROOT", &config.WebRoot)
	applyStringEnv("KEJILION_PANEL_PUBLIC_URL", &config.PublicURL)
	applyStringEnv("KEJILION_PANEL_COOKIE_NAME", &config.CookieName)
	applyStringEnv("KEJILION_PANEL_SESSION_TTL", &config.SessionTTLText)
	applyStringEnv("KEJILION_PANEL_LOGIN_WINDOW", &config.LoginWindowText)

	if value := strings.TrimSpace(os.Getenv("KEJILION_PANEL_SECURE_COOKIE")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse KEJILION_PANEL_SECURE_COOKIE: %w", err)
		}
		config.SecureCookie = parsed
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
	if config.CookieName == "" {
		if config.SecureCookie {
			config.CookieName = "__Host-kejilion_session"
		} else {
			config.CookieName = "kejilion_session"
		}
	}
	config.PublicURL = strings.TrimRight(strings.TrimSpace(config.PublicURL), "/")
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c Config) Validate() error {
	switch {
	case strings.TrimSpace(c.Listen) == "":
		return errors.New("listen address is required")
	case strings.TrimSpace(c.DataDir) == "" || !filepath.IsAbs(c.DataDir):
		return errors.New("dataDir must be absolute")
	case strings.TrimSpace(c.StorePath) == "" || !filepath.IsAbs(c.StorePath):
		return errors.New("storePath must be absolute")
	case strings.TrimSpace(c.BootstrapTokenPath) == "" || !filepath.IsAbs(c.BootstrapTokenPath):
		return errors.New("bootstrapTokenPath must be absolute")
	case strings.TrimSpace(c.AgentSocket) == "" || !filepath.IsAbs(c.AgentSocket):
		return errors.New("agentSocket must be absolute")
	case strings.TrimSpace(c.AgentTokenFile) == "" || !filepath.IsAbs(c.AgentTokenFile):
		return errors.New("agentTokenFile must be absolute")
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
	if c.SecureCookie && !strings.HasPrefix(c.CookieName, "__Host-") {
		return errors.New("secure cookie name must use the __Host- prefix")
	}
	if !c.SecureCookie && strings.HasPrefix(c.CookieName, "__Host-") {
		return errors.New("__Host- cookies require secureCookie=true")
	}
	if c.PublicURL != "" {
		parsed, err := url.Parse(c.PublicURL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return errors.New("publicUrl must be an absolute HTTP(S) URL")
		}
		if parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" {
			return errors.New("publicUrl must not contain a path, query, or fragment")
		}
		if c.SecureCookie && parsed.Scheme != "https" {
			return errors.New("secure cookies require an HTTPS publicUrl")
		}
	}
	return nil
}

func applyStringEnv(name string, target *string) {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		*target = value
	}
}
