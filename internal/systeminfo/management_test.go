package systeminfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadTimezonePrefersLocaltimeOverLegacyTimezoneFile(t *testing.T) {
	root := t.TempDir()
	etcRoot := filepath.Join(root, "etc")
	zonePath := filepath.Join(root, "usr", "share", "zoneinfo", "Asia", "Seoul")
	if err := os.MkdirAll(filepath.Dir(zonePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(zonePath, []byte("fixture"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(etcRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(etcRoot, "timezone"), []byte("Asia/Shanghai\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(zonePath, filepath.Join(etcRoot, "localtime")); err != nil {
		t.Fatal(err)
	}

	collector := &Collector{EtcRoot: etcRoot}
	if got := collector.readTimezone(); got != "Asia/Seoul" {
		t.Fatalf("readTimezone() = %q, want Asia/Seoul", got)
	}
}
