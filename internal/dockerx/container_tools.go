package dockerx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	maxContainerCommandBytes = 2048
	maxContainerCommandRun   = 20 * time.Second
)

var containerPathPattern = regexp.MustCompile(`^/(?:[A-Za-z0-9_.-]+/?){1,16}$`)
var containerEnvNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,127}$`)

type ContainerStats struct {
	ContainerID   string    `json:"containerId"`
	CPUPercent    float64   `json:"cpuPercent"`
	MemoryBytes   uint64    `json:"memoryBytes"`
	MemoryLimit   uint64    `json:"memoryLimitBytes"`
	MemoryPercent float64   `json:"memoryPercent"`
	NetworkRx     uint64    `json:"networkRxBytes"`
	NetworkTx     uint64    `json:"networkTxBytes"`
	BlockRead     uint64    `json:"blockReadBytes"`
	BlockWrite    uint64    `json:"blockWriteBytes"`
	PIDs          uint64    `json:"pids"`
	CollectedAt   time.Time `json:"collectedAt"`
}

type ContainerExecInput struct {
	ResourceVersion string `json:"resourceVersion"`
	Command         string `json:"command"`
}

type ContainerExecResult struct {
	ContainerID string    `json:"containerId"`
	ExitCode    int       `json:"exitCode"`
	Output      string    `json:"output"`
	Truncated   bool      `json:"truncated"`
	FinishedAt  time.Time `json:"finishedAt"`
}

type ContainerCreatePort struct {
	PrivatePort uint16 `json:"privatePort"`
	PublicPort  uint16 `json:"publicPort"`
	Protocol    string `json:"protocol,omitempty"`
	HostIP      string `json:"hostIp,omitempty"`
}

type ContainerCreateMount struct {
	Volume   string `json:"volume"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"readOnly,omitempty"`
}

type ContainerCreateEnvironment struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (c *Client) ContainerStats(ctx context.Context, id string) (ContainerStats, error) {
	if !containerIDPattern.MatchString(id) {
		return ContainerStats{}, errors.New("invalid container id")
	}
	if _, err := c.inspect(ctx, id); err != nil {
		return ContainerStats{}, err
	}
	var raw struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage  uint64   `json:"total_usage"`
				PercpuUsage []uint64 `json:"percpu_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
			OnlineCPUs     uint64 `json:"online_cpus"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64 `json:"usage"`
			Limit uint64 `json:"limit"`
			Stats struct {
				Cache             uint64 `json:"cache"`
				TotalInactiveFile uint64 `json:"total_inactive_file"`
				InactiveFile      uint64 `json:"inactive_file"`
			} `json:"stats"`
		} `json:"memory_stats"`
		Networks map[string]struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		} `json:"networks"`
		BlkioStats struct {
			IOServiceBytesRecursive []struct {
				Operation string `json:"op"`
				Value     uint64 `json:"value"`
			} `json:"io_service_bytes_recursive"`
		} `json:"blkio_stats"`
		PidsStats struct {
			Current uint64 `json:"current"`
		} `json:"pids_stats"`
	}
	query := url.Values{"stream": {"false"}, "one-shot": {"true"}}
	if err := c.getJSON(ctx, "/containers/"+id+"/stats?"+query.Encode(), &raw); err != nil {
		return ContainerStats{}, err
	}
	var cpuDelta, systemDelta uint64
	if raw.CPUStats.CPUUsage.TotalUsage >= raw.PreCPUStats.CPUUsage.TotalUsage {
		cpuDelta = raw.CPUStats.CPUUsage.TotalUsage - raw.PreCPUStats.CPUUsage.TotalUsage
	}
	if raw.CPUStats.SystemCPUUsage >= raw.PreCPUStats.SystemCPUUsage {
		systemDelta = raw.CPUStats.SystemCPUUsage - raw.PreCPUStats.SystemCPUUsage
	}
	onlineCPUs := raw.CPUStats.OnlineCPUs
	if onlineCPUs == 0 {
		onlineCPUs = uint64(len(raw.CPUStats.CPUUsage.PercpuUsage))
	}
	var cpuPercent float64
	if cpuDelta > 0 && systemDelta > 0 && onlineCPUs > 0 {
		cpuPercent = float64(cpuDelta) / float64(systemDelta) * float64(onlineCPUs) * 100
	}
	cache := raw.MemoryStats.Stats.TotalInactiveFile
	if cache == 0 {
		cache = raw.MemoryStats.Stats.InactiveFile
	}
	if cache == 0 {
		cache = raw.MemoryStats.Stats.Cache
	}
	memory := raw.MemoryStats.Usage
	if cache < memory {
		memory -= cache
	}
	var memoryPercent float64
	if raw.MemoryStats.Limit > 0 {
		memoryPercent = float64(memory) / float64(raw.MemoryStats.Limit) * 100
	}
	result := ContainerStats{
		ContainerID: id, CPUPercent: cpuPercent, MemoryBytes: memory,
		MemoryLimit: raw.MemoryStats.Limit, MemoryPercent: memoryPercent,
		PIDs: raw.PidsStats.Current, CollectedAt: c.now().UTC(),
	}
	for _, network := range raw.Networks {
		result.NetworkRx += network.RxBytes
		result.NetworkTx += network.TxBytes
	}
	for _, entry := range raw.BlkioStats.IOServiceBytesRecursive {
		switch strings.ToLower(entry.Operation) {
		case "read":
			result.BlockRead += entry.Value
		case "write":
			result.BlockWrite += entry.Value
		}
	}
	return result, nil
}

func (c *Client) ContainerExec(ctx context.Context, id string, input ContainerExecInput) (ContainerExecResult, error) {
	command := strings.TrimSpace(input.Command)
	if !validContainerCommand(command) {
		return ContainerExecResult{}, ErrUnsafeOrInvalidAction
	}
	if err := c.verifyContainerVersion(ctx, id, input.ResourceVersion); err != nil {
		return ContainerExecResult{}, err
	}
	inspect, err := c.inspect(ctx, id)
	if err != nil {
		return ContainerExecResult{}, err
	}
	if !inspect.State.Running || inspect.State.Paused || inspect.State.Restarting {
		return ContainerExecResult{}, ErrUnsafeOrInvalidAction
	}
	execContext, cancel := context.WithTimeout(ctx, maxContainerCommandRun)
	defer cancel()
	execID, err := c.createFixedExec(execContext, id, []string{"/bin/sh", "-lc", command}, nil)
	if err != nil {
		return ContainerExecResult{}, err
	}
	output, transportTruncated, err := c.startNginxExec(execContext, execID)
	if err != nil {
		return ContainerExecResult{}, err
	}
	state, err := c.inspectNginxExec(execContext, execID)
	if err != nil {
		return ContainerExecResult{}, err
	}
	if state.Running {
		return ContainerExecResult{}, fmt.Errorf("container command exceeded the %s safety limit", maxContainerCommandRun)
	}
	safeOutput, truncated := boundedRedactedNginxOutput(output, transportTruncated)
	return ContainerExecResult{
		ContainerID: id, ExitCode: state.ExitCode, Output: safeOutput,
		Truncated: truncated, FinishedAt: c.now().UTC(),
	}, nil
}

func validContainerCommand(value string) bool {
	if value == "" || len(value) > maxContainerCommandBytes {
		return false
	}
	for _, char := range value {
		if char == '\n' || char == '\r' || char == 0 || (unicode.IsControl(char) && char != '\t') {
			return false
		}
	}
	return true
}

func (c *Client) createManagedContainer(ctx context.Context, input MaintenanceInput) error {
	payload, err := c.containerCreatePayload(ctx, input)
	if err != nil {
		return err
	}
	var created struct {
		ID string `json:"Id"`
	}
	data, _, err := c.nginxDockerRequest(
		ctx,
		http.MethodPost,
		"/containers/create?name="+url.QueryEscape(input.Name),
		payload,
		8<<10,
	)
	if err != nil {
		return err
	}
	if jsonErr := decodeStrictDockerJSON(data, &created); jsonErr != nil ||
		!dockerExecIDPattern.MatchString(created.ID) {
		return errors.New("Docker returned an invalid container identity")
	}
	if err := c.post(ctx, "/containers/"+created.ID+"/start"); err != nil {
		rollbackContext, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		rollbackErr := c.dockerMutation(
			rollbackContext,
			http.MethodDelete,
			"/containers/"+created.ID+"?force=1&v=0",
			nil,
		)
		if rollbackErr != nil {
			return fmt.Errorf("start container failed and rollback needs attention: %w", err)
		}
		return fmt.Errorf("start container failed; created container rolled back: %w", err)
	}
	return nil
}

func (c *Client) containerCreatePayload(ctx context.Context, input MaintenanceInput) ([]byte, error) {
	if !dockerNamePattern.MatchString(input.Name) || !validImageReference(input.Image) ||
		len(input.Ports) > 16 || len(input.Mounts) > 16 ||
		len(input.Environment) > 64 || len(input.Command) > 32 {
		return nil, ErrInvalidDockerJob
	}
	switch input.RestartPolicy {
	case "", "no", "always", "unless-stopped", "on-failure":
	default:
		return nil, ErrInvalidDockerJob
	}
	if input.Network != "" && !dockerNamePattern.MatchString(input.Network) {
		return nil, ErrInvalidDockerJob
	}
	for _, value := range input.Command {
		if strings.TrimSpace(value) != value || len(value) == 0 || len(value) > 512 ||
			strings.ContainsAny(value, "\r\n\x00") {
			return nil, ErrInvalidDockerJob
		}
	}
	environment := make([]string, 0, len(input.Environment))
	seenEnvironment := make(map[string]bool)
	totalEnvironmentBytes := 0
	for _, variable := range input.Environment {
		name := strings.TrimSpace(variable.Name)
		if !containerEnvNamePattern.MatchString(name) || seenEnvironment[name] ||
			len(variable.Value) > 2048 || strings.ContainsAny(variable.Value, "\r\n\x00") {
			return nil, ErrInvalidDockerJob
		}
		totalEnvironmentBytes += len(name) + len(variable.Value) + 1
		if totalEnvironmentBytes > 32<<10 {
			return nil, ErrInvalidDockerJob
		}
		seenEnvironment[name] = true
		environment = append(environment, name+"="+variable.Value)
	}
	exposedPorts := make(map[string]any)
	portBindings := make(map[string][]map[string]string)
	seenPorts := make(map[string]bool)
	for _, port := range input.Ports {
		protocol := strings.ToLower(strings.TrimSpace(port.Protocol))
		if protocol == "" {
			protocol = "tcp"
		}
		if port.PrivatePort == 0 || port.PublicPort == 0 ||
			(protocol != "tcp" && protocol != "udp") ||
			(port.HostIP != "" && port.HostIP != "0.0.0.0" && port.HostIP != "127.0.0.1" &&
				port.HostIP != "::" && port.HostIP != "::1") {
			return nil, ErrInvalidDockerJob
		}
		key := fmt.Sprintf("%d/%s", port.PrivatePort, protocol)
		bindingKey := key + "\x00" + port.HostIP + "\x00" + fmt.Sprint(port.PublicPort)
		if seenPorts[bindingKey] {
			return nil, ErrInvalidDockerJob
		}
		seenPorts[bindingKey] = true
		exposedPorts[key] = struct{}{}
		portBindings[key] = append(portBindings[key], map[string]string{
			"HostIp": port.HostIP, "HostPort": fmt.Sprint(port.PublicPort),
		})
	}
	binds := make([]string, 0, len(input.Mounts))
	seenTargets := make(map[string]bool)
	existingVolumes := make(map[string]bool, len(input.Mounts))
	if len(input.Mounts) > 0 {
		volumes, err := c.Volumes(ctx)
		if err != nil {
			return nil, err
		}
		for _, volume := range volumes {
			existingVolumes[volume.Name] = true
		}
	}
	for _, mount := range input.Mounts {
		target := strings.TrimSpace(mount.Target)
		if !dockerNamePattern.MatchString(mount.Volume) || !existingVolumes[mount.Volume] ||
			!containerPathPattern.MatchString(target) || target == "/" ||
			seenTargets[target] {
			return nil, ErrInvalidDockerJob
		}
		seenTargets[target] = true
		binding := mount.Volume + ":" + target
		if mount.ReadOnly {
			binding += ":ro"
		}
		binds = append(binds, binding)
	}
	if input.Network == "host" || input.Network == "none" || input.Network == "kejilion-panel-network" {
		return nil, ErrProtectedDockerResource
	}
	if input.Network != "" && input.Network != "bridge" {
		found := false
		networks, networkErr := c.Networks(ctx)
		if networkErr != nil {
			return nil, networkErr
		}
		for _, network := range networks {
			if network.Name == input.Network {
				found = true
				break
			}
		}
		if !found {
			return nil, ErrDockerJobNotFound
		}
	}
	restartPolicy := input.RestartPolicy
	if restartPolicy == "" {
		restartPolicy = "unless-stopped"
	}
	payload := struct {
		Image        string            `json:"Image"`
		Cmd          []string          `json:"Cmd,omitempty"`
		Env          []string          `json:"Env,omitempty"`
		Labels       map[string]string `json:"Labels"`
		ExposedPorts map[string]any    `json:"ExposedPorts,omitempty"`
		HostConfig   struct {
			PortBindings  map[string][]map[string]string `json:"PortBindings,omitempty"`
			Binds         []string                       `json:"Binds,omitempty"`
			NetworkMode   string                         `json:"NetworkMode,omitempty"`
			RestartPolicy struct {
				Name string `json:"Name"`
			} `json:"RestartPolicy"`
		} `json:"HostConfig"`
	}{
		Image: input.Image,
		Cmd:   append([]string(nil), input.Command...),
		Env:   environment,
		Labels: map[string]string{
			"io.kejilion.panel.managed":    "true",
			"io.kejilion.panel.created-by": "kpanel",
		},
		ExposedPorts: exposedPorts,
	}
	payload.HostConfig.PortBindings = portBindings
	payload.HostConfig.Binds = binds
	payload.HostConfig.NetworkMode = input.Network
	payload.HostConfig.RestartPolicy.Name = restartPolicy
	return jsonMarshalDocker(payload)
}

func decodeStrictDockerJSON(data []byte, target any) error {
	return json.Unmarshal(data, target)
}

func jsonMarshalDocker(value any) ([]byte, error) {
	return json.Marshal(value)
}
