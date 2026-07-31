//go:build linux

package filemanager

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"golang.org/x/sys/unix"
)

func TestWriteTextPreservesExtendedAttributes(t *testing.T) {
	manager, root := newTestManager(t)
	filePath := filepath.Join(root, "labeled.txt")
	mustWrite(t, filePath, "before")
	if err := unix.Setxattr(filePath, "user.kpanel-test", []byte("preserved"), 0); err != nil {
		t.Skipf("extended attributes are unavailable: %v", err)
	}
	entry, err := manager.Stat("/labeled.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.WriteText(context.Background(), "/labeled.txt", contract.FileWriteRequest{
		Content: "after", ExpectedResourceVersion: entry.ResourceVersion,
	}); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 32)
	count, err := unix.Getxattr(filePath, "user.kpanel-test", buffer)
	if err != nil || string(buffer[:count]) != "preserved" {
		t.Fatalf("extended attribute was not preserved: %q err=%v", buffer[:max(count, 0)], err)
	}
}
