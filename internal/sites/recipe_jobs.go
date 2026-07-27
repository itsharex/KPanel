package sites

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

const maxRecipeJobBytes = 128 << 10

var (
	recipeJobIDPattern  = regexp.MustCompile(`^[a-f0-9]{32}$`)
	recipeProgressLine  = regexp.MustCompile(`^KPANEL_PROGRESS ([0-9]{1,3}) (.+)$`)
	recipeScriptLicense = regexp.MustCompile(`(?m)^permission_granted="true"\r?$`)
	recipeSelectors     = map[string]string{
		"discuz":    "3",
		"kodbox":    "4",
		"maccms":    "5",
		"dujiaoka":  "6",
		"flarum":    "7",
		"typecho":   "8",
		"linkstack": "9",
		"ai-prompt": "27",
	}
)

type RecipeJob struct {
	ID         string                `json:"id"`
	Domain     string                `json:"domain"`
	Recipe     string                `json:"recipe"`
	Status     string                `json:"status"`
	Stage      string                `json:"stage"`
	Progress   int                   `json:"progress"`
	Message    string                `json:"message,omitempty"`
	Site       *contract.SiteSummary `json:"site,omitempty"`
	CreatedAt  time.Time             `json:"createdAt"`
	StartedAt  *time.Time            `json:"startedAt,omitempty"`
	FinishedAt *time.Time            `json:"finishedAt,omitempty"`
}

type recipeJobRegistry struct {
	mu       sync.Mutex
	stateDir string
	jobs     map[string]RecipeJob
}

type scriptSiteInvocation struct {
	arguments   []string
	environment []string
	required    []string
	timeout     time.Duration
}

func newRecipeJobRegistry(stateDir string) *recipeJobRegistry {
	return &recipeJobRegistry{stateDir: stateDir, jobs: make(map[string]RecipeJob)}
}

func (m *Manager) ConfigureRecipeJobState(stateDir string) error {
	stateDir = filepath.Clean(stateDir)
	if !filepath.IsAbs(stateDir) || stateDir == string(filepath.Separator) {
		return errors.New("site recipe jobs require a dedicated absolute directory")
	}
	if err := ensureRecipeJobDirectory(stateDir); err != nil {
		return err
	}
	registry := newRecipeJobRegistry(stateDir)
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		job, readErr := registry.read(id)
		if readErr != nil {
			continue
		}
		if job.Status == "queued" || job.Status == "running" {
			finished := time.Now().UTC()
			job.Status = "failed"
			job.Stage = "interrupted"
			job.Progress = 100
			job.Message = "一键建站任务被 Agent 或服务器重启中断，请先核对实际站点产物"
			job.FinishedAt = &finished
			_ = registry.put(job)
		}
		registry.jobs[id] = job
	}
	m.recipeJobs = registry
	return nil
}

func ensureRecipeJobDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(path, 0o750); err != nil {
			return err
		}
		info, err = os.Lstat(path)
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("site recipe job directory is unavailable or unsafe")
	}
	return nil
}

func (m *Manager) RecipeWritable() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("%w: kejilion.sh recipes require Linux", ErrUnavailable)
	}
	if m.recipeJobs == nil || m.recipeJobs.stateDir == "" {
		return fmt.Errorf("%w: recipe background jobs are unavailable", ErrUnavailable)
	}
	if _, err := findSystemdRun(); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	_, err := findRecipeScript()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}

func (m *Manager) WordPressWritable(_ context.Context) error {
	return m.directSiteWritable(
		"KJ_WEB_NONINTERACTIVE",
		"KJ_WEB_RECIPE",
		"KJ_WEB_DOMAIN",
		`ldnmp_wp "${KJ_WEB_DOMAIN:-}"`,
	)
}

