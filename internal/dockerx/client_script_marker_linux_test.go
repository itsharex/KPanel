//go:build linux

package dockerx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScriptPortMarkerMakesStandaloneContainerManageable(t *testing.T) {
	appRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(appRoot, "script-app_port.conf"), []byte("8080\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	client := &Client{
		appRoot: appRoot, webRoot: t.TempDir(), stateRoot: t.TempDir(), now: time.Now,
	}
	raw := managedInspect(strings.Repeat("8", 64), "2026-01-01T00:00:00Z", 0)
	raw.Name = "/script-app"
	raw.Config.Labels = map[string]string{}
	summary := client.summaryFromInspect(raw)
	if summary.Ownership != "kejilion" ||
		!contains(summary.AllowedActions, "restart") ||
		!contains(summary.AllowedActions, "logs") {
		t.Fatalf("script-owned standalone container stayed read-only: %#v", summary)
	}
	raw.HostConfig.Privileged = true
	summary = client.summaryFromInspect(raw)
	if !contains(summary.AllowedActions, "restart") ||
		!contains(summary.AllowedActions, "logs") ||
		!contains(summary.AllowedActions, "exec") {
		t.Fatalf("runtime configuration drift disabled script container management: %#v", summary)
	}
}
