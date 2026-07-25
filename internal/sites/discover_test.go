package sites

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
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

func TestDiscoverReportsMissingArtifactDirectory(t *testing.T) {
	root := t.TempDir()
	copyFixtureTree(t, filepath.Join("testdata", "web"), root)
	if err := os.RemoveAll(filepath.Join(root, "html")); err != nil {
		t.Fatal(err)
	}
	if _, err := NewDiscoverer(root).Discover(); err == nil {
		t.Fatal("Discover() succeeded with a missing html directory")
	}
}

func TestTLSRejectsCertificateOutsideManagedRoot(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"conf.d", "html", "certs"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	discoverer := NewDiscoverer(root)
	status, artifact, _ := discoverer.discoverTLS(map[string][]string{
		"ssl_certificate": {"/etc/passwd"},
	}, "example.com")
	if status.Status != "untrusted_path" || artifact != nil {
		t.Fatalf("outside certificate status = %#v artifact=%#v", status, artifact)
	}
}

func TestTLSChecksHostnameAndPrivateKeyPresence(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"conf.d", "html", "certs"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	writeTestCertificate(t, filepath.Join(root, "certs"), "site.example.com", now)
	discoverer := &Discoverer{WebRoot: root, Now: func() time.Time { return now }}
	directives := map[string][]string{
		"ssl_certificate":     {"/etc/nginx/certs/site_cert.pem"},
		"ssl_certificate_key": {"/etc/nginx/certs/site_key.pem"},
	}
	status, _, _ := discoverer.discoverTLS(directives, "site.example.com")
	if status.Status != "valid" {
		t.Fatalf("matching certificate status = %#v", status)
	}
	status, _, _ = discoverer.discoverTLS(directives, "other.example.com")
	if status.Status != "hostname_mismatch" {
		t.Fatalf("mismatched certificate status = %#v", status)
	}
	directives["ssl_certificate_key"] = []string{"/etc/nginx/certs/missing_key.pem"}
	status, _, _ = discoverer.discoverTLS(directives, "site.example.com")
	if status.Status != "key_missing" {
		t.Fatalf("missing key status = %#v", status)
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
	if err := os.MkdirAll(filepath.Join(target, "certs"), 0o750); err != nil {
		t.Fatal(err)
	}
}

func writeTestCertificate(t *testing.T, root, domain string, now time.Time) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: domain},
		DNSNames:     []string{domain},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(filepath.Join(root, "site_cert.pem"), certPEM, 0o640); err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(filepath.Join(root, "site_key.pem"), keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
}
