package filemanager

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestExportZIPStreamsFoldersAndSelectionsWithoutCreatingAnArchive(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "folder"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "one.txt"), []byte("one"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "folder", "two.txt"), []byte("two"), 0644); err != nil {
		t.Fatal(err)
	}
	manager, err := New(Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	one, err := manager.Stat("/one.txt")
	if err != nil {
		t.Fatal(err)
	}
	folder, err := manager.Stat("/folder")
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := manager.ExportZIP(
		context.Background(),
		[]string{"/one.txt", "/folder"},
		map[string]string{"/one.txt": one.ResourceVersion, "/folder": folder.ResourceVersion},
		&output,
	); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	contents := make(map[string]string)
	for _, item := range reader.File {
		if item.FileInfo().IsDir() {
			continue
		}
		file, err := item.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, readErr := io.ReadAll(file)
		_ = file.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		contents[item.Name] = string(content)
	}
	if contents["one.txt"] != "one" || contents["folder/two.txt"] != "two" {
		t.Fatalf("archive contents=%#v", contents)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 2 {
		t.Fatalf("stream export changed source directory: entries=%#v err=%v", entries, err)
	}

	output.Reset()
	if err := manager.ExportZIP(
		context.Background(), []string{"/folder"},
		map[string]string{"/folder": folder.ResourceVersion}, &output,
	); err != nil {
		t.Fatal(err)
	}
	reader, err = zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil || len(reader.File) != 1 || reader.File[0].Name != "two.txt" {
		t.Fatalf("single folder archive=%#v err=%v", reader.File, err)
	}
}

func TestArchiveSourcesWithSameBaseNamesAreRenamedDeterministically(t *testing.T) {
	root := t.TempDir()
	fixtures := []struct {
		virtual string
		content string
	}{
		{virtual: "/a/nginx.conf", content: "a"},
		{virtual: "/b/nginx.conf", content: "b"},
		{virtual: "/c/nginx (2).conf", content: "existing suffix"},
		{virtual: "/d/NGINX.CONF", content: "case variant"},
	}
	for _, fixture := range fixtures {
		name := filepath.Join(root, rootName(fixture.virtual))
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(fixture.content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	manager, err := New(Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	sources := make([]string, 0, len(fixtures))
	versions := make(map[string]string, len(fixtures))
	for _, fixture := range fixtures {
		entry, statErr := manager.Stat(fixture.virtual)
		if statErr != nil {
			t.Fatal(statErr)
		}
		sources = append(sources, fixture.virtual)
		versions[fixture.virtual] = entry.ResourceVersion
	}
	expectedNames := []string{
		"nginx.conf",
		"nginx (3).conf",
		"nginx (2).conf",
		"NGINX (4).CONF",
	}
	expectedContents := map[string]string{
		"nginx.conf":     "a",
		"nginx (3).conf": "b",
		"nginx (2).conf": "existing suffix",
		"NGINX (4).CONF": "case variant",
	}
	assertArchive := func(label string, content []byte) {
		t.Helper()
		reader, readErr := zip.NewReader(bytes.NewReader(content), int64(len(content)))
		if readErr != nil {
			t.Fatalf("%s: open ZIP: %v", label, readErr)
		}
		if len(reader.File) != len(expectedNames) {
			t.Fatalf("%s: entries=%#v", label, reader.File)
		}
		for index, item := range reader.File {
			if item.Name != expectedNames[index] {
				t.Fatalf("%s: entry[%d]=%q want %q", label, index, item.Name, expectedNames[index])
			}
			file, openErr := item.Open()
			if openErr != nil {
				t.Fatal(openErr)
			}
			value, contentErr := io.ReadAll(file)
			_ = file.Close()
			if contentErr != nil || string(value) != expectedContents[item.Name] {
				t.Fatalf("%s: %q content=%q err=%v", label, item.Name, value, contentErr)
			}
		}
	}

	for run := 1; run <= 2; run++ {
		var output bytes.Buffer
		if err := manager.ExportZIP(context.Background(), sources, versions, &output); err != nil {
			t.Fatalf("export run %d: %v", run, err)
		}
		assertArchive("export", output.Bytes())
	}

	var rejected bytes.Buffer
	duplicate := []string{sources[0], sources[0]}
	if err := manager.ExportZIP(context.Background(), duplicate, versions, &rejected); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate source export error=%v", err)
	}
	if rejected.Len() != 0 {
		t.Fatalf("duplicate source export wrote %d bytes", rejected.Len())
	}
	if _, err := manager.compressArchive(
		context.Background(), duplicate, "/", "rejected.zip", archiveFormatZIP, versions,
	); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate source compression error=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "rejected.zip")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("duplicate source compression left output: %v", err)
	}

	if _, err := manager.compressArchive(
		context.Background(), sources, "/", "bundle.zip", archiveFormatZIP, versions,
	); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "bundle.zip"))
	if err != nil {
		t.Fatal(err)
	}
	assertArchive("compressed", content)
}

func TestExportZIPPreflightRejectsLateSymlinkWithoutOutput(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "folder"))
	mustWrite(t, filepath.Join(root, "folder", "a.txt"), "visible")
	if err := os.Symlink("a.txt", filepath.Join(root, "folder", "z-link")); err != nil {
		t.Fatal(err)
	}
	manager, err := New(Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	folder, err := manager.Stat("/folder")
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = manager.ExportZIP(
		context.Background(), []string{"/folder"},
		map[string]string{"/folder": folder.ResourceVersion}, &output,
	)
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("late symlink error=%v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("late symlink wrote %d bytes before rejection", output.Len())
	}
}

func TestExportZIPPreflightRejectsBudgetOverflowWithoutOutput(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "folder", "a.txt"), "1234")
	mustWrite(t, filepath.Join(root, "folder", "z.txt"), "5")
	manager, err := New(Config{Root: root, MaxCopyBytes: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	folder, err := manager.Stat("/folder")
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = manager.ExportZIP(
		context.Background(), []string{"/folder"},
		map[string]string{"/folder": folder.ResourceVersion}, &output,
	)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("budget overflow error=%v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("budget overflow wrote %d bytes before rejection", output.Len())
	}
}

func TestExportZIPSkipsProtectedDescendantsAndInternalFiles(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{
		filepath.Join(root, "folder"),
		filepath.Join(root, "folder", "protected"),
		filepath.Join(root, "folder", ".kpanel-upload-staged"),
	} {
		if err := os.Mkdir(directory, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range map[string]string{
		filepath.Join(root, "folder", "visible.txt"):                         "visible",
		filepath.Join(root, "folder", "protected", "secret.txt"):             "secret",
		filepath.Join(root, "folder", ".kpanel-edit-partial"):                "partial",
		filepath.Join(root, "folder", ".kpanel-upload-staged", "upload.tmp"): "upload",
	} {
		if err := os.WriteFile(name, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	manager, err := New(Config{
		Root:             root,
		ProtectedVirtual: []string{"/folder/protected"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	folder, err := manager.Stat("/folder")
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := manager.ExportZIP(
		context.Background(), []string{"/folder"},
		map[string]string{"/folder": folder.ResourceVersion}, &output,
	); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(reader.File) != 1 || reader.File[0].Name != "visible.txt" {
		t.Fatalf("archive exposed protected or internal entries: %#v", reader.File)
	}
	file, err := reader.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	content, readErr := io.ReadAll(file)
	_ = file.Close()
	if readErr != nil || string(content) != "visible" {
		t.Fatalf("visible content=%q err=%v", content, readErr)
	}
}

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
