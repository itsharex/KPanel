package dockerx

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var digestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

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
	var remote struct {
		Descriptor struct {
			Digest string `json:"digest"`
		} `json:"Descriptor"`
	}
	if err := c.getJSON(ctx, "/distribution/"+url.PathEscape(image)+"/json", &remote); err != nil {
		return ImageUpdateResult{}, err
	}
	if !digestPattern.MatchString(remote.Descriptor.Digest) {
		return ImageUpdateResult{}, errors.New("registry returned an invalid image digest")
	}
	localDigest := ""
	for _, value := range local.RepoDigests {
		_, digest, found := strings.Cut(value, "@")
		if found && digestPattern.MatchString(digest) {
			localDigest = digest
			if digest == remote.Descriptor.Digest {
				break
			}
		}
	}
	if localDigest == "" {
		return ImageUpdateResult{}, errors.New("local image does not expose a repository digest")
	}
	available := localDigest != remote.Descriptor.Digest
	status := "current"
	if available {
		status = "available"
	}
	return ImageUpdateResult{
		ContainerID: raw.ID, Image: image, Status: status, UpdateAvailable: available,
		LocalDigest: localDigest, RemoteDigest: remote.Descriptor.Digest,
		ResourceVersion: summary.ResourceVersion, CheckedAt: c.now().UTC(),
	}, nil
}
