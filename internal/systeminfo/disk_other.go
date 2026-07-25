//go:build !linux

package systeminfo

func diskUsage(_ string) (total, available uint64, ok bool) {
	return 0, 0, false
}
