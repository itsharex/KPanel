package systeminfo

import (
	"context"
	"time"
)

const (
	maxStorageEntries = 200_000
	maxStorageItems   = 64
)

type StorageUsageItem struct {
	Path      string `json:"path"`
	Kind      string `json:"kind"`
	SizeBytes uint64 `json:"sizeBytes"`
	Scanned   int    `json:"scanned"`
	Truncated bool   `json:"truncated"`
}

type StorageUsage struct {
	Path        string             `json:"path"`
	Items       []StorageUsageItem `json:"items"`
	Scanned     int                `json:"scanned"`
	Truncated   bool               `json:"truncated"`
	CollectedAt time.Time          `json:"collectedAt"`
}

func (c *Collector) StorageUsage(ctx context.Context, path string) (StorageUsage, error) {
	c.prepareDefaults()
	return c.collectStorageUsage(ctx, path)
}
