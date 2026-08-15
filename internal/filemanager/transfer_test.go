package filemanager

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDirectoryTransferRoundTripIsAtomic(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "app", "config"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "desktop"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app", "index.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app", "config", "site.conf"), []byte("enabled"), 0600); err != nil {
		t.Fatal(err)
	}
	manager, err := New(Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	source, err := manager.Stat("/app")
	if err != nil {
		t.Fatal(err)
	}
	var stream bytes.Buffer
	if _, err := manager.ExportDirectory(context.Background(), "/app", source.ResourceVersion, &stream); err != nil {
		t.Fatal(err)
	}
	entry, err := manager.ImportDirectory(context.Background(), "/desktop", "app", &stream)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Kind != "directory" || entry.Path != "/desktop/app" {
		t.Fatalf("unexpected imported entry: %#v", entry)
	}
	content, err := os.ReadFile(filepath.Join(root, "desktop", "app", "config", "site.conf"))
	if err != nil || string(content) != "enabled" {
		t.Fatalf("imported content=%q err=%v", content, err)
	}
	matches, err := filepath.Glob(filepath.Join(root, "desktop", ".kpanel-extract-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("transfer left temporary paths: %#v err=%v", matches, err)
	}
}

func TestDirectoryTransferRejectsStaleSourceAndDestinationCollision(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{"app", "desktop", filepath.Join("desktop", "app")} {
		if err := os.Mkdir(filepath.Join(root, directory), 0755); err != nil {
			t.Fatal(err)
		}
	}
	manager, err := New(Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	var stream bytes.Buffer
	if _, err := manager.ExportDirectory(context.Background(), "/app", "sha256:stale", &stream); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale export error=%v", err)
	}
	source, err := manager.Stat("/app")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ExportDirectory(context.Background(), "/app", source.ResourceVersion, &stream); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ImportDirectory(context.Background(), "/desktop", "app", &stream); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("collision import error=%v", err)
	}
}

func TestCancelledDirectoryImportCleansTemporaryDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "desktop"), 0755); err != nil {
		t.Fatal(err)
	}
	manager, err := New(Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := manager.ImportDirectory(ctx, "/desktop", "app", bytes.NewReader(make([]byte, 1024))); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled import error=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "desktop", "app")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled import exposed destination: %v", err)
	}
	matches, _ := filepath.Glob(filepath.Join(root, "desktop", ".kpanel-extract-*"))
	if len(matches) != 0 {
		t.Fatalf("cancelled import left temporary paths: %#v", matches)
	}
}
