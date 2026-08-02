package monitoring

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/dockerx"
)

const (
	recordVersion           = 1
	defaultRetentionDays    = 7
	defaultMaxContainers    = 32
	defaultMaxContainerJobs = 2
	defaultHostInterval     = time.Minute
	defaultContainerEvery   = 5 * time.Minute
	defaultSampleTimeout    = 30 * time.Second
	defaultMaxDailyBytes    = int64(4 << 20)
	defaultMaxStorageBytes  = int64(32 << 20)
	maxHistoryLineBytes     = 256 << 10
	maxHistorySeries        = 32
	maxScannedSeries        = 64
	maxHistoryPoints        = 720
)

var (
	metricFilePattern = regexp.MustCompile(`^metrics-(\d{4}-\d{2}-\d{2})\.jsonl$`)
	ErrBusy           = errors.New("monitoring history query is busy")
	ErrInvalidRange   = errors.New("invalid monitoring history range")
)

type SystemSource interface {
	CollectRuntime(context.Context) (contract.SystemSummary, error)
}

type DockerSource interface {
	RunningContainerStats(context.Context, int, int) (dockerx.ContainerMetricBatch, error)
}

type Config struct {
	StateDir                string
	System                  SystemSource
	Docker                  DockerSource
	OperatorLatency         OperatorLatencyProber
	Now                     func() time.Time
	HostInterval            time.Duration
	ContainerInterval       time.Duration
	OperatorLatencyInterval time.Duration
	SampleTimeout           time.Duration
	RetentionDays           int
	MaxContainers           int
	MaxDailyBytes           int64
	MaxStorageBytes         int64
}

type Service struct {
	stateDir                string
	system                  SystemSource
	docker                  DockerSource
	operatorLatency         OperatorLatencyProber
	now                     func() time.Time
	hostInterval            time.Duration
	containerInterval       time.Duration
	operatorLatencyInterval time.Duration
	sampleTimeout           time.Duration
	retentionDays           int
	maxContainers           int
	maxDailyBytes           int64
	maxStorageBytes         int64
	querySlots              chan struct{}

	mu                    sync.RWMutex
	status                contract.MonitoringStorageStatus
	nextContainerAt       time.Time
	nextOperatorLatencyAt time.Time
}

type diskRecord struct {
	Version            int                        `json:"v"`
	CollectedAt        time.Time                  `json:"at"`
	Host               diskHostPoint              `json:"h"`
	Containers         []diskContainerPoint       `json:"c,omitempty"`
	OperatorLatency    []diskOperatorLatencyPoint `json:"ol,omitempty"`
	ContainerSampled   bool                       `json:"cs,omitempty"`
	DockerAvailable    bool                       `json:"da,omitempty"`
	ContainerTotal     int                        `json:"ct,omitempty"`
	ContainerFailed    int                        `json:"cf,omitempty"`
	ContainerTruncated int                        `json:"cx,omitempty"`
}

type diskOperatorLatencyPoint struct {
	ID                  string  `json:"i"`
	LatencyMilliseconds float64 `json:"m,omitempty"`
	Reachable           bool    `json:"r,omitempty"`
}

type diskHostPoint struct {
	CPUPercent       float64 `json:"cp"`
	CPUCores         int     `json:"cc"`
	LoadOne          float64 `json:"l1"`
	LoadFive         float64 `json:"l5"`
	LoadFifteen      float64 `json:"l15"`
	MemoryUsedBytes  uint64  `json:"mu"`
	MemoryTotalBytes uint64  `json:"mt"`
	SwapUsedBytes    uint64  `json:"su"`
	SwapTotalBytes   uint64  `json:"st"`
	DiskUsedBytes    uint64  `json:"du"`
	DiskTotalBytes   uint64  `json:"dt"`
	DiskPercent      float64 `json:"dp"`
	DiskIOAvailable  bool    `json:"di,omitempty"`
	DiskReadBytes    uint64  `json:"dr,omitempty"`
	DiskWriteBytes   uint64  `json:"dw,omitempty"`
	NetworkRxBytes   uint64  `json:"nr"`
	NetworkTxBytes   uint64  `json:"nt"`
	TCPConnections   int     `json:"tc"`
	UDPConnections   int     `json:"uc"`
}

