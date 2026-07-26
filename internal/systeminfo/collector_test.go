package systeminfo

import (
	"context"
	"os"
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
	if got.Management.Swap.ActiveDevices != 1 ||
		!got.Management.Swap.FileExists ||
		!got.Management.Swap.FileActive ||
		got.Management.Swap.FileSizeBytes == 0 ||
		got.Management.Swap.OtherActiveDevices != 0 {
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

func TestReadSwapConfigurationSeparatesKejilionLegacyAndExternalSwap(t *testing.T) {
	root := t.TempDir()
	procRoot := filepath.Join(root, "proc")
	primaryPath := filepath.Join(root, "swapfile")
	legacyPath := filepath.Join(root, "legacy-swapfile")
	if err := os.MkdirAll(procRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, size := range map[string]int64{
		primaryPath: 1024 * 1024 * 1024,
		legacyPath:  2 * 1024 * 1024 * 1024,
	} {
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := file.Truncate(size); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	swaps := "Filename Type Size Used Priority\n" +
		primaryPath + " file 1048572 128 -2\n" +
		primaryPath + " file 2097148 0 -3\n" +
		"/dev/vda2 partition 524284 32 -4\n"
	if err := os.WriteFile(filepath.Join(procRoot, "swaps"), []byte(swaps), 0o600); err != nil {
		t.Fatal(err)
	}
	collector := &Collector{
		ProcRoot: procRoot, SwapPath: primaryPath, LegacySwapPath: legacyPath,
	}

	got := collector.readSwapConfiguration()
	if got.Path != primaryPath || !got.FileExists || !got.FileActive ||
		!got.LegacyExists || !got.LegacyActive ||
		got.ActiveDevices != 3 || got.OtherActiveDevices != 1 ||
		got.FileUsedBytes != 128*1024 ||
		got.OtherSwapTotalBytes != 524284*1024 {
		t.Fatalf("unexpected swap configuration: %#v", got)
	}
}

func TestCPUUsagePercentUsesIntervalDelta(t *testing.T) {
	before := cpuTimes{total: 1_000, idle: 800}
	after := cpuTimes{total: 1_200, idle: 900}
	if got := cpuUsagePercent(before, after); got != 50 {
		t.Fatalf("cpuUsagePercent() = %v, want 50", got)
	}
}
