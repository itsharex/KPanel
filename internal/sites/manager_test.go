//go:build linux

package sites

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

type fakeNginxController struct {
	mu         sync.Mutex
	readyErr   error
	testErrs   []error
	reloadErrs []error
	tests      int
	reloads    int
}

func (f *fakeNginxController) NginxReady(context.Context) error {
	return f.readyErr
}

func (f *fakeNginxController) NginxTest(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	index := f.tests
	f.tests++
	if index < len(f.testErrs) {
		return f.testErrs[index]
	}
	return nil
}

func (f *fakeNginxController) NginxReload(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	index := f.reloads
	f.reloads++
	if index < len(f.reloadErrs) {
		return f.reloadErrs[index]
	}
	return nil
}

func TestCreateStaticProducesCanonicalDiscoverableArtifacts(t *testing.T) {
	manager, nginx, root := newTestManager(t)
	enabled := true
	created, err := manager.Create(context.Background(), SiteInput{
		PrimaryDomain: "example.com",
		Aliases:       []string{"www.example.com"},
		Type:          "static",
		Enabled:       &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Origin != contract.OriginWeb || !containsString(created.AllowedActions, "update") {
		t.Fatalf("created site is not managed: %#v", created)
	}
	configPath := filepath.Join(root, "conf.d", "example.com.conf")
	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(config, []byte(managedMarker+"\n")) {
		t.Fatalf("managed marker missing: %s", config)
	}
	index, err := os.ReadFile(filepath.Join(root, "html", "example.com", "index.html"))
	if err != nil || !bytes.Equal(index, renderDefaultIndex("example.com")) {
		t.Fatalf("unexpected default index: err=%v body=%s", err, index)
	}
	discovered, err := NewDiscoverer(root).Discover()
	if err != nil {
		t.Fatal(err)
	}
	if len(discovered) != 1 || discovered[0].ID != created.ID ||
		discovered[0].ResourceVersion != created.ResourceVersion ||
		discovered[0].Origin != contract.OriginWeb {
		t.Fatalf("bidirectional discovery mismatch: %#v", discovered)
	}
	if nginx.tests != 2 || nginx.reloads != 1 {
		t.Fatalf("nginx calls tests=%d reloads=%d, want 2/1", nginx.tests, nginx.reloads)
	}
}

func TestCreateCollisionLeavesExistingArtifactsUnchanged(t *testing.T) {
	manager, nginx, root := newTestManager(t)
	path := filepath.Join(root, "conf.d", "collision.example.com.conf")
	original := []byte("server {\n    listen 80;\n    server_name collision.example.com;\n}\n")
	if err := os.WriteFile(path, original, 0o640); err != nil {
		t.Fatal(err)
	}
	_, err := manager.Create(context.Background(), SiteInput{
		PrimaryDomain: "collision.example.com", Type: "proxy", Upstream: "http://127.0.0.1:8080",
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("Create() error = %v, want conflict", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(after, original) {
		t.Fatalf("collision changed existing config: err=%v body=%q", readErr, after)
	}
	if nginx.reloads != 0 {
		t.Fatalf("collision triggered %d reloads", nginx.reloads)
	}
}

func TestConcurrentCreateDoesNotOverwriteExternalConfiguration(t *testing.T) {
	manager, nginx, root := newTestManager(t)
	configPath := filepath.Join(root, "conf.d", "concurrent.example.com.conf")
	external := []byte("# external concurrent create\nserver { listen 80; server_name concurrent.example.com; }\n")
	manager.testHook = func(stage, path string) {
		if stage == "before_config_publish" {
			if writeErr := os.WriteFile(path, external, 0o640); writeErr != nil {
				t.Errorf("inject external create: %v", writeErr)
			}
		}
	}
	_, err := manager.Create(context.Background(), SiteInput{
		PrimaryDomain: "concurrent.example.com", Type: "static",
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("Create() error = %v, want conflict", err)
	}
	after, readErr := os.ReadFile(configPath)
	if readErr != nil || !bytes.Equal(after, external) {
		t.Fatalf("external config was overwritten: err=%v body=%q", readErr, after)
	}
	if pathExists(filepath.Join(root, "html", "concurrent.example.com")) {
		t.Fatal("the transaction's static directory was not rolled back")
	}
	if nginx.reloads != 0 {
		t.Fatalf("concurrent conflict triggered %d reloads", nginx.reloads)
	}
}

func TestInvalidCandidateDoesNotReloadAndRollsBack(t *testing.T) {
	manager, nginx, root := newTestManager(t)
	nginx.testErrs = []error{nil, errors.New("candidate syntax error")}
	_, err := manager.Create(context.Background(), SiteInput{
		PrimaryDomain: "invalid.example.com", Type: "static",
	})
	if !errors.Is(err, ErrUnprocessable) {
		t.Fatalf("Create() error = %v, want unprocessable", err)
	}
	if pathExists(filepath.Join(root, "conf.d", "invalid.example.com.conf")) ||
		pathExists(filepath.Join(root, "html", "invalid.example.com")) {
		t.Fatal("failed candidate artifacts were not rolled back")
	}
	if nginx.reloads != 0 {
		t.Fatalf("invalid candidate triggered %d reloads", nginx.reloads)
	}
}

func TestExternalConcurrentUpdateIsRestoredWithoutOverwrite(t *testing.T) {
	manager, nginx, root := newTestManager(t)
	created, err := manager.Create(context.Background(), SiteInput{
		PrimaryDomain: "race.example.com", Type: "proxy", Upstream: "http://127.0.0.1:8080",
	})
	if err != nil {
		t.Fatal(err)
	}
	nginx.tests, nginx.reloads = 0, 0
	configPath := filepath.Join(root, "conf.d", "race.example.com.conf")
	external := []byte("# external concurrent edit\nserver { listen 80; server_name race.example.com; }\n")
	manager.testHook = func(stage, path string) {
		if stage == "before_exchange" {
			if writeErr := os.WriteFile(path, external, 0o640); writeErr != nil {
				t.Errorf("inject external edit: %v", writeErr)
			}
		}
	}
	_, err = manager.Update(context.Background(), created.ID, SiteInput{
		PrimaryDomain: "race.example.com", Aliases: []string{"www.race.example.com"},
		Type: "proxy", Upstream: "http://127.0.0.1:8080",
		ExpectedResourceVersion: created.ResourceVersion,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("Update() error = %v, want conflict", err)
	}
	after, readErr := os.ReadFile(configPath)
	if readErr != nil || !bytes.Equal(after, external) {
		t.Fatalf("external edit was overwritten: err=%v body=%q", readErr, after)
	}
	if nginx.reloads != 0 {
		t.Fatalf("concurrent conflict triggered %d reloads", nginx.reloads)
	}
}

func TestReloadFailureRestoresPreviousConfiguration(t *testing.T) {
	manager, nginx, root := newTestManager(t)
	created, err := manager.Create(context.Background(), SiteInput{
		PrimaryDomain: "rollback.example.com", Type: "proxy", Upstream: "http://127.0.0.1:8080",
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "conf.d", "rollback.example.com.conf")
	before, _ := os.ReadFile(path)
	nginx.tests, nginx.reloads = 0, 0
	nginx.testErrs = nil
	nginx.reloadErrs = []error{errors.New("reload failed"), nil}
	_, err = manager.Update(context.Background(), created.ID, SiteInput{
		PrimaryDomain:           "rollback.example.com",
		Aliases:                 []string{"www.rollback.example.com"},
		Type:                    "proxy",
		Upstream:                "http://127.0.0.1:8080",
		ExpectedResourceVersion: created.ResourceVersion,
	})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Update() error = %v, want unavailable after safe rollback", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(after, before) {
		t.Fatalf("previous config was not restored: err=%v", readErr)
	}
	if nginx.tests != 3 || nginx.reloads != 2 {
		t.Fatalf("rollback calls tests=%d reloads=%d, want 3/2", nginx.tests, nginx.reloads)
	}
}

func TestNonCanonicalManagedConfigCannotBeUpdated(t *testing.T) {
	manager, _, root := newTestManager(t)
	created, err := manager.Create(context.Background(), SiteInput{
		PrimaryDomain: "drift.example.com", Type: "proxy", Upstream: "http://127.0.0.1:8080",
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "conf.d", "drift.example.com.conf")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("# manual edit\n"); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	drifted, _ := os.ReadFile(path)
	discovered, discoverErr := manager.discoverer.Discover()
	if discoverErr != nil {
		t.Fatal(discoverErr)
	}
	if len(discovered) != 1 || discovered[0].Origin != contract.OriginWeb ||
		discovered[0].Consistency != contract.ConsistencyDrifted ||
		len(discovered[0].AllowedActions) != 0 {
		t.Fatalf("drifted Panel site ownership/action mismatch: %#v", discovered)
	}
	_, err = manager.Update(context.Background(), created.ID, SiteInput{
		PrimaryDomain: "drift.example.com", Type: "proxy", Upstream: "http://127.0.0.1:9090",
		ExpectedResourceVersion: created.ResourceVersion,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want forbidden", err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(after, drifted) {
		t.Fatal("non-canonical config was overwritten")
	}
}

func TestStrictSiteInputValidation(t *testing.T) {
	invalid := []SiteInput{
		{PrimaryDomain: "*.example.com", Type: "static"},
		{PrimaryDomain: "127.0.0.1", Type: "static"},
		{PrimaryDomain: "example.com/path", Type: "static"},
		{PrimaryDomain: "example.com ", Type: "static"},
		{PrimaryDomain: "example.com", Type: "proxy", Upstream: "http://user:pass@app:8080"},
		{PrimaryDomain: "example.com", Type: "proxy", Upstream: "http://app:8080/path"},
		{PrimaryDomain: "example.com", Type: "proxy", Upstream: "http://app.example:8080"},
		{PrimaryDomain: "example.com", Type: "proxy", Upstream: "http://8.8.8.8:53"},
		{PrimaryDomain: "example.com", Type: "proxy", Upstream: "http://127.0.0.1:0"},
	}
	for _, input := range invalid {
		if _, err := normalizeSiteInput(input); !errors.Is(err, ErrUnprocessable) {
			t.Errorf("normalizeSiteInput(%#v) error = %v, want unprocessable", input, err)
		}
	}
	valid := []string{
		"http://app:8080",
		"https://127.0.0.1",
		"http://10.0.0.2:65535",
		"http://172.16.0.2:80",
		"http://192.168.1.2:443",
		"http://[::1]:8080",
	}
	for _, upstream := range valid {
		if _, err := normalizeSiteInput(SiteInput{
			PrimaryDomain: "example.com", Type: "proxy", Upstream: upstream,
		}); err != nil {
			t.Errorf("valid upstream %q rejected: %v", upstream, err)
		}
	}
}

func newTestManager(t *testing.T) (*Manager, *fakeNginxController, string) {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"conf.d", "html", "certs"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	nginx := &fakeNginxController{}
	discoverer := NewDiscoverer(root)
	manager := NewManager(root, discoverer, nginx)
	return manager, nginx, root
}