type diskContainerPoint struct {
	ID               string  `json:"i"`
	Name             string  `json:"n"`
	Image            string  `json:"m"`
	CPUPercent       float64 `json:"cp"`
	MemoryBytes      uint64  `json:"mu"`
	MemoryLimitBytes uint64  `json:"ml"`
	MemoryPercent    float64 `json:"mp"`
	NetworkRxBytes   uint64  `json:"nr"`
	NetworkTxBytes   uint64  `json:"nt"`
	BlockReadBytes   uint64  `json:"br"`
	BlockWriteBytes  uint64  `json:"bw"`
	PIDs             uint64  `json:"p"`
}

func New(config Config) (*Service, error) {
	if strings.TrimSpace(config.StateDir) == "" {
		return nil, errors.New("monitoring state directory is required")
	}
	if !filepath.IsAbs(config.StateDir) {
		return nil, errors.New("monitoring state directory must be absolute")
	}
	if config.System == nil {
		return nil, errors.New("monitoring system source is required")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.HostInterval <= 0 {
		config.HostInterval = defaultHostInterval
	}
	if config.ContainerInterval <= 0 {
		config.ContainerInterval = defaultContainerEvery
	}
	if config.OperatorLatencyInterval <= 0 {
		config.OperatorLatencyInterval = defaultOperatorLatencyEvery
	}
	if config.SampleTimeout <= 0 {
		config.SampleTimeout = defaultSampleTimeout
	}
	if config.RetentionDays <= 0 {
		config.RetentionDays = defaultRetentionDays
	}
	if config.RetentionDays > defaultRetentionDays {
		return nil, fmt.Errorf("monitoring retention cannot exceed %d days", defaultRetentionDays)
	}
	if config.MaxContainers <= 0 {
		config.MaxContainers = defaultMaxContainers
	}
	if config.MaxContainers > 64 {
		return nil, errors.New("monitoring cannot sample more than 64 containers")
	}
	if config.MaxDailyBytes <= 0 {
		config.MaxDailyBytes = defaultMaxDailyBytes
	}
	if config.MaxDailyBytes > defaultMaxDailyBytes {
		return nil, errors.New("monitoring daily storage limit is too large")
	}
	if config.MaxStorageBytes <= 0 {
		config.MaxStorageBytes = defaultMaxStorageBytes
	}
	if config.MaxStorageBytes > defaultMaxStorageBytes {
		return nil, errors.New("monitoring storage limit is too large")
	}
	if err := os.MkdirAll(config.StateDir, 0o700); err != nil {
		return nil, fmt.Errorf("create monitoring state directory: %w", err)
	}
	if err := os.Chmod(config.StateDir, 0o700); err != nil {
		return nil, fmt.Errorf("protect monitoring state directory: %w", err)
	}
	service := &Service{
		stateDir: config.StateDir, system: config.System, docker: config.Docker,
		operatorLatency: config.OperatorLatency,
		now:             config.Now, hostInterval: config.HostInterval,
		containerInterval: config.ContainerInterval, sampleTimeout: config.SampleTimeout,
		operatorLatencyInterval: config.OperatorLatencyInterval,
		retentionDays:           config.RetentionDays, maxContainers: config.MaxContainers,
		maxDailyBytes: config.MaxDailyBytes, maxStorageBytes: config.MaxStorageBytes,
		querySlots: make(chan struct{}, 2),
	}
	service.status = service.baseStatus()
	bytes, limitReached, _ := service.prune(config.Now().UTC())
	service.status.StorageBytes = bytes
	service.status.StorageLimitReached = limitReached
	return service, nil
}

func (s *Service) baseStatus() contract.MonitoringStorageStatus {
	return contract.MonitoringStorageStatus{
		Enabled: true, RetentionDays: s.retentionDays,
		HostIntervalSeconds:            int(s.hostInterval.Seconds()),
		ContainerIntervalSeconds:       int(s.containerInterval.Seconds()),
		OperatorLatencyIntervalSeconds: int(s.operatorLatencyInterval.Seconds()),
		OperatorLatencyAvailable:       s.operatorLatency != nil,
		MaxContainers:                  s.maxContainers, MaxStorageBytes: s.maxStorageBytes,
	}
}

func (s *Service) Run(ctx context.Context) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			sampleContext, cancel := context.WithTimeout(ctx, s.sampleTimeout)
			_ = s.Sample(sampleContext)
			cancel()
			timer.Reset(s.hostInterval)
		}
	}
}

