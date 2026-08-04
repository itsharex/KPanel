package systeminfo

import (
	"context"
	"time"
)

const (
	maxProcessScan  = 8192
	maxProcessItems = 32
)

type ProcessMetric struct {
	PID            int     `json:"pid"`
	ParentPID      int     `json:"parentPid"`
	Name           string  `json:"name"`
	State          string  `json:"state"`
	UserID         int     `json:"userId"`
	CPUPercent     float64 `json:"cpuPercent"`
	MemoryBytes    uint64  `json:"memoryBytes"`
	StartTimeTicks uint64  `json:"startTimeTicks"`
}

type ProcessSnapshot struct {
	TopCPU         []ProcessMetric `json:"topCpu"`
	TopMemory      []ProcessMetric `json:"topMemory"`
	Scanned        int             `json:"scanned"`
	Truncated      bool            `json:"truncated"`
	SampleDuration time.Duration   `json:"sampleDuration"`
	CollectedAt    time.Time       `json:"collectedAt"`
}

func (c *Collector) Processes(ctx context.Context) (ProcessSnapshot, error) {
	c.prepareDefaults()
	return c.collectProcesses(ctx)
}
