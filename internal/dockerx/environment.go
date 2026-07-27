package dockerx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"time"
)

type EnvironmentInfo struct {
	Available       bool      `json:"available"`
	EngineVersion   string    `json:"engineVersion,omitempty"`
	StorageDriver   string    `json:"storageDriver,omitempty"`
	DataRoot        string    `json:"dataRoot,omitempty"`
	Containers      int       `json:"containers"`
	Images          int       `json:"images"`
	MirrorPreset    string    `json:"mirrorPreset"`
	RegistryMirrors []string  `json:"registryMirrors"`
	IPv6Enabled     bool      `json:"ipv6Enabled"`
	IPv6CIDR        string    `json:"ipv6Cidr,omitempty"`
	DaemonConfig    string    `json:"daemonConfig"`
	DaemonWarning   string    `json:"daemonWarning,omitempty"`
	ObservedAt      time.Time `json:"observedAt"`
}

func (c *Client) Environment(ctx context.Context) EnvironmentInfo {
	result := EnvironmentInfo{
		MirrorPreset: "official", RegistryMirrors: []string{},
		DaemonConfig: "missing", ObservedAt: c.now().UTC(),
	}
	var info struct {
		ServerVersion string `json:"ServerVersion"`
		DockerRootDir string `json:"DockerRootDir"`
		Driver        string `json:"Driver"`
		Containers    int    `json:"Containers"`
		Images        int    `json:"Images"`
	}
	if err := c.getJSON(ctx, "/info", &info); err == nil {
		result.Available = true
		result.EngineVersion = info.ServerVersion
		result.DataRoot = info.DockerRootDir
		result.StorageDriver = info.Driver
		result.Containers = info.Containers
		result.Images = info.Images
	}
	data, existed, err := readDockerDaemonConfig(c.daemonConfigPath)
	if err != nil {
		result.DaemonConfig = "invalid"
		result.DaemonWarning = safeDockerJobMessage(err)
		return result
	}
	if !existed || len(bytes.TrimSpace(data)) == 0 {
		return result
	}
	result.DaemonConfig = "valid"
	config, err := parseDockerDaemonConfig(data)
	if err != nil {
		result.DaemonConfig = "invalid"
		result.DaemonWarning = "Docker daemon.json 无法解析；面板不会覆盖现有配置"
		return result
	}
	result.IPv6Enabled, _ = config["ipv6"].(bool)
	result.IPv6CIDR, _ = config["fixed-cidr-v6"].(string)
	if raw, ok := config["registry-mirrors"].([]any); ok {
		for _, item := range raw {
			if value, ok := item.(string); ok && value != "" {
				result.RegistryMirrors = append(result.RegistryMirrors, value)
			}
		}
	}
	switch {
	case len(result.RegistryMirrors) == 0:
		result.MirrorPreset = "official"
	case sameStringSet(result.RegistryMirrors, kejilionDockerMirrors):
		result.MirrorPreset = "cn"
	default:
		result.MirrorPreset = "custom"
	}
	return result
}

func parseDockerDaemonConfig(data []byte) (map[string]any, error) {
	var config map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, errors.New("Docker daemon.json contains trailing data")
	}
	if config == nil {
		config = make(map[string]any)
	}
	return config, nil
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for index := range leftCopy {
		if leftCopy[index] != rightCopy[index] {
			return false
		}
	}
	return true
}
