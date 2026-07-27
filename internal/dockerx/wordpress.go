package dockerx

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/sites"
)

const (
	wordPressMySQLContainer = "mysql"
	wordPressCertbotImage   = "certbot/certbot@sha256:34ee91d2f43008eb78a007d22f23ed4b2eaa9a454cb27ca2c042b49527a695b4"
)

var (
	wordPressDatabasePattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)
	wordPressUserPattern     = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)
	wordPressDomainPattern   = regexp.MustCompile(
		`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`,
	)
)

type wordPressMySQLRuntime struct {
	container containerInspect
	root      string
	user      string
	password  string
}

func (c *Client) WordPressReady(ctx context.Context) error {
	nginx, err := c.readyNginx(ctx)
	if err != nil {
		return err
	}
	for _, requirement := range []struct {
		hostPath      string
		containerPath string
	}{
		{pathpkg.Join(c.webRoot, "certs"), "/etc/nginx/certs"},
		{pathpkg.Join(c.webRoot, "letsencrypt"), "/var/www/letsencrypt"},
	} {
		if !c.hasExactSafeNginxBind(
			nginx.Mounts,
			requirement.hostPath,
			requirement.containerPath,
			true,
		) {
			return fmt.Errorf(
				"Nginx does not expose the kejilion.sh binding %s -> %s",
				requirement.hostPath,
				requirement.containerPath,
			)
		}
	}
	if err := c.readyWordPressServices(ctx); err != nil {
		if composeErr := c.validateWordPressCompose(ctx); composeErr != nil {
			return fmt.Errorf("LDNMP is not running and its Compose artifact is incompatible: %v", composeErr)
		}
	}
	for _, requirement := range []struct {
		path      string
		directory bool
	}{
		{filepath.FromSlash(pathpkg.Join(c.webRoot, "certs")), true},
		{filepath.FromSlash(pathpkg.Join(c.webRoot, "letsencrypt")), true},
	} {
		info, err := os.Lstat(requirement.path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("required WordPress directory %s is unavailable or unsafe", requirement.path)
		}
	}
	return nil
}

func (c *Client) PrepareWordPressDatabase(
	ctx context.Context,
	database string,
) (sites.WordPressDatabaseCredentials, error) {
	if !wordPressDatabasePattern.MatchString(database) {
		return sites.WordPressDatabaseCredentials{}, errors.New("invalid WordPress database identity")
	}
	if err := c.ensureWordPressLDNMP(ctx); err != nil {
		return sites.WordPressDatabaseCredentials{}, err
	}
	runtime, err := c.readyWordPressMySQL(ctx)
	if err != nil {
		return sites.WordPressDatabaseCredentials{}, err
	}
	if err := c.runWordPressMySQL(ctx, runtime, "CREATE DATABASE `"+database+"`;"); err != nil {
		return sites.WordPressDatabaseCredentials{}, err
	}
	if err := c.runWordPressMySQL(
		ctx,
		runtime,
		"GRANT ALL PRIVILEGES ON `"+database+"`.* TO '"+runtime.user+"'@'%';",
	); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if cleanupErr := c.runWordPressMySQL(
			cleanupCtx,
			runtime,
			"DROP DATABASE IF EXISTS `"+database+"`;",
		); cleanupErr != nil {
			return sites.WordPressDatabaseCredentials{}, errors.New(
				"grant WordPress database access failed and the new database requires manual cleanup",
			)
		}
		return sites.WordPressDatabaseCredentials{}, errors.New(
			"grant WordPress database access failed; the new database was rolled back",
		)
	}
	return sites.WordPressDatabaseCredentials{
		Name: database, User: runtime.user, Password: runtime.password,
	}, nil
}

func (c *Client) RollbackWordPressDatabase(ctx context.Context, database string) error {
	if !wordPressDatabasePattern.MatchString(database) {
		return errors.New("invalid WordPress database identity")
	}
	runtime, err := c.readyWordPressMySQL(ctx)
	if err != nil {
		return err
	}
	return c.runWordPressMySQL(ctx, runtime, "DROP DATABASE IF EXISTS `"+database+"`;")
}

