package dockerx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	maxAppDockerResponse = 8 << 20
	appPullTimeout       = 10 * time.Minute
)

var (
	appNamePattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{0,62}$`)
	appTokenPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)
	appImagePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9./_:@-]{0,254}$`)
)

type DeclarativeAppSpec struct {
	Token         string
	ContainerName string
	Image         string
	ContainerPort uint16
}

type AppMutationResult struct {
	ContainerID     string `json:"containerId,omitempty"`
	Action          string `json:"action"`
	Status          string `json:"status"`
	ResourceVersion string `json:"resourceVersion,omitempty"`
}

func (c *Client) LifecycleDeclarativeApp(
	ctx context.Context,
	spec DeclarativeAppSpec,
	action string,
	expectedVersion string,
) (ActionResult, error) {
	if action != "start" && action != "stop" && action != "restart" {
		return ActionResult{}, ErrActionUnsupported
	}
	if expectedVersion == "" {
		return ActionResult{}, ErrVersionRequired
	}
	c.lifecycle.Lock()
	defer c.lifecycle.Unlock()
	current, summary, _, _, err := c.verifiedDeclarativeContainer(ctx, spec, expectedVersion)
	if err != nil {
		return ActionResult{}, err
	}
	if !contains(c.summaryFromInspect(current).AllowedActions, action) {
		return ActionResult{}, ErrActionUnsupported
	}
	endpoint := "/containers/" + current.ID + "/" + action
	if action == "stop" || action == "restart" {
		endpoint += "?t=10"
	}
	if err := c.post(ctx, endpoint); err != nil {
		return ActionResult{}, err
	}
	version := summary.ResourceVersion
	if updated, inspectErr := c.inspectSummary(ctx, current.ID); inspectErr == nil {
		version = updated.ResourceVersion
	}
	return ActionResult{
		ContainerID: current.ID, Action: action, Status: "completed",
		ResourceVersion: version,
	}, nil
}

func (c *Client) InstallDeclarativeApp(
	ctx context.Context,
	spec DeclarativeAppSpec,
	hostPort uint16,
	accessMode string,
) (AppMutationResult, error) {
	if err := validateDeclarativeSpec(spec, hostPort, accessMode); err != nil {
		return AppMutationResult{}, err
	}
	c.lifecycle.Lock()
	defer c.lifecycle.Unlock()
	if _, err := c.inspect(ctx, spec.ContainerName); err == nil {
		return AppMutationResult{}, ErrAppConflict
	} else if !isDockerStatus(err, http.StatusNotFound) {
		return AppMutationResult{}, err
	}
	if err := c.pullImage(ctx, spec.Image); err != nil {
		return AppMutationResult{}, err
	}
	containerID, err := c.createDeclarativeContainer(ctx, spec, spec.Image, hostPort, accessMode, nil)
	if err != nil {
		return AppMutationResult{}, err
	}
	if err := c.post(ctx, "/containers/"+containerID+"/start"); err != nil {
		if cleanupErr := c.deleteContainer(ctx, containerID); cleanupErr != nil {
			return AppMutationResult{}, fmt.Errorf(
				"%w: application start failed and cleanup failed: %v",
				ErrAppNeedsAttention,
				cleanupErr,
			)
		}
		return AppMutationResult{}, err
	}
	summary, err := c.inspectSummary(ctx, containerID)
	if err != nil {
		_ = c.stopContainer(ctx, containerID)
		if cleanupErr := c.deleteContainer(ctx, containerID); cleanupErr != nil {
			return AppMutationResult{}, fmt.Errorf(
				"%w: installed container verification failed and rollback failed: %v",
				ErrAppNeedsAttention,
				cleanupErr,
			)
		}
		return AppMutationResult{}, fmt.Errorf(
			"%w: installed container verification failed; container rolled back: %v",
			ErrAppRolledBack,
			err,
		)
	}
	return AppMutationResult{
		ContainerID: summary.ID, Action: "install", Status: "completed",
		ResourceVersion: summary.ResourceVersion,
	}, nil
}

