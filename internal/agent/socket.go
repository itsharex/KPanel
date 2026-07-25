package agent

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"
)

// ListenUnix creates the Agent's only listener. The socket and parent directory
// are restricted to root and the dedicated panel group.
func ListenUnix(socketPath, groupName string) (net.Listener, func(), error) {
	if !filepath.IsAbs(socketPath) {
		return nil, nil, errors.New("Agent socket path must be absolute")
	}
	if groupName == "" {
		return nil, nil, errors.New("Agent socket group is required")
	}
	group, err := user.LookupGroup(groupName)
	if err != nil {
		return nil, nil, fmt.Errorf("lookup Agent socket group %q: %w", groupName, err)
	}
	gid, err := strconv.Atoi(group.Gid)
	if err != nil {
		return nil, nil, fmt.Errorf("parse Agent socket group id: %w", err)
	}
	parent := filepath.Dir(socketPath)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return nil, nil, fmt.Errorf("create Agent socket directory: %w", err)
	}
	if err := os.Chown(parent, 0, gid); err != nil {
		return nil, nil, fmt.Errorf("set Agent socket directory group: %w", err)
	}
	if err := os.Chmod(parent, 0o750); err != nil {
		return nil, nil, fmt.Errorf("set Agent socket directory permissions: %w", err)
	}
	if err := removeStaleSocket(socketPath); err != nil {
		return nil, nil, err
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, nil, fmt.Errorf("listen on Agent Unix Socket: %w", err)
	}
	fail := func(err error) (net.Listener, func(), error) {
		_ = listener.Close()
		_ = os.Remove(socketPath)
		return nil, nil, err
	}
	if err := os.Chown(socketPath, 0, gid); err != nil {
		return fail(fmt.Errorf("set Agent socket group: %w", err))
	}
	if err := os.Chmod(socketPath, 0o660); err != nil {
		return fail(fmt.Errorf("set Agent socket permissions: %w", err))
	}
	createdInfo, _ := os.Lstat(socketPath)
	cleanup := func() {
		_ = listener.Close()
		current, statErr := os.Lstat(socketPath)
		if statErr == nil && createdInfo != nil && os.SameFile(createdInfo, current) {
			_ = os.Remove(socketPath)
		}
	}
	return listener, cleanup, nil
}

func PrepareTokenFile(path, groupName string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("Agent token file path must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("read Agent token metadata: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("Agent token must be a regular non-symlink file")
	}
	group, err := user.LookupGroup(groupName)
	if err != nil {
		return nil, fmt.Errorf("lookup Agent token group %q: %w", groupName, err)
	}
	gid, err := strconv.Atoi(group.Gid)
	if err != nil {
		return nil, fmt.Errorf("parse Agent token group id: %w", err)
	}
	if err := os.Chown(path, 0, gid); err != nil {
		return nil, fmt.Errorf("set Agent token owner: %w", err)
	}
	if err := os.Chmod(path, 0o640); err != nil {
		return nil, fmt.Errorf("set Agent token permissions: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Agent token: %w", err)
	}
	for len(data) > 0 && (data[len(data)-1] == '\n' || data[len(data)-1] == '\r') {
		data = data[:len(data)-1]
	}
	if len(data) < 32 || len(data) > 4096 {
		return nil, errors.New("Agent token length must be between 32 and 4096 bytes")
	}
	return data, nil
}

func removeStaleSocket(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect existing Agent socket: %w", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return errors.New("refusing to replace non-socket Agent path")
	}
	connection, dialErr := net.DialTimeout("unix", path, 250*time.Millisecond)
	if dialErr == nil {
		_ = connection.Close()
		return errors.New("another Agent is already listening on the configured socket")
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove stale Agent socket: %w", err)
	}
	return nil
}
