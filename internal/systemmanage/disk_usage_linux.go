//go:build linux

package systemmanage

import "golang.org/x/sys/unix"

func diskMountUsage(path string) (*uint64, *uint64, *uint64, *float64) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil || stat.Blocks == 0 || stat.Bsize <= 0 {
		return nil, nil, nil, nil
	}
	blockSize := uint64(stat.Bsize)
	total := stat.Blocks * blockSize
	available := stat.Bavail * blockSize
	free := stat.Bfree * blockSize
	used := uint64(0)
	if total >= free {
		used = total - free
	}
	percent := float64(used) * 100 / float64(total)
	return &total, &used, &available, &percent
}