func (c *Client) PrepareWordPressCertificate(ctx context.Context, domain string) error {
	if len(domain) > 253 || !wordPressDomainPattern.MatchString(domain) {
		return errors.New("invalid WordPress certificate domain")
	}
	certDestination := filepath.FromSlash(pathpkg.Join(c.webRoot, "certs", domain+"_cert.pem"))
	keyDestination := filepath.FromSlash(pathpkg.Join(c.webRoot, "certs", domain+"_key.pem"))
	for _, path := range []string{certDestination, keyDestination} {
		if _, err := os.Lstat(path); err == nil {
			return errors.New("WordPress certificate destination already exists")
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	certSource, keySource, err := wordPressCertificateSources(domain)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if issueErr := c.issueWordPressCertificate(ctx, domain); issueErr != nil {
			return issueErr
		}
		certSource, keySource, err = wordPressCertificateSources(domain)
		if err != nil {
			return fmt.Errorf("issued certificate artifacts are unavailable: %w", err)
		}
	}
	cert, err := readWordPressCertificateFile(certSource, 4<<20)
	if err != nil {
		return err
	}
	key, err := readWordPressCertificateFile(keySource, 1<<20)
	if err != nil {
		return err
	}
	if err := validateWordPressKeyPair(domain, cert, key); err != nil {
		return err
	}
	if err := publishCertificateFile(certDestination, cert, 0o644); err != nil {
		return err
	}
	if err := publishCertificateFile(keyDestination, key, 0o600); err != nil {
		_ = removeMatchingCertificateFile(certDestination, cert)
		return err
	}
	return nil
}

func (c *Client) RollbackWordPressCertificate(_ context.Context, domain string) error {
	if len(domain) > 253 || !wordPressDomainPattern.MatchString(domain) {
		return errors.New("invalid WordPress certificate domain")
	}
	certSource, keySource, err := wordPressCertificateSources(domain)
	if err != nil {
		return err
	}
	cert, err := readWordPressCertificateFile(certSource, 4<<20)
	if err != nil {
		return err
	}
	key, err := readWordPressCertificateFile(keySource, 1<<20)
	if err != nil {
		return err
	}
	certDestination := filepath.FromSlash(pathpkg.Join(c.webRoot, "certs", domain+"_cert.pem"))
	keyDestination := filepath.FromSlash(pathpkg.Join(c.webRoot, "certs", domain+"_key.pem"))
	if err := removeMatchingCertificateFile(keyDestination, key); err != nil {
		return err
	}
	return removeMatchingCertificateFile(certDestination, cert)
}

func (c *Client) readyWordPressMySQL(ctx context.Context) (wordPressMySQLRuntime, error) {
	raw, err := c.inspect(ctx, wordPressMySQLContainer)
	if err != nil {
		return wordPressMySQLRuntime{}, fmt.Errorf("inspect fixed MySQL container: %w", err)
	}
	if strings.TrimPrefix(raw.Name, "/") != wordPressMySQLContainer ||
		!dockerExecIDPattern.MatchString(raw.ID) ||
		!raw.State.Running || raw.State.Status != "running" || raw.State.Paused || raw.State.Restarting {
		return wordPressMySQLRuntime{}, errors.New("fixed MySQL container is not running normally")
	}
	if raw.State.Health != nil && raw.State.Health.Status != "healthy" {
		return wordPressMySQLRuntime{}, errors.New("fixed MySQL container is not healthy")
	}
	expectedSource := pathpkg.Join(c.webRoot, "mysql")
	mountOK := false
	for _, mount := range raw.Mounts {
		if mount.Type == "bind" && mount.RW &&
			pathpkg.Clean(mount.Source) == expectedSource &&
			pathpkg.Clean(mount.Destination) == "/var/lib/mysql" {
			mountOK = true
		}
	}
	if !mountOK {
		return wordPressMySQLRuntime{}, errors.New("MySQL does not use the kejilion.sh data binding")
	}
	values := make(map[string]string)
	for _, value := range raw.Config.Env {
		key, content, ok := strings.Cut(value, "=")
		if ok && (key == "MYSQL_ROOT_PASSWORD" || key == "MYSQL_USER" || key == "MYSQL_PASSWORD") {
			values[key] = content
		}
	}
	if values["MYSQL_ROOT_PASSWORD"] == "" || values["MYSQL_PASSWORD"] == "" ||
		!wordPressUserPattern.MatchString(values["MYSQL_USER"]) {
		return wordPressMySQLRuntime{}, errors.New("MySQL credentials do not match the kejilion.sh contract")
	}
	return wordPressMySQLRuntime{
		container: raw,
		root:      values["MYSQL_ROOT_PASSWORD"],
		user:      values["MYSQL_USER"],
		password:  values["MYSQL_PASSWORD"],
	}, nil
}

func (c *Client) readyWordPressServices(ctx context.Context) error {
	if _, err := c.readyWordPressMySQL(ctx); err != nil {
		return err
	}
	nginx, err := c.readyNginx(ctx)
	if err != nil {
		return err
	}
	for _, name := range []string{"php", "php74", "redis"} {
		raw, err := c.inspect(ctx, name)
		if err != nil {
			return fmt.Errorf("inspect fixed %s container: %w", name, err)
		}
		if strings.TrimPrefix(raw.Name, "/") != name ||
			!dockerExecIDPattern.MatchString(raw.ID) ||
			!raw.State.Running || raw.State.Status != "running" ||
			raw.State.Paused || raw.State.Restarting {
			return fmt.Errorf("fixed %s container is not running normally", name)
		}
		if name == "php" || name == "php74" {
			target := "/run/" + name
			source, ok := dockerVolumeSource(raw.Mounts, target)
			if !ok || !dockerUsesVolumeSource(nginx.Mounts, source, target) {
				return fmt.Errorf("fixed %s does not share its kejilion.sh socket with Nginx", name)
			}
		}
	}
	return nil
}

func (c *Client) ensureWordPressLDNMP(ctx context.Context) error {
	if err := c.readyWordPressServices(ctx); err == nil {
		return nil
	}
	if err := c.validateWordPressCompose(ctx); err != nil {
		return err
	}
	_, _, err := c.runWordPressDockerCLI(
		ctx,
		"compose", "-f", filepath.FromSlash(pathpkg.Join(c.webRoot, "docker-compose.yml")),
		"up", "-d", "--no-recreate", "mysql", "redis", "php", "php74",
	)
	if err != nil {
		return fmt.Errorf("start the fixed kejilion.sh LDNMP services: %w", err)
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := c.readyWordPressServices(ctx); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("LDNMP services did not become ready before the install timeout")
		case <-ticker.C:
		}
	}
}

