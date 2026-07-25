package systeminfo

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestCollectorReadsLinuxFixtures(t *testing.T) {
	root := filepath.Join("testdata", "root")
	collector := &Collector{
		ProcRoot: filepath.Join(root, "proc"),
		EtcRoot:  filepath.Join(root, "etc"),
		Now:      func() time.Time { return time.Unix(1_700_000_000, 0) },
	}
	got, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if got.OS != "Fixture Linux 1" || got.Kernel != "6.8.0-fixture" {
		t.Fatalf("unexpected OS data: %#v", got)
	}
	if got.CPU.Cores != 2 || got.CPU.UsagePercent != 20 {
		t.Fatalf("unexpected CPU: %#v", got.CPU)
	}
	if got.Memory.TotalBytes != 8*1024*1024 || got.Memory.UsedBytes != 6*1024*1024 {
		t.Fatalf("unexpected memory: %#v", got.Memory)
	}
	if got.Network.ReceivedBytes != 3000 || got.Network.SentBytes != 7000 {
		t.Fatalf("unexpected network: %#v", got.Network)
	}
	if got.UptimeSeconds != 12345 {
		t.Fatalf("unexpected uptime: %d", got.UptimeSeconds)
	}
}
