//go:build !linux

package systemmanage

func diskMountUsage(string) (*uint64, *uint64, *uint64, *float64) {
	return nil, nil, nil, nil
}