type wordPressComposeConfig struct {
	Services map[string]struct {
		ContainerName string            `json:"container_name"`
		Image         string            `json:"image"`
		Command       json.RawMessage   `json:"command"`
		Entrypoint    json.RawMessage   `json:"entrypoint"`
		Environment   map[string]string `json:"environment"`
		Privileged    bool              `json:"privileged"`
		NetworkMode   string            `json:"network_mode"`
		Restart       string            `json:"restart"`
		Volumes       []struct {
			Type   string `json:"type"`
			Source string `json:"source"`
			Target string `json:"target"`
		} `json:"volumes"`
	} `json:"services"`
}

func (c *Client) validateWordPressCompose(ctx context.Context) error {
	composePath := filepath.FromSlash(pathpkg.Join(c.webRoot, "docker-compose.yml"))
	info, err := os.Lstat(composePath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
		info.Size() < 1 || info.Size() > 256<<10 {
		return errors.New("docker-compose.yml is unavailable or unsafe")
	}
	output, _, err := c.runWordPressDockerCLI(
		ctx, "compose", "-f", composePath, "config", "--format", "json",
	)
	if err != nil {
		return fmt.Errorf("validate docker-compose.yml: %w", err)
	}
	var config wordPressComposeConfig
	if err := json.Unmarshal(output, &config); err != nil {
		return errors.New("Docker Compose returned invalid JSON")
	}
	nginx, ok := config.Services["nginx"]
	if !ok || nginx.ContainerName != "nginx" ||
		!composeHasBinding(
			nginx.Volumes,
			pathpkg.Join(c.webRoot, "certs"),
			"/etc/nginx/certs",
		) ||
		!composeHasBinding(
			nginx.Volumes,
			pathpkg.Join(c.webRoot, "letsencrypt"),
			"/var/www/letsencrypt",
		) {
		return errors.New("Compose service nginx does not match the kejilion.sh WordPress contract")
	}
	requirements := map[string]struct {
		source       string
		target       string
		needsSecrets bool
	}{
		"mysql": {pathpkg.Join(c.webRoot, "mysql"), "/var/lib/mysql", true},
		"redis": {pathpkg.Join(c.webRoot, "redis"), "/data", false},
		"php":   {pathpkg.Join(c.webRoot, "html"), "/var/www/html", false},
		"php74": {pathpkg.Join(c.webRoot, "html"), "/var/www/html", false},
	}
	for name, requirement := range requirements {
		service, ok := config.Services[name]
		if !ok || service.ContainerName != name ||
			!composeHasBinding(service.Volumes, requirement.source, requirement.target) {
			return fmt.Errorf("Compose service %s does not match the kejilion.sh contract", name)
		}
		if requirement.needsSecrets &&
			(service.Environment["MYSQL_ROOT_PASSWORD"] == "" ||
				service.Environment["MYSQL_PASSWORD"] == "" ||
				!wordPressUserPattern.MatchString(service.Environment["MYSQL_USER"])) {
			return errors.New("Compose MySQL credentials do not match the kejilion.sh contract")
		}
	}
	for _, name := range []string{"php", "php74"} {
		target := "/run/" + name
		phpSource, ok := composeVolumeSource(config.Services[name].Volumes, target)
		if !ok || !composeUsesVolumeSource(nginx.Volumes, phpSource, target) {
			return fmt.Errorf("Compose service %s does not share its socket with nginx", name)
		}
	}
	return nil
}

func (c *Client) runWordPressDockerCLI(
	ctx context.Context,
	arguments ...string,
) ([]byte, []byte, error) {
	const dockerPath = "/usr/bin/docker"
	info, err := os.Lstat(dockerPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o022 != 0 {
		return nil, nil, errors.New("fixed Docker CLI is unavailable or writable by an untrusted user")
	}
	command := exec.CommandContext(
		ctx, dockerPath,
		append([]string{"--host", "unix:///var/run/docker.sock"}, arguments...)...,
	)
	command.Dir = filepath.FromSlash(c.webRoot)
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		switch key {
		case "DOCKER_HOST", "DOCKER_CONTEXT", "COMPOSE_FILE", "COMPOSE_PROJECT_NAME":
			continue
		default:
			command.Env = append(command.Env, value)
		}
	}
	stdout := &boundedCommandBuffer{limit: 2 << 20}
	stderr := &boundedCommandBuffer{limit: 64 << 10}
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return stdout.Bytes(), stderr.Bytes(), errors.New("fixed Docker Compose operation failed")
	}
	if stdout.overflow || stderr.overflow {
		return nil, nil, errors.New("fixed Docker Compose output exceeded the safety limit")
	}
	return stdout.Bytes(), stderr.Bytes(), nil
}

