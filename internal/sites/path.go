package sites

import (
	"path/filepath"
	"strings"
)

func pathWithin(candidate, root string) bool {
	candidate = filepath.Clean(candidate)
	root = filepath.Clean(root)
	return candidate == root || strings.HasPrefix(candidate, root+string(filepath.Separator))
}
