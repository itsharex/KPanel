//go:build linux

package systeminfo

import "syscall"

func diskUsage(path string) (total, available uint64, ok bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, false
	}
	if stat.Bsize <= 0 {
		return 0, 0, false
	}
	blockSize := uint64(stat.Bsize)
	if stat.Blocks > ^uint64(0)/blockSize || stat.Bavail > ^uint64(0)/blockSize {
		return 0, 0, false
	}
	return stat.Blocks * blockSize, stat.Bavail * blockSize, true
}