func (m *Manager) ProxyWritable() error {
	return m.directSiteWritable(
		"KJ_WEB_NONINTERACTIVE",
		"KJ_WEB_RECIPE",
		"KJ_WEB_DOMAIN",
		"KJ_WEB_PROXY_HOST",
		"KJ_WEB_PROXY_PORT",
		`ldnmp_Proxy "${KJ_WEB_DOMAIN:-}" "${KJ_WEB_PROXY_HOST:-}" "${KJ_WEB_PROXY_PORT:-}"`,
	)
}

func (m *Manager) directSiteWritable(required ...string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("%w: kejilion.sh website commands require Linux", ErrUnavailable)
	}
	if m.recipeJobs == nil || m.recipeJobs.stateDir == "" {
		return fmt.Errorf("%w: website background jobs are unavailable", ErrUnavailable)
	}
	if _, err := findSystemdRun(); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if _, err := findTrustedKejilionScript(required...); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}

func (m *Manager) StartRecipe(_ context.Context, input SiteInput) (RecipeJob, error) {
	domain, selector, err := normalizeRecipeInput(input)
	if err != nil {
		return RecipeJob{}, err
	}
	if err := m.RecipeWritable(); err != nil {
		return RecipeJob{}, err
	}
	siteWriteMutex.Lock()
	defer siteWriteMutex.Unlock()
	if err := m.checkCollisions(managedSpec{Primary: domain, Kind: contract.SitePHP}, ""); err != nil {
		return RecipeJob{}, err
	}
	if m.recipeJobs.hasActive() {
		return RecipeJob{}, fmt.Errorf("%w: another one-click website task is running", ErrConflict)
	}
	var identity [16]byte
	if _, err := rand.Read(identity[:]); err != nil {
		return RecipeJob{}, err
	}
	job := RecipeJob{
		ID: hex.EncodeToString(identity[:]), Domain: domain, Recipe: input.Recipe,
		Status: "queued", Stage: "queued", Progress: 0,
		Message: "一键建站任务已进入后台队列", CreatedAt: time.Now().UTC(),
	}
	if err := m.recipeJobs.put(job); err != nil {
		return RecipeJob{}, fmt.Errorf("%w: persist recipe job: %v", ErrNeedsAttention, err)
	}
	go m.runRecipeJob(job.ID, scriptSiteInvocation{
		arguments:   []string{"web"},
		environment: []string{"KJ_WEB_NONINTERACTIVE=1", "KJ_WEB_RECIPE=" + selector, "KJ_WEB_DOMAIN=" + domain},
		required:    []string{"KJ_WEB_NONINTERACTIVE", "KJ_WEB_RECIPE", "KJ_WEB_DOMAIN"},
		timeout:     60 * time.Minute,
	})
	return job, nil
}

func (m *Manager) StartWordPress(ctx context.Context, input SiteInput) (RecipeJob, error) {
	spec, err := normalizeWordPressInput(input)
	if err != nil {
		return RecipeJob{}, err
	}
	if err := m.WordPressWritable(ctx); err != nil {
		return RecipeJob{}, err
	}
	return m.startDirectSiteJob(
		spec,
		"wordpress",
		wordPressInvocation(spec.Primary),
	)
}

func (m *Manager) StartProxy(input SiteInput) (RecipeJob, error) {
	spec, host, port, err := normalizeDirectProxyInput(input)
	if err != nil {
		return RecipeJob{}, err
	}
	if err := m.ProxyWritable(); err != nil {
		return RecipeJob{}, err
	}
	return m.startDirectSiteJob(
		spec,
		"reverse-proxy",
		proxyInvocation(spec.Primary, host, port),
	)
}

func wordPressInvocation(domain string) scriptSiteInvocation {
	return scriptSiteInvocation{
		arguments:   []string{"web"},
		environment: []string{"KJ_WEB_NONINTERACTIVE=1", "KJ_WEB_RECIPE=2", "KJ_WEB_DOMAIN=" + domain},
		required: []string{
			"KJ_WEB_NONINTERACTIVE",
			"KJ_WEB_RECIPE",
			"KJ_WEB_DOMAIN",
			`ldnmp_wp "${KJ_WEB_DOMAIN:-}"`,
		},
		timeout: 60 * time.Minute,
	}
}

