//go:build linux

package store

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

type fileProcessLock struct {
	file *os.File
}

func acquireProcessLock(path string) (processLock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open store lock: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("protect store lock: %w", err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return nil, fmt.Errorf("%w: %s", ErrStoreLocked, path)
		}
		return nil, fmt.Errorf("lock store: %w", err)
	}
	return &fileProcessLock{file: file}, nil
}

func (l *fileProcessLock) Close() error {
	unlockErr := unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
	closeErr := l.file.Close()
	return errors.Join(unlockErr, closeErr)
}
