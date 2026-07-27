//go:build !linux

package dockerx

import "os"

func fileNumericOwnership(_ os.FileInfo) (int, int, error) {
	return 0, 0, nil
}

func applyNumericOwnership(_ string, _, _ int) error {
	return nil
}

func currentNumericOwnership() (int, int) {
	return 0, 0
}