func proxyInvocation(domain, host, port string) scriptSiteInvocation {
	return scriptSiteInvocation{
		arguments: []string{"web"},
		environment: []string{
			"KJ_WEB_NONINTERACTIVE=1",
			"KJ_WEB_RECIPE=23",
			"KJ_WEB_DOMAIN=" + domain,
			"KJ_WEB_PROXY_HOST=" + host,
			"KJ_WEB_PROXY_PORT=" + port,
		},
		required: []string{
			"KJ_WEB_NONINTERACTIVE",
			"KJ_WEB_RECIPE",
			"KJ_WEB_DOMAIN",
			"KJ_WEB_PROXY_HOST",
			"KJ_WEB_PROXY_PORT",
			`ldnmp_Proxy "${KJ_WEB_DOMAIN:-}" "${KJ_WEB_PROXY_HOST:-}" "${KJ_WEB_PROXY_PORT:-}"`,
		},
		timeout: 45 * time.Minute,
	}
}

func (m *Manager) startDirectSiteJob(
	spec managedSpec,
	recipe string,
	invocation scriptSiteInvocation,
) (RecipeJob, error) {
	siteWriteMutex.Lock()
	defer siteWriteMutex.Unlock()
	if err := m.checkCollisions(spec, ""); err != nil {
		return RecipeJob{}, err
	}
	if m.recipeJobs.hasActive() {
		return RecipeJob{}, fmt.Errorf("%w: another kejilion.sh website task is running", ErrConflict)
	}
	var identity [16]byte
	if _, err := rand.Read(identity[:]); err != nil {
		return RecipeJob{}, err
	}
	job := RecipeJob{
		ID: hex.EncodeToString(identity[:]), Domain: spec.Primary, Recipe: recipe,
		Status: "queued", Stage: "queued", Progress: 0,
		Message: "kejilion.sh 原生建站任务已进入后台队列", CreatedAt: time.Now().UTC(),
	}
	if err := m.recipeJobs.put(job); err != nil {
		return RecipeJob{}, fmt.Errorf("%w: persist website job: %v", ErrNeedsAttention, err)
	}
	go m.runRecipeJob(job.ID, invocation)
	return job, nil
}

func normalizeRecipeInput(input SiteInput) (string, string, error) {
	if input.Type != "recipe" || input.Recipe == "" {
		return "", "", fmt.Errorf("%w: a one-click recipe is required", ErrInvalidInput)
	}
	selector, ok := recipeSelectors[input.Recipe]
	if !ok {
		return "", "", fmt.Errorf("%w: unsupported one-click recipe", ErrUnprocessable)
	}
	domain, err := normalizeFQDN(input.PrimaryDomain)
	if err != nil {
		return "", "", err
	}
	if len(input.Aliases) > 0 || input.Upstream != "" || len(input.Upstreams) > 0 ||
		input.RedirectTarget != "" || input.RedirectCode != 0 || input.PHPVersion != "" ||
		input.ExpectedResourceVersion != "" || (input.Enabled != nil && !*input.Enabled) {
		return "", "", fmt.Errorf("%w: recipe does not accept generic site settings", ErrUnprocessable)
	}
	return domain, selector, nil
}

func normalizeWordPressInput(input SiteInput) (managedSpec, error) {
	primary, err := normalizeFQDN(input.PrimaryDomain)
	if err != nil {
		return managedSpec{}, err
	}
	if input.Type != "wordpress" {
		return managedSpec{}, fmt.Errorf("%w: WordPress type is required", ErrInvalidInput)
	}
	if input.Enabled != nil && !*input.Enabled {
		return managedSpec{}, fmt.Errorf("%w: disabling WordPress is not supported", ErrUnprocessable)
	}
	if input.ExpectedResourceVersion != "" {
		return managedSpec{}, fmt.Errorf("%w: expectedResourceVersion is not valid for create", ErrInvalidInput)
	}
	if len(input.Aliases) != 0 || strings.TrimSpace(input.Upstream) != "" ||
		len(input.Upstreams) != 0 || strings.TrimSpace(input.RedirectTarget) != "" ||
		input.RedirectCode != 0 || strings.TrimSpace(input.PHPVersion) != "" ||
		input.Recipe != "" {
		return managedSpec{}, fmt.Errorf(
			"%w: kejilion.sh WordPress accepts only one primary domain",
			ErrUnprocessable,
		)
	}
	return managedSpec{Primary: primary, Kind: contract.SiteWordPress}, nil
}

