//go:build linux

package hostpty

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

type linuxProcess struct {
	*os.File
	command *exec.Cmd
}

func startPlatform(command *exec.Cmd, rows, columns uint16) (Process, error) {
	masterFD, err := unix.Open(
		"/dev/ptmx",
		unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC,
		0,
	)
	if err != nil {
		return nil, err
	}
	master := os.NewFile(uintptr(masterFD), "/dev/ptmx")
	closeMaster := true
	defer func() {
		if closeMaster {
			_ = master.Close()
		}
	}()
	if err := unix.IoctlSetPointerInt(masterFD, unix.TIOCSPTLCK, 0); err != nil {
		return nil, err
	}
	number, err := unix.IoctlGetInt(masterFD, unix.TIOCGPTN)
	if err != nil {
		return nil, err
	}
	slave, err := os.OpenFile(
		fmt.Sprintf("/dev/pts/%d", number),
		os.O_RDWR|syscall.O_NOCTTY,
		0,
	)
	if err != nil {
		return nil, err
	}
	defer slave.Close()
	if err := unix.IoctlSetWinsize(
		masterFD,
		unix.TIOCSWINSZ,
		&unix.Winsize{Row: rows, Col: columns},
	); err != nil {
		return nil, err
	}

	command.Stdin = slave
	command.Stdout = slave
	command.Stderr = slave
	command.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    0,
	}
	if err := command.Start(); err != nil {
		return nil, err
	}
	closeMaster = false
	return &linuxProcess{File: master, command: command}, nil
}

func (process *linuxProcess) Wait() error {
	return process.command.Wait()
}

func (process *linuxProcess) Kill() error {
	if process.command.Process == nil {
		return nil
	}
	return process.command.Process.Kill()
}

func createPlatformInput(path string) error {
	_ = unix.Unlink(path)
	return unix.Mkfifo(path, 0o600)
}

func openPlatformInput(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDWR, 0)
}

func writePlatformInput(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

func removePlatformInput(path string) error {
	err := unix.Unlink(path)
	if err == unix.ENOENT {
		return nil
	}
	return err
}

func isPlatformEnd(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, syscall.EIO)
}