func (c *Client) UpdateDeclarativeApp(
	ctx context.Context,
	spec DeclarativeAppSpec,
	expectedVersion string,
) (AppMutationResult, error) {
	if err := validateDeclarativeSpec(spec, 1024, "direct"); err != nil {
		return AppMutationResult{}, err
	}
	if expectedVersion == "" {
		return AppMutationResult{}, ErrVersionRequired
	}
	c.lifecycle.Lock()
	defer c.lifecycle.Unlock()
	current, _, hostPort, accessMode, err := c.verifiedDeclarativeContainer(ctx, spec, expectedVersion)
	if err != nil {
		return AppMutationResult{}, err
	}
	if hostPort == 0 {
		return AppMutationResult{}, fmt.Errorf(
			"%w: the application container does not expose the expected TCP port",
			ErrAppConflict,
		)
	}
	if err := c.pullImage(ctx, spec.Image); err != nil {
		return AppMutationResult{}, err
	}
	updated, err := c.replaceDeclarativeContainer(ctx, spec, current, hostPort, accessMode, spec.Image, "update")
	if err != nil {
		return AppMutationResult{}, err
	}
	return AppMutationResult{
		ContainerID: updated.ID, Action: "update", Status: "completed",
		ResourceVersion: updated.ResourceVersion,
	}, nil
}

func (c *Client) SetDeclarativeAppAccess(
	ctx context.Context,
	spec DeclarativeAppSpec,
	expectedVersion string,
	accessMode string,
) (AppMutationResult, error) {
	if accessMode != "direct" && accessMode != "domain_only" {
		return AppMutationResult{}, ErrActionUnsupported
	}
	if expectedVersion == "" {
		return AppMutationResult{}, ErrVersionRequired
	}
	c.lifecycle.Lock()
	defer c.lifecycle.Unlock()
	current, _, hostPort, currentAccess, err := c.verifiedDeclarativeContainer(ctx, spec, expectedVersion)
	if err != nil {
		return AppMutationResult{}, err
	}
	if hostPort == 0 {
		return AppMutationResult{}, fmt.Errorf(
			"%w: the application container does not expose the expected TCP port",
			ErrAppConflict,
		)
	}
	if currentAccess == accessMode {
		summary := c.summaryFromInspect(current)
		return AppMutationResult{
			ContainerID: summary.ID, Action: "direct_access", Status: "unchanged",
			ResourceVersion: summary.ResourceVersion,
		}, nil
	}
	updated, err := c.replaceDeclarativeContainer(
		ctx,
		spec,
		current,
		hostPort,
		accessMode,
		current.Config.Image,
		"direct_access",
	)
	if err != nil {
		return AppMutationResult{}, err
	}
	return AppMutationResult{
		ContainerID: updated.ID, Action: "direct_access", Status: "completed",
		ResourceVersion: updated.ResourceVersion,
	}, nil
}

func (c *Client) UninstallDeclarativeApp(
	ctx context.Context,
	spec DeclarativeAppSpec,
	expectedVersion string,
) (AppMutationResult, error) {
	if expectedVersion == "" {
		return AppMutationResult{}, ErrVersionRequired
	}
	c.lifecycle.Lock()
	defer c.lifecycle.Unlock()
	current, summary, _, _, err := c.verifiedDeclarativeContainer(ctx, spec, expectedVersion)
	if err != nil {
		return AppMutationResult{}, err
	}
	wasRunning := current.State.Running
	if wasRunning {
		if err := c.stopContainer(ctx, current.ID); err != nil {
			return AppMutationResult{}, err
		}
	}
	if err := c.deleteContainer(ctx, current.ID); err != nil {
		if wasRunning {
			_ = c.post(ctx, "/containers/"+current.ID+"/start")
		}
		return AppMutationResult{}, err
	}
	return AppMutationResult{
		ContainerID: summary.ID, Action: "uninstall", Status: "completed",
	}, nil
}

func (c *Client) inspectSummary(ctx context.Context, id string) (contractSummary, error) {
	raw, err := c.inspect(ctx, id)
	if err != nil {
		return contractSummary{}, err
	}
	summary := c.summaryFromInspect(raw)
	return contractSummary{ID: summary.ID, ResourceVersion: summary.ResourceVersion}, nil
}

type contractSummary struct {
	ID              string
	ResourceVersion string
}

