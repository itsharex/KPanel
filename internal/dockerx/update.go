package dockerx

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	digestPattern        = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
	dockerHubPathPattern = regexp.MustCompile(
		`^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*$`,
	)
	dockerTagPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)
)

const (
	officialUpdateCheckTimeout = 4 * time.Second
	countryLookupTimeout       = 250 * time.Millisecond
	acceleratorCheckTimeout    = 4 * time.Second
)

var dockerHubUpdateAccelerators = []string{
	"docker.1ms.run",
	"gh.kejilion.pro",
}

type ImageUpdateResult struct {
	ContainerID     string    `json:"containerId"`
	Image           string    `json:"image"`
	Status          string    `json:"status"`
	UpdateAvailable bool      `json:"updateAvailable"`
	LocalDigest     string    `json:"localDigest,omitempty"`
	RemoteDigest    string    `json:"remoteDigest,omitempty"`
	ResourceVersion string    `json:"resourceVersion"`
	CheckedAt       time.Time `json:"checkedAt"`
}

func (c *Client) CheckContainerImageUpdate(
	ctx context.Context,
	id string,
	expectedVersion string,
) (ImageUpdateResult, error) {
	if !containerIDPattern.MatchString(id) {
		return ImageUpdateResult{}, errors.New("invalid container id")
	}
	if expectedVersion == "" {
		return ImageUpdateResult{}, ErrVersionRequired
	}
	raw, err := c.inspect(ctx, id)
	if err != nil {
		return ImageUpdateResult{}, err
	}
	summary := c.summaryFromInspect(raw)
	if summary.ResourceVersion != expectedVersion {
		return ImageUpdateResult{}, ErrResourceConflict
	}
	image := strings.TrimSpace(raw.Config.Image)
	localImage := image
	if strings.HasPrefix(image, "sha256:") &&
		raw.Config.Labels["io.kejilion.panel.managed"] == "true" {
		image = strings.TrimSpace(raw.Config.Labels["io.kejilion.panel.image"])
		localImage = raw.Image
	}
	if image == "" || strings.Contains(image, "@sha256:") || localImage == "" {
		return ImageUpdateResult{}, ErrActionUnsupported
	}
	var local struct {
		RepoDigests []string `json:"RepoDigests"`
	}
	if err := c.getJSON(ctx, "/images/"+url.PathEscape(localImage)+"/json", &local); err != nil {
		return ImageUpdateResult{}, err
	}
	remoteDigest, err := c.remoteImageDigestForUpdate(ctx, image)
	if err != nil {
		return ImageUpdateResult{}, err
	}
	localDigest := ""
	for _, value := range local.RepoDigests {
		_, digest, found := strings.Cut(value, "@")
		if found && digestPattern.MatchString(digest) {
			localDigest = digest
			if digest == remoteDigest {
				break
			}
		}
	}
	if localDigest == "" {
		return ImageUpdateResult{}, errors.New("local image does not expose a repository digest")
	}
	available := localDigest != remoteDigest
	status := "current"
	if available {
		status = "available"
	}
	return ImageUpdateResult{
		ContainerID: raw.ID, Image: image, Status: status, UpdateAvailable: available,
		LocalDigest: localDigest, RemoteDigest: remoteDigest,
		ResourceVersion: summary.ResourceVersion, CheckedAt: c.now().UTC(),
	}, nil
}

func (c *Client) remoteImageDigestForUpdate(ctx context.Context, image string) (string, error) {
	repository, tag, dockerHubImage := normalizedDockerHubReference(image)
	if !dockerHubImage || c.imageUpdateCountry == nil {
		return c.distributionDigest(ctx, image)
	}

	countryContext, countryCancel := context.WithTimeout(ctx, countryLookupTimeout)
	country, countryErr := c.imageUpdateCountry(countryContext)
	countryCancel()
	country = strings.ToUpper(strings.TrimSpace(country))
	if countryErr == nil && country != "" && country != "CN" && country != "HK" {
		return c.distributionDigest(ctx, image)
	}

	officialContext, cancel := context.WithTimeout(ctx, officialUpdateCheckTimeout)
	digest, officialErr := c.distributionDigest(officialContext, image)
	cancel()
	if officialErr == nil {
		return digest, nil
	}

	for _, accelerator := range dockerHubUpdateAccelerators {
		fallbackContext, fallbackCancel := context.WithTimeout(ctx, acceleratorCheckTimeout)
		fallbackImage := accelerator + "/" + repository + ":" + tag
		fallbackDigest, err := c.distributionDigest(fallbackContext, fallbackImage)
		fallbackCancel()
		if err == nil {
			return fallbackDigest, nil
		}
	}
	return "", fmt.Errorf("Docker Hub update check failed after accelerated fallback: %w", officialErr)
}

func (c *Client) distributionDigest(ctx context.Context, image string) (string, error) {
	var remote struct {
		Descriptor struct {
			Digest string `json:"digest"`
		} `json:"Descriptor"`
	}
	if err := c.getJSON(ctx, "/distribution/"+url.PathEscape(image)+"/json", &remote); err != nil {
		return "", err
	}
	if !digestPattern.MatchString(remote.Descriptor.Digest) {
		return "", errors.New("registry returned an invalid image digest")
	}
	return remote.Descriptor.Digest, nil
}

func normalizedDockerHubReference(image string) (string, string, bool) {
	value := strings.TrimSpace(image)
	if value == "" || strings.Contains(value, "@") || strings.ContainsAny(value, "?#\\") {
		return "", "", false
	}
	parts := strings.Split(value, "/")
	if len(parts) > 1 && isRegistryComponent(parts[0]) {
		switch parts[0] {
		case "docker.io", "index.docker.io", "registry-1.docker.io":
			parts = parts[1:]
		default:
			return "", "", false
		}
	}
	if len(parts) == 0 {
		return "", "", false
	}
	last := parts[len(parts)-1]
	tag := "latest"
	if index := strings.LastIndexByte(last, ':'); index >= 0 {
		tag = last[index+1:]
		parts[len(parts)-1] = last[:index]
	}
	if len(parts) == 1 {
		parts = append([]string{"library"}, parts...)
	}
	repository := strings.Join(parts, "/")
	if len(repository) > 255 || !dockerHubPathPattern.MatchString(repository) ||
		!dockerTagPattern.MatchString(tag) {
		return "", "", false
	}
	return repository, tag, true
}

func isRegistryComponent(value string) bool {
	return value == "localhost" || strings.ContainsAny(value, ".:")
}
