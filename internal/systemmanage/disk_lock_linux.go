//go:build linux

package systemmanage

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func (m *Manager) acquireDiskProcessLock(ctx context.Context) (func(), error) {
	if err := m.ensureDiskJobDir(); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(m.diskJobDir()+"/writer.lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	for {
		if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err == nil {
			return func() {
				_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
				_ = file.Close()
			}, nil
		} else if err != unix.EWOULDBLOCK && err != unix.EAGAIN {
			_ = file.Close()
			return nil, fmt.Errorf("lock disk writer: %w", err)
		}
		select {
		case <-ctx.Done():
			_ = file.Close()
			return nil, ctx.Err()
		case <-deadline.C:
			_ = file.Close()
			return nil, fmt.Errorf("%w: disk writer lock is busy", ErrConflict)
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func syncDiskStateDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
