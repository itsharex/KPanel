//go:build linux

package sites

import (
	"os"
	"syscall"
)

func recipeScriptOwnerTrusted(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == 0
}
