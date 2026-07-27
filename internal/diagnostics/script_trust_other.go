//go:build !linux

package diagnostics

import "os"

func trustedScriptOwner(os.FileInfo) bool {
	return true
}
