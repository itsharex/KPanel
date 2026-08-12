//go:build !linux

package dockerx

import (
	"context"
	"errors"
)

func runFixedDockerComposeCommand(context.Context, ...string) ([]byte, error) {
	return nil, errors.New("Docker Compose deployment is supported only on Linux")
}
