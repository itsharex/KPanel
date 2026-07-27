package appmarket

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

func TestInventoryUsesDockerAppServiceAsMainContainer(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "appno.txt"), []byte("81\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	docker := &fakeDocker{containers: []contract.ContainerSummary{{
		ID: strings.Repeat("c", 64), Name: "jitsi-web-1",
		Image: "jitsi/web:latest", State: "running", Status: "Up",
		Ports: []contract.PortBinding{{
			PrivatePort: 80, PublicPort: 8081, IP: "0.0.0.0", Type: "tcp",
		}},
		ResourceVersion: "sha256:" + strings.Repeat("d", 64),
		AllowedActions:  []string{"stop", "restart"},
	}}}
	service, err := New(docker, root)
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeJobRunner{}
	if err := service.configureJobs(
		filepath.Join(root, "jobs"),
		filepath.Join(root, "kejilion-agent"),
		runner,
	); err != nil {
		t.Fatal(err)
	}
	service.scriptFinder = func() (string, error) { return "/usr/local/bin/k", nil }
	service.scriptManageFinder = func() (string, error) { return "/usr/local/bin/k", nil }
	item, err := service.Find(context.Background(), "builtin-81")
	if err != nil {
		t.Fatal(err)
	}
	if !item.Runtime.Installed || item.Runtime.ContainerName != "jitsi-web-1" ||
		!item.Capabilities["update"].Enabled {
		t.Fatalf("docker_app_service alias was not manageable: %#v", item)
	}
}

func TestRemoteCatalogDynamicallyReplacesThirdPartyEntries(t *testing.T) {
	embedded, _, _, err := LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	payload := remotePayloadFromCatalog(embedded)
	removed := ""
	apps := make([]App, 0, len(payload.Apps))
	for _, app := range payload.Apps {
		if app.Source == "thirdparty" && removed == "" {
			removed = app.Token
			continue
		}
		apps = append(apps, app)
	}
	apps = append(apps, App{
		ID: "thirdparty-new-safe-app", Source: "thirdparty", Token: "new-safe-app",
		NameZH: "新入驻应用", NameEN: "New Safe App", Description: "动态目录测试",
		DescriptionEN: "Dynamic catalog test", Category: "commtools",
		Website: "https://example.com", Icon: "icons/new-safe-app.webp", Slug: "new-safe-app",
	})
	payload.Apps = apps
	payload.Meta.ThirdParty = len(apps) - payload.Meta.Builtin
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	remote, err := decodeRemoteCatalog(
		[]byte("<script>window.__APPS__ = " + string(encoded) + ";\n  </script>"),
	)
	if err != nil {
		t.Fatal(err)
	}
	merged := mergeRemoteThirdParty(embedded, remote)
	foundNew := false
	foundRemoved := false
	for _, app := range merged.Apps {
		if app.Token == "new-safe-app" {
			foundNew = app.Icon == genericThirdPartyIcon && app.IconSHA256 == ""
		}
		if app.Token == removed {
			foundRemoved = true
		}
	}
	if !foundNew || foundRemoved || len(merged.Apps) != len(embedded.Apps) {
		t.Fatalf(
			"dynamic merge failed: new=%v removedStillPresent=%v count=%d",
			foundNew, foundRemoved, len(merged.Apps),
		)
	}
}

func TestRemoteCatalogFallsBackToLastKnownGood(t *testing.T) {
	embedded, _, _, err := LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	fetcher := func(context.Context) (Catalog, error) {
		if calls.Add(1) == 1 {
			return embedded, nil
		}
		return Catalog{}, errors.New("upstream unavailable")
	}
	service, err := newService(&fakeDocker{}, t.TempDir(), fetcher)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	first := service.currentCatalog(context.Background())
	if first.Mode != "embedded" || first.Warning != "" {
		t.Fatalf("cold catalog should return embedded immediately: %#v", first)
	}
	first = waitForCatalogState(t, service, "live", "")
	if calls.Load() != 1 {
		t.Fatalf("unexpected first refresh calls=%d", calls.Load())
	}
	now = now.Add(remoteCatalogTTL + time.Second)
	second := service.currentCatalog(context.Background())
	if second.Mode != "cached" {
		t.Fatalf("stale catalog should remain immediately usable: %#v", second)
	}
	second = waitForCatalogState(t, service, "cached", "warning")
	if second.Warning == "" || calls.Load() != 2 {
		t.Fatalf("unexpected cached catalog state: %#v calls=%d", second, calls.Load())
	}
}

func TestRemoteCatalogColdReadDoesNotWaitForNetwork(t *testing.T) {
	embedded, _, _, err := LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	fetcher := func(context.Context) (Catalog, error) {
		close(started)
		<-release
		return embedded, nil
	}
	service, err := newService(&fakeDocker{}, t.TempDir(), fetcher)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	snapshot := service.currentCatalog(context.Background())
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("cold catalog read blocked for %s", elapsed)
	}
	if snapshot.Mode != "embedded" {
		t.Fatalf("cold catalog mode = %s", snapshot.Mode)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("catalog refresh did not start")
	}
	close(release)
	_ = waitForCatalogState(t, service, "live", "")
}

func TestCompatibilityFilesReconcileScriptAndManualDrift(t *testing.T) {
	root := t.TempDir()
	service := &Service{appRoot: root}
	spec := declarativeSpecs["speedtest"]
	item := Summary{App: App{Num: 28, Token: spec.Token}}
	portPath := filepath.Join(root, spec.ContainerName+"_port.conf")
	markerPath := filepath.Join(root, "appno.txt")
	if err := os.WriteFile(portPath, []byte("changed-by-script\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(markerPath, []byte("custom marker from script\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := service.addCompatibilityFiles(item, spec, 18028); err != nil {
		t.Fatal(err)
	}
	portData, err := os.ReadFile(portPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(portData) != "18028\n" {
		t.Fatalf("port compatibility data = %q", portData)
	}
	markerData, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(markerData) != "custom marker from script\n28\n" {
		t.Fatalf("marker compatibility data = %q", markerData)
	}

	if err := os.WriteFile(portPath, []byte("changed-again\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := service.removeCompatibilityFiles(item, spec); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(portPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("port compatibility artifact still exists: %v", err)
	}
	markerData, err = os.ReadFile(markerPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(markerData) != "custom marker from script\n" {
		t.Fatalf("unrelated script marker was changed: %q", markerData)
	}
}

func waitForCatalogState(t *testing.T, service *Service, mode, warning string) catalogSnapshot {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snapshot := service.currentCatalog(context.Background())
		if snapshot.Mode == mode && (warning == "" || snapshot.Warning != "") {
			return snapshot
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("catalog did not reach mode %s", mode)
	return catalogSnapshot{}
}

func remotePayloadFromCatalog(catalog Catalog) remoteCatalogPayload {
	payload := remoteCatalogPayload{
		Meta:       remoteCatalogMeta{Builtin: 115, Source: catalog.Upstream},
		Categories: append([]Category(nil), catalog.Categories...),
		Apps:       append([]App(nil), catalog.Apps...),
	}
	for index := range payload.Apps {
		app := &payload.Apps[index]
		app.Icon = "icons/" + app.Slug + ".webp"
		app.IconSHA256 = ""
		if app.Source == "thirdparty" {
			payload.Meta.ThirdParty++
		}
	}
	return payload
}