func (c *Client) verifiedDeclarativeContainer(
	ctx context.Context,
	spec DeclarativeAppSpec,
	expectedVersion string,
) (containerInspect, contractSummary, uint16, string, error) {
	raw, err := c.inspect(ctx, spec.ContainerName)
	if err != nil {
		return containerInspect{}, contractSummary{}, 0, "", err
	}
	summary := c.summaryFromInspect(raw)
	if summary.ResourceVersion != expectedVersion {
		return containerInspect{}, contractSummary{}, 0, "", ErrResourceConflict
	}
	if strings.TrimPrefix(raw.Name, "/") != spec.ContainerName {
		return containerInspect{}, contractSummary{}, 0, "", ErrAppConflict
	}
	hostPort, accessMode, _ := declarativePortBinding(raw, spec)
	return raw, contractSummary{ID: summary.ID, ResourceVersion: summary.ResourceVersion}, hostPort, accessMode, nil
}

func declarativePortBinding(raw containerInspect, spec DeclarativeAppSpec) (uint16, string, bool) {
	if strings.TrimPrefix(raw.Name, "/") != spec.ContainerName {
		return 0, "", false
	}
	key := strconv.Itoa(int(spec.ContainerPort)) + "/tcp"
	bindings, ok := raw.NetworkSettings.Ports[key]
	if !ok {
		return 0, "", false
	}
	for _, binding := range bindings {
		port, err := strconv.ParseUint(binding.HostPort, 10, 16)
		if err != nil || port < 1 {
			continue
		}
		hostIP := strings.TrimSpace(binding.HostIP)
		if hostIP == "127.0.0.1" || hostIP == "::1" {
			return uint16(port), "domain_only", true
		}
		return uint16(port), "direct", true
	}
	return 0, "", false
}

func (c *Client) replaceDeclarativeContainer(
	ctx context.Context,
	spec DeclarativeAppSpec,
	current containerInspect,
	hostPort uint16,
	accessMode string,
	newImage string,
	action string,
) (contractSummary, error) {
	wasRunning := current.State.Running
	oldImage := current.Image
	oldLabels := current.Config.Labels
	oldAccess := accessModeFromInspect(current, spec)
	if wasRunning {
		if err := c.stopContainer(ctx, current.ID); err != nil {
			return contractSummary{}, err
		}
	}
	if err := c.deleteContainer(ctx, current.ID); err != nil {
		if wasRunning {
			_ = c.post(ctx, "/containers/"+current.ID+"/start")
		}
		return contractSummary{}, err
	}
	newID, createErr := c.createDeclarativeContainer(ctx, spec, newImage, hostPort, accessMode, nil)
	if createErr == nil && wasRunning {
		createErr = c.post(ctx, "/containers/"+newID+"/start")
	}
	if createErr == nil {
		updated, inspectErr := c.inspectSummary(ctx, newID)
		if inspectErr == nil {
			return updated, nil
		}
		createErr = fmt.Errorf("verify replacement container: %w", inspectErr)
	}
	if newID != "" {
		_ = c.stopContainer(ctx, newID)
		if cleanupErr := c.deleteContainer(ctx, newID); cleanupErr != nil {
			return contractSummary{}, fmt.Errorf(
				"%w: %s failed and the replacement container could not be removed: %v",
				ErrAppNeedsAttention,
				action,
				cleanupErr,
			)
		}
	}
	rollbackID, rollbackErr := c.createDeclarativeContainer(ctx, spec, oldImage, hostPort, oldAccess, oldLabels)
	if rollbackErr == nil && wasRunning {
		rollbackErr = c.post(ctx, "/containers/"+rollbackID+"/start")
	}
	if rollbackErr != nil {
		return contractSummary{}, fmt.Errorf("%w: %s failed and rollback failed: %v", ErrAppNeedsAttention, action, rollbackErr)
	}
	return contractSummary{}, fmt.Errorf("%w: %s failed; previous container restored: %v", ErrAppRolledBack, action, createErr)
}

func accessModeFromInspect(raw containerInspect, spec DeclarativeAppSpec) string {
	_, access, ok := declarativePortBinding(raw, spec)
	if !ok {
		return "direct"
	}
	return access
}

