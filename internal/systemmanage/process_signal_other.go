//go:build !linux

package systemmanage

import "errors"

const processSignalSupported = false

func platformProcessSignaler(int, string) error {
	return errors.New("process signaling requires Linux")
}
