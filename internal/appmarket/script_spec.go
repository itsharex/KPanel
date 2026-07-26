package appmarket

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const maxScriptAppConfigBytes = 256 << 10

var (
	dockerObjectNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
	scriptAssignmentPattern = regexp.MustCompile(
		`^(?:local[[:space:]]+)?(docker_name|docker_app_service)[[:space:]]*=[[:space:]]*(?:"([A-Za-z0-9][A-Za-z0-9_.-]{0,127})"|'([A-Za-z0-9][A-Za-z0-9_.-]{0,127})'|([A-Za-z0-9][A-Za-z0-9_.-]{0,127}))[[:space:]]*(?:#.*)?$`,
	)
	scriptAssignmentPrefixPattern = regexp.MustCompile(
		`^(?:local[[:space:]]+)?(docker_name|docker_app_service)[[:space:]]*=`,
	)
)

type scriptAppSpec struct {
	Container string
	Service   string
	Adapter   string
}

func (spec scriptAppSpec) runtimeContainer() string {
	if spec.Service != "" {
		return spec.Service
	}
	return spec.Container
}

func (s *Service) readThirdPartyScriptSpec(token string) (scriptAppSpec, error) {
	if !tokenPattern.MatchString(token) {
		return scriptAppSpec{}, errors.New("invalid application token")
	}
	path := filepath.Join(s.scriptAppRoot, token+".conf")
	info, err := os.Lstat(path)
	if err != nil {
		return scriptAppSpec{}, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
		info.Size() <= 0 || info.Size() > maxScriptAppConfigBytes ||
		!s.fileOwnerTrusted(info) ||
		(runtime.GOOS == "linux" && info.Mode().Perm()&0o022 != 0) {
		return scriptAppSpec{}, errors.New("application configuration is not a trusted regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return scriptAppSpec{}, err
	}
	defer file.Close()

	spec := scriptAppSpec{}
	assignments := make(map[string]bool, 2)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4<<10), maxScriptAppConfigBytes)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch line {
		case "docker_app", "docker_app_plus":
			if spec.Adapter != "" {
				return scriptAppSpec{}, errors.New("application adapter is duplicated")
			}
			spec.Adapter = line
			continue
		}
		matches := scriptAssignmentPattern.FindStringSubmatch(line)
		if len(matches) == 5 {
			key := matches[1]
			if assignments[key] {
				return scriptAppSpec{}, fmt.Errorf("%s is duplicated", key)
			}
			assignments[key] = true
			value := matches[2] + matches[3] + matches[4]
			if !dockerObjectNamePattern.MatchString(value) {
				return scriptAppSpec{}, fmt.Errorf("%s is invalid", key)
			}
			if key == "docker_name" {
				spec.Container = value
			} else {
				spec.Service = value
			}
			continue
		}
		if scriptAssignmentPrefixPattern.MatchString(line) {
			return scriptAppSpec{}, errors.New("dynamic container assignments are not accepted")
		}
	}
	if err := scanner.Err(); err != nil {
		return scriptAppSpec{}, err
	}
	if spec.Container == "" || spec.Adapter == "" {
		return scriptAppSpec{}, errors.New("application container metadata is incomplete")
	}
	return spec, nil
}

func (s *Service) readScriptAccessMode(containerName string) (string, bool) {
	if !dockerObjectNamePattern.MatchString(containerName) {
		return "", false
	}
	path := filepath.Join(s.appRoot, containerName+"_access.conf")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
		info.Size() <= 0 || info.Size() > 64 ||
		!s.fileOwnerTrusted(info) ||
		(runtime.GOOS == "linux" && info.Mode().Perm()&0o022 != 0) {
		return "", false
	}
	value, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	mode := strings.TrimSpace(string(value))
	if mode != "direct" && mode != "domain_only" {
		return "", false
	}
	return mode, true
}
