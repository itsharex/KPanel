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
	skipExisting := make(map[string]bool)
	mergeAppNo := false
	for _, name := range topLevels {
		if name == "kpanel" || name == "kpanel_port.conf" || name == ".kpanel-backups" ||
			name == "." || name == ".." || !dockerBackupTopPattern.MatchString(name) {
			return errors.New("Docker backup contains an unsafe top-level path")
		}
		source := filepath.Join(stageRoot, "docker", name)
		target := filepath.Join(appRoot, name)
		targetInfo, targetErr := os.Lstat(target)
		if errors.Is(targetErr, os.ErrNotExist) {
			continue
		}
		if targetErr != nil || targetInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("restore conflict: /home/docker/%s is unavailable or unsafe", name)
		}
		sourceInfo, sourceErr := os.Lstat(source)
		if sourceErr != nil {
			return sourceErr
		}
		if name == "appno.txt" && sourceInfo.Mode().IsRegular() && targetInfo.Mode().IsRegular() {
			mergeAppNo = true
			continue
		}
		equal, compareErr := sameRegularFile(source, target)
		if compareErr != nil || !equal {
			return fmt.Errorf("restore conflict: /home/docker/%s already exists", name)
		}
		skipExisting[name] = true
	}
	var created []string
	for _, name := range topLevels {
		if skipExisting[name] || (mergeAppNo && name == "appno.txt") {
			continue
		}
		select {
		case <-ctx.Done():
			rollbackRestoredDockerPaths(created, appRoot)
			return ctx.Err()
		default:
		}
		source := filepath.Join(stageRoot, "docker", name)
		target := filepath.Join(appRoot, name)
		if err := copyRestoredDockerTree(source, target); err != nil {
			rollbackRestoredDockerPaths(append(created, target), appRoot)
			return err
		}
		created = append(created, target)
	}
	if mergeAppNo {
		if err := mergeDockerAppMarkers(
			filepath.Join(stageRoot, "docker", "appno.txt"),
			filepath.Join(appRoot, "appno.txt"),
		); err != nil {
			rollbackRestoredDockerPaths(created, appRoot)
			return err
		}
	}
	return syncDirectoryPath(appRoot)
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
			parts[0] == "kpanel" || parts[0] == "kpanel_port.conf" ||
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

func sameRegularFile(left, right string) (bool, error) {
	leftInfo, err := os.Lstat(left)
	if err != nil {
		return false, err
	}
	rightInfo, err := os.Lstat(right)
	if err != nil {
		return false, err
	}
	if !leftInfo.Mode().IsRegular() || !rightInfo.Mode().IsRegular() ||
		leftInfo.Size() != rightInfo.Size() || leftInfo.Size() > 1<<20 {
		return false, nil
	}
	leftData, err := os.ReadFile(left)
	if err != nil {
		return false, err
	}
	rightData, err := os.ReadFile(right)
	if err != nil {
		return false, err
	}
	return string(leftData) == string(rightData), nil
}

func mergeDockerAppMarkers(source, target string) error {
	sourceData, err := os.ReadFile(source)
	if err != nil || len(sourceData) > 1<<20 {
		return errors.New("backup appno.txt is unavailable or unsafe")
	}
	targetInfo, err := os.Lstat(target)
	if err != nil || !targetInfo.Mode().IsRegular() ||
		targetInfo.Mode()&os.ModeSymlink != 0 || targetInfo.Size() > 1<<20 {
		return errors.New("existing appno.txt is unavailable or unsafe")
	}
	targetData, err := os.ReadFile(target)
	if err != nil {
		return err
	}
	seen := make(map[string]bool)
	var merged []string
	for _, data := range [][]byte{targetData, sourceData} {
		for _, line := range strings.Split(string(data), "\n") {
			value := strings.TrimSpace(line)
			if value == "" {
				continue
			}
			if !dockerBackupTopPattern.MatchString(value) || seen[value] {
				if seen[value] {
					continue
				}
				return errors.New("appno.txt contains an invalid application marker")
			}
			seen[value] = true
			merged = append(merged, value)
		}
	}
	data := []byte(strings.Join(merged, "\n") + "\n")
	return atomicWriteRestoredFile(target, data, targetInfo)
}

func atomicWriteRestoredFile(path string, data []byte, original os.FileInfo) error {
	parent := filepath.Dir(filepath.Clean(path))
	temp, err := os.CreateTemp(parent, ".kpanel-restore-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(original.Mode().Perm()); err != nil {
		_ = temp.Close()
		return err
	}
	uid, gid, err := fileNumericOwnership(original)
	if err != nil {
		_ = temp.Close()
		return err
	}
	if err := applyNumericOwnership(tempPath, uid, gid); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	return syncDirectoryPath(parent)
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

func rollbackRestoredDockerPaths(paths []string, appRoot string) {
	appRoot = filepath.Clean(appRoot)
	for index := len(paths) - 1; index >= 0; index-- {
		path := filepath.Clean(paths[index])
		if filepath.IsAbs(path) && path != string(filepath.Separator) &&
			pathWithin(path, appRoot) && path != appRoot {
			_ = os.RemoveAll(path)
		}
	}
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
