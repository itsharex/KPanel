package agent

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kejilion/kejilion-panel/internal/appmarket"
	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/dockerx"
	"github.com/kejilion/kejilion-panel/internal/sites"
	"github.com/kejilion/kejilion-panel/internal/systeminfo"
	"github.com/kejilion/kejilion-panel/internal/systemmanage"
)

const maxAgentRequestBytes = 64 << 10

type Config struct {
	Token           []byte
	Version         string
	ProtocolVersion string
	WebRoot         string
	StateDir        string
	System          *systeminfo.Collector
	SystemManager   *systemmanage.Manager
	Sites           *sites.Discoverer
	SitesManager    *sites.Manager
	Docker          *dockerx.Client
	AppMarket       *appmarket.Service
	Now             func() time.Time
}

type Server struct {
	tokenHash       [32]byte
	version         string
	protocolVersion string
	webRoot         string
	system          *systeminfo.Collector
	systemManager   *systemmanage.Manager
	sites           *sites.Discoverer
	sitesManager    *sites.Manager
	docker          *dockerx.Client
	appMarket       *appmarket.Service
	now             func() time.Time
}

func NewServer(config Config) (*Server, error) {
	if len(config.Token) < 32 {
		return nil, errors.New("Agent token must contain at least 32 bytes")
	}
	if config.Version == "" || config.ProtocolVersion == "" {
		return nil, errors.New("Agent version and protocol version are required")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.System == nil {
		config.System = systeminfo.NewCollector()
	}
	if config.SystemManager == nil {
		config.SystemManager = systemmanage.NewManager(systemmanage.Config{Enabled: false})
	}
	if config.Sites == nil {
		config.Sites = sites.NewDiscoverer(config.WebRoot)
	}
	if config.Docker == nil {
		return nil, errors.New("Docker client is required")
	}
	if config.StateDir != "" {
		if err := config.Docker.ConfigureJobs(filepath.Join(config.StateDir, "docker-jobs")); err != nil {
			return nil, fmt.Errorf("initialize Docker job state: %w", err)
		}
	}
	if config.SitesManager == nil {
		config.SitesManager = sites.NewManager(config.WebRoot, config.Sites, config.Docker)
		if config.StateDir != "" {
			if err := config.SitesManager.ConfigureWordPressJobState(
				filepath.Join(config.StateDir, "wordpress-jobs"),
			); err != nil {
				return nil, fmt.Errorf("initialize WordPress job state: %w", err)
			}
			if err := config.SitesManager.ConfigureRecipeJobState(
				filepath.Join(config.StateDir, "site-recipe-jobs"),
			); err != nil {
				return nil, fmt.Errorf("initialize site recipe job state: %w", err)
			}
		}
	}
	if config.AppMarket == nil {
		var err error
		config.AppMarket, err = appmarket.New(config.Docker, "/home/docker")
		if err != nil {
			return nil, fmt.Errorf("initialize application market: %w", err)
		}
	}
	return &Server{
		tokenHash:       sha256.Sum256(config.Token),
		version:         config.Version,
		protocolVersion: config.ProtocolVersion,
		webRoot:         config.WebRoot,
		system:          config.System,
		systemManager:   config.SystemManager,
		sites:           config.Sites,
		sitesManager:    config.SitesManager,
		docker:          config.Docker,
		appMarket:       config.AppMarket,
		now:             config.Now,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := requestID()
	w.Header().Set("X-Request-ID", requestID)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
	if !s.authorized(r) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="kejilion-agent"`)
		writeProblem(w, requestID, http.StatusUnauthorized, "agent_unauthorized", "认证失败", "")
		return
	}

	switch {
	case r.URL.Path == "/v1/health":
		s.requireMethod(w, r, requestID, http.MethodGet, s.health)
	case r.URL.Path == "/v1/capabilities":
		s.requireMethod(w, r, requestID, http.MethodGet, s.capabilities)
	case r.URL.Path == "/v1/system/summary":
		s.requireMethod(w, r, requestID, http.MethodGet, s.systemSummary)
	case r.URL.Path == "/v1/system/public-network":
		s.requireMethod(w, r, requestID, http.MethodGet, s.publicNetwork)
	case r.URL.Path == "/v1/system/actions":
		s.requireMethod(w, r, requestID, http.MethodPost, s.systemAction)
	case r.URL.Path == "/v1/sites":
		s.siteCollection(w, r, requestID)
	case strings.HasPrefix(r.URL.Path, "/v1/site-installations/"):
		s.siteInstallation(w, r, requestID)
	case strings.HasPrefix(r.URL.Path, "/v1/sites/"):
		s.siteOperation(w, r, requestID)
	case r.URL.Path == "/v1/apps":
		s.requireMethod(w, r, requestID, http.MethodGet, s.appList)
	case r.URL.Path == "/v1/app-jobs":
		s.requireMethod(w, r, requestID, http.MethodGet, s.appJobList)
	case strings.HasPrefix(r.URL.Path, "/v1/app-jobs/"):
		s.requireMethod(w, r, requestID, http.MethodGet, s.appJob)
	case strings.HasPrefix(r.URL.Path, "/v1/apps/"):
		s.appOperation(w, r, requestID)
	case r.URL.Path == "/v1/docker/summary":
		s.requireMethod(w, r, requestID, http.MethodGet, s.dockerSummary)
	case r.URL.Path == "/v1/docker/environment":
		s.requireMethod(w, r, requestID, http.MethodGet, s.dockerEnvironment)
	case r.URL.Path == "/v1/docker/containers":
		s.requireMethod(w, r, requestID, http.MethodGet, s.containerList)
	case r.URL.Path == "/v1/docker/images":
		s.requireMethod(w, r, requestID, http.MethodGet, s.imageList)
	case r.URL.Path == "/v1/docker/networks":
		s.requireMethod(w, r, requestID, http.MethodGet, s.networkList)
	case r.URL.Path == "/v1/docker/volumes":
		s.requireMethod(w, r, requestID, http.MethodGet, s.volumeList)
	case r.URL.Path == "/v1/docker/backups":
		s.requireMethod(w, r, requestID, http.MethodGet, s.dockerBackupList)
	case r.URL.Path == "/v1/docker/jobs":
		s.requireMethod(w, r, requestID, http.MethodGet, s.dockerJobList)
	case strings.HasPrefix(r.URL.Path, "/v1/docker/jobs/"):
		s.requireMethod(w, r, requestID, http.MethodGet, s.dockerJob)
	case r.URL.Path == "/v1/docker/tasks":
		s.requireMethod(w, r, requestID, http.MethodPost, s.dockerTask)
	case strings.HasPrefix(r.URL.Path, "/v1/docker/containers/"):
		s.containerOperation(w, r, requestID)
	default:
		writeProblem(w, requestID, http.StatusNotFound, "not_found", "资源不存在", "")
	}
}

func (s *Server) authorized(r *http.Request) bool {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") || strings.Contains(header, ",") {
		return false
	}
	token := strings.TrimPrefix(header, "Bearer ")
	if token == "" || strings.TrimSpace(token) != token {
		return false
	}
	candidate := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(candidate[:], s.tokenHash[:]) == 1
}

func (s *Server) requireMethod(w http.ResponseWriter, r *http.Request, requestID, method string, next http.HandlerFunc) {
	if r.Method != method {
		w.Header().Set("Allow", method)
		writeProblem(w, requestID, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不允许", "")
		return
	}
	next(w, r)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 800*time.Millisecond)
	defer cancel()
	var reasons []string
	if err := availableWebRoot(s.webRoot); err != nil {
		reasons = append(reasons, "web_root_unavailable")
	}
	if err := s.docker.Ping(ctx); err != nil {
		reasons = append(reasons, "docker_unavailable")
	}
	status := "ok"
	if len(reasons) > 0 {
		status = "degraded"
	}
	writeJSON(w, http.StatusOK, contract.AgentHealth{
		Status: status, Version: s.version, ProtocolVersion: s.protocolVersion,
		ReadOnly: false, Reasons: reasons, CheckedAt: s.now().UTC(),
	})
}

func (s *Server) capabilities(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var (
		dockerAvailable   bool
		siteErr           error
		siteWriteErr      error
		wordPressWriteErr error
		recipeWriteErr    error
	)
	var checks sync.WaitGroup
	checks.Add(5)
	go func() {
		defer checks.Done()
		pingContext, pingCancel := context.WithTimeout(ctx, 800*time.Millisecond)
		defer pingCancel()
		dockerAvailable = s.docker.Ping(pingContext) == nil
	}()
	go func() {
		defer checks.Done()
		siteErr = availableWebRoot(s.webRoot)
	}()
	go func() {
		defer checks.Done()
		siteWriteErr = s.sitesManager.Writable(ctx)
	}()
	go func() {
		defer checks.Done()
		wordPressWriteErr = s.sitesManager.WordPressWritable(ctx)
	}()
	go func() {
		defer checks.Done()
		recipeWriteErr = s.sitesManager.RecipeWritable()
	}()
	checks.Wait()
	items := []contract.Capability{
		{ID: "system.read", Enabled: true, Methods: []string{"GET"}},
		{ID: "apps.read", Enabled: dockerAvailable, Reason: reasonUnless(dockerAvailable, "Docker Engine 不可用"), Methods: []string{"GET"}},
		{ID: "apps.lifecycle", Enabled: dockerAvailable, Reason: reasonUnless(dockerAvailable, "Docker Engine 不可用"), Methods: []string{"POST"}},
		{ID: "apps.install", Enabled: dockerAvailable, Reason: reasonUnless(dockerAvailable, "Docker Engine 不可用"), Methods: []string{"POST"}},
		{ID: "sites.read", Enabled: siteErr == nil, Reason: reasonIf(siteErr, "Kejilion Web 根目录不可用"), Methods: []string{"GET"}},
		{ID: "docker.read", Enabled: dockerAvailable, Reason: reasonUnless(dockerAvailable, "Docker Engine 不可用"), Methods: []string{"GET"}},
		{ID: "docker.logs", Enabled: dockerAvailable, Reason: reasonUnless(dockerAvailable, "Docker Engine 不可用"), Methods: []string{"GET"}},
		{ID: "docker.lifecycle", Enabled: dockerAvailable, Reason: reasonUnless(dockerAvailable, "Docker Engine 不可用"), Methods: []string{"POST"}},
		{ID: "docker.exec", Enabled: dockerAvailable, Reason: reasonUnless(dockerAvailable, "Docker Engine 不可用"), Methods: []string{"POST"}},
		{ID: "docker.maintenance", Enabled: dockerAvailable, Reason: reasonUnless(dockerAvailable, "Docker Engine 不可用"), Methods: []string{"GET", "POST"}},
		{ID: "sites.write", Enabled: siteWriteErr == nil, Reason: reasonIf(siteWriteErr, "网站写入依赖未就绪"), Methods: []string{"POST", "PATCH"}},
		{ID: "sites.wordpress.install", Enabled: wordPressWriteErr == nil, Reason: reasonIf(wordPressWriteErr, "WordPress 一键搭建条件不满足"), Methods: []string{"POST"}},
		{ID: "sites.recipes.install", Enabled: recipeWriteErr == nil, Reason: reasonIf(recipeWriteErr, "kejilion.sh 一键建站协议不可用"), Methods: []string{"POST"}},
	}
	items = append(items, s.systemManager.Capabilities()...)
	writeJSON(w, http.StatusOK, contract.PageResult[contract.Capability]{Items: items})
}

func availableWebRoot(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("web root is not a directory")
	}
	return nil
}

func (s *Server) systemSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.system.Collect(r.Context())
	if err != nil && summary.Hostname == "" {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "system_unavailable", "系统状态不可用", "")
		return
	}
	summary.Management.Maintenance = s.systemManager.MaintenanceStatus()
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) publicNetwork(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3500*time.Millisecond)
	defer cancel()
	writeJSON(w, http.StatusOK, s.system.PublicNetwork(ctx))
}

func (s *Server) systemAction(w http.ResponseWriter, r *http.Request) {
	requestID := requestIDFrom(w)
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_system_action", "系统操作 URL 无效", "")
		return
	}
	var input contract.SystemActionRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_request", "请求格式无效", "")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	result, err := s.systemManager.Execute(ctx, input)
	if err != nil {
		status, code, title := http.StatusServiceUnavailable, "system_action_failed", "系统操作失败"
		switch {
		case errors.Is(err, systemmanage.ErrInvalidInput):
			status, code, title = http.StatusUnprocessableEntity, "invalid_system_action", "系统操作参数无效"
		case errors.Is(err, systemmanage.ErrDisabled), errors.Is(err, systemmanage.ErrUnsupported):
			status, code, title = http.StatusForbidden, "system_action_unavailable", "系统操作不可用"
		case errors.Is(err, systemmanage.ErrConflict):
			status, code, title = http.StatusConflict, "system_action_conflict", "系统配置发生冲突"
		case errors.Is(err, systemmanage.ErrRolledBack):
			status, code, title = http.StatusUnprocessableEntity, "system_action_rolled_back", "系统操作失败并已回滚"
		case errors.Is(err, systemmanage.ErrNeedsAttention):
			status, code, title = http.StatusServiceUnavailable, "system_action_needs_attention", "系统操作需要人工检查"
		}
		writeProblem(w, requestID, status, code, title, safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) siteList(w http.ResponseWriter, _ *http.Request) {
	items, err := s.sites.Discover()
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "sites_unavailable", "网站状态不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, contract.PageResult[contract.SiteSummary]{Items: items})
}

func (s *Server) siteCollection(w http.ResponseWriter, r *http.Request, requestID string) {
	switch r.Method {
	case http.MethodGet:
		s.siteList(w, r)
	case http.MethodPost:
		if r.URL.RawPath != "" || r.URL.RawQuery != "" {
			writeProblem(w, requestID, http.StatusBadRequest, "invalid_site_request", "网站写入 URL 无效", "")
			return
		}
		var input sites.SiteInput
		if err := decodeJSON(w, r, &input); err != nil {
			writeProblem(w, requestID, http.StatusBadRequest, "invalid_request", "请求格式无效", "")
			return
		}
		if input.Type == "wordpress" {
			job, err := s.sitesManager.StartWordPress(r.Context(), input)
			if err != nil {
				s.writeSiteError(w, requestID, err)
				return
			}
			writeJSON(w, http.StatusAccepted, job)
			return
		}
		if input.Type == "recipe" {
			job, err := s.sitesManager.StartRecipe(r.Context(), input)
			if err != nil {
				s.writeSiteError(w, requestID, err)
				return
			}
			writeJSON(w, http.StatusAccepted, job)
			return
		}
		result, err := s.sitesManager.Create(r.Context(), input)
		if err != nil {
			s.writeSiteError(w, requestID, err)
			return
		}
		writeJSON(w, http.StatusCreated, result)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		writeProblem(w, requestID, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不允许", "")
	}
}

func (s *Server) siteInstallation(w http.ResponseWriter, r *http.Request, requestID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeProblem(w, requestID, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不允许", "")
		return
	}
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_site_installation_request", "安装任务 URL 无效", "")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/site-installations/")
	job, err := s.sitesManager.InstallationJob(id)
	if err != nil {
		writeProblem(w, requestID, http.StatusNotFound, "not_found", "安装任务不存在", "")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) siteOperation(w http.ResponseWriter, r *http.Request, requestID string) {
	if (r.Method == http.MethodPatch || r.Method == http.MethodDelete) &&
		(r.URL.RawPath != "" || r.URL.RawQuery != "") {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_site_request", "网站写入 URL 无效", "")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/sites/")
	if !validSiteID(id) {
		writeProblem(w, requestID, http.StatusNotFound, "not_found", "资源不存在", "")
		return
	}
	if r.Method != http.MethodPatch && r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodPatch+", "+http.MethodDelete)
		writeProblem(w, requestID, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不允许", "")
		return
	}
	if r.Method == http.MethodDelete {
		var input sites.DeleteInput
		if err := decodeJSON(w, r, &input); err != nil {
			writeProblem(w, requestID, http.StatusBadRequest, "invalid_request", "请求格式无效", "")
			return
		}
		result, err := s.sitesManager.DeleteWithOptions(r.Context(), id, input)
		if err != nil {
			s.writeSiteError(w, requestID, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	var input sites.SiteInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_request", "请求格式无效", "")
		return
	}
	result, err := s.sitesManager.Update(r.Context(), id, input)
	if err != nil {
		s.writeSiteError(w, requestID, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) writeSiteError(w http.ResponseWriter, requestID string, err error) {
	status, code, title := http.StatusServiceUnavailable, "sites_unavailable", "网站写入暂不可用"
	switch {
	case errors.Is(err, sites.ErrInvalidInput):
		status, code, title = http.StatusBadRequest, "invalid_site_request", "网站请求无效"
	case errors.Is(err, sites.ErrForbidden):
		status, code, title = http.StatusUnprocessableEntity, "site_action_unsupported", "当前网站结构没有对应操作适配器"
	case errors.Is(err, sites.ErrConflict):
		status, code, title = http.StatusConflict, "resource_conflict", "网站资源发生冲突"
	case errors.Is(err, sites.ErrUnprocessable):
		status, code, title = http.StatusUnprocessableEntity, "site_validation_failed", "网站配置验证失败"
	case errors.Is(err, sites.ErrNeedsAttention):
		status, code, title = http.StatusServiceUnavailable, "site_needs_attention", "网站操作需要人工检查"
	case errors.Is(err, sites.ErrUnavailable):
		status, code, title = http.StatusServiceUnavailable, "sites_unavailable", "网站写入暂不可用"
	}
	writeProblem(w, requestID, status, code, title, safeDetail(err))
}

func validSiteID(value string) bool {
	if len(value) != 32 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func (s *Server) appList(w http.ResponseWriter, r *http.Request) {
	inventory, err := s.appMarket.Inventory(r.Context())
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "apps_unavailable", "应用市场状态不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, inventory)
}

func (s *Server) appJobList(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		writeProblem(w, requestIDFrom(w), http.StatusBadRequest, "invalid_app_job_request", "应用任务 URL 无效", "")
		return
	}
	writeJSON(w, http.StatusOK, contract.PageResult[appmarket.AppJob]{
		Items: s.appMarket.AppJobs(),
	})
}

func (s *Server) appJob(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		writeProblem(w, requestIDFrom(w), http.StatusBadRequest, "invalid_app_job_request", "应用任务 URL 无效", "")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/app-jobs/")
	job, err := s.appMarket.AppJob(id)
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusNotFound, "app_job_not_found", "应用任务不存在", "")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) appOperation(w http.ResponseWriter, r *http.Request, requestID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeProblem(w, requestID, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不允许", "")
		return
	}
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_app_request", "应用操作 URL 无效", "")
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/v1/apps/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		writeProblem(w, requestID, http.StatusNotFound, "not_found", "资源不存在", "")
		return
	}
	id, action := parts[0], parts[1]
	timeout := 2 * time.Minute
	if action == "install" || action == "update" {
		timeout = 12 * time.Minute
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	if action == "install" {
		var input appmarket.InstallInput
		if err := decodeJSON(w, r, &input); err != nil {
			writeProblem(w, requestID, http.StatusBadRequest, "invalid_request", "请求格式无效", "")
			return
		}
		result, err := s.appMarket.StartInstall(ctx, id, input)
		if err != nil {
			s.writeAppError(w, requestID, err)
			return
		}
		writeJSON(w, http.StatusAccepted, result)
		return
	}

	var input appmarket.MutationInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_request", "请求格式无效", "")
		return
	}
	if action == "start" || action == "stop" || action == "restart" {
		result, err := s.appMarket.Lifecycle(ctx, id, action, input.ResourceVersion)
		if err != nil {
			s.writeAppError(w, requestID, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if action == "check_update" {
		result, err := s.appMarket.CheckUpdate(ctx, id, input.ResourceVersion)
		if err != nil {
			s.writeAppError(w, requestID, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if action != "update" && action != "uninstall" && action != "direct_access" {
		writeProblem(w, requestID, http.StatusNotFound, "not_found", "资源不存在", "")
		return
	}
	job, scriptBacked, err := s.appMarket.StartScriptMutation(ctx, id, action, input)
	if scriptBacked {
		if err != nil {
			s.writeAppError(w, requestID, err)
			return
		}
		writeJSON(w, http.StatusAccepted, job)
		return
	}
	result, err := s.appMarket.Mutate(ctx, id, action, input)
	if err != nil {
		s.writeAppError(w, requestID, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) writeAppError(w http.ResponseWriter, requestID string, err error) {
	status, code, title := http.StatusServiceUnavailable, "app_action_failed", "应用操作失败"
	switch {
	case errors.Is(err, appmarket.ErrNotFound):
		status, code, title = http.StatusNotFound, "app_not_found", "应用不存在"
	case errors.Is(err, appmarket.ErrForbidden), errors.Is(err, appmarket.ErrUnsupported),
		errors.Is(err, dockerx.ErrRuntimeContract), errors.Is(err, dockerx.ErrActionUnsupported):
		status, code, title = http.StatusUnprocessableEntity, "app_action_unsupported", "当前应用状态或适配器不支持此操作"
	case errors.Is(err, dockerx.ErrVersionRequired):
		status, code, title = http.StatusBadRequest, "resource_version_required", "必须提供资源版本"
	case errors.Is(err, appmarket.ErrConflict),
		errors.Is(err, dockerx.ErrResourceConflict), errors.Is(err, dockerx.ErrAppConflict):
		status, code, title = http.StatusConflict, "resource_conflict", "应用资源已发生变化"
	case errors.Is(err, appmarket.ErrRolledBack), errors.Is(err, dockerx.ErrAppRolledBack):
		status, code, title = http.StatusUnprocessableEntity, "app_action_rolled_back", "应用操作失败并已回滚"
	case errors.Is(err, appmarket.ErrNeedsAttention), errors.Is(err, dockerx.ErrAppNeedsAttention):
		status, code, title = http.StatusServiceUnavailable, "app_needs_attention", "应用操作需要人工检查"
	}
	writeProblem(w, requestID, status, code, title, safeDetail(err))
}

func (s *Server) dockerSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.docker.Summary(r.Context())
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "docker_unavailable", "Docker Engine 不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) dockerEnvironment(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.docker.Environment(r.Context()))
}

func (s *Server) containerList(w http.ResponseWriter, r *http.Request) {
	items, err := s.docker.Containers(r.Context())
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "docker_unavailable", "容器列表不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, contract.PageResult[contract.ContainerSummary]{Items: items})
}

func (s *Server) imageList(w http.ResponseWriter, r *http.Request) {
	items, err := s.docker.Images(r.Context())
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "docker_unavailable", "镜像列表不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, contract.PageResult[dockerx.ImageSummary]{Items: items})
}

func (s *Server) networkList(w http.ResponseWriter, r *http.Request) {
	items, err := s.docker.Networks(r.Context())
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "docker_unavailable", "网络列表不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, contract.PageResult[dockerx.NetworkSummary]{Items: items})
}

func (s *Server) volumeList(w http.ResponseWriter, r *http.Request) {
	items, err := s.docker.Volumes(r.Context())
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "docker_unavailable", "存储卷列表不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, contract.PageResult[dockerx.VolumeSummary]{Items: items})
}

func (s *Server) dockerBackupList(w http.ResponseWriter, _ *http.Request) {
	items, err := s.docker.DockerBackups()
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "docker_backups_unavailable", "Docker 备份列表不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, contract.PageResult[dockerx.DockerBackup]{Items: items})
}

func (s *Server) dockerJobList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, contract.PageResult[dockerx.MaintenanceJob]{
		Items: s.docker.MaintenanceJobs(),
	})
}

func (s *Server) dockerJob(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/docker/jobs/")
	job, err := s.docker.MaintenanceJob(id)
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusNotFound, "docker_job_not_found", "Docker 任务不存在", "")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) dockerTask(w http.ResponseWriter, r *http.Request) {
	var input dockerx.MaintenanceInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusBadRequest, "invalid_request", "请求格式无效", "")
		return
	}
	job, err := s.docker.StartMaintenance(r.Context(), input)
	if err != nil {
		status, code, title := http.StatusUnprocessableEntity, "docker_task_invalid", "Docker 任务请求无效"
		switch {
		case errors.Is(err, dockerx.ErrDockerJobConflict):
			status, code, title = http.StatusConflict, "docker_task_conflict", "已有 Docker 后台任务正在运行"
		case errors.Is(err, dockerx.ErrResourceConflict):
			status, code, title = http.StatusConflict, "resource_conflict", "Docker 资源已发生变化"
		case errors.Is(err, dockerx.ErrDockerJobNotFound):
			status, code, title = http.StatusNotFound, "docker_resource_not_found", "Docker 资源不存在"
		}
		writeProblem(w, requestIDFrom(w), status, code, title, safeDetail(err))
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func (s *Server) containerOperation(w http.ResponseWriter, r *http.Request, requestID string) {
	rest := strings.TrimPrefix(r.URL.Path, "/v1/docker/containers/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		writeProblem(w, requestID, http.StatusNotFound, "not_found", "资源不存在", "")
		return
	}
	id, action := parts[0], parts[1]
	if action == "logs" {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeProblem(w, requestID, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不允许", "")
			return
		}
		tail, _ := strconv.Atoi(r.URL.Query().Get("tail"))
		logs, err := s.docker.ContainerLogs(r.Context(), id, tail)
		if err != nil {
			if errors.Is(err, dockerx.ErrRuntimeContract) {
				writeProblem(w, requestID, http.StatusUnprocessableEntity, "container_logs_unsupported", "当前容器运行契约无法读取日志", "")
				return
			}
			writeProblem(w, requestID, http.StatusBadGateway, "docker_logs_failed", "容器日志不可用", safeDetail(err))
			return
		}
		writeJSON(w, http.StatusOK, logs)
		return
	}
	if action == "stats" {
		if r.Method != http.MethodGet || r.URL.RawPath != "" || r.URL.RawQuery != "" {
			w.Header().Set("Allow", http.MethodGet)
			writeProblem(w, requestID, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不允许", "")
			return
		}
		stats, err := s.docker.ContainerStats(r.Context(), id)
		if err != nil {
			writeProblem(w, requestID, http.StatusBadGateway, "docker_stats_failed", "容器性能数据不可用", safeDetail(err))
			return
		}
		writeJSON(w, http.StatusOK, stats)
		return
	}
	if action == "exec" {
		if r.Method != http.MethodPost || r.URL.RawPath != "" || r.URL.RawQuery != "" {
			w.Header().Set("Allow", http.MethodPost)
			writeProblem(w, requestID, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不允许", "")
			return
		}
		var input dockerx.ContainerExecInput
		if err := decodeJSON(w, r, &input); err != nil {
			writeProblem(w, requestID, http.StatusBadRequest, "invalid_request", "请求格式无效", "")
			return
		}
		result, err := s.docker.ContainerExec(r.Context(), id, input)
		if err != nil {
			status, code, title := http.StatusUnprocessableEntity, "container_exec_rejected", "容器控制台操作被拒绝"
			switch {
			case errors.Is(err, dockerx.ErrResourceConflict):
				status, code, title = http.StatusConflict, "resource_conflict", "资源已被其他操作修改"
			case errors.Is(err, dockerx.ErrRuntimeContract), errors.Is(err, dockerx.ErrActionUnsupported):
				status, code, title = http.StatusUnprocessableEntity, "container_exec_unsupported", "当前容器状态不支持控制台"
			case errors.Is(err, dockerx.ErrVersionRequired):
				status, code, title = http.StatusBadRequest, "resource_version_required", "必须提供资源版本"
			}
			writeProblem(w, requestID, status, code, title, safeDetail(err))
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeProblem(w, requestID, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不允许", "")
		return
	}
	var input struct {
		ResourceVersion string `json:"resourceVersion"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_request", "请求格式无效", "")
		return
	}
	result, err := s.docker.Lifecycle(r.Context(), id, action, input.ResourceVersion)
	if err != nil {
		status, code, title := http.StatusBadRequest, "docker_action_rejected", "容器操作被拒绝"
		switch {
		case errors.Is(err, dockerx.ErrResourceConflict):
			status, code, title = http.StatusConflict, "resource_conflict", "资源已被其他操作修改"
		case errors.Is(err, dockerx.ErrRuntimeContract), errors.Is(err, dockerx.ErrActionUnsupported):
			status, code, title = http.StatusUnprocessableEntity, "container_action_unsupported", "当前容器状态或运行契约不支持此操作"
		case errors.Is(err, dockerx.ErrVersionRequired):
			status, code, title = http.StatusBadRequest, "resource_version_required", "必须提供资源版本"
		}
		writeProblem(w, requestID, status, code, title, safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxAgentRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra interface{}
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeProblem(w http.ResponseWriter, requestID string, status int, code, title, detail string) {
	writeJSON(w, status, contract.Problem{
		Type: "about:blank", Title: title, Status: status, Code: code,
		Detail: detail, RequestID: requestID, Retryable: status >= 500,
	})
}

func requestID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(value[:])
}

func requestIDFrom(w http.ResponseWriter) string {
	return w.Header().Get("X-Request-ID")
}

func safeDetail(err error) string {
	if err == nil {
		return ""
	}
	value := err.Error()
	if len(value) > 500 {
		value = value[:500]
	}
	return value
}

func reasonIf(err error, reason string) string {
	if err != nil {
		return reason
	}
	return ""
}

func reasonUnless(value bool, reason string) string {
	if !value {
		return reason
	}
	if strings.Contains(reason, "仅") {
		return reason
	}
	return ""
}
