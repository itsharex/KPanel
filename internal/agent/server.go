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
	"strconv"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/dockerx"
	"github.com/kejilion/kejilion-panel/internal/sites"
	"github.com/kejilion/kejilion-panel/internal/systeminfo"
)

const maxAgentRequestBytes = 64 << 10

type Config struct {
	Token           []byte
	Version         string
	ProtocolVersion string
	WebRoot         string
	System          *systeminfo.Collector
	Sites           *sites.Discoverer
	Docker          *dockerx.Client
	Now             func() time.Time
}

type Server struct {
	tokenHash       [32]byte
	version         string
	protocolVersion string
	webRoot         string
	system          *systeminfo.Collector
	sites           *sites.Discoverer
	docker          *dockerx.Client
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
	if config.Sites == nil {
		config.Sites = sites.NewDiscoverer(config.WebRoot)
	}
	if config.Docker == nil {
		return nil, errors.New("Docker client is required")
	}
	return &Server{
		tokenHash:       sha256.Sum256(config.Token),
		version:         config.Version,
		protocolVersion: config.ProtocolVersion,
		webRoot:         config.WebRoot,
		system:          config.System,
		sites:           config.Sites,
		docker:          config.Docker,
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
	case r.URL.Path == "/v1/sites":
		s.requireMethod(w, r, requestID, http.MethodGet, s.siteList)
	case r.URL.Path == "/v1/docker/summary":
		s.requireMethod(w, r, requestID, http.MethodGet, s.dockerSummary)
	case r.URL.Path == "/v1/docker/containers":
		s.requireMethod(w, r, requestID, http.MethodGet, s.containerList)
	case r.URL.Path == "/v1/docker/images":
		s.requireMethod(w, r, requestID, http.MethodGet, s.imageList)
	case r.URL.Path == "/v1/docker/networks":
		s.requireMethod(w, r, requestID, http.MethodGet, s.networkList)
	case r.URL.Path == "/v1/docker/volumes":
		s.requireMethod(w, r, requestID, http.MethodGet, s.volumeList)
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
	if _, err := os.Stat(s.webRoot); err != nil {
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
	ctx, cancel := context.WithTimeout(r.Context(), 800*time.Millisecond)
	defer cancel()
	dockerAvailable := s.docker.Ping(ctx) == nil
	_, siteErr := os.Stat(s.webRoot)
	items := []contract.Capability{
		{ID: "system.read", Enabled: true, Methods: []string{"GET"}},
		{ID: "sites.read", Enabled: siteErr == nil, Reason: reasonIf(siteErr, "Kejilion Web 根目录不可用"), Methods: []string{"GET"}},
		{ID: "docker.read", Enabled: dockerAvailable, Reason: reasonUnless(dockerAvailable, "Docker Engine 不可用"), Methods: []string{"GET"}},
		{ID: "docker.logs", Enabled: dockerAvailable, Reason: reasonUnless(dockerAvailable, "Docker Engine 不可用"), Methods: []string{"GET"}},
		{ID: "docker.lifecycle", Enabled: dockerAvailable, Reason: reasonUnless(dockerAvailable, "仅安全识别的 Kejilion 容器可操作"), Methods: []string{"POST"}},
		{ID: "sites.write", Enabled: false, Reason: "首版 Agent 暂不写入站点", Methods: []string{"POST", "PUT"}},
	}
	writeJSON(w, http.StatusOK, contract.PageResult[contract.Capability]{Items: items})
}

func (s *Server) systemSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.system.Collect(r.Context())
	if err != nil && summary.Hostname == "" {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "system_unavailable", "系统状态不可用", "")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) siteList(w http.ResponseWriter, _ *http.Request) {
	items, err := s.sites.Discover()
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "sites_unavailable", "网站状态不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, contract.PageResult[contract.SiteSummary]{Items: items})
}

func (s *Server) dockerSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.docker.Summary(r.Context())
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "docker_unavailable", "Docker Engine 不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, summary)
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
			writeProblem(w, requestID, http.StatusBadGateway, "docker_logs_failed", "容器日志不可用", safeDetail(err))
			return
		}
		writeJSON(w, http.StatusOK, logs)
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
		case errors.Is(err, dockerx.ErrReadOnlyContainer), errors.Is(err, dockerx.ErrUnsafeOrInvalidAction):
			status, code, title = http.StatusForbidden, "container_read_only", "该容器只能查看"
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
