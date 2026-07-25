//go:build linux

package sites

import "golang.org/x/sys/unix"

func atomicNoReplace(source, target string) error {
	return unix.Renameat2(unix.AT_FDCWD, source, unix.AT_FDCWD, target, unix.RENAME_NOREPLACE)
}

func atomicExchange(left, right string) error {
	return unix.Renameat2(unix.AT_FDCWD, left, unix.AT_FDCWD, right, unix.RENAME_EXCHANGE)
}

func atomicSiteWritesSupported() bool {
	return true
}
