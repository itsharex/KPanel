//go:build !linux

package sites

import "os"

func recipeScriptOwnerTrusted(os.FileInfo) bool {
	return false
}
