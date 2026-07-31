package contract

import "time"

type MonitoringHostPoint struct {
	CollectedAt      time.Time `json:"collectedAt"`
	CPUPercent       float64   `json:"cpuPercent"`
	CPUCores         int       `json:"cpuCores"`
	LoadOne          float64   `json:"loadOne"`
	LoadFive         float64   `json:"loadFive"`
	LoadFifteen      float64   `json:"loadFifteen"`
	MemoryUsedBytes  uint64    `json:"memoryUsedBytes"`
	MemoryTotalBytes uint64    `json:"memoryTotalBytes"`
	SwapUsedBytes    uint64    `json:"swapUsedBytes"`
	SwapTotalBytes   uint64    `json:"swapTotalBytes"`
	DiskUsedBytes    uint64    `json:"diskUsedBytes"`
	DiskTotalBytes   uint64    `json:"diskTotalBytes"`
	DiskPercent      float64   `json:"diskPercent"`
	DiskIOAvailable  bool      `json:"diskIoAvailable"`
	DiskReadBytes    uint64    `json:"diskReadBytes"`
	DiskWriteBytes   uint64    `json:"diskWriteBytes"`
	DiskReadRate     float64   `json:"diskReadBytesPerSecond"`
	DiskWriteRate    float64   `json:"diskWriteBytesPerSecond"`
	NetworkRxBytes   uint64    `json:"networkRxBytes"`
	NetworkTxBytes   uint64    `json:"networkTxBytes"`
	NetworkRxRate    float64   `json:"networkRxBytesPerSecond"`
	NetworkTxRate    float64   `json:"networkTxBytesPerSecond"`
	TCPConnections   int       `json:"tcpConnections"`
	UDPConnections   int       `json:"udpConnections"`
}

type MonitoringContainerPoint struct {
	CollectedAt      time.Time `json:"collectedAt"`
	CPUPercent       float64   `json:"cpuPercent"`
	MemoryBytes      uint64    `json:"memoryBytes"`
	MemoryLimitBytes uint64    `json:"memoryLimitBytes"`
	MemoryPercent    float64   `json:"memoryPercent"`
	NetworkRxBytes   uint64    `json:"networkRxBytes"`
	NetworkTxBytes   uint64    `json:"networkTxBytes"`
	NetworkRxRate    float64   `json:"networkRxBytesPerSecond"`
	NetworkTxRate    float64   `json:"networkTxBytesPerSecond"`
	BlockReadBytes   uint64    `json:"blockReadBytes"`
	BlockWriteBytes  uint64    `json:"blockWriteBytes"`
	BlockReadRate    float64   `json:"blockReadBytesPerSecond"`
	BlockWriteRate   float64   `json:"blockWriteBytesPerSecond"`
	PIDs             uint64    `json:"pids"`
}

type MonitoringContainerSeries struct {
	ContainerID string                     `json:"containerId"`
	Name        string                     `json:"name"`
	Image       string                     `json:"image"`
	Points      []MonitoringContainerPoint `json:"points"`
}

type MonitoringStorageStatus struct {
	Enabled                  bool       `json:"enabled"`
	RetentionDays            int        `json:"retentionDays"`
	HostIntervalSeconds      int        `json:"hostIntervalSeconds"`
	ContainerIntervalSeconds int        `json:"containerIntervalSeconds"`
	MaxContainers            int        `json:"maxContainers"`
	StorageBytes             int64      `json:"storageBytes"`
	MaxStorageBytes          int64      `json:"maxStorageBytes"`
	LastSampleAt             *time.Time `json:"lastSampleAt,omitempty"`
	LastError                string     `json:"lastError,omitempty"`
	LastContainerTotal       int        `json:"lastContainerTotal"`
	LastContainerRecorded    int        `json:"lastContainerRecorded"`
	LastContainerFailed      int        `json:"lastContainerFailed"`
	LastContainerTruncated   int        `json:"lastContainerTruncated"`
	LastDockerAvailable      bool       `json:"lastDockerAvailable"`
	StorageLimitReached      bool       `json:"storageLimitReached"`
}

type MonitoringHistory struct {
	Range           string                      `json:"range"`
	StartedAt       time.Time                   `json:"startedAt"`
	EndedAt         time.Time                   `json:"endedAt"`
	BucketSeconds   int                         `json:"bucketSeconds"`
	Host            []MonitoringHostPoint       `json:"host"`
	Containers      []MonitoringContainerSeries `json:"containers"`
	Storage         MonitoringStorageStatus     `json:"storage"`
	ScannedBytes    int64                       `json:"scannedBytes"`
	SkippedLines    int                         `json:"skippedLines"`
	TruncatedSeries int                         `json:"truncatedSeries"`
}
