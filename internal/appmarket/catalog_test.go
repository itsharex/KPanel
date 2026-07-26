package appmarket

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/dockerx"
)

type fakeDocker struct {
	containers []contract.ContainerSummary
}

func (f *fakeDocker) Containers(context.Context) ([]contract.ContainerSummary, error) {
	return append([]contract.ContainerSummary(nil), f.containers...), nil
}

func (f *fakeDocker) Lifecycle(context.Context, string, string, string) (dockerx.ActionResult, error) {
	return dockerx.ActionResult{}, nil
}

func (f *fakeDocker) LifecycleDeclarativeApp(
	context.Context,
	dockerx.DeclarativeAppSpec,
	string,
	string,
) (dockerx.ActionResult, error) {
	return dockerx.ActionResult{}, nil
}

func (f *fakeDocker) InstallDeclarativeApp(
	context.Context,
	dockerx.DeclarativeAppSpec,
	uint16,
	string,
) (dockerx.AppMutationResult, error) {
	return dockerx.AppMutationResult{}, nil
}

func (f *fakeDocker) UpdateDeclarativeApp(
	context.Context,
	dockerx.DeclarativeAppSpec,
	string,
) (dockerx.AppMutationResult, error) {
	return dockerx.AppMutationResult{}, nil
}

func (f *fakeDocker) SetDeclarativeAppAccess(
	context.Context,
	dockerx.DeclarativeAppSpec,
	string,
	string,
) (dockerx.AppMutationResult, error) {
	return dockerx.AppMutationResult{}, nil
}

func (f *fakeDocker) UninstallDeclarativeApp(
	context.Context,
	dockerx.DeclarativeAppSpec,
	string,
) (dockerx.AppMutationResult, error) {
	return dockerx.AppMutationResult{}, nil
}

func (f *fakeDocker) CheckContainerImageUpdate(
	context.Context,
	string,
	string,
) (dockerx.ImageUpdateResult, error) {
	return dockerx.ImageUpdateResult{}, nil
}

func TestEmbeddedCatalogMatchesAuditedApplicationMarket(t *testing.T) {
	catalog, legacy, scriptSHA256, err := LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Apps) != 146 || len(legacy) != 115 {
		t.Fatalf("catalog counts = %d/%d, want 146/115", len(catalog.Apps), len(legacy))
	}
	if !strings.HasPrefix(catalog.Source, "https://app.kejilion.sh") ||
		len(scriptSHA256) != 64 {
		t.Fatalf("catalog provenance is incomplete: source=%q hash=%q", catalog.Source, scriptSHA256)
	}
	icons := make(map[string]bool, len(catalog.Apps))
	for _, app := range catalog.Apps {
		if !strings.HasPrefix(app.Icon, "/app-icons/") || len(app.IconSHA256) != 64 {
			t.Fatalf("invalid local icon metadata for %s", app.ID)
		}
		if icons[app.Icon] {
			t.Fatalf("duplicate icon path %s", app.Icon)
		}
		icons[app.Icon] = true
	}
}

func TestInventoryCombinesDockerTruthAndScriptMarker(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "appno.txt"), []byte("28\n64\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	docker := &fakeDocker{containers: []contract.ContainerSummary{{
		ID: strings.Repeat("a", 64), Name: "speedtest",
		Image: "ghcr.io/librespeed/speedtest", State: "running", Status: "Up",
		Ports:  []contract.PortBinding{{PrivatePort: 8080, PublicPort: 8028, IP: "0.0.0.0", Type: "tcp"}},
		Mounts: []contract.Mount{}, ResourceVersion: "sha256:" + strings.Repeat("b", 64),
		AllowedActions: []string{"restart", "stop"},
	}}}
	service, err := New(docker, root)
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := service.Inventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Items) != 146 || inventory.Installed != 2 || inventory.Running != 1 {
		t.Fatalf("inventory counts are wrong: %#v", inventory)
	}
	var speedtest, itTools Summary
	for _, item := range inventory.Items {
		switch item.Token {
		case "speedtest":
			speedtest = item
		case "it-tools":
			itTools = item
		}
	}
	if speedtest.Runtime.State != "running" ||
		!speedtest.Capabilities["update"].Enabled ||
		!speedtest.Capabilities["direct_access"].Enabled {
		t.Fatalf("speedtest was not safely manageable: %#v", speedtest)
	}
	if itTools.Runtime.State != "unknown" ||
		itTools.Capabilities["start"].Enabled ||
		itTools.Runtime.Warning == "" {
		t.Fatalf("marker-only application was not degraded safely: %#v", itTools)
	}
}
