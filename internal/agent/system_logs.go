package agent

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/systemmanage"
)

const (
	systemLogSummaryTimeout = 8 * time.Second
	systemLogReadTimeout    = 6 * time.Second
)

func (s *Server) systemLogsSummary(w http.ResponseWriter, r *http.Request) {
	requestID := requestIDFrom(w)
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		writeProblem(w, requestID, http.StatusUnprocessableEntity, "invalid_system_log_query", "系统日志查询参数无效", "")
		return
	}
	if !s.acquireSystemLogs(w, requestID) {
		return
	}
	defer func() { <-s.systemLogsGate }()
	ctx, cancel := context.WithTimeout(r.Context(), systemLogSummaryTimeout)
	defer cancel()
	result, err := s.systemManager.SystemLogSummary(ctx)
	if err != nil {
		s.writeSystemLogError(w, requestID, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) systemLogs(w http.ResponseWriter, r *http.Request) {
	requestID := requestIDFrom(w)
	if r.URL.RawPath != "" {
		writeProblem(w, requestID, http.StatusUnprocessableEntity, "invalid_system_log_query", "系统日志查询参数无效", "")
		return
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		writeProblem(w, requestID, http.StatusUnprocessableEntity, "invalid_system_log_query", "系统日志查询参数无效", "")
		return
	}
	query, field, detail := contract.ParseSystemLogQuery(values)
	if field != "" {
		writeProblem(w, requestID, http.StatusUnprocessableEntity, "invalid_system_log_query", "系统日志查询参数无效", field+": "+detail)
		return
	}
	if !s.acquireSystemLogs(w, requestID) {
		return
	}
	defer func() { <-s.systemLogsGate }()
	ctx, cancel := context.WithTimeout(r.Context(), systemLogReadTimeout)
	defer cancel()
	result, err := s.systemManager.SystemLogs(ctx, query)
	if err != nil {
		s.writeSystemLogError(w, requestID, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) acquireSystemLogs(w http.ResponseWriter, requestID string) bool {
	select {
	case s.systemLogsGate <- struct{}{}:
		return true
	default:
		writeProblem(w, requestID, http.StatusTooManyRequests, "system_logs_busy", "另一项系统日志查询正在执行", "")
		return false
	}
}

func (s *Server) writeSystemLogError(w http.ResponseWriter, requestID string, err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		writeProblem(w, requestID, http.StatusGatewayTimeout, "system_logs_timeout", "系统日志查询超时", "")
		return
	}
	if errors.Is(err, systemmanage.ErrInvalidInput) {
		writeProblem(w, requestID, http.StatusUnprocessableEntity, "invalid_system_log_query", "系统日志查询参数无效", safeDetail(err))
		return
	}
	writeProblem(w, requestID, http.StatusServiceUnavailable, "system_logs_unavailable", "系统日志暂时不可用", safeDetail(err))
}