func (s *Service) Sample(ctx context.Context) error {
	now := s.now().UTC()
	summary, systemErr := s.system.CollectRuntime(ctx)
	if systemErr != nil {
		s.recordFailure(systemErr)
		return fmt.Errorf("collect host monitoring sample: %w", systemErr)
	}
	record := diskRecord{
		Version: recordVersion, CollectedAt: now,
		Host:       hostPoint(summary, now),
		Containers: []diskContainerPoint{},
	}

	s.mu.Lock()
	includeContainers := s.nextContainerAt.IsZero() || !now.Before(s.nextContainerAt)
	if includeContainers {
		s.nextContainerAt = now.Add(s.containerInterval)
	}
	includeOperatorLatency := s.operatorLatency != nil &&
		(s.nextOperatorLatencyAt.IsZero() || !now.Before(s.nextOperatorLatencyAt))
	if includeOperatorLatency {
		s.nextOperatorLatencyAt = now.Add(s.operatorLatencyInterval)
	}
	s.mu.Unlock()

	var operatorLatency <-chan []operatorLatencyResult
	if includeOperatorLatency {
		result := make(chan []operatorLatencyResult, 1)
		operatorLatency = result
		go func() {
			result <- collectOperatorLatency(ctx, s.operatorLatency)
		}()
	}

	var dockerErr error
	if includeContainers && s.docker != nil {
		record.ContainerSampled = true
		dockerContext, cancel := context.WithTimeout(ctx, s.sampleTimeout)
		batch, err := s.docker.RunningContainerStats(
			dockerContext,
			s.maxContainers,
			defaultMaxContainerJobs,
		)
		cancel()
		if err != nil {
			dockerErr = err
		} else {
			record.DockerAvailable = true
			record.ContainerTotal = batch.Total
			record.ContainerFailed = batch.Failed
			record.ContainerTruncated = batch.Truncated
			record.Containers = make([]diskContainerPoint, 0, len(batch.Items))
			for _, item := range batch.Items {
				record.Containers = append(record.Containers, diskContainerPoint{
					ID: item.ContainerID, Name: boundedText(item.Name, 128),
					Image: boundedText(item.Image, 256), CPUPercent: item.CPUPercent,
					MemoryBytes: item.MemoryBytes, MemoryLimitBytes: item.MemoryLimit,
					MemoryPercent: item.MemoryPercent, NetworkRxBytes: item.NetworkRx,
					NetworkTxBytes: item.NetworkTx, BlockReadBytes: item.BlockRead,
					BlockWriteBytes: item.BlockWrite, PIDs: item.PIDs,
				})
			}
		}
	}

	operatorLatencySuccessful := 0
	operatorLatencyFailed := 0
	if operatorLatency != nil {
		for _, item := range <-operatorLatency {
			point := diskOperatorLatencyPoint{ID: item.target.ID, Reachable: item.reachable}
			if item.reachable {
				point.LatencyMilliseconds = item.milliseconds
				operatorLatencySuccessful++
			} else {
				operatorLatencyFailed++
			}
			record.OperatorLatency = append(record.OperatorLatency, point)
		}
	}

	if err := s.appendRecord(record); err != nil {
		s.recordFailure(err)
		return err
	}
	bytes, limitReached, pruneErr := s.prune(now)
	if pruneErr != nil {
		s.recordFailure(pruneErr)
		return pruneErr
	}
	s.mu.Lock()
	status := s.baseStatus()
	status.StorageBytes = bytes
	status.StorageLimitReached = limitReached
	status.LastSampleAt = &now
	status.LastDockerAvailable = record.DockerAvailable
	status.LastContainerTotal = record.ContainerTotal
	status.LastContainerRecorded = len(record.Containers)
	status.LastContainerFailed = record.ContainerFailed
	status.LastContainerTruncated = record.ContainerTruncated
	if includeOperatorLatency {
		status.LastOperatorLatencyAt = &now
		status.LastOperatorLatencySuccessful = operatorLatencySuccessful
		status.LastOperatorLatencyFailed = operatorLatencyFailed
	} else {
		status.LastOperatorLatencyAt = s.status.LastOperatorLatencyAt
		status.LastOperatorLatencySuccessful = s.status.LastOperatorLatencySuccessful
		status.LastOperatorLatencyFailed = s.status.LastOperatorLatencyFailed
	}
	if dockerErr != nil {
		status.LastError = boundedError(dockerErr)
	}
	s.status = status
	s.mu.Unlock()
	return nil
}

