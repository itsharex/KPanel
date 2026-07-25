//go:build windows

package store

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

type fileProcessLock struct {
	file       *os.File
	overlapped windows.Overlapped
}

func acquireProcessLock(path string) (processLock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open store lock: %w", err)
	}
	lock := &fileProcessLock{file: file}
	err = windows.LockFileEx(
		windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1,
		0,
		&lock.overlapped,
	)
	if err != nil {
		_ = file.Close()
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) ||
			errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
			return nil, fmt.Errorf("%w: %s", ErrStoreLocked, path)
		}
		return nil, fmt.Errorf("lock store: %w", err)
	}
	return lock, nil
}

func (l *fileProcessLock) Close() error {
	unlockErr := windows.UnlockFileEx(
		windows.Handle(l.file.Fd()),
		0,
		1,
		0,
		&l.overlapped,
	)
	closeErr := l.file.Close()
	return errors.Join(unlockErr, closeErr)
}
