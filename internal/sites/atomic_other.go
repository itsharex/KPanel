//go:build !linux

package sites

import "errors"

func atomicNoReplace(_, _ string) error {
	return errors.New("atomic no-replace publication is only supported on Linux")
}

func atomicExchange(_, _ string) error {
	return errors.New("atomic exchange is only supported on Linux")
}

func atomicSiteWritesSupported() bool {
	return false
}