func hostPoint(summary contract.SystemSummary, _ time.Time) diskHostPoint {
	disk := rootDisk(summary.Disks)
	return diskHostPoint{
		CPUPercent: summary.CPU.UsagePercent, CPUCores: summary.CPU.Cores,
		LoadOne: summary.Load.One, LoadFive: summary.Load.Five, LoadFifteen: summary.Load.Fifteen,
		MemoryUsedBytes: summary.Memory.UsedBytes, MemoryTotalBytes: summary.Memory.TotalBytes,
		SwapUsedBytes: summary.Memory.SwapUsedBytes, SwapTotalBytes: summary.Memory.SwapTotalBytes,
		DiskUsedBytes: disk.UsedBytes, DiskTotalBytes: disk.TotalBytes, DiskPercent: disk.UsagePercent,
		DiskIOAvailable: summary.DiskIO.Available,
		DiskReadBytes:   summary.DiskIO.ReadBytes, DiskWriteBytes: summary.DiskIO.WriteBytes,
		NetworkRxBytes: summary.Network.ReceivedBytes, NetworkTxBytes: summary.Network.SentBytes,
		TCPConnections: summary.Network.TCPConnections, UDPConnections: summary.Network.UDPConnections,
	}
}

func (point diskHostPoint) contractPoint(at time.Time) contract.MonitoringHostPoint {
	return contract.MonitoringHostPoint{
		CollectedAt: at,
		CPUPercent:  point.CPUPercent, CPUCores: point.CPUCores,
		LoadOne: point.LoadOne, LoadFive: point.LoadFive, LoadFifteen: point.LoadFifteen,
		MemoryUsedBytes: point.MemoryUsedBytes, MemoryTotalBytes: point.MemoryTotalBytes,
		SwapUsedBytes: point.SwapUsedBytes, SwapTotalBytes: point.SwapTotalBytes,
		DiskUsedBytes: point.DiskUsedBytes, DiskTotalBytes: point.DiskTotalBytes,
		DiskPercent:     point.DiskPercent,
		DiskIOAvailable: point.DiskIOAvailable,
		DiskReadBytes:   point.DiskReadBytes, DiskWriteBytes: point.DiskWriteBytes,
		NetworkRxBytes: point.NetworkRxBytes, NetworkTxBytes: point.NetworkTxBytes,
		TCPConnections: point.TCPConnections, UDPConnections: point.UDPConnections,
	}
}

func rootDisk(disks []contract.DiskSummary) contract.DiskSummary {
	for _, disk := range disks {
		if disk.MountPoint == "/" {
			return disk
		}
	}
	var selected contract.DiskSummary
	for _, disk := range disks {
		if disk.TotalBytes > selected.TotalBytes {
			selected = disk
		}
	}
	return selected
}

func (s *Service) appendRecord(record diskRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode monitoring sample: %w", err)
	}
	if len(data)+1 > maxHistoryLineBytes {
		return errors.New("monitoring sample exceeds the per-record limit")
	}
	path := filepath.Join(s.stateDir, "metrics-"+record.CollectedAt.Format("2006-01-02")+".jsonl")
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("monitoring shard is not a regular file")
		}
		if info.Size()+int64(len(data)+1) > s.maxDailyBytes {
			s.setStorageLimit(info.Size())
			return errors.New("monitoring daily storage limit reached")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect monitoring shard: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open monitoring shard: %w", err)
	}
	defer file.Close()
	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("protect monitoring shard: %w", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append monitoring sample: %w", err)
	}
	return nil
}

func (s *Service) setStorageLimit(bytes int64) {
	s.mu.Lock()
	s.status.StorageBytes = bytes
	s.status.StorageLimitReached = true
	s.status.LastError = "历史监控存储已达到固定上限"
	s.mu.Unlock()
}

func (s *Service) recordFailure(err error) {
	s.mu.Lock()
	s.status.LastError = boundedError(err)
	s.mu.Unlock()
}

func boundedError(err error) string {
	if err == nil {
		return ""
	}
	return boundedText(strings.NewReplacer("\r", " ", "\n", " ").Replace(err.Error()), 256)
}

