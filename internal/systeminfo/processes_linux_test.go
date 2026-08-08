//go:build linux

package systeminfo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestParseProcessStatHandlesSpacesAndIdentity(t *testing.T) {
	value, err := parseProcessStat("123 (worker pool) S 7 0 0 0 0 0 0 0 0 0 10 5 0 0 20 -5 8 0 99")
	if err != nil {
		t.Fatal(err)
	}
	if value.pid != 123 || value.parentPID != 7 || value.name != "worker pool" || value.ticks != 15 ||
		value.startTicks != 99 || value.threads != 8 || value.nice != -5 {
		t.Fatalf("process stat=%#v", value)
	}
}

func TestQueryProcessesCapsReturnedRows(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "stat"), []byte("cpu 100 0 50 500 0 0 0 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for pid := 1; pid <= MaxProcessResults+44; pid++ {
		processRoot := filepath.Join(root, strconv.Itoa(pid))
		if err := os.Mkdir(processRoot, 0o700); err != nil {
			t.Fatal(err)
		}
		stat := fmt.Sprintf("%d (worker-%d) S 1 0 0 0 0 0 0 0 0 0 20 10 0 0 20 0 1 0 %d", pid, pid, pid+100)
		if err := os.WriteFile(filepath.Join(processRoot, "stat"), []byte(stat), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	collector := NewCollector()
	collector.ProcRoot = root
	collector.EtcRoot = root
	collector.ProcessSampleInterval = time.Millisecond
	result, err := collector.QueryProcesses(context.Background(), ProcessQuery{
		Sort: "pid", Order: "asc", Limit: MaxProcessResults,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != MaxProcessResults || result.Total != MaxProcessResults+44 || !result.Truncated {
		t.Fatalf("bounded process snapshot items=%d total=%d truncated=%v", len(result.Items), result.Total, result.Truncated)
	}
}

func TestFilterAndSortProcessesSearchesIdentityAndSortsDeterministically(t *testing.T) {
	items := []ProcessMetric{
		{PID: 30, Name: "nginx worker", User: "www-data", CPUPercent: 2, MemoryBytes: 300, Threads: 4},
		{PID: 20, Name: "nginx master", User: "root", CPUPercent: 7, MemoryBytes: 100, Threads: 2},
		{PID: 10, Name: "postgres", User: "postgres", CPUPercent: 9, MemoryBytes: 900, Threads: 8},
	}
	result := filterAndSortProcesses(items, ProcessQuery{Search: "NGINX", Sort: "memory", Order: "desc"})
	if len(result) != 2 || result[0].PID != 30 || result[1].PID != 20 {
		t.Fatalf("filtered processes=%#v", result)
	}
	result = filterAndSortProcesses(items, ProcessQuery{Search: "root", Sort: "cpu", Order: "desc"})
	if len(result) != 1 || result[0].PID != 20 {
		t.Fatalf("user-filtered processes=%#v", result)
	}
	result = filterAndSortProcesses(items, ProcessQuery{Search: "10", Sort: "pid", Order: "asc"})
	if len(result) != 1 || result[0].PID != 10 {
		t.Fatalf("pid-filtered processes=%#v", result)
	}
}

func BenchmarkFilterAndSortProcesses8192(b *testing.B) {
	items := make([]ProcessMetric, maxProcessScan)
	for index := range items {
		items[index] = ProcessMetric{
			PID: index + 1, Name: "worker", User: "service",
			CPUPercent: float64(index % 128), MemoryBytes: uint64(index) * 4096, Threads: index % 32,
		}
	}
	query := ProcessQuery{Search: "worker", Sort: "cpu", Order: "desc", Limit: MaxProcessResults}
	b.ResetTimer()
	for range b.N {
		result := filterAndSortProcesses(items, query)
		if len(result) != len(items) {
			b.Fatal("unexpected process count")
		}
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
	stat := "42 (nginx worker) S 1 0 0 0 0 0 0 0 0 0 20 10 0 0 0 0 0 0 123"
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
