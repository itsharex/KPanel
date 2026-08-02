package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/dockerx"
)

type fakeSystemSource struct {
	summary contract.SystemSummary
	err     error
}

func (source fakeSystemSource) CollectRuntime(context.Context) (contract.SystemSummary, error) {
	return source.summary, source.err
}

type fakeDockerSource struct {
	batch dockerx.ContainerMetricBatch
	err   error
	calls int
}

type fakeOperatorLatencyProber struct {
	mu        sync.Mutex
	active    int
	maxActive int
	calls     []string
	failures  map[string]bool
}

func (prober *fakeOperatorLatencyProber) Probe(
	ctx context.Context,
	address string,
) (time.Duration, error) {
	prober.mu.Lock()
	prober.active++
	if prober.active > prober.maxActive {
		prober.maxActive = prober.active
	}
	prober.calls = append(prober.calls, address)
	prober.mu.Unlock()
	select {
	case <-time.After(time.Millisecond):
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	prober.mu.Lock()
	prober.active--
	fails := prober.failures[address]
	prober.mu.Unlock()
	if fails {
		return 0, errors.New("unreachable")
	}
	return time.Duration(len(address)) * time.Millisecond, nil
}

func (source *fakeDockerSource) RunningContainerStats(
	context.Context,
	int,
	int,
) (dockerx.ContainerMetricBatch, error) {
	source.calls++
	return source.batch, source.err
}

func testSummary(rx, tx uint64) contract.SystemSummary {
	return contract.SystemSummary{
		CPU:  contract.CPUSummary{Cores: 4, UsagePercent: 25},
		Load: contract.LoadSummary{One: 0.5, Five: 0.4, Fifteen: 0.3},
		Memory: contract.MemorySummary{
			UsedBytes: 2 << 30, TotalBytes: 8 << 30,
			SwapUsedBytes: 128 << 20, SwapTotalBytes: 1 << 30,
		},
		Disks: []contract.DiskSummary{{
			MountPoint: "/", UsedBytes: 20 << 30, TotalBytes: 100 << 30, UsagePercent: 20,
		}},
		DiskIO: contract.DiskIOSummary{
			Available: true, ReadBytes: rx * 2, WriteBytes: tx * 2,
		},
		Network: contract.NetworkSummary{
			ReceivedBytes: rx, SentBytes: tx, TCPConnections: 12, UDPConnections: 3,
		},
	}
}

func TestSampleAndHistoryPersistBoundedHostAndContainerMetrics(t *testing.T) {
	current := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	system := &fakeSystemSource{summary: testSummary(1_000, 2_000)}
	docker := &fakeDockerSource{batch: dockerx.ContainerMetricBatch{
		Total: 1,
		Items: []dockerx.ContainerMetricSample{{
			Name: "web", Image: "nginx:latest",
			ContainerStats: dockerx.ContainerStats{
				ContainerID: strings.Repeat("a", 64), CPUPercent: 12,
				MemoryBytes: 128 << 20, MemoryLimit: 1 << 30, MemoryPercent: 12.5,
				NetworkRx: 100, NetworkTx: 200, BlockRead: 1_000, BlockWrite: 2_000, PIDs: 5,
			},
		}},
	}}
	service, err := New(Config{
		StateDir: t.TempDir(), System: system, Docker: docker,
		Now:               func() time.Time { return current },
		ContainerInterval: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Sample(context.Background()); err != nil {
		t.Fatal(err)
	}
	current = current.Add(time.Minute)
	system.summary = testSummary(1_600, 2_900)
	docker.batch.Items[0].NetworkRx = 400
	docker.batch.Items[0].NetworkTx = 500
	docker.batch.Items[0].BlockRead = 1_600
	docker.batch.Items[0].BlockWrite = 3_200
	if err := service.Sample(context.Background()); err != nil {
		t.Fatal(err)
	}

	history, err := service.History(context.Background(), "1h")
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Host) != 2 || history.Host[1].NetworkRxRate != 10 ||
		history.Host[1].NetworkTxRate != 15 || history.Host[1].DiskReadRate != 20 ||
		history.Host[1].DiskWriteRate != 30 {
		t.Fatalf("unexpected host history: %#v", history.Host)
	}
	if len(history.Containers) != 1 || len(history.Containers[0].Points) != 2 ||
		history.Containers[0].Points[1].NetworkRxRate != 5 ||
		history.Containers[0].Points[1].BlockReadRate != 10 ||
		history.Containers[0].Points[1].BlockWriteRate != 20 {
		t.Fatalf("unexpected container history: %#v", history.Containers)
	}
	if history.Storage.StorageBytes <= 0 || history.Storage.LastContainerRecorded != 1 ||
		docker.calls != 2 {
		t.Fatalf("unexpected storage status: %#v calls=%d", history.Storage, docker.calls)
	}
	info, err := os.Stat(filepath.Join(service.stateDir, "metrics-2026-07-31.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("history shard permissions = %o", info.Mode().Perm())
	}
}

func TestOperatorLatencyCollectionIsFixedBoundedAndPersisted(t *testing.T) {
	current := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	failedAddress := operatorLatencyTargets[2].Address
	prober := &fakeOperatorLatencyProber{failures: map[string]bool{failedAddress: true}}
	service, err := New(Config{
		StateDir: t.TempDir(), System: fakeSystemSource{summary: testSummary(1, 2)},
		Now: func() time.Time { return current }, OperatorLatency: prober,
		OperatorLatencyInterval: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Sample(context.Background()); err != nil {
		t.Fatal(err)
	}
	status := service.Status()
	if !status.OperatorLatencyAvailable || status.OperatorLatencyIntervalSeconds != 60 ||
		status.LastOperatorLatencySuccessful != len(operatorLatencyTargets)-1 ||
		status.LastOperatorLatencyFailed != 1 {
		t.Fatalf("unexpected operator latency status: %#v", status)
	}
	prober.mu.Lock()
	callCount, maxActive := len(prober.calls), prober.maxActive
	prober.mu.Unlock()
	if callCount != len(operatorLatencyTargets) || maxActive > operatorProbeWorkers {
		t.Fatalf("operator probe calls=%d max active=%d", callCount, maxActive)
	}
	history, err := service.History(context.Background(), "1h")
	if err != nil {
		t.Fatal(err)
	}
	if len(history.OperatorLatency) != len(operatorLatencyTargets) {
		t.Fatalf("operator series count=%d", len(history.OperatorLatency))
	}
	for index, series := range history.OperatorLatency {
		target := operatorLatencyTargets[index]
		if series.ID != target.ID || series.Operator != target.Operator ||
			series.Region != target.Region || series.Address != target.Address || len(series.Points) != 1 {
			t.Fatalf("unexpected operator series at %d: %#v", index, series)
		}
		if target.Address == failedAddress {
			if series.Points[0].LatencyMilliseconds != nil {
				t.Fatalf("failed probe was recorded as latency: %#v", series.Points[0])
			}
		} else if series.Points[0].LatencyMilliseconds == nil {
			t.Fatalf("successful probe was recorded as missing: %#v", series.Points[0])
		}
	}
}

func TestOperatorLatencyCatalogIsNineFixedIPv4Targets(t *testing.T) {
	if len(operatorLatencyTargets) != 9 {
		t.Fatalf("operator target count=%d", len(operatorLatencyTargets))
	}
	seen := make(map[string]struct{}, len(operatorLatencyTargets))
	counts := make(map[string]int)
	for _, target := range operatorLatencyTargets {
		if target.ID != fmt.Sprintf("%s-%s", target.Operator, target.Region) {
			t.Fatalf("target id mismatch: %#v", target)
		}
		if ip := net.ParseIP(target.Address); ip == nil || ip.To4() == nil {
			t.Fatalf("target is not fixed IPv4: %#v", target)
		}
		if _, exists := seen[target.Address]; exists {
			t.Fatalf("duplicate target address: %s", target.Address)
		}
		seen[target.Address] = struct{}{}
		counts[target.Operator]++
	}
	for _, operator := range []string{"telecom", "unicom", "mobile"} {
		if counts[operator] != 3 {
			t.Fatalf("operator %s target count=%d", operator, counts[operator])
		}
	}
}

func TestDNSRootNSQueryUsesBoundedValidatedPacket(t *testing.T) {
	query := dnsRootNSQuery(0x1234)
	if len(query) != 17 || query[0] != 0x12 || query[1] != 0x34 ||
		query[2] != 0x01 || query[5] != 0x01 || query[12] != 0 ||
		query[13] != 0 || query[14] != 2 || query[15] != 0 || query[16] != 1 {
		t.Fatalf("unexpected DNS query: %v", query)
	}
}

func TestSortContainerSeriesPrioritizesFreshMemoryThenCPU(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	series := []contract.MonitoringContainerSeries{
		{ContainerID: "empty", Name: "empty"},
		{
			ContainerID: "same-zeta", Name: "zeta",
			Points: []contract.MonitoringContainerPoint{{
				CollectedAt: now, MemoryBytes: 3 << 30, CPUPercent: 5,
			}},
		},
		{
			ContainerID: "same-alpha", Name: "alpha",
			Points: []contract.MonitoringContainerPoint{{
				CollectedAt: now, MemoryBytes: 3 << 30, CPUPercent: 5,
			}},
		},
		{
			ContainerID: "older", Name: "older",
			Points: []contract.MonitoringContainerPoint{{
				CollectedAt: now.Add(-5 * time.Minute), MemoryBytes: 8 << 30, CPUPercent: 100,
			}},
		},
		{
			ContainerID: "lower-memory", Name: "lower-memory",
			Points: []contract.MonitoringContainerPoint{{
				CollectedAt: now, MemoryBytes: 1 << 30, CPUPercent: 90,
			}},
		},
		{
			ContainerID: "lower-cpu", Name: "lower-cpu",
			Points: []contract.MonitoringContainerPoint{{
				CollectedAt: now, MemoryBytes: 2 << 30, CPUPercent: 20,
			}},
		},
		{
			ContainerID: "higher-cpu", Name: "higher-cpu",
			Points: []contract.MonitoringContainerPoint{{
				CollectedAt: now, MemoryBytes: 2 << 30, CPUPercent: 40,
			}},
		},
	}

	sortContainerSeries(series)
	want := []string{"same-alpha", "same-zeta", "higher-cpu", "lower-cpu", "lower-memory", "older", "empty"}
	for index, containerID := range want {
		if series[index].ContainerID != containerID {
			t.Fatalf("container order at %d = %q, want %q: %#v", index, series[index].ContainerID, containerID, series)
		}
	}
}

func TestHistorySkipsCorruptRecordsAndRejectsUnknownRange(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	service, err := New(Config{
		StateDir: dir,
		System:   fakeSystemSource{summary: testSummary(0, 0)},
		Now:      func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "metrics-2026-07-31.jsonl"),
		[]byte("{not-json}\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	history, err := service.History(context.Background(), "24h")
	if err != nil {
		t.Fatal(err)
	}
	if history.SkippedLines != 1 || len(history.Host) != 0 {
		t.Fatalf("corrupt history result = %#v", history)
	}
	if _, err := service.History(context.Background(), "30d"); !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("unknown range error = %v", err)
	}
}

func TestPruneRemovesExpiredShardButPreservesUnrelatedFiles(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	oldShard := filepath.Join(dir, "metrics-2026-07-20.jsonl")
	unrelated := filepath.Join(dir, "keep.txt")
	if err := os.WriteFile(oldShard, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unrelated, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(Config{
		StateDir: dir,
		System:   fakeSystemSource{summary: testSummary(0, 0)},
		Now:      func() time.Time { return now },
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(oldShard); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expired shard remains: %v", err)
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("unrelated file was removed: %v", err)
	}
}

func TestAppendRejectsSymlinkShard(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional privileges on Windows")
	}
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("protected"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "metrics-2026-07-31.jsonl")); err != nil {
		t.Fatal(err)
	}
	service, err := New(Config{
		StateDir: dir,
		System:   fakeSystemSource{summary: testSummary(0, 0)},
		Now:      func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Sample(context.Background()); err == nil ||
		!strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("symlink shard error = %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "protected" {
		t.Fatalf("symlink target changed: %q, %v", data, err)
	}
}

func TestCompactRecordFitsDailyStorageBudgetAtMaximumContainerCount(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	hostOnly := diskRecord{
		Version: recordVersion, CollectedAt: now, Host: hostPoint(testSummary(1, 2), now),
	}
	full := hostOnly
	full.ContainerSampled = true
	full.DockerAvailable = true
	full.ContainerTotal = defaultMaxContainers
	for _, target := range operatorLatencyTargets {
		full.OperatorLatency = append(full.OperatorLatency, diskOperatorLatencyPoint{
			ID: target.ID, LatencyMilliseconds: 999.9, Reachable: true,
		})
	}
	for index := 0; index < defaultMaxContainers; index++ {
		full.Containers = append(full.Containers, diskContainerPoint{
			ID: strings.Repeat("a", 64), Name: "container-name-1234567890",
			Image:      "registry.example.test/namespace/application:latest",
			CPUPercent: 100, MemoryBytes: 1 << 30, MemoryLimitBytes: 2 << 30,
			MemoryPercent: 50, NetworkRxBytes: 1 << 40, NetworkTxBytes: 1 << 40,
			BlockReadBytes: 1 << 40, BlockWriteBytes: 1 << 40, PIDs: 999,
		})
	}
	hostData, err := json.Marshal(hostOnly)
	if err != nil {
		t.Fatal(err)
	}
	fullData, err := json.Marshal(full)
	if err != nil {
		t.Fatal(err)
	}
	projectedDailyBytes := int64(len(hostData)+1)*1152 + int64(len(fullData)+1)*288
	t.Logf(
		"maximum sampling projection=%d bytes/day host=%d bytes full=%d bytes",
		projectedDailyBytes, len(hostData), len(fullData),
	)
	if projectedDailyBytes >= defaultMaxDailyBytes {
		t.Fatalf(
			"maximum sampling projection = %d bytes (host=%d full=%d), budget=%d",
			projectedDailyBytes, len(hostData), len(fullData), defaultMaxDailyBytes,
		)
	}
}

func TestBucketAggregationKeepsResourcePeaksAndLatestCounters(t *testing.T) {
	at := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	host := appendHostBucket(nil, contract.MonitoringHostPoint{
		CollectedAt: at, MemoryUsedBytes: 4, MemoryTotalBytes: 10,
		SwapUsedBytes: 1, SwapTotalBytes: 10,
		DiskUsedBytes: 4, DiskTotalBytes: 10, DiskPercent: 40,
		NetworkRxBytes: 100, DiskReadRate: 2, DiskWriteRate: 3,
	}, time.Minute)
	host = appendHostBucket(host, contract.MonitoringHostPoint{
		CollectedAt: at.Add(30 * time.Second), MemoryUsedBytes: 6, MemoryTotalBytes: 20,
		SwapUsedBytes: 8, SwapTotalBytes: 10,
		DiskUsedBytes: 9, DiskTotalBytes: 10, DiskPercent: 90,
		NetworkRxBytes: 200, DiskReadRate: 8, DiskWriteRate: 9,
	}, time.Minute)
	if host[0].MemoryUsedBytes != 4 || host[0].SwapUsedBytes != 8 ||
		host[0].DiskUsedBytes != 9 || host[0].NetworkRxBytes != 200 ||
		host[0].DiskReadRate != 8 || host[0].DiskWriteRate != 9 {
		t.Fatalf("host bucket did not preserve peaks and latest counter: %#v", host[0])
	}

	container := appendContainerBucket(nil, contract.MonitoringContainerPoint{
		CollectedAt: at, MemoryBytes: 4, MemoryLimitBytes: 10, MemoryPercent: 40,
		NetworkRxBytes: 100, BlockReadRate: 2, BlockWriteRate: 3,
	}, time.Minute)
	container = appendContainerBucket(container, contract.MonitoringContainerPoint{
		CollectedAt: at.Add(30 * time.Second), MemoryBytes: 6, MemoryLimitBytes: 20,
		MemoryPercent: 30, NetworkRxBytes: 200, BlockReadRate: 8, BlockWriteRate: 9,
	}, time.Minute)
	if container[0].MemoryBytes != 4 || container[0].MemoryPercent != 40 ||
		container[0].NetworkRxBytes != 200 || container[0].BlockReadRate != 8 ||
		container[0].BlockWriteRate != 9 {
		t.Fatalf("container bucket did not preserve peak and latest counter: %#v", container[0])
	}

	latencyA := 12.5
	latencyB := 30.25
	latency := appendOperatorLatencyBucket(nil, contract.MonitoringOperatorLatencyPoint{
		CollectedAt: at, LatencyMilliseconds: &latencyA,
	}, time.Minute)
	latency = appendOperatorLatencyBucket(latency, contract.MonitoringOperatorLatencyPoint{
		CollectedAt: at.Add(20 * time.Second), LatencyMilliseconds: nil,
	}, time.Minute)
	latency = appendOperatorLatencyBucket(latency, contract.MonitoringOperatorLatencyPoint{
		CollectedAt: at.Add(40 * time.Second), LatencyMilliseconds: &latencyB,
	}, time.Minute)
	if len(latency) != 1 || latency[0].LatencyMilliseconds == nil ||
		*latency[0].LatencyMilliseconds != latencyB ||
		!latency[0].CollectedAt.Equal(at.Add(40*time.Second)) {
		t.Fatalf("latency bucket did not preserve successful peak: %#v", latency)
	}
}

func TestMaximumSevenDayResponseFitsPanelAgentLimit(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	history := contract.MonitoringHistory{
		Range: "7d", StartedAt: now.Add(-7 * 24 * time.Hour), EndedAt: now,
		BucketSeconds:   int((30 * time.Minute).Seconds()),
		Host:            make([]contract.MonitoringHostPoint, 7*24*2),
		Containers:      make([]contract.MonitoringContainerSeries, maxHistorySeries),
		OperatorLatency: operatorLatencyCatalog(),
	}
	for seriesIndex := range history.OperatorLatency {
		series := &history.OperatorLatency[seriesIndex]
		series.Points = make([]contract.MonitoringOperatorLatencyPoint, 7*24*2)
		for pointIndex := range series.Points {
			value := 999.9
			series.Points[pointIndex] = contract.MonitoringOperatorLatencyPoint{
				CollectedAt:         now.Add(-time.Duration(pointIndex) * 30 * time.Minute),
				LatencyMilliseconds: &value,
			}
		}
	}
	for index := range history.Host {
		history.Host[index] = contract.MonitoringHostPoint{
			CollectedAt: now.Add(-time.Duration(index) * 30 * time.Minute),
			CPUPercent:  100, CPUCores: 64, MemoryUsedBytes: 1 << 40,
			MemoryTotalBytes: 2 << 40, DiskUsedBytes: 1 << 40, DiskTotalBytes: 2 << 40,
			NetworkRxBytes: 1 << 50, NetworkTxBytes: 1 << 50,
			NetworkRxRate: 1 << 30, NetworkTxRate: 1 << 30,
		}
	}
	for seriesIndex := range history.Containers {
		series := &history.Containers[seriesIndex]
		series.ContainerID = strings.Repeat("a", 64)
		series.Name = "container-name-1234567890"
		series.Image = "registry.example.test/namespace/application:latest"
		series.Points = make([]contract.MonitoringContainerPoint, 7*24*2)
		for pointIndex := range series.Points {
			series.Points[pointIndex] = contract.MonitoringContainerPoint{
				CollectedAt: now.Add(-time.Duration(pointIndex) * 30 * time.Minute),
				CPUPercent:  100, MemoryBytes: 1 << 40, MemoryLimitBytes: 2 << 40,
				MemoryPercent: 50, NetworkRxBytes: 1 << 50, NetworkTxBytes: 1 << 50,
				NetworkRxRate: 1 << 30, NetworkTxRate: 1 << 30,
				BlockReadBytes: 1 << 50, BlockWriteBytes: 1 << 50, PIDs: 999,
			}
		}
	}
	data, err := json.Marshal(history)
	if err != nil {
		t.Fatal(err)
	}
	const panelAgentResponseLimit = 8 << 20
	t.Logf("maximum seven-day response=%d bytes", len(data))
	if len(data) >= panelAgentResponseLimit {
		t.Fatalf("maximum history response = %d bytes, limit=%d", len(data), panelAgentResponseLimit)
	}
}

func BenchmarkHistorySevenDays(b *testing.B) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	service, err := New(Config{
		StateDir: b.TempDir(),
		System:   fakeSystemSource{summary: testSummary(0, 0)},
		Now:      func() time.Time { return now },
	})
	if err != nil {
		b.Fatal(err)
	}
	for index := 0; index < 7*24*60; index++ {
		at := now.Add(-time.Duration(index) * time.Minute)
		record := diskRecord{Version: recordVersion, CollectedAt: at, Host: hostPoint(testSummary(uint64(index), uint64(index)), at)}
		if err := service.appendRecord(record); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := service.History(context.Background(), "7d"); err != nil {
			b.Fatal(err)
		}
	}
}
