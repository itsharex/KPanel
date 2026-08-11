package panel

import (
	"path/filepath"
	"testing"
)

func TestLoadConfigAllowIPHostsEnvironment(t *testing.T) {
	t.Setenv("KEJILION_PANEL_CONFIG", "")
	t.Setenv("KEJILION_PANEL_ALLOW_IP_HOSTS", "true")
	config, err := LoadConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if !config.AllowIPHosts {
		t.Fatal("KEJILION_PANEL_ALLOW_IP_HOSTS did not enable literal IP hosts")
	}

	t.Setenv("KEJILION_PANEL_ALLOW_IP_HOSTS", "invalid")
	if _, err := LoadConfig(""); err == nil {
		t.Fatal("invalid KEJILION_PANEL_ALLOW_IP_HOSTS was accepted")
	}
}

func TestConfigRejectsNonOriginPublicURLs(t *testing.T) {
	t.Parallel()
	for _, publicURL := range []string{
		"https://user@panel.example.com",
		"https://panel.example.com/path",
		"https://panel.example.com?query=1",
		"https://panel.example.com#fragment",
		"https://panel.example.com:65536",
	} {
		config := validTestConfig()
		config.PublicURL = publicURL
		if err := config.Validate(); err == nil {
			t.Errorf("Validate accepted %q", publicURL)
		}
	}
}

func TestConfigAcceptsHTTPSOrigin(t *testing.T) {
	t.Parallel()
	config := validTestConfig()
	config.PublicURL = "https://panel.example.com:8443"
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate rejected HTTPS origin: %v", err)
	}
}

func TestConfigRejectsWebRootSecretOverlap(t *testing.T) {
	t.Parallel()
	for name, mutate := range map[string]func(*Config){
		"relative": func(config *Config) {
			config.WebRoot = "relative/web"
		},
		"data child": func(config *Config) {
			config.WebRoot = "/var/lib/kejilion-panel/web"
		},
		"data parent": func(config *Config) {
			config.WebRoot = "/var/lib"
		},
		"agent token parent": func(config *Config) {
			config.WebRoot = "/run"
		},
	} {
		t.Run(name, func(t *testing.T) {
			config := validTestConfig()
			mutate(&config)
			if err := config.Validate(); err == nil {
				t.Fatal("Validate accepted an unsafe webRoot")
			}
		})
	}
}

func TestConfigRejectsUnsafeTOTPKeyPaths(t *testing.T) {
	t.Parallel()
	for name, keyPath := range map[string]string{
		"relative":        "totp-encryption.key",
		"store collision": "/var/lib/kejilion-panel/panel-state.json",
		"web root child":  "/app/web/totp-encryption.key",
	} {
		t.Run(name, func(t *testing.T) {
			config := validTestConfig()
			config.TOTPKeyPath = keyPath
			if err := config.Validate(); err == nil {
				t.Fatal("Validate accepted an unsafe TOTP encryption key path")
			}
		})
	}
}

func TestConfigRejectsUnsafeCookieNames(t *testing.T) {
	t.Parallel()
	for _, cookieName := range []string{
		"__Host-invalid cookie",
		"__Host-kejilion_csrf",
		"__Host-invalid;cookie",
	} {
		config := validTestConfig()
		config.CookieName = cookieName
		if err := config.Validate(); err == nil {
			t.Errorf("Validate accepted cookie name %q", cookieName)
		}
	}
}

func TestConfigValidatesTrustedProxyCIDRs(t *testing.T) {
	t.Parallel()
	config := validTestConfig()
	config.TrustedProxyCIDRs = []string{"127.0.0.0/8", "172.17.0.0/16", "::1/128"}
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate rejected trusted proxy CIDRs: %v", err)
	}
	config.TrustedProxyCIDRs = []string{"not-a-cidr"}
	if err := config.Validate(); err == nil {
		t.Fatal("Validate accepted an invalid trusted proxy CIDR")
	}
}

func TestConfigRequiresIsolatedBrowserRelayOriginAndSecret(t *testing.T) {
	directory := t.TempDir()
	base := validTestConfig()
	base.DataDir = filepath.Join(directory, "data")
	base.StorePath = filepath.Join(base.DataDir, "panel-state.json")
	base.BootstrapTokenPath = filepath.Join(base.DataDir, "bootstrap.token")
	base.TOTPKeyPath = filepath.Join(base.DataDir, "totp.key")
	base.AgentSocket = filepath.Join(directory, "run", "agent.sock")
	base.AgentTokenFile = filepath.Join(directory, "run", "agent.token")
	base.WebRoot = filepath.Join(directory, "web")
	base.PublicURL = "https://panel.example.com"

	valid := base
	valid.BrowserRelayURL = "https://browser.example.com"
	valid.BrowserRelaySecretFile = filepath.Join(directory, "run", "browser.secret")
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid browser relay config rejected: %v", err)
	}

	for name, mutate := range map[string]func(*Config){
		"missing secret": func(config *Config) { config.BrowserRelayURL = "https://browser.example.com" },
		"missing URL":    func(config *Config) { config.BrowserRelaySecretFile = filepath.Join(directory, "browser.secret") },
		"same origin": func(config *Config) {
			config.BrowserRelayURL = config.PublicURL
			config.BrowserRelaySecretFile = filepath.Join(directory, "browser.secret")
		},
		"same default port origin": func(config *Config) {
			config.BrowserRelayURL = "https://PANEL.example.com:443"
			config.BrowserRelaySecretFile = filepath.Join(directory, "browser.secret")
		},
		"URL path": func(config *Config) {
			config.BrowserRelayURL = "https://browser.example.com/path"
			config.BrowserRelaySecretFile = filepath.Join(directory, "browser.secret")
		},
		"relative secret": func(config *Config) {
			config.BrowserRelayURL = "https://browser.example.com"
			config.BrowserRelaySecretFile = "browser.secret"
		},
	} {
		t.Run(name, func(t *testing.T) {
			config := base
			mutate(&config)
			if err := config.Validate(); err == nil {
				t.Fatal("invalid browser relay config accepted")
			}
		})
	}
}

func validTestConfig() Config {
	config := DefaultConfig()
	config.StorePath = "/var/lib/kejilion-panel/panel-state.json"
	config.BootstrapTokenPath = "/var/lib/kejilion-panel/bootstrap.token"
	config.TOTPKeyPath = "/var/lib/kejilion-panel/totp-encryption.key"
	config.CookieName = "__Host-kejilion_session"
	return config
}
