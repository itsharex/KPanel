package appmarket

import (
	"io"
	"os/exec"
)

type terminalProcess interface {
	io.ReadWriteCloser
	Wait() error
	Kill() error
}

func startTerminalProcess(command *exec.Cmd, rows, columns uint16) (terminalProcess, error) {
	return startPlatformTerminalProcess(command, rows, columns)
}
