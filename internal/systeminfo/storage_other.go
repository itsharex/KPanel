//go:build !linux

package systeminfo

import (
	"context"
	"errors"
)

func (c *Collector) collectStorageUsage(context.Context, string) (StorageUsage, error) {
	return StorageUsage{}, errors.New("storage analysis requires Linux")
}
