package dockerx

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	nginxContainerName        = "nginx"
	nginxComposeWorkingDir    = "/home/web"
	nginxMainConfigPath       = "/etc/nginx/nginx.conf"
	nginxConfDirectoryPath    = "/etc/nginx/conf.d"
	nginxHTMLDirectoryPath    = "/var/www/html"
	maxNginxMainConfigBytes   = 1 << 20
	maxNginxExecResponseBytes = 64 << 10
	maxNginxExecOutputLines   = 1000
)

var (
	dockerExecIDPattern     = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
	nginxConfIncludePattern = regexp.MustCompile(
		`(?m)^[\t ]*include[\t ]+/etc/nginx/conf[.]d/[*][.]conf[\t ]*;[\t ]*(?:#[^\r\n]*)?\r?$`,
	)

	ErrNginxNotReady      = errors.New("managed nginx container is not ready")
	ErrInvalidDockerExec  = errors.New("Docker returned an invalid exec identity")
	ErrNginxExecRunning   = errors.New("nginx exec did not finish")
	ErrNginxCommandFailed = errors.New("nginx command failed")
)

// NginxExecError reports a completed, fixed nginx command with a non-zero exit
// status. Output is size-limited and redacted before it is exposed.
type NginxExecError struct {
	Operation string
	ExitCode  int
	Output    string
	Truncated bool
}

func (e *NginxExecError) Error() string {
	message := fmt.Sprintf("nginx %s failed with exit code %d", e.Operation, e.ExitCode)
	if e.Output != "" {
		message += ": " + e.Output
	}
	return message
}

func (e *NginxExecError) Unwrap() error {
	return ErrNginxCommandFailed
}

type nginxExecOperation uint8

const (
	nginxExecTest nginxExecOperation = iota + 1
	nginxExecReload
)

func (operation nginxExecOperation) nameAndCommand() (string, []string, error) {
	switch operation {
	case nginxExecTest:
		return "test", []string{"nginx", "-t"}, nil
	case nginxExecReload:
		return "reload", []string{"nginx", "-s", "reload"}, nil
	default:
		return "", nil, errors.New("unsupported internal nginx operation")
	}
}

// NginxReady verifies that the fixed nginx container is running, managed by
// Kejilion, and does not have a configuration that makes control unsafe.
func (c *Client) NginxReady(ctx context.Context) error {
	_, err := c.readyNginx(ctx)
	return err
}

// NginxTest runs exactly: nginx -t.
func (c *Client) NginxTest(ctx context.Context) error {
	return c.runNginxExec(ctx, nginxExecTest)
}

// NginxReload runs exactly: nginx -s reload.
func (c *Client) NginxReload(ctx context.Context) error {
	return c.runNginxExec(ctx, nginxExecReload)
}

func (c *Client) readyNginx(ctx context.Context) (containerInspect, error) {
	raw, err := c.inspect(ctx, nginxContainerName)
	if err != nil {
		return containerInspect{}, fmt.Errorf("%w: inspect fixed container %s: %w", ErrNginxNotReady, nginxContainerName, err)
	}
	if strings.TrimPrefix(raw.Name, "/") != nginxContainerName || !dockerExecIDPattern.MatchString(raw.ID) {
		return containerInspect{}, fmt.Errorf("%w: fixed nginx container identity was not established", ErrReadOnlyContainer)
	}
	if !raw.State.Running || raw.State.Status != "running" || raw.State.Paused || raw.State.Restarting {
		return containerInspect{}, fmt.Errorf("%w: fixed nginx container is not running normally", ErrNginxNotReady)
	}
	if !managedNginxLabels(raw.Config.Labels) {
		return containerInspect{}, fmt.Errorf("%w: nginx requires %s or %s=%s",
			ErrReadOnlyContainer,
			"com.docker.compose.project.working_dir=/home/web",
			"io.kejilion.panel.managed",
			"true",
		)
	}
	if reason := c.unsafeNginxReason(raw); reason != "" {
		return containerInspect{}, fmt.Errorf("%w: nginx container failed the safety check", ErrUnsafeOrInvalidAction)
	}
	if err := c.verifyNginxArtifactBindings(raw); err != nil {
		return containerInspect{}, err
	}
	return raw, nil
}

func managedNginxLabels(labels map[string]string) bool {
	if labels["io.kejilion.panel.managed"] == "true" {
		return true
	}
	workdir := labels["com.docker.compose.project.working_dir"]
	return pathpkg.IsAbs(workdir) && pathpkg.Clean(workdir) == nginxComposeWorkingDir
}

