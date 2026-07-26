//go:build linux

package appmarket

import (
	"os"
	"syscall"
)

func trustedFileOwner(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == 0
}
