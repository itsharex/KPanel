package dockerx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const maxComposeSourceBytes = 24 << 10

var composeProjectPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)

func (c *Client) validateComposeDeploymentInput(ctx context.Context, input MaintenanceInput) error {
	project := strings.TrimSpace(input.Name)
	source := strings.TrimSpace(input.Compose)
	if project != input.Name || !composeProjectPattern.MatchString(project) || source == "" ||
		len(input.Compose) > maxComposeSourceBytes || !utf8.ValidString(input.Compose) ||
		strings.ContainsRune(input.Compose, 0) {
		return ErrInvalidDockerJob
	}
	root, err := c.resolvedDockerAppRoot()
	if err != nil {
		return err
	}
	target := filepath.Join(root, project)
	if !pathWithin(target, root) || target == root {
		return ErrInvalidDockerJob
	}
	if _, statErr := os.Lstat(target); statErr == nil {
		return ErrResourceConflict
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	containers, err := c.Containers(ctx)
	if err != nil {
		return err
	}
	for _, container := range containers {
		if container.ComposeProject == project {
			return ErrResourceConflict
		}
	}
	return nil
}

func (c *Client) deployComposeProject(ctx context.Context, input MaintenanceInput) error {
	if err := c.validateComposeDeploymentInput(ctx, input); err != nil {
		return err
	}
	root, err := c.resolvedDockerAppRoot()
	if err != nil {
		return err
	}
	projectDir := filepath.Join(root, input.Name)
	if err := os.Mkdir(projectDir, 0o750); err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrResourceConflict
		}
		return err
	}
	composePath := filepath.Join(projectDir, "docker-compose.yml")
	source := input.Compose
	if !strings.HasSuffix(source, "\n") {
		source += "\n"
	}
	file, err := os.OpenFile(composePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		_ = cleanupComposeProjectDirectory(root, projectDir)
		return err
	}
	if _, err = file.WriteString(source); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = cleanupComposeProjectDirectory(root, projectDir)
		return err
	}
	if err := syncDirectoryPath(projectDir); err != nil {
		_ = cleanupComposeProjectDirectory(root, projectDir)
		return err
	}

	base := []string{
		"compose", "--project-directory", projectDir,
		"--file", composePath, "--project-name", input.Name,
	}
	services, err := c.runCompose(ctx, append(base, "config", "--services")...)
	if err != nil {
		_ = cleanupComposeProjectDirectory(root, projectDir)
		return fmt.Errorf("Compose configuration is invalid: %w", err)
	}
	if len(strings.Fields(string(services))) == 0 {
		_ = cleanupComposeProjectDirectory(root, projectDir)
		return errors.New("Compose configuration does not define an active service")
	}
	if _, err := c.runCompose(ctx, append(base, "up", "--detach")...); err != nil {
		return c.rollbackComposeDeployment(root, projectDir, base, "start Compose project", err)
	}
	containerIDs, err := c.runCompose(ctx, append(base, "ps", "--all", "--quiet")...)
	if err != nil {
		return c.rollbackComposeDeployment(root, projectDir, base, "verify Compose project", err)
	}
	validContainer := false
	for _, value := range strings.Fields(string(containerIDs)) {
		if containerIDPattern.MatchString(value) {
			validContainer = true
			break
		}
	}
	if !validContainer {
		return c.rollbackComposeDeployment(
			root, projectDir, base, "verify Compose project",
			errors.New("Docker Compose did not return a created container"),
		)
	}
	if err := syncDirectoryPath(root); err != nil {
		return fmt.Errorf("Compose project started but directory durability needs attention: %w", err)
	}
	return nil
}

func (c *Client) rollbackComposeDeployment(
	root string,
	projectDir string,
	base []string,
	step string,
	cause error,
) error {
	rollbackContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, rollbackErr := c.runCompose(rollbackContext, append(base, "down", "--remove-orphans")...)
	if rollbackErr != nil {
		return fmt.Errorf("%s failed and rollback needs attention: %w", step, cause)
	}
	if cleanupErr := cleanupComposeProjectDirectory(root, projectDir); cleanupErr != nil {
		return fmt.Errorf("%s failed; containers rolled back but project cleanup needs attention: %w", step, cause)
	}
	return fmt.Errorf("%s failed; Compose project rolled back: %w", step, cause)
}

func (c *Client) runCompose(ctx context.Context, arguments ...string) ([]byte, error) {
	run := c.composeCommand
	if run == nil {
		run = runFixedDockerComposeCommand
	}
	output, err := run(ctx, arguments...)
	if err == nil {
		return output, nil
	}
	detail := strings.TrimSpace(redactText(string(output)))
	if len(detail) > 400 {
		detail = detail[:400]
	}
	if detail == "" {
		return output, err
	}
	return output, fmt.Errorf("%w: %s", err, detail)
}

func cleanupComposeProjectDirectory(root, target string) error {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if target == root || !pathWithin(target, root) {
		return errors.New("Compose cleanup target is unsafe")
	}
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("Compose cleanup target is unavailable or unsafe")
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	return syncDirectoryPath(root)
}