func boundedText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func (s *Service) prune(now time.Time) (int64, bool, error) {
	entries, err := os.ReadDir(s.stateDir)
	if err != nil {
		return 0, false, fmt.Errorf("read monitoring state directory: %w", err)
	}
	type shard struct {
		name string
		date time.Time
		size int64
	}
	cutoffDate := midnightUTC(now).AddDate(0, 0, -(s.retentionDays - 1))
	shards := make([]shard, 0, len(entries))
	var total int64
	for _, entry := range entries {
		match := metricFilePattern.FindStringSubmatch(entry.Name())
		if len(match) != 2 {
			continue
		}
		path := filepath.Join(s.stateDir, entry.Name())
		info, statErr := os.Lstat(path)
		if statErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		date, parseErr := time.Parse("2006-01-02", match[1])
		if parseErr != nil {
			continue
		}
		if date.Before(cutoffDate) {
			if removeErr := os.Remove(path); removeErr != nil {
				return total, false, fmt.Errorf("remove expired monitoring shard: %w", removeErr)
			}
			continue
		}
		shards = append(shards, shard{name: entry.Name(), date: date, size: info.Size()})
		total += info.Size()
	}
	sort.Slice(shards, func(i, j int) bool { return shards[i].date.Before(shards[j].date) })
	current := "metrics-" + now.Format("2006-01-02") + ".jsonl"
	for _, item := range shards {
		if total <= s.maxStorageBytes {
			break
		}
		if item.name == current {
			continue
		}
		if err := os.Remove(filepath.Join(s.stateDir, item.name)); err != nil {
			return total, true, fmt.Errorf("remove excess monitoring shard: %w", err)
		}
		total -= item.size
	}
	return total, total >= s.maxStorageBytes, nil
}

