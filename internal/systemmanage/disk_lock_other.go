//go:build !linux

package systemmanage

import "context"

func (m *Manager) acquireDiskProcessLock(context.Context) (func(), error) {
	return func() {}, nil
}

func syncDiskStateDirectory(string) error { return nil }