// unsafeNginxReason intentionally permits host networking because the legacy
// Kejilion nginx service uses it. The general lifecycle safety policy remains
// stricter and continues to reject host networking in unsafeReason.
func (c *Client) unsafeNginxReason(raw containerInspect) string {
	host := raw.HostConfig
	switch {
	case host.Privileged:
		return "container uses privileged mode"
	case host.PidMode == "host" || host.IpcMode == "host" || host.UTSMode == "host" || host.UsernsMode == "host":
		return "container shares a host namespace"
	case len(host.CapAdd) > 0:
		return "container adds capabilities"
	case len(host.Devices) > 0:
		return "container maps host devices"
	}
	for _, option := range host.SecurityOpt {
		lower := strings.ToLower(option)
		if strings.Contains(lower, "unconfined") || strings.Contains(lower, "disable") {
			return "container disables a security policy"
		}
	}
	for _, mount := range raw.Mounts {
		source := pathpkg.Clean(mount.Source)
		destination := pathpkg.Clean(mount.Destination)
		if pathpkg.Base(source) == "docker.sock" || pathpkg.Base(destination) == "docker.sock" {
			return "container mounts the Docker Socket"
		}
		if mount.Type == "bind" && !c.provenWithin(source, c.webRoot, false) {
			return "container binds a path outside the managed web root"
		}
	}
	return ""
}

func (c *Client) verifyNginxArtifactBindings(raw containerInspect) error {
	requirements := []struct {
		hostPath      string
		containerPath string
		directory     bool
	}{
		{pathpkg.Join(c.webRoot, "nginx.conf"), nginxMainConfigPath, false},
		{pathpkg.Join(c.webRoot, "conf.d"), nginxConfDirectoryPath, true},
		{pathpkg.Join(c.webRoot, "html"), nginxHTMLDirectoryPath, true},
	}
	for _, requirement := range requirements {
		if !c.hasExactSafeNginxBind(raw.Mounts, requirement.hostPath, requirement.containerPath, requirement.directory) {
			return fmt.Errorf(
				"%w: nginx requires the exact managed bind %s -> %s",
				ErrReadOnlyContainer,
				requirement.hostPath,
				requirement.containerPath,
			)
		}
	}
	mainConfig, err := readBoundedRegularFile(
		filepath.FromSlash(pathpkg.Join(c.webRoot, "nginx.conf")),
		maxNginxMainConfigBytes,
	)
	if err != nil {
		return fmt.Errorf("%w: safely read managed nginx.conf: %v", ErrReadOnlyContainer, err)
	}
	if !nginxConfIncludePattern.Match(mainConfig) {
		return fmt.Errorf(
			"%w: managed nginx.conf does not include %s/*.conf",
			ErrReadOnlyContainer,
			nginxConfDirectoryPath,
		)
	}
	return nil
}

func (c *Client) hasExactSafeNginxBind(
	mounts []dockerMount,
	hostPath string,
	containerPath string,
	directory bool,
) bool {
	hostPath = pathpkg.Clean(hostPath)
	containerPath = pathpkg.Clean(containerPath)
	if !pathpkg.IsAbs(hostPath) || !pathpkg.IsAbs(containerPath) ||
		!c.provenWithin(hostPath, c.webRoot, false) {
		return false
	}
	info, err := os.Lstat(filepath.FromSlash(hostPath))
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return false
	}
	if directory && !info.IsDir() {
		return false
	}
	if !directory && !info.Mode().IsRegular() {
		return false
	}
	for _, mount := range mounts {
		if mount.Type == "bind" &&
			pathpkg.Clean(mount.Source) == hostPath &&
			pathpkg.Clean(mount.Destination) == containerPath {
			return true
		}
	}
	return false
}

func readBoundedRegularFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("file is not a regular non-symlink file")
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("file exceeds %d bytes", limit)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
		return nil, errors.New("file changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file exceeds %d bytes", limit)
	}
	return data, nil
}

func (c *Client) runNginxExec(ctx context.Context, operation nginxExecOperation) error {
	name, _, err := operation.nameAndCommand()
	if err != nil {
		return err
	}
	container, err := c.readyNginx(ctx)
	if err != nil {
		return err
	}
	execID, err := c.createNginxExec(ctx, container.ID, operation)
	if err != nil {
		return fmt.Errorf("create nginx %s exec: %w", name, err)
	}
	output, outputTruncated, err := c.startNginxExec(ctx, execID)
	if err != nil {
		return fmt.Errorf("start nginx %s exec: %w", name, err)
	}
	state, err := c.inspectNginxExec(ctx, execID)
	if err != nil {
		return fmt.Errorf("inspect nginx %s exec: %w", name, err)
	}
	if state.Running {
		return fmt.Errorf("%w: %s", ErrNginxExecRunning, name)
	}
	if state.ExitCode != 0 {
		safeOutput, truncated := boundedRedactedNginxOutput(output, outputTruncated)
		return &NginxExecError{
			Operation: name,
			ExitCode:  state.ExitCode,
			Output:    safeOutput,
			Truncated: truncated,
		}
	}
	return nil
}

