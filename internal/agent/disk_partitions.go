package agent

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/systemmanage"
)

func (s *Server) diskPartitions(w http.ResponseWriter, r *http.Request) {
	requestID := requestIDFrom(w)
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_disk_partition_url", "磁盘分区 URL 无效", "")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
	defer cancel()
	snapshot, err := s.systemManager.DiskPartitions(ctx)
	if err != nil {
		writeProblem(w, requestID, http.StatusServiceUnavailable, "disk_partitions_unavailable", "磁盘分区状态不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) diskPartitionAction(w http.ResponseWriter, r *http.Request) {
	requestID := requestIDFrom(w)
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_disk_partition_action_url", "磁盘分区操作 URL 无效", "")
		return
	}
	var input contract.DiskPartitionActionRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_request", "请求格式无效", "")
		return
	}
	if field, detail := contract.ValidateDiskPartitionAction(&input); field != "" {
		writeProblem(w, requestID, http.StatusUnprocessableEntity, "invalid_disk_partition_action", "磁盘分区操作参数无效", field+": "+detail)
		return
	}
	// Once accepted this is a durable host operation. A disconnected browser
	// must not cancel the inspect/persist/launch transaction.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 50*time.Second)
	defer cancel()
	job, err := s.systemManager.StartDiskPartitionAction(ctx, input)
	if err != nil {
		status, code, title, retryable := diskPartitionProblem(err)
		writeProblemWithRetryable(w, requestID, status, code, title, safeDetail(err), retryable)
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func diskPartitionProblem(err error) (int, string, string, bool) {
	switch {
	case errors.Is(err, systemmanage.ErrInvalidInput):
		return http.StatusUnprocessableEntity, "invalid_disk_partition_action", "磁盘分区操作参数无效", false
	case errors.Is(err, systemmanage.ErrDisabled):
		return http.StatusForbidden, "disk_partition_write_disabled", "磁盘分区写入未启用", false
	case errors.Is(err, systemmanage.ErrConflict):
		return http.StatusConflict, "disk_partition_conflict", "磁盘状态已变化或任务冲突", false
	case errors.Is(err, systemmanage.ErrUnsupported):
		return http.StatusServiceUnavailable, "disk_partition_adapter_unavailable", "磁盘分区适配器不可用", true
	default:
		return http.StatusServiceUnavailable, "disk_partition_action_failed", "磁盘分区操作失败", true
	}
}