func normalizeDirectProxyInput(input SiteInput) (managedSpec, string, string, error) {
	spec, err := normalizeSiteInput(input)
	if err != nil {
		return managedSpec{}, "", "", err
	}
	if input.Type != "proxy" || spec.Kind != contract.SiteReverseProxy {
		return managedSpec{}, "", "", fmt.Errorf("%w: IP and port proxy type is required", ErrInvalidInput)
	}
	if input.ExpectedResourceVersion != "" {
		return managedSpec{}, "", "", fmt.Errorf(
			"%w: expectedResourceVersion is not valid for create",
			ErrInvalidInput,
		)
	}
	if len(spec.Aliases) != 0 {
		return managedSpec{}, "", "", fmt.Errorf(
			"%w: kejilion.sh IP and port proxy accepts one primary domain",
			ErrUnprocessable,
		)
	}
	parsed, err := url.Parse(spec.Upstream)
	if err != nil || parsed.Scheme != "http" {
		return managedSpec{}, "", "", fmt.Errorf(
			"%w: kejilion.sh IP and port proxy requires an http upstream",
			ErrUnprocessable,
		)
	}
	host, port := parsed.Hostname(), parsed.Port()
	if host == "" || port == "" || strings.Contains(host, ":") {
		return managedSpec{}, "", "", fmt.Errorf(
			"%w: kejilion.sh IP and port proxy requires an IPv4 or hostname with an explicit port",
			ErrUnprocessable,
		)
	}
	return spec, host, port, nil
}

func (m *Manager) RecipeJob(id string) (RecipeJob, error) {
	if m.recipeJobs == nil {
		return RecipeJob{}, ErrConflict
	}
	return m.recipeJobs.read(id)
}

func (m *Manager) InstallationJob(id string) (any, error) {
	if job, err := m.RecipeJob(id); err == nil {
		return job, nil
	}
	return nil, ErrConflict
}

