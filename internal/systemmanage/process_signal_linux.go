//go:build linux

package systemmanage

import "syscall"

const processSignalSupported = true

func platformProcessSignaler(pid int, signal string) error {
	value := syscall.SIGTERM
	if signal == "kill" {
		value = syscall.SIGKILL
	}
	return syscall.Kill(pid, value)
}
