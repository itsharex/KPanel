//go:build windows

package filemanager

import "os"

func fileOwner(os.FileInfo) (string, string) {
	return "", ""
}

func preserveOwnership(_ string, _ os.FileInfo) error {
	return nil
}
