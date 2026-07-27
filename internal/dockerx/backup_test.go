package dockerx

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRestoreDockerBackupMergesScriptMarkersWithoutOverwritingProjects(t *testing.T) {
	client := &Client{
		stateRoot: t.TempDir(),
		appRoot:   t.TempDir(),
		now:       time.Now,
	}
	if err := os.WriteFile(filepath.Join(client.appRoot, "appno.txt"), []byte("kpanel\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	id := "docker-20260727T120000Z-deadbeef.tar.gz"
	backupRoot := client.dockerBackupRoot()
	if err := os.MkdirAll(backupRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	writeDockerBackupFixture(t, filepath.Join(backupRoot, id), map[string]string{
		"docker/appno.txt":                  "101\n102\n",
		"docker/example/docker-compose.yml": "services:\n  app:\n    image: nginx:alpine\n",
	})
	if err := client.restoreDockerBackup(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	markers, err := os.ReadFile(filepath.Join(client.appRoot, "appno.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(markers) != "kpanel\n101\n102\n" {
		t.Fatalf("merged app markers = %q", markers)
	}
	compose, err := os.ReadFile(filepath.Join(client.appRoot, "example", "docker-compose.yml"))
	if err != nil || !strings.Contains(string(compose), "nginx:alpine") {
		t.Fatalf("restored Compose project = %q, %v", compose, err)
	}
}

func TestRestoreDockerBackupRejectsExistingProjectBeforeWriting(t *testing.T) {
	client := &Client{stateRoot: t.TempDir(), appRoot: t.TempDir(), now: time.Now}
	if err := os.Mkdir(filepath.Join(client.appRoot, "example"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(client.appRoot, "example", "existing"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	id := "docker-20260727T120001Z-cafebabe.tar.gz"
	backupRoot := client.dockerBackupRoot()
	if err := os.MkdirAll(backupRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	writeDockerBackupFixture(t, filepath.Join(backupRoot, id), map[string]string{
		"docker/example/new": "reject",
		"docker/other/file":  "must not be written",
	})
	err := client.restoreDockerBackup(context.Background(), id)
	if err == nil || !strings.Contains(err.Error(), "restore conflict") {
		t.Fatalf("restore conflict error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(client.appRoot, "other")); !os.IsNotExist(statErr) {
		t.Fatalf("restore wrote another project before rejecting conflict: %v", statErr)
	}
	data, readErr := os.ReadFile(filepath.Join(client.appRoot, "example", "existing"))
	if readErr != nil || string(data) != "keep" {
		t.Fatalf("existing project changed: %q, %v", data, readErr)
	}
}

func TestExtractDockerBackupRejectsLinks(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "unsafe.tar.gz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: "docker/example/link", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd", Mode: 0o777,
	}); err != nil {
		t.Fatal(err)
	}
	_ = tarWriter.Close()
	_ = gzipWriter.Close()
	_ = file.Close()
	if _, err := extractDockerBackup(context.Background(), path, t.TempDir()); err == nil ||
		!strings.Contains(err.Error(), "links") {
		t.Fatalf("unsafe link error = %v", err)
	}
}

func writeDockerBackupFixture(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, content := range entries {
		data := []byte(content)
		if err := tarWriter.WriteHeader(&tar.Header{
			Name: name, Typeflag: tar.TypeReg, Mode: 0o600, Size: int64(len(data)),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
