//go:build linux

package dockerx

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDockerBackupAndRestoreSupportScriptSymlinkRoot(t *testing.T) {
	base := t.TempDir()
	realRoot := filepath.Join(base, "volume", "docker")
	if err := os.MkdirAll(filepath.Join(realRoot, "example"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(realRoot, "example", "compose.yml"),
		[]byte("services: {}\n"),
		0o640,
	); err != nil {
		t.Fatal(err)
	}
	linkRoot := filepath.Join(base, "home-docker")
	if err := os.Symlink(realRoot, linkRoot); err != nil {
		t.Fatal(err)
	}
	client := &Client{appRoot: linkRoot, stateRoot: t.TempDir(), now: time.Now}
	resultPath, err := client.createDockerBackup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(resultPath, linkRoot+string(filepath.Separator)) {
		t.Fatalf("backup result path = %q, want logical script root", resultPath)
	}
	if err := os.RemoveAll(filepath.Join(realRoot, "example")); err != nil {
		t.Fatal(err)
	}
	if err := client.restoreDockerBackup(context.Background(), filepath.Base(resultPath)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(realRoot, "example", "compose.yml")); err != nil {
		t.Fatalf("restore through symlink root failed: %v", err)
	}
}
