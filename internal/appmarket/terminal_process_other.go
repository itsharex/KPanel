//go:build !linux

package appmarket

import (
	"errors"
	"io"
	"os"
	"os/exec"
)

func startPlatformTerminalProcess(
	*exec.Cmd,
	uint16,
	uint16,
) (terminalProcess, error) {
	return nil, errors.New("interactive application terminals require Linux")
}

func createTerminalInput(string) error {
	return nil
}

func openTerminalInput(string) (*os.File, error) {
	return nil, errors.New("interactive application terminals require Linux")
}

func writeTerminalInput(string, []byte) error {
	return errors.New("interactive application terminals require Linux")
}

func removeTerminalInput(string) error {
	return nil
}

func isTerminalEnd(err error) bool {
	return errors.Is(err, io.EOF)
}
