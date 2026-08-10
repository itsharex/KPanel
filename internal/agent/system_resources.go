package agent

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/systemmanage"
)

func (s *Server) systemHosts(w http.ResponseWriter, r *http.Request) {
	if !validSystemResourceURL(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	snapshot, err := s.systemManager.Hosts(ctx)
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "system_hosts_unavailable", "hosts 状态不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) systemCron(w http.ResponseWriter, r *http.Request) {
	if !validSystemResourceURL(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	snapshot, err := s.systemManager.Cron(ctx)
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "system_cron_unavailable", "定时任务状态不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) systemNetworkInterfaces(w http.ResponseWriter, r *http.Request) {
	if !validSystemResourceURL(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	snapshot, err := s.systemManager.NetworkInterfaces(ctx)
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "system_network_interfaces_unavailable", "网卡状态不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) systemFirewall(w http.ResponseWriter, r *http.Request) {
	if !validSystemResourceURL(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	snapshot, err := s.systemManager.Firewall(ctx)
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "system_firewall_unavailable", "防火墙状态不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func validSystemResourceURL(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.RawPath == "" && r.URL.RawQuery == "" {
		return true
	}
	writeProblem(w, requestIDFrom(w), http.StatusBadRequest, "invalid_system_resource_url", "系统资源 URL 无效", "")
	return false
}

func (s *Server) systemResourceAction(w http.ResponseWriter, r *http.Request) {
	requestID := requestIDFrom(w)
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_system_resource_action", "系统资源操作 URL 无效", "")
		return
	}
	var input contract.SystemResourceActionRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_request", "请求格式无效", "")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Second)
	defer cancel()
	result, err := s.systemManager.ExecuteSystemResourceAction(ctx, input)
	if err != nil {
		status, code, title, retryable := systemResourceProblem(err)
		writeProblemWithRetryable(w, requestID, status, code, title, safeDetail(err), retryable)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func systemResourceProblem(err error) (int, string, string, bool) {
	status, code, title := http.StatusServiceUnavailable, "system_resource_action_failed", "系统资源操作失败"
	retryable := true
	switch {
	case errors.Is(err, systemmanage.ErrInvalidInput):
		status, code, title, retryable = http.StatusUnprocessableEntity, "invalid_system_resource_action", "系统资源操作参数无效", false
	case errors.Is(err, systemmanage.ErrDisabled):
		status, code, title, retryable = http.StatusForbidden, "system_resource_write_disabled", "系统资源写入未启用", false
	case errors.Is(err, systemmanage.ErrConflict):
		status, code, title, retryable = http.StatusConflict, "system_resource_conflict", "系统资源版本发生冲突", false
	case errors.Is(err, systemmanage.ErrRolledBack):
		code, title = "system_resource_rolled_back", "系统资源操作失败并已回滚"
	case errors.Is(err, systemmanage.ErrRollbackFailed):
		code, title, retryable = "system_resource_rollback_failed", "系统资源回滚失败，需要人工检查", false
	case errors.Is(err, systemmanage.ErrNeedsAttention):
		code, title, retryable = "system_resource_needs_attention", "系统资源操作需要人工检查", false
	case errors.Is(err, systemmanage.ErrUnsupported):
		code, title = "system_resource_adapter_unavailable", "系统资源适配器不可用"
	}
	return status, code, title, retryable
}