func midnightUTC(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

type rangeSpec struct {
	name     string
	duration time.Duration
	bucket   time.Duration
}

func parseRange(value string) (rangeSpec, error) {
	switch value {
	case "", "24h":
		return rangeSpec{name: "24h", duration: 24 * time.Hour, bucket: 2 * time.Minute}, nil
	case "1h":
		return rangeSpec{name: "1h", duration: time.Hour, bucket: time.Minute}, nil
	case "6h":
		return rangeSpec{name: "6h", duration: 6 * time.Hour, bucket: time.Minute}, nil
	case "7d":
		return rangeSpec{name: "7d", duration: 7 * 24 * time.Hour, bucket: 30 * time.Minute}, nil
	default:
		return rangeSpec{}, ErrInvalidRange
	}
}

func (s *Service) History(ctx context.Context, requestedRange string) (contract.MonitoringHistory, error) {
	spec, err := parseRange(requestedRange)
	if err != nil {
		return contract.MonitoringHistory{}, err
	}
	select {
	case s.querySlots <- struct{}{}:
		defer func() { <-s.querySlots }()
	default:
		return contract.MonitoringHistory{}, ErrBusy
	}
	now := s.now().UTC()
	start := now.Add(-spec.duration)
	result := contract.MonitoringHistory{
		Range: spec.name, StartedAt: start, EndedAt: now,
		BucketSeconds:   int(spec.bucket.Seconds()),
		Host:            []contract.MonitoringHostPoint{},
		Containers:      []contract.MonitoringContainerSeries{},
		OperatorLatency: operatorLatencyCatalog(),
		Storage:         s.Status(),
	}
	hostPoints := make([]contract.MonitoringHostPoint, 0, maxHistoryPoints)
	containerPoints := make(map[string]*contract.MonitoringContainerSeries)
	var previousHost *contract.MonitoringHostPoint
	previousContainers := make(map[string]contract.MonitoringContainerPoint)
	operatorLatencySeries := make(map[string]*contract.MonitoringOperatorLatencySeries)
	for index := range result.OperatorLatency {
		series := &result.OperatorLatency[index]
		operatorLatencySeries[series.ID] = series
	}
	scanned, skipped, err := s.scanRecords(ctx, start, now, func(record diskRecord) {
		host := record.Host.contractPoint(record.CollectedAt)
		if previousHost != nil {
			seconds := host.CollectedAt.Sub(previousHost.CollectedAt).Seconds()
			host.NetworkRxRate = counterRate(previousHost.NetworkRxBytes, host.NetworkRxBytes, seconds)
			host.NetworkTxRate = counterRate(previousHost.NetworkTxBytes, host.NetworkTxBytes, seconds)
			if previousHost.DiskIOAvailable && host.DiskIOAvailable {
				host.DiskReadRate = counterRate(previousHost.DiskReadBytes, host.DiskReadBytes, seconds)
				host.DiskWriteRate = counterRate(previousHost.DiskWriteBytes, host.DiskWriteBytes, seconds)
			}
		}
		hostCopy := host
		previousHost = &hostCopy
		hostPoints = appendHostBucket(hostPoints, host, spec.bucket)
		for _, container := range record.Containers {
			if _, exists := containerPoints[container.ID]; !exists && len(containerPoints) >= maxScannedSeries {
				result.TruncatedSeries++
				continue
			}
			point := contract.MonitoringContainerPoint{
				CollectedAt: record.CollectedAt, CPUPercent: container.CPUPercent,
				MemoryBytes: container.MemoryBytes, MemoryLimitBytes: container.MemoryLimitBytes,
				MemoryPercent: container.MemoryPercent, NetworkRxBytes: container.NetworkRxBytes,
				NetworkTxBytes: container.NetworkTxBytes, BlockReadBytes: container.BlockReadBytes,
				BlockWriteBytes: container.BlockWriteBytes, PIDs: container.PIDs,
			}
			if previous, ok := previousContainers[container.ID]; ok {
				seconds := point.CollectedAt.Sub(previous.CollectedAt).Seconds()
				point.NetworkRxRate = counterRate(previous.NetworkRxBytes, point.NetworkRxBytes, seconds)
				point.NetworkTxRate = counterRate(previous.NetworkTxBytes, point.NetworkTxBytes, seconds)
				point.BlockReadRate = counterRate(previous.BlockReadBytes, point.BlockReadBytes, seconds)
				point.BlockWriteRate = counterRate(previous.BlockWriteBytes, point.BlockWriteBytes, seconds)
			}
			previousContainers[container.ID] = point
			series := containerPoints[container.ID]
			if series == nil {
				series = &contract.MonitoringContainerSeries{
					ContainerID: container.ID, Name: container.Name, Image: container.Image,
					Points: []contract.MonitoringContainerPoint{},
				}
				containerPoints[container.ID] = series
			}
			series.Name = container.Name
			series.Image = container.Image
			series.Points = appendContainerBucket(series.Points, point, spec.bucket)
		}
		for _, latency := range record.OperatorLatency {
			series := operatorLatencySeries[latency.ID]
			if series == nil {
				continue
			}
			point := contract.MonitoringOperatorLatencyPoint{CollectedAt: record.CollectedAt}
			if latency.Reachable {
				value := latency.LatencyMilliseconds
				point.LatencyMilliseconds = &value
			}
			series.Points = appendOperatorLatencyBucket(series.Points, point, spec.bucket)
		}
	})
	if err != nil {
		return contract.MonitoringHistory{}, err
	}
	result.ScannedBytes = scanned
	result.SkippedLines = skipped
	result.Host = hostPoints
	series := make([]contract.MonitoringContainerSeries, 0, len(containerPoints))
	for _, item := range containerPoints {
		series = append(series, *item)
	}
	sortContainerSeries(series)
	if len(series) > maxHistorySeries {
		result.TruncatedSeries += len(series) - maxHistorySeries
		series = series[:maxHistorySeries]
	}
	result.Containers = series
	return result, nil
}

func (s *Service) Status() contract.MonitoringStorageStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *Service) scanRecords(
	ctx context.Context,
	start time.Time,
	end time.Time,
	consume func(diskRecord),
) (int64, int, error) {
	entries, err := os.ReadDir(s.stateDir)
	if err != nil {
		return 0, 0, fmt.Errorf("read monitoring shards: %w", err)
	}
	type namedShard struct {
		name string
		date time.Time
		size int64
	}
	shards := make([]namedShard, 0, len(entries))
	for _, entry := range entries {
		match := metricFilePattern.FindStringSubmatch(entry.Name())
		if len(match) != 2 {
			continue
		}
		date, parseErr := time.Parse("2006-01-02", match[1])
		if parseErr != nil || date.Before(midnightUTC(start)) || date.After(midnightUTC(end)) {
			continue
		}
		path := filepath.Join(s.stateDir, entry.Name())
		info, statErr := os.Lstat(path)
		if statErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		shards = append(shards, namedShard{name: entry.Name(), date: date, size: info.Size()})
	}
	sort.Slice(shards, func(i, j int) bool { return shards[i].date.Before(shards[j].date) })
	var scanned int64
	skipped := 0
	for _, shard := range shards {
		if err := ctx.Err(); err != nil {
			return scanned, skipped, err
		}
		if scanned+shard.size > s.maxStorageBytes {
			break
		}
		path := filepath.Join(s.stateDir, shard.name)
		file, openErr := os.Open(path)
		if openErr != nil {
			return scanned, skipped, fmt.Errorf("open monitoring shard: %w", openErr)
		}
		scanner := bufio.NewScanner(io.LimitReader(file, shard.size))
		scanner.Buffer(make([]byte, 64<<10), maxHistoryLineBytes)
		for scanner.Scan() {
			if err := ctx.Err(); err != nil {
				_ = file.Close()
				return scanned, skipped, err
			}
			var record diskRecord
			if err := json.Unmarshal(scanner.Bytes(), &record); err != nil ||
				record.Version != recordVersion {
				skipped++
				continue
			}
			if record.CollectedAt.Before(start) || record.CollectedAt.After(end) {
				continue
			}
			consume(record)
		}
		if scanErr := scanner.Err(); scanErr != nil {
			skipped++
		}
		if closeErr := file.Close(); closeErr != nil {
			return scanned, skipped, fmt.Errorf("close monitoring shard: %w", closeErr)
		}
		scanned += shard.size
	}
	return scanned, skipped, nil
}

