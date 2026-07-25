//go:build !linux

package agent

import (
	"errors"
	"os"
)

func validateTokenMetadata(info os.FileInfo, _ int) error {
	if info.Mode().Perm() != 0o640 {
		return errors.New("Agent token permissions must be exactly 0640")
	}
	return nil
}