type boundedCommandBuffer struct {
	data     bytes.Buffer
	limit    int
	overflow bool
}

func (buffer *boundedCommandBuffer) Write(content []byte) (int, error) {
	original := len(content)
	remaining := buffer.limit - buffer.data.Len()
	if remaining <= 0 {
		buffer.overflow = true
		return original, nil
	}
	if len(content) > remaining {
		buffer.overflow = true
		content = content[:remaining]
	}
	_, _ = buffer.data.Write(content)
	return original, nil
}

func (buffer *boundedCommandBuffer) Bytes() []byte {
	return buffer.data.Bytes()
}

func composeHasBinding(volumes []struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Target string `json:"target"`
}, source, target string) bool {
	for _, volume := range volumes {
		if volume.Type == "bind" &&
			pathpkg.Clean(volume.Source) == pathpkg.Clean(source) &&
			pathpkg.Clean(volume.Target) == pathpkg.Clean(target) {
			return true
		}
	}
	return false
}

func composeVolumeSource(volumes []struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Target string `json:"target"`
}, target string) (string, bool) {
	for _, volume := range volumes {
		if volume.Type == "volume" && volume.Source != "" &&
			pathpkg.Clean(volume.Target) == pathpkg.Clean(target) {
			return volume.Source, true
		}
	}
	return "", false
}