func counterRate(previous uint64, current uint64, seconds float64) float64 {
	if seconds <= 0 || current < previous {
		return 0
	}
	return float64(current-previous) / seconds
}

func operatorLatencyCatalog() []contract.MonitoringOperatorLatencySeries {
	series := make([]contract.MonitoringOperatorLatencySeries, 0, len(operatorLatencyTargets))
	for _, target := range operatorLatencyTargets {
		series = append(series, contract.MonitoringOperatorLatencySeries{
			ID: target.ID, Operator: target.Operator, Region: target.Region,
			Address: target.Address, Points: []contract.MonitoringOperatorLatencyPoint{},
		})
	}
	return series
}

func appendOperatorLatencyBucket(
	points []contract.MonitoringOperatorLatencyPoint,
	point contract.MonitoringOperatorLatencyPoint,
	width time.Duration,
) []contract.MonitoringOperatorLatencyPoint {
	bucket := point.CollectedAt.Unix() / int64(width.Seconds())
	if len(points) == 0 ||
		points[len(points)-1].CollectedAt.Unix()/int64(width.Seconds()) != bucket {
		points = append(points, point)
		if len(points) > maxHistoryPoints {
			copy(points, points[len(points)-maxHistoryPoints:])
			points = points[:maxHistoryPoints]
		}
		return points
	}
	last := &points[len(points)-1]
	if point.LatencyMilliseconds != nil &&
		(last.LatencyMilliseconds == nil || *point.LatencyMilliseconds > *last.LatencyMilliseconds) {
		*last = point
	} else if last.LatencyMilliseconds == nil {
		last.CollectedAt = point.CollectedAt
	}
	return points
}

