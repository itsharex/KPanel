//go:build !linux

package dockerx

func daemonProcessRunning(_ string) bool {
	return false
}
