//go:build linux

package dockerx

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

func daemonProcessRunning(pidFile string) bool {
	info, err := os.Lstat(pidFile)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > 64 {
		return false
	}
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 1 {
		return false
	}
	comm, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/comm")
	if err != nil || strings.TrimSpace(string(comm)) != "dockerd" {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}
