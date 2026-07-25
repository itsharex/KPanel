//go:build !linux && !windows

package store

import (
	"errors"
	"fmt"
	"os"
)

type fileProcessLock struct {
	file *os.File
	path string
}

func acquireProcessLock(path string) (processLock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("%w: %s", ErrStoreLocked, path)
		}
		return nil, fmt.Errorf("create store lock: %w", err)
	}
	return &fileProcessLock{file: file, path: path}, nil
}

func (l *fileProcessLock) Close() error {
	return errors.Join(l.file.Close(), os.Remove(l.path))
}
