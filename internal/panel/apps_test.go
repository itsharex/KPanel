package panel

import (
	"strings"
	"testing"
)

func TestAllowedAppActionPath(t *testing.T) {
	id := "builtin-64"
	path, gotID, action, ok := allowedAppActionPath("/api/v1/apps/" + id + "/install")
	if !ok || gotID != id || action != "install" || path != "/v1/apps/"+id+"/install" {
		t.Fatalf("valid app path rejected: %q %q %q %v", path, gotID, action, ok)
	}
	for _, value := range []string{
		"/api/v1/apps/not valid/install",
		"/api/v1/apps/" + id + "/exec",
		"/api/v1/apps/" + id + "/install/extra",
	} {
		if _, _, _, ok := allowedAppActionPath(value); ok {
			t.Fatalf("unsafe app path accepted: %s", value)
		}
	}
}

func TestValidateAppActionInput(t *testing.T) {
	validVersion := "sha256:" + strings.Repeat("a", 64)
	if field, _ := validateAppActionInput("install", appActionInput{
		HostPort:   optionalInt{Value: 80, Set: true},
		AccessMode: optionalString{Value: "domain_only", Set: true},
	}); field != "" {
		t.Fatalf("script-compatible privileged port was rejected on %s", field)
	}
	if field, _ := validateAppActionInput("update", appActionInput{
		ResourceVersion: optionalString{Value: validVersion, Set: true},
	}); field != "" {
		t.Fatalf("valid update rejected on %s", field)
	}
	if field, _ := validateAppActionInput("direct_access", appActionInput{
		ResourceVersion: optionalString{Value: validVersion, Set: true},
		AccessMode:      optionalString{Value: "public", Set: true},
	}); field != "accessMode" {
		t.Fatalf("unsafe access mode rejected on %q", field)
	}
}