func composeUsesVolumeSource(volumes []struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Target string `json:"target"`
}, source, target string) bool {
	for _, volume := range volumes {
		if volume.Type == "volume" && volume.Source == source &&
			pathpkg.Clean(volume.Target) == pathpkg.Clean(target) {
			return true
		}
	}
	return false
}

func dockerVolumeSource(volumes []dockerMount, target string) (string, bool) {
	for _, volume := range volumes {
		if volume.Type == "volume" && volume.RW && volume.Source != "" &&
			pathpkg.Clean(volume.Destination) == pathpkg.Clean(target) {
			return volume.Source, true
		}
	}
	return "", false
}

func dockerUsesVolumeSource(volumes []dockerMount, source, target string) bool {
	for _, volume := range volumes {
		if volume.Type == "volume" && volume.RW && volume.Source == source &&
			pathpkg.Clean(volume.Destination) == pathpkg.Clean(target) {
			return true
		}
	}
	return false
}

func (c *Client) runWordPressMySQL(
	ctx context.Context,
	runtime wordPressMySQLRuntime,
	sql string,
) error {
	payload, err := json.Marshal(struct {
		AttachStdout bool     `json:"AttachStdout"`
		AttachStderr bool     `json:"AttachStderr"`
		Tty          bool     `json:"Tty"`
		Env          []string `json:"Env"`
		Cmd          []string `json:"Cmd"`
	}{
		AttachStdout: true,
		AttachStderr: true,
		Env:          []string{"MYSQL_PWD=" + runtime.root},
		Cmd:          []string{"mysql", "-u", "root", "-e", sql},
	})
	if err != nil {
		return err
	}
	data, _, err := c.nginxDockerRequest(
		ctx, http.MethodPost, "/containers/"+runtime.container.ID+"/exec",
		payload, 4<<10,
	)
	if err != nil {
		return err
	}
	var created struct {
		ID string `json:"Id"`
	}
	if err := json.Unmarshal(data, &created); err != nil ||
		!dockerExecIDPattern.MatchString(created.ID) {
		return ErrInvalidDockerExec
	}
	if _, _, err := c.startNginxExec(ctx, created.ID); err != nil {
		return err
	}
	state, err := c.inspectNginxExec(ctx, created.ID)
	if err != nil {
		return err
	}
	if state.Running {
		return ErrNginxExecRunning
	}
	if state.ExitCode != 0 {
		return errors.New("fixed MySQL operation failed")
	}
	return nil
}

