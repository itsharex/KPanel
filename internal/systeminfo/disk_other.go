//go:build !linux

package systeminfo

func diskUsage(_ string) (total, used uint64, usagePercent float64, ok bool) {
	return 0, 0, 0, false
}
