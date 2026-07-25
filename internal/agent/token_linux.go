//go:build linux

package agent

import (
	"errors"
	"os"
	"syscall"
)

func validateTokenMetadata(info os.FileInfo, expectedGID int) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("cannot inspect Agent token ownership")
	}
	if stat.Uid != 0 || int(stat.Gid) != expectedGID {
		return errors.New("Agent token must be owned by root and the configured panel group")
	}
	if info.Mode().Perm() != 0o640 {
		return errors.New("Agent token permissions must be exactly 0640")
	}
	return nil
}