func (c *Client) issueWordPressCertificate(ctx context.Context, domain string) error {
	for _, directory := range []string{
		"/etc/letsencrypt",
		filepath.FromSlash(pathpkg.Join(c.webRoot, "letsencrypt")),
	} {
		if err := ensureWordPressDirectory(directory); err != nil {
			return err
		}
	}
	if err := c.pullImage(ctx, wordPressCertbotImage); err != nil {
		return fmt.Errorf("pull fixed Certbot image: %w", err)
	}
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	name := "kpanel-certbot-" + hex.EncodeToString(random)
	payload := struct {
		Image      string            `json:"Image"`
		Cmd        []string          `json:"Cmd"`
		Labels     map[string]string `json:"Labels"`
		HostConfig struct {
			Binds []string `json:"Binds"`
		} `json:"HostConfig"`
	}{
		Image: wordPressCertbotImage,
		Cmd: []string{
			"certonly", "--webroot", "--webroot-path", "/var/www/letsencrypt",
			"--domain", domain, "--email", "your@email.com", "--agree-tos",
			"--no-eff-email", "--non-interactive", "--key-type", "ecdsa",
		},
		Labels: map[string]string{
			"io.kejilion.panel.managed": "true",
			"io.kejilion.panel.job":     "wordpress-certificate",
		},
	}
	payload.HostConfig.Binds = []string{
		"/etc/letsencrypt:/etc/letsencrypt",
		pathpkg.Join(c.webRoot, "letsencrypt") + ":/var/www/letsencrypt",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	data, err := c.appDockerRequest(
		ctx, http.MethodPost, "/containers/create?name="+name,
		body, 64<<10, c.httpClient,
	)
	if err != nil {
		return err
	}
	var created struct {
		ID string `json:"Id"`
	}
	if err := json.Unmarshal(data, &created); err != nil ||
		!containerIDPattern.MatchString(created.ID) {
		return errors.New("Docker returned an invalid Certbot container identity")
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, _ = c.appDockerRequest(
			cleanupCtx, http.MethodDelete, "/containers/"+created.ID+"?v=0&force=1",
			nil, 64<<10, c.httpClient,
		)
	}()
	if err := c.post(ctx, "/containers/"+created.ID+"/start"); err != nil {
		return err
	}
	waitClient := *c.httpClient
	waitClient.Timeout = 6 * time.Minute
	waitData, err := c.appDockerRequest(
		ctx, http.MethodPost, "/containers/"+created.ID+"/wait?condition=not-running",
		nil, 64<<10, &waitClient,
	)
	if err != nil {
		return err
	}
	var waitResult struct {
		StatusCode int `json:"StatusCode"`
	}
	if err := json.Unmarshal(waitData, &waitResult); err != nil {
		return err
	}
	if waitResult.StatusCode != 0 {
		return errors.New("certificate authority rejected the WordPress certificate request")
	}
	return nil
}

func ensureWordPressDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(path, 0o750); err != nil {
			return err
		}
		info, err = os.Lstat(path)
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("certificate directory %s is unavailable or unsafe", path)
	}
	return nil
}

func wordPressCertificateSources(domain string) (string, string, error) {
	root := filepath.Clean("/etc/letsencrypt")
	live := filepath.Join(root, "live", domain)
	cert, err := safeResolvedCertificatePath(filepath.Join(live, "fullchain.pem"), root)
	if err != nil {
		return "", "", err
	}
	key, err := safeResolvedCertificatePath(filepath.Join(live, "privkey.pem"), root)
	if err != nil {
		return "", "", err
	}
	return cert, key, nil
}

func safeResolvedCertificatePath(path, root string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	resolved = filepath.Clean(resolved)
	root = filepath.Clean(root)
	if resolved == root || !strings.HasPrefix(resolved, root+string(filepath.Separator)) {
		return "", errors.New("certificate source escaped /etc/letsencrypt")
	}
	info, err := os.Lstat(resolved)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("certificate source is not a regular file")
	}
	return resolved, nil
}

func readWordPressCertificateFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
		info.Size() < 1 || info.Size() > limit {
		return nil, errors.New("certificate artifact is unavailable or unsafe")
	}
	return os.ReadFile(path)
}

func validateWordPressKeyPair(domain string, certificate, key []byte) error {
	pair, err := tls.X509KeyPair(certificate, key)
	if err != nil || len(pair.Certificate) == 0 {
		return errors.New("certificate and private key do not form a valid pair")
	}
	parsed, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return err
	}
	now := time.Now()
	if now.Before(parsed.NotBefore) || !now.Before(parsed.NotAfter) {
		return errors.New("certificate is not currently valid")
	}
	if err := parsed.VerifyHostname(domain); err != nil {
		return errors.New("certificate does not cover the WordPress domain")
	}
	return nil
}

func publishCertificateFile(destination string, content []byte, mode os.FileMode) error {
	directory := filepath.Dir(destination)
	file, err := os.CreateTemp(directory, "."+filepath.Base(destination)+".kp-*")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Link(temp, destination); err != nil {
		return err
	}
	return nil
}

func removeMatchingCertificateFile(path string, expected []byte) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("certificate rollback target is unsafe")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(content, expected) {
		return errors.New("certificate rollback target changed externally")
	}
	return os.Remove(path)
}
