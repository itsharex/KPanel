//go:build linux

package systemmanage

import (
	"os"

	"golang.org/x/sys/unix"
)

func renameReplace(source, target string) error {
	if _, err := os.Lstat(target); err == nil {
		return unix.Renameat2(unix.AT_FDCWD, source, unix.AT_FDCWD, target, 0)
	}
	return os.Rename(source, target)
}
