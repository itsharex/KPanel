//go:build !linux

package systemmanage

import "os"

func dnsScriptOwnerTrusted(os.FileInfo) bool {
	return false
}
