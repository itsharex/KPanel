//go:build linux

package systeminfo

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseProcessStatHandlesSpacesAndIdentity(t *testing.T) {
	value, err := parseProcessStat("123 (worker pool) S 7 0 0 0 0 0 0 0 0 0 10 5 0 0 0 0 0 99")
	if err != nil {
		t.Fatal(err)
	}
	if value.pid != 123 || value.parentPID != 7 || value.name != "worker pool" || value.ticks != 15 || value.startTicks != 99 {
		t.Fatalf("process stat=%#v", value)
	}
}

func TestCollectProcessesReturnsBoundedNonSensitiveMetrics(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "stat"), []byte("cpu 100 0 50 500 0 0 0 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	processRoot := filepath.Join(root, "42")
	if err := os.Mkdir(processRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	stat := "42 (nginx worker) S 1 0 0 0 0 0 0 0 0 0 20 10 0 0 0 0 0 123"
	if err := os.WriteFile(filepath.Join(processRoot, "stat"), []byte(stat), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(processRoot, "status"), []byte("Uid:\t1000\t1000\t1000\t1000\nVmRSS:\t64 kB\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	collector := NewCollector()
	collector.ProcRoot = root
	collector.ProcessSampleInterval = time.Millisecond
	result, err := collector.Processes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.TopCPU) != 1 || len(result.TopMemory) != 1 || result.TopCPU[0].PID != 42 || result.TopCPU[0].Name != "nginx worker" || result.TopCPU[0].UserID != 1000 || result.TopCPU[0].MemoryBytes != 64*1024 {
		t.Fatalf("process snapshot=%#v", result)
	}
}
