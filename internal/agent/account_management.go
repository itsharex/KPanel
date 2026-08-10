package agent

import (
	"context"
	"net/http"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func (s *Server) systemAccounts(w http.ResponseWriter, r *http.Request) {
	if !validSystemResourceURL(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	snapshot, err := s.systemManager.Accounts(ctx)
	if err != nil {
		writeProblem(w, requestIDFrom(w), http.StatusServiceUnavailable, "system_accounts_unavailable", "账户状态不可用", safeDetail(err))
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) systemAccountAction(w http.ResponseWriter, r *http.Request) {
	requestID := requestIDFrom(w)
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_account_action", "账户操作 URL 无效", "")
		return
	}
	var input contract.AccountManagementActionRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_request", "请求格式无效", "")
		return
	}
	if field, detail := contract.ValidateAccountManagementAction(&input); field != "" {
		writeProblem(w, requestID, http.StatusUnprocessableEntity, "invalid_account_action", "账户操作无效", field+": "+detail)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Second)
	defer cancel()
	result, err := s.systemManager.ExecuteAccountManagementAction(ctx, input)
	if err != nil {
		status, code, title, retryable := systemResourceProblem(err)
		writeProblemWithRetryable(w, requestID, status, code, title, safeDetail(err), retryable)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
