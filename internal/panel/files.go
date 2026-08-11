package panel

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/filemanager"
	"github.com/kejilion/kejilion-panel/internal/httpstream"
)

const (
	panelFileTransferIdleTimeout = 45 * time.Second
	panelFileTransferMaxDuration = 2 * time.Hour
	fileDownloadTicketTTL        = 5 * time.Minute
	maxFileDownloadTickets       = 128
)

type fileDownloadTicket struct {
	Path      string
	ExpiresAt time.Time
}

type fileDownloadTicketRequest struct {
	Path string `json:"path"`
}

type fileDownloadTicketResponse struct {
	DownloadURL string    `json:"downloadUrl"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

var errFileDownloadTicketLimit = errors.New("file download ticket limit reached")

type agentStreamAPI interface {
	OpenStream(
		ctx context.Context,
		method, path, rawQuery, requestID string,
		body io.Reader,
		headers http.Header,
		contentLength int64,
	) (*http.Response, error)
}

func (s *Server) handleFileList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		s.writeProblem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", "")
		return
	}
	if r.URL.RawPath != "" || !strictPanelQuery(r.URL.Query(), "path", "limit", "offset", "search") {
		s.writeProblem(w, r, http.StatusBadRequest, "file_query_invalid", "文件查询参数无效", "")
		return
	}
	_, _, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	response, err := s.agent.Get(r.Context(), "/v1/files", r.URL.RawQuery, requestID(r))
	if err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Agent unavailable", "")
		return
	}
	s.writeAgentResponse(w, r, response)
}

func (s *Server) handleFileTrashList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		s.writeProblem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", "")
		return
	}
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		s.writeProblem(w, r, http.StatusBadRequest, "file_query_invalid", "回收站查询参数无效", "")
		return
	}
	if _, _, ok := s.requireSession(w, r); !ok {
		return
	}
	response, err := s.agent.Get(r.Context(), "/v1/files/trash", "", requestID(r))
	if err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Agent unavailable", "")
		return
	}
	s.writeAgentResponse(w, r, response)
}

func (s *Server) handleFileContent(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		s.handleFileDownload(w, r)
	case http.MethodPut:
		s.handleFileWrite(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead+", "+http.MethodPut)
		s.writeProblem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", "")
	}
}

func (s *Server) handleFileDownload(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawPath != "" || !strictPanelQuery(r.URL.Query(), "path", "disposition", "mode", "version") {
		s.writeProblem(w, r, http.StatusBadRequest, "file_query_invalid", "文件查询参数无效", "")
		return
	}
	_, _, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	s.streamFileDownload(w, r, r.URL.RawQuery)
}

func (s *Server) handleFileDownloadTicketCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		s.writeProblem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", "")
		return
	}
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		s.writeProblem(w, r, http.StatusBadRequest, "file_query_invalid", "文件查询参数无效", "")
		return
	}
	if !s.checkOrigin(w, r) {
		return
	}
	_, session, ok := s.requireSession(w, r)
	if !ok || !s.checkCSRF(w, r, session) {
		return
	}
	var input fileDownloadTicketRequest
	if err := s.decodeJSON(w, r, &input); err != nil {
		return
	}
	if !validFileDownloadPath(input.Path) {
		s.writeValidationProblem(w, r, "path", "file path must be a canonical absolute path")
		return
	}
	token, expiresAt, err := s.issueFileDownloadTicket(input.Path)
	if errors.Is(err, errFileDownloadTicketLimit) {
		s.writeProblem(w, r, http.StatusTooManyRequests, "file_download_ticket_limit", "下载请求过多，请稍后重试", "")
		return
	}
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, "file_download_ticket_unavailable", "无法创建下载链接", "")
		return
	}
	s.writeJSON(w, http.StatusCreated, fileDownloadTicketResponse{
		DownloadURL: "/api/v1/files/download/" + token,
		ExpiresAt:   expiresAt,
	})
}

func (s *Server) handleFileDownloadTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
		s.writeProblem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", "")
		return
	}
	if r.URL.RawQuery != "" {
		s.writeProblem(w, r, http.StatusNotFound, "file_download_not_found", "下载链接无效或已过期", "")
		return
	}
	key, ok := fileDownloadTicketKey(strings.TrimPrefix(r.URL.Path, "/api/v1/files/download/"))
	if !ok {
		s.writeProblem(w, r, http.StatusNotFound, "file_download_not_found", "下载链接无效或已过期", "")
		return
	}
	ticket, ok := s.lookupFileDownloadTicket(key)
	if !ok {
		s.writeProblem(w, r, http.StatusNotFound, "file_download_not_found", "下载链接无效或已过期", "")
		return
	}
	query := url.Values{"path": []string{ticket.Path}, "disposition": []string{"attachment"}}
	s.streamFileDownload(w, r, query.Encode())
}

func (s *Server) streamFileDownload(w http.ResponseWriter, r *http.Request, rawQuery string) {
	streamer, ok := s.agent.(agentStreamAPI)
	if !ok {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_stream_unavailable", "Agent 文件流不可用", "")
		return
	}
	headers := make(http.Header)
	for _, key := range []string{"Range", "If-None-Match", "If-Modified-Since", "If-Range"} {
		if value := r.Header.Get(key); value != "" {
			headers.Set(key, value)
		}
	}
	transferContext, cancel := context.WithTimeout(r.Context(), panelFileTransferMaxDuration)
	defer cancel()
	response, err := streamer.OpenStream(
		transferContext, r.Method, "/v1/files/content", rawQuery,
		requestID(r), http.NoBody, headers, 0,
	)
	if err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Agent unavailable", "")
		return
	}
	defer response.Body.Close()
	copyFileHeaders(w.Header(), response.Header)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Pragma", "no-cache")
	writer := httpstream.NewIdleResponseWriter(
		transferContext, w, panelFileTransferIdleTimeout,
	)
	writer.WriteHeader(response.StatusCode)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = io.CopyBuffer(writer, response.Body, make([]byte, 64<<10))
}

func validFileDownloadPath(value string) bool {
	return value != "" && len(value) <= 4096 && strings.HasPrefix(value, "/") &&
		!strings.ContainsAny(value, "\\\x00") && path.Clean(value) == value
}

func isFileDownloadTicketPath(requestPath string) bool {
	if !strings.HasPrefix(requestPath, "/api/v1/files/download/") {
		return false
	}
	_, ok := fileDownloadTicketKey(strings.TrimPrefix(requestPath, "/api/v1/files/download/"))
	return ok
}

func fileDownloadTicketKey(token string) ([32]byte, bool) {
	if len(token) != 43 {
		return [32]byte{}, false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != 32 {
		return [32]byte{}, false
	}
	return sha256.Sum256(decoded), true
}

func (s *Server) issueFileDownloadTicket(filePath string) (string, time.Time, error) {
	now := time.Now().UTC()
	s.downloadTicketMu.Lock()
	defer s.downloadTicketMu.Unlock()
	if s.downloadTickets == nil {
		s.downloadTickets = make(map[[32]byte]fileDownloadTicket)
	}
	for key, ticket := range s.downloadTickets {
		if !ticket.ExpiresAt.After(now) {
			delete(s.downloadTickets, key)
		}
	}
	if len(s.downloadTickets) >= maxFileDownloadTickets {
		return "", time.Time{}, errFileDownloadTicketLimit
	}
	for range 3 {
		value := make([]byte, 32)
		if _, err := rand.Read(value); err != nil {
			return "", time.Time{}, err
		}
		token := base64.RawURLEncoding.EncodeToString(value)
		key := sha256.Sum256(value)
		if _, exists := s.downloadTickets[key]; exists {
			continue
		}
		expiresAt := now.Add(fileDownloadTicketTTL)
		s.downloadTickets[key] = fileDownloadTicket{Path: filePath, ExpiresAt: expiresAt}
		return token, expiresAt, nil
	}
	return "", time.Time{}, errors.New("file download ticket collision")
}

func (s *Server) lookupFileDownloadTicket(key [32]byte) (fileDownloadTicket, bool) {
	now := time.Now().UTC()
	s.downloadTicketMu.Lock()
	defer s.downloadTicketMu.Unlock()
	ticket, ok := s.downloadTickets[key]
	if !ok || !ticket.ExpiresAt.After(now) {
		delete(s.downloadTickets, key)
		return fileDownloadTicket{}, false
	}
	return ticket, true
}

func (s *Server) handleFileWrite(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawPath != "" || !strictPanelQuery(r.URL.Query(), "path") {
		s.writeProblem(w, r, http.StatusBadRequest, "file_query_invalid", "文件查询参数无效", "")
		return
	}
	if !s.checkOrigin(w, r) {
		return
	}
	_, session, ok := s.requireSession(w, r)
	if !ok || !s.checkCSRF(w, r, session) {
		return
	}
	if mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mediaType != "application/json" {
		s.writeProblem(w, r, http.StatusUnsupportedMediaType, "json_required", "JSON request body required", "")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, filemanager.MaxTextBytes+(64<<10))
	var input contract.FileWriteRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		s.writeProblem(w, r, http.StatusBadRequest, "file_content_invalid", "文件内容无效", "")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		s.writeProblem(w, r, http.StatusBadRequest, "file_content_invalid", "文件内容无效", "")
		return
	}
	body, err := json.Marshal(input)
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, "request_encoding_failed", "Request encoding failed", "")
		return
	}
	target := r.URL.Query().Get("path")
	change := map[string]any{"bytes": len(input.Content), "resourceVersion": input.ExpectedResourceVersion}
	if err := s.audit(r, session.User.ID, "file.write", "file", target, "intent", change); err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "audit_unavailable", "Audit storage unavailable", "")
		return
	}
	response, err := s.agent.Do(
		r.Context(), http.MethodPut, "/v1/files/content", r.URL.RawQuery, requestID(r), body,
	)
	if err != nil {
		_ = s.audit(r, session.User.ID, "file.write", "file", target, "failure", change)
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Agent unavailable", "")
		return
	}
	result := "failure"
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		result = "success"
	}
	_ = s.audit(r, session.User.ID, "file.write", "file", target, result, change)
	s.writeAgentResponse(w, r, response)
}

func (s *Server) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		s.writeProblem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", "")
		return
	}
	if r.URL.RawPath != "" || !strictPanelQuery(r.URL.Query(), "path", "name", "overwrite") {
		s.writeProblem(w, r, http.StatusBadRequest, "file_query_invalid", "上传参数无效", "")
		return
	}
	if !s.checkOrigin(w, r) {
		return
	}
	_, session, ok := s.requireSession(w, r)
	if !ok || !s.checkCSRF(w, r, session) {
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/octet-stream" {
		s.writeProblem(w, r, http.StatusUnsupportedMediaType, "binary_required", "上传必须使用二进制内容", "")
		return
	}
	if r.ContentLength > filemanager.MaxUploadBytes {
		s.writeProblem(w, r, http.StatusRequestEntityTooLarge, "file_too_large", "文件超过 512 MiB", "")
		return
	}
	streamer, ok := s.agent.(agentStreamAPI)
	if !ok {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_stream_unavailable", "Agent 文件流不可用", "")
		return
	}
	target := path.Join(r.URL.Query().Get("path"), r.URL.Query().Get("name"))
	change := map[string]any{"bytes": r.ContentLength, "overwrite": r.URL.Query().Get("overwrite") == "true"}
	if err := s.audit(r, session.User.ID, "file.upload", "file", target, "intent", change); err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "audit_unavailable", "Audit storage unavailable", "")
		return
	}
	headers := make(http.Header)
	headers.Set("Content-Type", "application/octet-stream")
	transferContext, cancel := context.WithTimeout(r.Context(), panelFileTransferMaxDuration)
	defer cancel()
	content := httpstream.NewIdleReader(
		transferContext, w, r.Body, panelFileTransferIdleTimeout,
	)
	response, err := streamer.OpenStream(
		transferContext, http.MethodPost, "/v1/files/upload", r.URL.RawQuery,
		requestID(r), content, headers, r.ContentLength,
	)
	if err != nil {
		_ = s.audit(r, session.User.ID, "file.upload", "file", target, "failure", change)
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Agent unavailable", "")
		return
	}
	defer response.Body.Close()
	result := "failure"
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		result = "success"
	}
	_ = s.audit(r, session.User.ID, "file.upload", "file", target, result, change)
	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json; charset=utf-8"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(response.StatusCode)
	_, _ = io.CopyBuffer(w, io.LimitReader(response.Body, 1<<20), make([]byte, 32<<10))
}

func (s *Server) handleFileAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		s.writeProblem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", "")
		return
	}
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		s.writeProblem(w, r, http.StatusBadRequest, "file_query_invalid", "文件操作参数无效", "")
		return
	}
	if !s.checkOrigin(w, r) {
		return
	}
	_, session, ok := s.requireSession(w, r)
	if !ok || !s.checkCSRF(w, r, session) {
		return
	}
	var input contract.FileActionRequest
	if err := s.decodeJSON(w, r, &input); err != nil {
		return
	}
	if !allowedFileAction(input.Action) {
		s.writeValidationProblem(w, r, "action", "unsupported file action")
		return
	}
	body, err := json.Marshal(input)
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, "request_encoding_failed", "Request encoding failed", "")
		return
	}
	target := input.Target
	if target == "" && len(input.Sources) > 0 {
		target = input.Sources[0]
	} else if target == "" && len(input.TrashIDs) > 0 {
		target = input.TrashIDs[0]
	}
	if input.Action == "trash_empty" {
		target = "file-trash"
	}
	itemCount := len(input.Sources)
	if len(input.TrashIDs) > 0 {
		itemCount = len(input.TrashIDs)
	}
	change := map[string]any{"action": input.Action, "items": itemCount}
	if input.Action == "compress" || input.Action == "extract" {
		change["format"] = input.Format
		change["name"] = input.Name
	}
	if err := s.audit(r, session.User.ID, "file."+input.Action, "file", target, "intent", change); err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "audit_unavailable", "Audit storage unavailable", "")
		return
	}
	response, err := s.doFileAction(r, input.Action, body)
	if err != nil {
		_ = s.audit(r, session.User.ID, "file."+input.Action, "file", target, "failure", change)
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Agent unavailable", "")
		return
	}
	result := "failure"
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		result = "success"
		var actionResult contract.FileActionResult
		if json.Unmarshal(response.Body, &actionResult) == nil && len(actionResult.Failed) > 0 {
			result = "partial_failure"
		}
	}
	_ = s.audit(r, session.User.ID, "file."+input.Action, "file", target, result, change)
	s.writeAgentResponse(w, r, response)
}

func (s *Server) doFileAction(r *http.Request, action string, body []byte) (AgentResponse, error) {
	if action != "compress" && action != "extract" {
		return s.agent.Do(
			r.Context(), http.MethodPost, "/v1/files/actions", "", requestID(r), body,
		)
	}
	streamer, ok := s.agent.(agentStreamAPI)
	if !ok {
		return s.agent.Do(
			r.Context(), http.MethodPost, "/v1/files/actions", "", requestID(r), body,
		)
	}
	headers := make(http.Header)
	headers.Set("Accept", "application/json")
	headers.Set("Content-Type", "application/json")
	response, err := streamer.OpenStream(
		r.Context(), http.MethodPost, "/v1/files/actions", "", requestID(r),
		bytes.NewReader(body), headers, int64(len(body)),
	)
	if err != nil {
		return AgentResponse{}, err
	}
	defer response.Body.Close()
	const maxArchiveActionResponse = 1 << 20
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxArchiveActionResponse+1))
	if err != nil {
		return AgentResponse{}, err
	}
	if len(responseBody) > maxArchiveActionResponse {
		return AgentResponse{}, errors.New("agent archive response exceeds configured limit")
	}
	return AgentResponse{
		StatusCode:  response.StatusCode,
		ContentType: response.Header.Get("Content-Type"),
		Body:        responseBody,
	}, nil
}

func strictPanelQuery(values map[string][]string, allowed ...string) bool {
	keys := make(map[string]struct{}, len(allowed))
	for _, value := range allowed {
		keys[value] = struct{}{}
	}
	for key, values := range values {
		if _, ok := keys[key]; !ok || len(values) != 1 {
			return false
		}
	}
	return true
}

func allowedFileAction(action string) bool {
	switch action {
	case "mkdir", "rename", "copy", "move", "trash", "chmod",
		"compress", "extract", "trash_restore", "trash_delete", "trash_empty":
		return true
	default:
		return false
	}
}

func copyFileHeaders(target, source http.Header) {
	for _, key := range []string{
		"Accept-Ranges", "Content-Disposition", "Content-Length", "Content-Range",
		"Content-Type", "ETag", "Last-Modified",
	} {
		if value := source.Get(key); value != "" {
			target.Set(key, value)
		}
	}
	target.Set("X-Content-Type-Options", "nosniff")
	target.Set("Content-Security-Policy", "default-src 'none'; sandbox")
}