func (m *Manager) runRecipeJob(id string, invocation scriptSiteInvocation) {
	job, err := m.recipeJobs.read(id)
	if err != nil {
		return
	}
	script, err := findTrustedKejilionScript(invocation.required...)
	if err != nil {
		m.failRecipeJob(job, "script_unavailable", err)
		return
	}
	systemdRun, err := findSystemdRun()
	if err != nil {
		m.failRecipeJob(job, "runner_unavailable", err)
		return
	}
	started := time.Now().UTC()
	job.Status = "running"
	job.Stage = "preflight"
	job.Progress = 1
	job.Message = "正在启动 kejilion.sh 原生一键建站流程"
	job.StartedAt = &started
	if m.recipeJobs.put(job) != nil {
		return
	}

	timeout := invocation.timeout
	if timeout <= 0 {
		timeout = 60 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(
		ctx,
		systemdRun,
		systemdSiteArguments(job, script, invocation)...,
	)
	command.Env = os.Environ()
	output, err := command.StdoutPipe()
	if err != nil {
		m.failRecipeJob(job, "start_failed", err)
		return
	}
	command.Stderr = command.Stdout
	if err := command.Start(); err != nil {
		m.failRecipeJob(job, "start_failed", err)
		return
	}
	scanner := bufio.NewScanner(output)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	outputLines := 0
	for scanner.Scan() {
		matches := recipeProgressLine.FindStringSubmatch(scanner.Text())
		if len(matches) == 3 {
			progress, _ := strconv.Atoi(matches[1])
			job.Progress = min(max(progress, 0), 100)
			job.Stage = recipeJobStage(job.Progress)
			job.Message = safeRecipeMessage(matches[2])
			_ = m.recipeJobs.put(job)
			continue
		}
		outputLines++
		if outputLines%8 == 0 && job.Progress < 88 {
			job.Progress = min(job.Progress+3, 88)
			job.Stage = "installing"
			job.Message = fmt.Sprintf(
				"kejilion.sh 原生%s流程正在执行（已处理 %d 行输出）",
				scriptSiteLabel(job.Recipe),
				outputLines,
			)
			_ = m.recipeJobs.put(job)
		}
	}
	waitErr := command.Wait()
	if scanErr := scanner.Err(); scanErr != nil && waitErr == nil {
		waitErr = scanErr
	}
	if waitErr != nil {
		m.failRecipeJob(job, "failed", waitErr)
		return
	}
	items, err := m.discoverer.Discover()
	if err != nil {
		m.failRecipeJob(job, "reconcile_failed", err)
		return
	}
	for index := range items {
		if items[index].PrimaryDomain == job.Domain {
			job.Site = &items[index]
			break
		}
	}
	if job.Site == nil {
		m.failRecipeJob(job, "reconcile_failed", errors.New("site artifacts were not discovered"))
		return
	}
	finished := time.Now().UTC()
	job.Status = "succeeded"
	job.Stage = "completed"
	job.Progress = 100
	job.Message = "kejilion.sh 原生" + scriptSiteLabel(job.Recipe) + "产物已完成对账"
	job.FinishedAt = &finished
	_ = m.recipeJobs.put(job)
}

func systemdSiteArguments(
	job RecipeJob,
	script string,
	invocation scriptSiteInvocation,
) []string {
	timeout := invocation.timeout
	if timeout <= 0 {
		timeout = 60 * time.Minute
	}
	arguments := []string{
		"--unit=kpanel-site-" + job.ID,
		"--wait",
		"--pipe",
		"--collect",
		"--quiet",
		"--property=Type=exec",
		"--property=TimeoutStartSec=" + strconv.FormatInt(int64(timeout.Seconds()), 10) + "s",
		"--property=TimeoutStopSec=30s",
		"--property=User=root",
		"--property=UMask=0027",
		"--property=NoNewPrivileges=no",
		"--property=ProtectSystem=no",
		"--property=ProtectHome=no",
		"--property=PrivateTmp=no",
		"--property=PrivateDevices=no",
		"--property=RestrictNamespaces=no",
		"--property=RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6 AF_NETLINK",
		"--setenv=LC_ALL=C.UTF-8",
		"--setenv=LANG=C.UTF-8",
	}
	for _, value := range invocation.environment {
		arguments = append(arguments, "--setenv="+value)
	}
	arguments = append(arguments, "--", "/bin/bash", script)
	return append(arguments, invocation.arguments...)
}

func scriptSiteLabel(recipe string) string {
	switch recipe {
	case "wordpress":
		return " WordPress 建站"
	case "reverse-proxy":
		return " IP+端口反向代理"
	default:
		return "一键建站"
	}
}

func (m *Manager) failRecipeJob(job RecipeJob, stage string, cause error) {
	finished := time.Now().UTC()
	job.Status = "failed"
	job.Stage = stage
	job.Progress = 100
	job.Message = "一键建站失败；未展示脚本原始输出，以免泄露数据库凭证。请核对域名、证书和任务产物"
	if errors.Is(cause, context.DeadlineExceeded) {
		job.Message = "一键建站超过 60 分钟并已终止，请核对实际产物"
	}
	job.FinishedAt = &finished
	_ = m.recipeJobs.put(job)
}

func findRecipeScript() (string, error) {
	return findTrustedKejilionScript(
		"KJ_WEB_NONINTERACTIVE",
		"KJ_WEB_RECIPE",
		"KJ_WEB_DOMAIN",
	)
}

func findTrustedKejilionScript(required ...string) (string, error) {
	candidates := []string{"/usr/local/bin/k", "/usr/bin/k", "/root/kejilion.sh"}
	if path, err := exec.LookPath("k"); err == nil {
		candidates = append([]string{path}, candidates...)
	}
	seen := make(map[string]bool)
	for _, candidate := range candidates {
		resolved, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			continue
		}
		resolved = filepath.Clean(resolved)
		if seen[resolved] {
			continue
		}
		seen[resolved] = true
		info, err := os.Stat(resolved)
		if err != nil || !info.Mode().IsRegular() || info.Size() < 1024 || info.Size() > 4<<20 ||
			info.Mode().Perm()&0o022 != 0 || !recipeScriptOwnerTrusted(info) {
			continue
		}
		content, err := os.ReadFile(resolved)
		if err != nil {
			continue
		}
		value := string(content)
		if recipeScriptLicense.Match(content) && containsAll(value, required) {
			return resolved, nil
		}
	}
	return "", errors.New("a trusted kejilion.sh website command was not found")
}

