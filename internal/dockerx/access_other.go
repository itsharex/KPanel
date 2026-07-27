//go:build !linux

package dockerx

import (
	"context"
	"errors"
)

func (c *Client) updateContainerAccess(
	context.Context,
	string,
	string,
	bool,
	string,
) error {
	return errors.New("Docker access control is supported only on Linux")
}

func runFixedDockerHostCommand(
	context.Context,
	string,
	...string,
) ([]byte, error) {
	return nil, errors.New("Docker host commands are supported only on Linux")
}
