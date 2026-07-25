package sites

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func TestDiscoverStaticReverseAndUnknown(t *testing.T) {
	root := t.TempDir()
	copyFixtureTree(t, filepath.Join("testdata", "web"), root)
	d := &Discoverer{WebRoot: root, Now: func() time.Time { return time.Unix(1_700_000_000, 0) }}
	got, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	byDomain := make(map[string]contract.SiteSummary)
	for _, site := range got {
		byDomain[site.PrimaryDomain] = site
	}
	if site := byDomain["static.example.com"]; site.Kind != contract.SiteStatic ||
		site.DocumentRoot != filepath.Join(root, "html", "static.example.com") ||
		site.ResourceVersion == "" {
		t.Fatalf("unexpected static site: %#v", site)
	}
	if site := byDomain["proxy.example.com"]; site.Kind != contract.SiteReverseProxy ||
		site.Target != "127.0.0.1:8080, 127.0.0.1:8081" {
		t.Fatalf("unexpected proxy site: %#v", site)
	}
	if site := byDomain["unknown.example.com"]; site.Kind != contract.SiteUnknown ||
		site.Consistency != contract.ConsistencyReadOnly || len(site.AllowedActions) != 0 {
		t.Fatalf("unexpected unknown site: %#v", site)
	}
}

func TestConfigHashChangesResourceVersion(t *testing.T) {
	root := t.TempDir()
	copyFixtureTree(t, filepath.Join("testdata", "web"), root)
	d := NewDiscoverer(root)
	before, _ := d.Discover()
	path := filepath.Join(root, "conf.d", "static.example.com.conf")
	if err := os.WriteFile(path, []byte("server { server_name static.example.com; root /var/www/html/changed; }\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	after, _ := d.Discover()
	if findSite(before, "static.example.com").ResourceVersion == findSite(after, "static.example.com").ResourceVersion {
		t.Fatal("resourceVersion did not change with config")
	}
}

func findSite(sites []contract.SiteSummary, domain string) contract.SiteSummary {
	for _, site := range sites {
		if site.PrimaryDomain == domain {
			return site
		}
	}
	return contract.SiteSummary{}
}

func copyFixtureTree(t *testing.T, source, target string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, _ := filepath.Rel(source, path)
		dest := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0o750)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0o640)
	})
	if err != nil {
		t.Fatal(err)
	}
}