func appendHostBucket(
	points []contract.MonitoringHostPoint,
	point contract.MonitoringHostPoint,
	width time.Duration,
) []contract.MonitoringHostPoint {
	bucket := point.CollectedAt.Unix() / int64(width.Seconds())
	if len(points) == 0 ||
		points[len(points)-1].CollectedAt.Unix()/int64(width.Seconds()) != bucket {
		points = append(points, point)
		if len(points) > maxHistoryPoints {
			copy(points, points[len(points)-maxHistoryPoints:])
			points = points[:maxHistoryPoints]
		}
		return points
	}
	last := &points[len(points)-1]
	if point.CPUPercent > last.CPUPercent {
		last.CPUPercent = point.CPUPercent
	}
	if ratio(point.MemoryUsedBytes, point.MemoryTotalBytes) >
		ratio(last.MemoryUsedBytes, last.MemoryTotalBytes) {
		last.MemoryUsedBytes = point.MemoryUsedBytes
		last.MemoryTotalBytes = point.MemoryTotalBytes
	}
	if ratio(point.SwapUsedBytes, point.SwapTotalBytes) >
		ratio(last.SwapUsedBytes, last.SwapTotalBytes) {
		last.SwapUsedBytes = point.SwapUsedBytes
		last.SwapTotalBytes = point.SwapTotalBytes
	}
	if point.DiskPercent > last.DiskPercent {
		last.DiskUsedBytes = point.DiskUsedBytes
		last.DiskTotalBytes = point.DiskTotalBytes
		last.DiskPercent = point.DiskPercent
	}
	if point.DiskReadRate > last.DiskReadRate {
		last.DiskReadRate = point.DiskReadRate
	}
	if point.DiskWriteRate > last.DiskWriteRate {
		last.DiskWriteRate = point.DiskWriteRate
	}
	if point.NetworkRxRate > last.NetworkRxRate {
		last.NetworkRxRate = point.NetworkRxRate
	}
	if point.NetworkTxRate > last.NetworkTxRate {
		last.NetworkTxRate = point.NetworkTxRate
	}
	last.CollectedAt = point.CollectedAt
	last.NetworkRxBytes = point.NetworkRxBytes
	last.NetworkTxBytes = point.NetworkTxBytes
	last.DiskIOAvailable = point.DiskIOAvailable
	last.DiskReadBytes = point.DiskReadBytes
	last.DiskWriteBytes = point.DiskWriteBytes
	last.LoadOne = point.LoadOne
	last.LoadFive = point.LoadFive
	last.LoadFifteen = point.LoadFifteen
	last.TCPConnections = point.TCPConnections
	last.UDPConnections = point.UDPConnections
	return points
}

func appendContainerBucket(
	points []contract.MonitoringContainerPoint,
	point contract.MonitoringContainerPoint,
	width time.Duration,
) []contract.MonitoringContainerPoint {
	bucket := point.CollectedAt.Unix() / int64(width.Seconds())
	if len(points) == 0 ||
		points[len(points)-1].CollectedAt.Unix()/int64(width.Seconds()) != bucket {
		points = append(points, point)
		if len(points) > maxHistoryPoints {
			copy(points, points[len(points)-maxHistoryPoints:])
			points = points[:maxHistoryPoints]
		}
		return points
	}
	last := &points[len(points)-1]
	if point.CPUPercent > last.CPUPercent {
		last.CPUPercent = point.CPUPercent
	}
	if point.MemoryPercent > last.MemoryPercent {
		last.MemoryBytes = point.MemoryBytes
		last.MemoryLimitBytes = point.MemoryLimitBytes
		last.MemoryPercent = point.MemoryPercent
	}
	if point.NetworkRxRate > last.NetworkRxRate {
		last.NetworkRxRate = point.NetworkRxRate
	}
	if point.NetworkTxRate > last.NetworkTxRate {
		last.NetworkTxRate = point.NetworkTxRate
	}
	if point.BlockReadRate > last.BlockReadRate {
		last.BlockReadRate = point.BlockReadRate
	}
	if point.BlockWriteRate > last.BlockWriteRate {
		last.BlockWriteRate = point.BlockWriteRate
	}
	last.CollectedAt = point.CollectedAt
	last.NetworkRxBytes = point.NetworkRxBytes
	last.NetworkTxBytes = point.NetworkTxBytes
	last.BlockReadBytes = point.BlockReadBytes
	last.BlockWriteBytes = point.BlockWriteBytes
	last.PIDs = point.PIDs
	return points
}

func ratio(used uint64, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(used) / float64(total)
}

func sortContainerSeries(series []contract.MonitoringContainerSeries) {
	sort.Slice(series, func(i, j int) bool {
		left, leftOK := latestContainerPoint(series[i])
		right, rightOK := latestContainerPoint(series[j])
		if leftOK != rightOK {
			return leftOK
		}
		if leftOK && !left.CollectedAt.Equal(right.CollectedAt) {
			return left.CollectedAt.After(right.CollectedAt)
		}
		if leftOK && left.MemoryBytes != right.MemoryBytes {
			return left.MemoryBytes > right.MemoryBytes
		}
		if leftOK && left.CPUPercent != right.CPUPercent {
			return left.CPUPercent > right.CPUPercent
		}
		if series[i].Name != series[j].Name {
			return series[i].Name < series[j].Name
		}
		return series[i].ContainerID < series[j].ContainerID
	})
}

func latestContainerPoint(series contract.MonitoringContainerSeries) (contract.MonitoringContainerPoint, bool) {
	if len(series.Points) == 0 {
		return contract.MonitoringContainerPoint{}, false
	}
	return series.Points[len(series.Points)-1], true
}
