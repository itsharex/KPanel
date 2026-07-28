//go:build linux

package webenv

import (
	"os"
	"syscall"
)

func ownerTrusted(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == 0
}

func currentEUID() int { return os.Geteuid() }
