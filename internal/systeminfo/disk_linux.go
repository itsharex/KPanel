//go:build linux

package systeminfo

import "syscall"

func diskUsage(path string) (total, used uint64, usagePercent float64, ok bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0, false
	}
	if stat.Bsize <= 0 {
		return 0, 0, 0, false
	}
	return diskCapacity(stat.Blocks, stat.Bfree, stat.Bavail, uint64(stat.Bsize))
}
