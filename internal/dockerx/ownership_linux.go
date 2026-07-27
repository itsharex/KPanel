//go:build linux

package dockerx

import (
	"errors"
	"os"
	"syscall"
)

func fileNumericOwnership(info os.FileInfo) (int, int, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, errors.New("file numeric ownership is unavailable")
	}
	return int(stat.Uid), int(stat.Gid), nil
}

func applyNumericOwnership(path string, uid, gid int) error {
	if uid < 0 || gid < 0 {
		return errors.New("file numeric ownership is invalid")
	}
	return os.Chown(path, uid, gid)
}

func currentNumericOwnership() (int, int) {
	return os.Getuid(), os.Getgid()
}
