//go:build linux

package dockerx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

const maxComposeCommandOutput = 256 << 10

type composeOutputBuffer struct {
	bytes.Buffer
}

func (buffer *composeOutputBuffer) Write(value []byte) (int, error) {
	written := len(value)
	remaining := maxComposeCommandOutput - buffer.Len()
	if remaining > 0 {
		if len(value) > remaining {
			value = value[:remaining]
		}
		_, _ = buffer.Buffer.Write(value)
	}
	return written, nil
}

func runFixedDockerComposeCommand(ctx context.Context, arguments ...string) ([]byte, error) {
	for _, candidate := range []string{"/usr/bin/docker", "/bin/docker"} {
		resolved, err := trustedDockerHostExecutable(candidate)
		if err != nil {
			continue
		}
		var output composeOutputBuffer
		command := exec.CommandContext(ctx, resolved, arguments...)
		command.Stdout = &output
		command.Stderr = &output
		if err := command.Run(); err != nil {
			return output.Bytes(), fmt.Errorf("docker compose failed: %w", err)
		}
		return output.Bytes(), nil
	}
	return nil, errors.New("docker compose executable is unavailable")
}
