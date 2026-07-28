package panel

import "testing"

func TestAllowedWebEnvironmentPaths(t *testing.T) {
	valid := map[string]string{
		"/api/v1/web-environment":                                                "/v1/web-environment",
		"/api/v1/web-environment/catalog":                                        "/v1/web-environment/catalog",
		"/api/v1/web-environment/backups":                                        "/v1/web-environment/backups",
		"/api/v1/web-environment/jobs":                                           "/v1/web-environment/jobs",
		"/api/v1/web-environment/jobs/0123456789abcdef0123456789abcdef":          "/v1/web-environment/jobs/0123456789abcdef0123456789abcdef",
		"/api/v1/web-environment/jobs/0123456789abcdef0123456789abcdef/terminal": "/v1/web-environment/jobs/0123456789abcdef0123456789abcdef/terminal",
		"/api/v1/web-environment/jobs/0123456789abcdef0123456789abcdef/input":    "/v1/web-environment/jobs/0123456789abcdef0123456789abcdef/input",
		"/api/v1/web-environment/backups/web_20260728112233.tar.gz":              "/v1/web-environment/backups/web_20260728112233.tar.gz",
	}
	for publicPath, expected := range valid {
		got, ok := allowedAgentPath(publicPath)
		if !ok || got != expected {
			t.Fatalf("allowedAgentPath(%q) = %q, %v; want %q", publicPath, got, ok, expected)
		}
	}
}

func TestWebEnvironmentPathsRejectTraversalAndUnknownActions(t *testing.T) {
	for _, path := range []string{
		"/api/v1/web-environment/backups/../../etc/shadow",
		"/api/v1/web-environment/backups/web_latest.tar.gz",
		"/api/v1/web-environment/jobs/not-a-job",
		"/api/v1/web-environment/jobs/0123456789abcdef0123456789abcdef/stop",
	} {
		if got, ok := allowedAgentPath(path); ok {
			t.Fatalf("allowedAgentPath(%q) unexpectedly allowed %q", path, got)
		}
	}
}
