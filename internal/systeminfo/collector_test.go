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
	if got.CPU.Cores != 2 || got.CPU.UsagePercent != 0 {
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
	if len(got.Management.SSH.Ports) != 1 || got.Management.SSH.Ports[0] != 2222 {
		t.Fatalf("unexpected SSH configuration: %#v", got.Management.SSH)
	}
	if len(got.Management.DNS.Servers) != 2 || got.Management.DNS.Servers[0] != "1.1.1.1" {
		t.Fatalf("unexpected DNS configuration: %#v", got.Management.DNS)
	}
	if got.Management.Timezone != "Asia/Shanghai" || got.Management.IPPreference != "ipv4" {
		t.Fatalf("unexpected regional configuration: %#v", got.Management)
	}
	if got.Management.PackageManager != "apt" ||
		len(got.Management.PackageSources) != 1 ||
		got.Management.PackageSources[0] != "mirrors.example.test" {
		t.Fatalf("unexpected package sources: %#v", got.Management)
	}
	if got.Management.Swap.ActiveDevices != 1 {
		t.Fatalf("unexpected swap state: %#v", got.Management.Swap)
	}
	if !got.Management.KernelOptimization.Enabled ||
		got.Management.KernelOptimization.Profile != "网站优化模式" {
		t.Fatalf("unexpected kernel optimization: %#v", got.Management.KernelOptimization)
	}
	if !got.Management.BBR.Enabled || !got.Management.BBR.Supported ||
		got.Management.BBR.DefaultQDisc != "fq" {
		t.Fatalf("unexpected BBR state: %#v", got.Management.BBR)
	}
}

func TestCPUUsagePercentUsesIntervalDelta(t *testing.T) {
	before := cpuTimes{total: 1_000, idle: 800}
	after := cpuTimes{total: 1_200, idle: 900}
	if got := cpuUsagePercent(before, after); got != 50 {
		t.Fatalf("cpuUsagePercent() = %v, want 50", got)
	}
}
