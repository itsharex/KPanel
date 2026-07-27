package dockerx

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	maxDockerBackupEntries = 100_000
	maxDockerBackupBytes   = int64(50 << 30)
)

var (
	dockerBackupIDPattern  = regexp.MustCompile(`^docker-[0-9]{8}T[0-9]{6}Z-[a-f0-9]{8}\.tar\.gz$`)
	dockerBackupTopPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
	migrationHostPattern   = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9.-]{0,251}[A-Za-z0-9])?$`)
	migrationUserPattern   = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
)

type DockerBackup struct {
	ID        string    `json:"id"`
	SizeBytes int64     `json:"sizeBytes"`
	CreatedAt time.Time `json:"createdAt"`
	Format    string    `json:"format"`
}

func (c *Client) DockerBackups() ([]DockerBackup, error) {
	root := c.dockerBackupRoot()
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return []DockerBackup{}, nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("Docker backup directory is unavailable or unsafe")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	result := make([]DockerBackup, 0, len(entries))
	for _, entry := range entries {
		if !dockerBackupIDPattern.MatchString(entry.Name()) || entry.IsDir() ||
			entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		entryInfo, statErr := entry.Info()
		if statErr != nil || !entryInfo.Mode().IsRegular() ||
			entryInfo.Size() <= 0 || entryInfo.Size() > maxDockerBackupBytes {
			continue
		}
		result = append(result, DockerBackup{
			ID: entry.Name(), SizeBytes: entryInfo.Size(),
			CreatedAt: entryInfo.ModTime().UTC(), Format: "kpanel-home-docker-v1",
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (c *Client) dockerBackupPath(id string) (string, error) {
	if !dockerBackupIDPattern.MatchString(id) {
		return "", ErrDockerJobNotFound
	}
	root := c.dockerBackupRoot()
	path := filepath.Join(root, id)
	if filepath.Dir(path) != root {
		return "", ErrDockerJobNotFound
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
		info.Size() <= 0 || info.Size() > maxDockerBackupBytes {
		return "", ErrDockerJobNotFound
	}
	return path, nil
}

func (c *Client) dockerBackupRoot() string {
	return filepath.Join(filepath.Clean(c.appRoot), ".kpanel-backups")
}

func (c *Client) resolvedDockerAppRoot() (string, error) {
	root := filepath.Clean(c.appRoot)
	if !filepath.IsAbs(root) || root == string(filepath.Separator) {
		return "", errors.New("Docker application root is unsafe")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", errors.New("Docker application root is unavailable or unsafe")
	}
	resolved = filepath.Clean(resolved)
	if !filepath.IsAbs(resolved) || resolved == string(filepath.Separator) {
		return "", errors.New("Docker application root resolved to an unsafe path")
	}
	info, err := os.Lstat(resolved)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("Docker application root is unavailable or unsafe")
	}
	return resolved, nil
}

func (c *Client) restoreDockerBackup(ctx context.Context, id string) error {
	archivePath, err := c.dockerBackupPath(id)
	if err != nil {
		return err
	}
	stageRoot, err := os.MkdirTemp(filepath.Clean(c.stateRoot), ".docker-restore-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stageRoot)
	if err := os.Chmod(stageRoot, 0o700); err != nil {
		return err
	}
	topLevels, err := extractDockerBackup(ctx, archivePath, stageRoot)
	if err != nil {
		return err
	}
	appRoot, err := c.resolvedDockerAppRoot()
	if err != nil {
		return err
	}
	rollbackRoot, err := os.MkdirTemp(appRoot, ".kpanel-restore-rollback-*")
	if err != nil {
		return err
	}
	if err := os.Chmod(rollbackRoot, 0o700); err != nil {
		_ = os.RemoveAll(rollbackRoot)
		return err
	}
	var replacements []dockerRestoreReplacement
	for _, name := range topLevels {
		if name == ".kpanel-backups" ||
			name == "." || name == ".." || !dockerBackupTopPattern.MatchString(name) {
			_ = os.RemoveAll(rollbackRoot)
			return errors.New("Docker backup contains an unsafe top-level path")
		}
	}
	for _, name := range topLevels {
		select {
		case <-ctx.Done():
			rollbackDockerRestore(replacements, rollbackRoot, appRoot)
			return ctx.Err()
		default:
		}
		source := filepath.Join(stageRoot, "docker", name)
		target := filepath.Join(appRoot, name)
		replacement := dockerRestoreReplacement{target: target}
		if _, targetErr := os.Lstat(target); targetErr == nil {
			replacement.previous = filepath.Join(rollbackRoot, name)
			if err := os.Rename(target, replacement.previous); err != nil {
				rollbackDockerRestore(replacements, rollbackRoot, appRoot)
				return fmt.Errorf("stage existing /home/docker/%s for rollback: %w", name, err)
			}
		} else if !errors.Is(targetErr, os.ErrNotExist) {
			rollbackDockerRestore(replacements, rollbackRoot, appRoot)
			return fmt.Errorf("inspect existing /home/docker/%s: %w", name, targetErr)
		}
		replacements = append(replacements, replacement)
		if err := copyRestoredDockerTree(source, target); err != nil {
			rollbackDockerRestore(replacements, rollbackRoot, appRoot)
			return err
		}
	}
	if err := syncDirectoryPath(appRoot); err != nil {
		rollbackDockerRestore(replacements, rollbackRoot, appRoot)
		return err
	}
	if err := os.RemoveAll(rollbackRoot); err != nil {
		return fmt.Errorf("restore completed but previous data cleanup failed: %w", err)
	}
	return syncDirectoryPath(appRoot)
}

type dockerRestoreReplacement struct {
	target   string
	previous string
}

func rollbackDockerRestore(replacements []dockerRestoreReplacement, rollbackRoot, appRoot string) {
	for index := len(replacements) - 1; index >= 0; index-- {
		replacement := replacements[index]
		if filepath.IsAbs(replacement.target) && pathWithin(replacement.target, appRoot) &&
			replacement.target != appRoot {
			_ = os.RemoveAll(replacement.target)
		}
		if replacement.previous != "" && pathWithin(replacement.previous, rollbackRoot) {
			_ = os.Rename(replacement.previous, replacement.target)
		}
	}
	_ = os.RemoveAll(rollbackRoot)
	_ = syncDirectoryPath(appRoot)
}

func extractDockerBackup(
	ctx context.Context,
	archivePath string,
	stageRoot string,
) ([]string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, errors.New("Docker backup is not a valid gzip archive")
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	topLevels := make(map[string]bool)
	var entries int
	var total int64
	for {
		header, nextErr := reader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return nil, errors.New("Docker backup tar stream is invalid")
		}
		entries++
		if entries > maxDockerBackupEntries {
			return nil, errors.New("Docker backup contains too many entries")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(header.Name)))
		if clean == "." || clean == "docker" {
			continue
		}
		if strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "../") ||
			!strings.HasPrefix(clean, "docker/") {
			return nil, errors.New("Docker backup contains an unsafe path")
		}
		relative := strings.TrimPrefix(clean, "docker/")
		parts := strings.Split(relative, "/")
		if len(parts) == 0 || !dockerBackupTopPattern.MatchString(parts[0]) ||
			parts[0] == "." || parts[0] == ".." ||
			parts[0] == ".kpanel-backups" {
			return nil, errors.New("Docker backup contains an unsafe application path")
		}
		topLevels[parts[0]] = true
		if header.Size < 0 || header.Size > 10<<30 || total+header.Size > maxDockerBackupBytes {
			return nil, errors.New("Docker backup exceeds the restore safety limit")
		}
		if header.Uid < 0 || header.Gid < 0 ||
			header.Uid > 1<<31-1 || header.Gid > 1<<31-1 {
			return nil, errors.New("Docker backup contains invalid numeric ownership")
		}
		total += header.Size
		target := filepath.Join(stageRoot, filepath.FromSlash(clean))
		if !pathWithin(target, stageRoot) {
			return nil, errors.New("Docker backup path escaped the staging directory")
		}
		mode := os.FileMode(header.Mode).Perm() & 0o777
		switch header.Typeflag {
		case tar.TypeDir:
			if mode == 0 {
				mode = 0o750
			}
			if err := os.MkdirAll(target, mode); err != nil {
				return nil, err
			}
			if err := applyNumericOwnership(target, header.Uid, header.Gid); err != nil {
				return nil, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
				return nil, err
			}
			if mode == 0 {
				mode = 0o640
			}
			output, openErr := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
			if openErr != nil {
				return nil, openErr
			}
			written, copyErr := io.Copy(output, io.LimitReader(reader, header.Size+1))
			syncErr := output.Sync()
			closeErr := output.Close()
			if copyErr != nil || written != header.Size || syncErr != nil || closeErr != nil {
				return nil, errors.New("Docker backup entry could not be restored safely")
			}
			if err := applyNumericOwnership(target, header.Uid, header.Gid); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("Docker backup contains links or unsupported filesystem objects")
		}
	}
	names := make([]string, 0, len(topLevels))
	for name := range topLevels {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, errors.New("Docker backup does not contain application data")
	}
	return names, nil
}

func pathWithin(candidate, root string) bool {
	candidate = filepath.Clean(candidate)
	root = filepath.Clean(root)
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func copyRestoredDockerTree(source, target string) error {
	sourceInfo, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if sourceInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("restore source contains a symbolic link")
	}
	if sourceInfo.IsDir() {
		if err := os.Mkdir(target, sourceInfo.Mode().Perm()); err != nil {
			return err
		}
		uid, gid, err := fileNumericOwnership(sourceInfo)
		if err != nil {
			return err
		}
		if err := applyNumericOwnership(target, uid, gid); err != nil {
			return err
		}
		entries, err := os.ReadDir(source)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyRestoredDockerTree(
				filepath.Join(source, entry.Name()),
				filepath.Join(target, entry.Name()),
			); err != nil {
				return err
			}
		}
		return syncDirectoryPath(target)
	}
	if !sourceInfo.Mode().IsRegular() {
		return errors.New("restore source contains an unsupported filesystem object")
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, sourceInfo.Mode().Perm())
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(output, io.LimitReader(input, sourceInfo.Size()+1))
	syncErr := output.Sync()
	closeErr := output.Close()
	if copyErr != nil || written != sourceInfo.Size() || syncErr != nil || closeErr != nil {
		return errors.New("restore file copy failed")
	}
	uid, gid, err := fileNumericOwnership(sourceInfo)
	if err != nil {
		return err
	}
	return applyNumericOwnership(target, uid, gid)
}

func validMigrationHost(value string) bool {
	value = strings.TrimSpace(value)
	if parsed := net.ParseIP(value); parsed != nil {
		return true
	}
	return len(value) <= 253 && migrationHostPattern.MatchString(value) &&
		!strings.Contains(value, "..")
}

func (c *Client) migrateDockerBackup(
	ctx context.Context,
	id string,
	host string,
	user string,
	port int,
) (string, error) {
	path, err := c.dockerBackupPath(id)
	if err != nil {
		return "", err
	}
	if !validMigrationHost(host) || !migrationUserPattern.MatchString(user) ||
		port < 1 || port > 65535 {
		return "", ErrInvalidDockerJob
	}
	run := c.hostCommand
	if run == nil {
		run = runFixedDockerHostCommand
	}
	destinationHost := host
	if parsed := net.ParseIP(host); parsed != nil && parsed.To4() == nil {
		destinationHost = "[" + host + "]"
	}
	destination := user + "@" + destinationHost + ":/tmp/" + id
	_, err = run(
		ctx,
		"scp",
		"-P", strconv.Itoa(port),
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "ConnectTimeout=10",
		"--",
		path,
		destination,
	)
	if err != nil {
		return "", fmt.Errorf("migrate Docker backup: %w", err)
	}
	return destination, nil
}
