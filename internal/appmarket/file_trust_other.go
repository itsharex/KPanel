//go:build !linux

package appmarket

import "os"

func trustedFileOwner(os.FileInfo) bool {
	return true
}