func (c *Client) createNginxExec(ctx context.Context, containerID string, operation nginxExecOperation) (string, error) {
	if !dockerExecIDPattern.MatchString(containerID) {
		return "", fmt.Errorf("%w: invalid nginx container id", ErrReadOnlyContainer)
	}
	_, command, err := operation.nameAndCommand()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(struct {
		AttachStdout bool     `json:"AttachStdout"`
		AttachStderr bool     `json:"AttachStderr"`
		Tty          bool     `json:"Tty"`
		Cmd          []string `json:"Cmd"`
	}{
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
		Cmd:          command,
	})
	if err != nil {
		return "", fmt.Errorf("encode Docker exec create request: %w", err)
	}
	data, _, err := c.nginxDockerRequest(
		ctx,
		http.MethodPost,
		"/containers/"+containerID+"/exec",
		payload,
		4<<10,
	)
	if err != nil {
		return "", err
	}
	var response struct {
		ID string `json:"Id"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return "", fmt.Errorf("decode Docker exec create response: %w", err)
	}
	if !dockerExecIDPattern.MatchString(response.ID) {
		return "", ErrInvalidDockerExec
	}
	return response.ID, nil
}

func (c *Client) startNginxExec(ctx context.Context, execID string) ([]byte, bool, error) {
	if !dockerExecIDPattern.MatchString(execID) {
		return nil, false, ErrInvalidDockerExec
	}
	payload := []byte(`{"Detach":false,"Tty":false}`)
	return c.nginxDockerRequest(
		ctx,
		http.MethodPost,
		"/exec/"+execID+"/start",
		payload,
		maxNginxExecResponseBytes,
	)
}

type nginxExecState struct {
	ID       string `json:"ID"`
	Running  bool   `json:"Running"`
	ExitCode int    `json:"ExitCode"`
}

func (c *Client) inspectNginxExec(ctx context.Context, execID string) (nginxExecState, error) {
	if !dockerExecIDPattern.MatchString(execID) {
		return nginxExecState{}, ErrInvalidDockerExec
	}
	data, _, err := c.nginxDockerRequest(
		ctx,
		http.MethodGet,
		"/exec/"+execID+"/json",
		nil,
		4<<10,
	)
	if err != nil {
		return nginxExecState{}, err
	}
	var state nginxExecState
	if err := json.Unmarshal(data, &state); err != nil {
		return nginxExecState{}, fmt.Errorf("decode Docker exec inspect response: %w", err)
	}
	if state.ID != execID || !dockerExecIDPattern.MatchString(state.ID) {
		return nginxExecState{}, ErrInvalidDockerExec
	}
	return state, nil
}

func (c *Client) nginxDockerRequest(
	ctx context.Context,
	method string,
	path string,
	payload []byte,
	limit int64,
) ([]byte, bool, error) {
	var body io.Reader = http.NoBody
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, false, err
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, false, fmt.Errorf("Docker API unavailable: %w", err)
	}
	defer response.Body.Close()
	data, readErr := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if readErr != nil {
		return nil, false, fmt.Errorf("read Docker response: %w", readErr)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, false, dockerError(response.StatusCode, data)
	}
	truncated := int64(len(data)) > limit
	if truncated {
		if _, err := io.Copy(io.Discard, response.Body); err != nil {
			return nil, false, fmt.Errorf("drain Docker response: %w", err)
		}
		data = data[:limit]
	}
	return data, truncated, nil
}

func boundedRedactedNginxOutput(data []byte, transportTruncated bool) (string, bool) {
	plain := demuxNginxExecStream(data)
	output := strings.TrimSpace(strings.Join(redactLines(plain, maxNginxExecOutputLines), "\n"))
	truncated := transportTruncated
	if len(output) > maxNginxExecResponseBytes {
		output = truncateUTF8(output, maxNginxExecResponseBytes)
		truncated = true
	}
	return output, truncated
}

func demuxNginxExecStream(data []byte) []byte {
	var output bytes.Buffer
	framed := false
	for len(data) >= 8 && (data[0] == 0 || data[0] == 1 || data[0] == 2) {
		framed = true
		frameLength := binary.BigEndian.Uint32(data[4:8])
		data = data[8:]
		if uint64(frameLength) > uint64(len(data)) {
			output.Write(data)
			return output.Bytes()
		}
		length := int(frameLength)
		output.Write(data[:length])
		data = data[length:]
	}
	if !framed {
		return data
	}
	output.Write(data)
	return output.Bytes()
}

func truncateUTF8(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	end := limit
	for end > 0 && !utf8.ValidString(value[:end]) {
		end--
	}
	return value[:end]
}
