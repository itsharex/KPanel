package systeminfo

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	maxProcessScan        = 8192
	maxProcessItems       = 32
	MaxProcessResults     = 256
	DefaultProcessResults = 200
	MaxProcessSearchBytes = 128
)

type ProcessQuery struct {
	Search string
	Sort   string
	Order  string
	Limit  int
}

type ProcessMetric struct {
	PID            int     `json:"pid"`
	ParentPID      int     `json:"parentPid"`
	Name           string  `json:"name"`
	State          string  `json:"state"`
	UserID         int     `json:"userId"`
	User           string  `json:"user,omitempty"`
	CPUPercent     float64 `json:"cpuPercent"`
	MemoryBytes    uint64  `json:"memoryBytes"`
	Threads        int     `json:"threads"`
	Nice           int     `json:"nice"`
	StartTimeTicks uint64  `json:"startTimeTicks"`
}

type ProcessSummary struct {
	CPUPercent       float64 `json:"cpuPercent"`
	MemoryUsedBytes  uint64  `json:"memoryUsedBytes"`
	MemoryTotalBytes uint64  `json:"memoryTotalBytes"`
	Total            int     `json:"total"`
	Running          int     `json:"running"`
	Sleeping         int     `json:"sleeping"`
	Stopped          int     `json:"stopped"`
	Zombie           int     `json:"zombie"`
}

type ProcessSnapshot struct {
	TopCPU         []ProcessMetric `json:"topCpu"`
	TopMemory      []ProcessMetric `json:"topMemory"`
	Items          []ProcessMetric `json:"items,omitempty"`
	Total          int             `json:"total,omitempty"`
	Summary        ProcessSummary  `json:"summary"`
	Scanned        int             `json:"scanned"`
	Truncated      bool            `json:"truncated"`
	SampleDuration time.Duration   `json:"sampleDuration"`
	CollectedAt    time.Time       `json:"collectedAt"`
}

func (c *Collector) Processes(ctx context.Context) (ProcessSnapshot, error) {
	c.prepareDefaults()
	return c.collectProcesses(ctx, ProcessQuery{})
}

func (c *Collector) QueryProcesses(ctx context.Context, query ProcessQuery) (ProcessSnapshot, error) {
	c.prepareDefaults()
	normalized, err := NormalizeProcessQuery(query)
	if err != nil {
		return ProcessSnapshot{}, err
	}
	return c.collectProcesses(ctx, normalized)
}

func NormalizeProcessQuery(query ProcessQuery) (ProcessQuery, error) {
	query.Search = strings.TrimSpace(query.Search)
	query.Sort = strings.ToLower(strings.TrimSpace(query.Sort))
	query.Order = strings.ToLower(strings.TrimSpace(query.Order))
	if len(query.Search) > MaxProcessSearchBytes || strings.ContainsAny(query.Search, "\x00\r\n") {
		return ProcessQuery{}, errors.New("invalid process search")
	}
	if query.Sort == "" {
		query.Sort = "cpu"
	}
	switch query.Sort {
	case "cpu", "memory", "pid", "name", "user", "state", "threads":
	default:
		return ProcessQuery{}, errors.New("invalid process sort")
	}
	if query.Order == "" {
		query.Order = "desc"
	}
	if query.Order != "asc" && query.Order != "desc" {
		return ProcessQuery{}, errors.New("invalid process order")
	}
	if query.Limit == 0 {
		query.Limit = DefaultProcessResults
	}
	if query.Limit < 1 || query.Limit > MaxProcessResults {
		return ProcessQuery{}, errors.New("invalid process limit")
	}
	return query, nil
}
