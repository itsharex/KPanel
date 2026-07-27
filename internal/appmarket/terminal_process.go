package appmarket

import (
	"os"
	"os/exec"

	"github.com/kejilion/kejilion-panel/internal/hostpty"
)

type terminalProcess = hostpty.Process

func startTerminalProcess(command *exec.Cmd, rows, columns uint16) (terminalProcess, error) {
	return hostpty.Start(command, rows, columns)
}

func createTerminalInput(path string) error {
	return hostpty.CreateInput(path)
}

func openTerminalInput(path string) (*os.File, error) {
	return hostpty.OpenInput(path)
}

func writeTerminalInput(path string, data []byte) error {
	return hostpty.WriteInput(path, data)
}

func removeTerminalInput(path string) error {
	return hostpty.RemoveInput(path)
}

func isTerminalEnd(err error) bool {
	return hostpty.IsEnd(err)
}