func (c *Client) createDeclarativeContainer(
	ctx context.Context,
	spec DeclarativeAppSpec,
	image string,
	hostPort uint16,
	accessMode string,
	labels map[string]string,
) (string, error) {
	hostIP := "0.0.0.0"
	if accessMode == "domain_only" {
		hostIP = "127.0.0.1"
	}
	managedLabels := make(map[string]string, len(labels)+3)
	for key, value := range labels {
		managedLabels[key] = value
	}
	managedLabels["io.kejilion.panel.managed"] = "true"
	managedLabels["io.kejilion.panel.app"] = spec.Token
	managedLabels["io.kejilion.panel.image"] = spec.Image
	managedLabels["io.kejilion.panel.spec"] = "stateless-v1"
	containerPort := strconv.Itoa(int(spec.ContainerPort)) + "/tcp"
	payload := struct {
		Image        string                 `json:"Image"`
		Labels       map[string]string      `json:"Labels"`
		ExposedPorts map[string]interface{} `json:"ExposedPorts"`
		HostConfig   struct {
			PortBindings map[string][]struct {
				HostIP   string `json:"HostIp"`
				HostPort string `json:"HostPort"`
			} `json:"PortBindings"`
			RestartPolicy struct {
				Name string `json:"Name"`
			} `json:"RestartPolicy"`
		} `json:"HostConfig"`
	}{
		Image: image, Labels: managedLabels,
		ExposedPorts: map[string]interface{}{containerPort: struct{}{}},
	}
	payload.HostConfig.PortBindings = map[string][]struct {
		HostIP   string `json:"HostIp"`
		HostPort string `json:"HostPort"`
	}{
		containerPort: {{HostIP: hostIP, HostPort: strconv.Itoa(int(hostPort))}},
	}
	payload.HostConfig.RestartPolicy.Name = "always"
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	data, err := c.appDockerRequest(
		ctx, http.MethodPost, "/containers/create?name="+url.QueryEscape(spec.ContainerName),
		body, maxAppDockerResponse, c.httpClient,
	)
	if err != nil {
		return "", err
	}
	var response struct {
		ID string `json:"Id"`
	}
	if err := json.Unmarshal(data, &response); err != nil || !containerIDPattern.MatchString(response.ID) {
		return "", errors.New("Docker returned an invalid application container identity")
	}
	return response.ID, nil
}

func (c *Client) pullImage(ctx context.Context, image string) error {
	client := *c.httpClient
	client.Timeout = appPullTimeout
	path := "/images/create?fromImage=" + url.QueryEscape(image)
	_, err := c.appDockerRequest(ctx, http.MethodPost, path, nil, maxAppDockerResponse, &client)
	return err
}

func (c *Client) stopContainer(ctx context.Context, id string) error {
	return c.post(ctx, "/containers/"+id+"/stop?t=20")
}

func (c *Client) deleteContainer(ctx context.Context, id string) error {
	_, err := c.appDockerRequest(
		ctx, http.MethodDelete, "/containers/"+id+"?v=0&force=0",
		nil, 64<<10, c.httpClient,
	)
	return err
}

func (c *Client) appDockerRequest(
	ctx context.Context,
	method string,
	path string,
	payload []byte,
	limit int64,
	client *http.Client,
) ([]byte, error) {
	var body io.Reader = http.NoBody
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("Docker API unavailable: %w", err)
	}
	defer response.Body.Close()
	data, readErr := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if readErr != nil {
		return nil, fmt.Errorf("read Docker response: %w", readErr)
	}
	if int64(len(data)) > limit {
		return nil, errors.New("Docker application response exceeded the safety limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, dockerError(response.StatusCode, data)
	}
	return data, nil
}

func validateDeclarativeSpec(spec DeclarativeAppSpec, hostPort uint16, accessMode string) error {
	if !appTokenPattern.MatchString(spec.Token) || !appNamePattern.MatchString(spec.ContainerName) ||
		!appImagePattern.MatchString(spec.Image) || spec.ContainerPort == 0 ||
		hostPort == 0 || (accessMode != "direct" && accessMode != "domain_only") {
		return ErrActionUnsupported
	}
	return nil
}

var (
	ErrAppConflict       = errors.New("application container already exists")
	ErrAppRolledBack     = errors.New("application action failed and was rolled back")
	ErrAppNeedsAttention = errors.New("application action requires manual attention")
)
