//go:build !linux

package systeminfo

import (
	"context"
	"errors"
)

func (c *Collector) collectProcesses(context.Context, ProcessQuery) (ProcessSnapshot, error) {
	return ProcessSnapshot{}, errors.New("process metrics require Linux")
}
