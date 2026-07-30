package filemanager

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func newTestManager(t *testing.T) (*Manager, string) {
	t.Helper()
	root := t.TempDir()
	manager, err := New(Config{
		Root: root,
		ProtectedVirtual: []string{
			"/docker/kpanel",
			"/.kpanel-trash",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return manager, root
}

func TestListHidesProtectedDirectoryAndSortsDirectoriesFirst(t *testing.T) {
	manager, root := newTestManager(t)
	mustMkdirAll(t, filepath.Join(root, "docker", "kpanel"))
	mustMkdirAll(t, filepath.Join(root, "website"))
	mustWrite(t, filepath.Join(root, "a.txt"), "hello")

	result, err := manager.List(context.Background(), "/", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 3 || result.Entries[0].Kind != "directory" {
		t.Fatalf("unexpected directory listing: %#v", result.Entries)
	}
	docker, err := manager.List(context.Background(), "/docker", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(docker.Entries) != 0 {
		t.Fatalf("protected directory leaked: %#v", docker.Entries)
	}
}

func TestRejectsTraversalProtectedPathsAndSymlinks(t *testing.T) {
	manager, root := newTestManager(t)
	mustMkdirAll(t, filepath.Join(root, "docker", "kpanel"))
	for _, value := range []string{"/../etc", `/docker\kpanel`, "/docker/kpanel"} {
		if _, err := manager.List(context.Background(), value, 10); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
	if runtime.GOOS == "windows" {
		return
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.List(context.Background(), "/escape", 10); !errors.Is(err, ErrSymlink) {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

func TestWriteTextIsAtomicAndRequiresResourceVersion(t *testing.T) {
	manager, root := newTestManager(t)
	mustWrite(t, filepath.Join(root, "config.json"), `{"enabled":false}`)
	entry, err := manager.Stat("/config.json")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := manager.WriteText(context.Background(), "/config.json", contract.FileWriteRequest{
		Content: `{"enabled":true}`, ExpectedResourceVersion: entry.ResourceVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ResourceVersion == entry.ResourceVersion {
		t.Fatal("resource version did not change")
	}
	content, err := os.ReadFile(filepath.Join(root, "config.json"))
	if err != nil || string(content) != `{"enabled":true}` {
		t.Fatalf("unexpected content %q, err=%v", content, err)
	}
	if _, err := manager.WriteText(context.Background(), "/config.json", contract.FileWriteRequest{
		Content: "stale", ExpectedResourceVersion: entry.ResourceVersion,
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected stale write conflict, got %v", err)
	}
}

func TestActiveSVGContentUsesTheTextEditor(t *testing.T) {
	manager, root := newTestManager(t)
	mustWrite(t, filepath.Join(root, "icon.svg"), `<svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	entry, err := manager.Stat("/icon.svg")
	if err != nil {
		t.Fatal(err)
	}
	if !entry.Editable || !entry.Previewable {
		t.Fatalf("SVG viewer policy = %#v", entry)
	}
}

func TestUploadCopyMoveChmodAndTrash(t *testing.T) {
	manager, root := newTestManager(t)
	mustMkdirAll(t, filepath.Join(root, "source"))
	mustMkdirAll(t, filepath.Join(root, "target"))
	entry, err := manager.Upload(
		context.Background(), "/source", "hello.txt",
		strings.NewReader("hello"), 5, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if entry.SizeBytes != 5 {
		t.Fatalf("unexpected upload entry: %#v", entry)
	}
	if _, err := manager.Action(context.Background(), contract.FileActionRequest{
		Action: "copy", Sources: []string{"/source/hello.txt"}, Target: "/target",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "target", "hello.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Action(context.Background(), contract.FileActionRequest{
		Action: "chmod", Sources: []string{"/target/hello.txt"}, Mode: "640",
	}); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(filepath.Join(root, "target", "hello.txt"))
		if info.Mode().Perm() != 0640 {
			t.Fatalf("unexpected mode %o", info.Mode().Perm())
		}
	}
	if _, err := manager.Action(context.Background(), contract.FileActionRequest{
		Action: "trash", Sources: []string{"/target/hello.txt"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "target", "hello.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file was not moved to trash: %v", err)
	}
	trashEntries, err := os.ReadDir(filepath.Join(root, ".kpanel-trash", "files"))
	if err != nil || len(trashEntries) != 1 {
		t.Fatalf("unexpected trash: %#v, err=%v", trashEntries, err)
	}
}

func TestLimitsUploadsAndBatchOperations(t *testing.T) {
	manager, _ := newTestManager(t)
	if _, err := manager.Upload(
		context.Background(), "/", "large.bin",
		bytes.NewReader(nil), MaxUploadBytes+1, false,
	); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("expected upload limit, got %v", err)
	}
	sources := make([]string, MaxBatchItems+1)
	if _, err := manager.Action(context.Background(), contract.FileActionRequest{
		Action: "trash", Sources: sources,
	}); !errors.Is(err, ErrBatchTooLarge) {
		t.Fatalf("expected batch limit, got %v", err)
	}
}

func TestBatchActionReportsPartialResultWithoutLosingSuccesses(t *testing.T) {
	manager, root := newTestManager(t)
	mustWrite(t, filepath.Join(root, "keep.txt"), "keep")

	result, err := manager.Action(context.Background(), contract.FileActionRequest{
		Action: "trash", Sources: []string{"/keep.txt", "/missing.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Succeeded) != 1 || result.Succeeded[0].Path != "/keep.txt" {
		t.Fatalf("unexpected successes: %#v", result.Succeeded)
	}
	if len(result.Failed) != 1 || result.Failed[0].Path != "/missing.txt" {
		t.Fatalf("unexpected failures: %#v", result.Failed)
	}
	if _, err := os.Stat(filepath.Join(root, "keep.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("successful item was not moved: %v", err)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