func findSystemdRun() (string, error) {
	for _, candidate := range []string{"/usr/bin/systemd-run", "/bin/systemd-run"} {
		info, err := os.Stat(candidate)
		if err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o022 == 0 &&
			recipeScriptOwnerTrusted(info) {
			return candidate, nil
		}
	}
	return "", errors.New("trusted systemd-run is unavailable")
}

func containsAll(value string, required []string) bool {
	for _, token := range required {
		if token == "" || !strings.Contains(value, token) {
			return false
		}
	}
	return true
}

func recipeJobStage(progress int) string {
	switch {
	case progress < 10:
		return "preflight"
	case progress < 90:
		return "installing"
	case progress < 100:
		return "reconciling"
	default:
		return "completed"
	}
}

func safeRecipeMessage(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 300 {
		value = value[:300]
	}
	return value
}

func (registry *recipeJobRegistry) hasActive() bool {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	for _, job := range registry.jobs {
		if job.Status == "queued" || job.Status == "running" {
			return true
		}
	}
	return false
}

func (registry *recipeJobRegistry) put(job RecipeJob) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if !recipeJobIDPattern.MatchString(job.ID) {
		return errors.New("invalid recipe job identity")
	}
	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if len(data) > maxRecipeJobBytes {
		return errors.New("recipe job state exceeds the safety limit")
	}
	temp, err := os.CreateTemp(registry.stateDir, "."+job.ID+".*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
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
	target := registry.path(job.ID)
	if runtime.GOOS == "windows" {
		_ = os.Remove(target)
	}
	if err := os.Rename(tempPath, target); err != nil {
		return err
	}
	registry.jobs[job.ID] = job
	registry.pruneLocked()
	return nil
}

func (registry *recipeJobRegistry) read(id string) (RecipeJob, error) {
	if !recipeJobIDPattern.MatchString(id) {
		return RecipeJob{}, ErrConflict
	}
	info, err := os.Lstat(registry.path(id))
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
		info.Size() <= 0 || info.Size() > maxRecipeJobBytes {
		return RecipeJob{}, ErrConflict
	}
	data, err := os.ReadFile(registry.path(id))
	if err != nil {
		return RecipeJob{}, err
	}
	var job RecipeJob
	if json.Unmarshal(data, &job) != nil || job.ID != id {
		return RecipeJob{}, ErrConflict
	}
	return job, nil
}

func (registry *recipeJobRegistry) path(id string) string {
	return filepath.Join(registry.stateDir, id+".json")
}

func (registry *recipeJobRegistry) pruneLocked() {
	jobs := make([]RecipeJob, 0, len(registry.jobs))
	for _, job := range registry.jobs {
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
	})
	if len(jobs) <= 100 {
		return
	}
	for _, job := range jobs[100:] {
		if job.Status == "queued" || job.Status == "running" {
			continue
		}
		delete(registry.jobs, job.ID)
		_ = os.Remove(registry.path(job.ID))
	}
}
