package panel

import (
	"strings"
	"testing"
)

func TestDiagnosticRoutesOnlyAcceptFixedIdentifiers(t *testing.T) {
	id := strings.Repeat("a", 32)
	for publicPath, expected := range map[string]string{
		"/api/v1/diagnostics":           "/v1/diagnostics",
		"/api/v1/diagnostic-jobs":       "/v1/diagnostic-jobs",
		"/api/v1/diagnostic-jobs/" + id: "/v1/diagnostic-jobs/" + id,
	} {
		path, ok := allowedAgentPath(publicPath)
		if !ok || path != expected {
			t.Fatalf("allowedAgentPath(%q) = %q, %v", publicPath, path, ok)
		}
	}
	for _, value := range []string{
		"/api/v1/diagnostic-jobs/",
		"/api/v1/diagnostic-jobs/" + strings.Repeat("a", 31),
		"/api/v1/diagnostic-jobs/" + id + "/extra",
		"/api/v1/diagnostic-jobs/" + strings.Repeat("A", 32),
	} {
		if _, ok := allowedAgentPath(value); ok {
			t.Fatalf("unsafe diagnostic route accepted: %s", value)
		}
	}
	for _, value := range []string{"chatgpt", "ip-quality", "nodequality"} {
		if !diagnosticCheckIDPattern.MatchString(value) {
			t.Fatalf("valid diagnostic check ID rejected: %s", value)
		}
	}
	for _, value := range []string{"", "../root", "yabs;id", strings.Repeat("a", 49)} {
		if diagnosticCheckIDPattern.MatchString(value) {
			t.Fatalf("unsafe diagnostic check ID accepted: %s", value)
		}
	}
}
