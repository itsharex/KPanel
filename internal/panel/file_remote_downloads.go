package panel

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/filemanager"
	"github.com/kejilion/kejilion-panel/internal/httpstream"
	"github.com/kejilion/kejilion-panel/internal/remotedownload"
)

const maxPanelRemoteDownloads = 2

func (s *Server) handleFileRemoteDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		s.writeProblem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", "")
		return
	}
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		s.writeProblem(w, r, http.StatusBadRequest, "file_query_invalid", "远程下载参数无效", "")
		return
	}
	if !s.checkOrigin(w, r) {
		return
	}
	_, session, ok := s.requireSession(w, r)
	if !ok || !s.checkCSRF(w, r, session) {
		return
	}
	var input contract.FileRemoteDownloadRequest
	if err := s.decodeJSON(w, r, &input); err != nil {
		return
	}
	parsedURL, err := remotedownload.ValidateURL(input.URL)
	if err != nil || !validFileDownloadPath(input.TargetDirectory) ||
		(input.Name != "" && !validFileRemoteDownloadName(input.Name)) {
		s.writeProblem(w, r, http.StatusUnprocessableEntity, "remote_download_invalid", "请检查下载地址、目标目录和保存名称。", "")
		return
	}
	select {
	case s.remoteDownloadGate <- struct{}{}:
		defer func() { <-s.remoteDownloadGate }()
	default:
		w.Header().Set("Retry-After", "1")
		s.writeProblem(w, r, http.StatusTooManyRequests, "remote_download_busy", "当前远程下载较多，请稍后重试。", "")
		return
	}

	change := map[string]any{
		"source":          remotedownload.SourceDisplay(parsedURL),
		"targetDirectory": input.TargetDirectory,
	}
	if input.Name != "" {
		change["requestedName"] = input.Name
	}
	if err := s.audit(r, session.User.ID, "file.remote_download", "directory", input.TargetDirectory, "intent", change); err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "audit_unavailable", "Audit storage unavailable", "")
		return
	}

	transferContext, cancel := context.WithTimeout(r.Context(), panelFileTransferMaxDuration)
	defer cancel()
	// The work context has a hard deadline, but the response writer must remain
	// usable long enough to deliver that deadline as the terminal stream event.
	output := httpstream.NewIdleResponseWriter(r.Context(), w, panelFileTransferIdleTimeout)
	responseController := http.NewResponseController(output)
	var encoder *json.Encoder
	streamStarted := false
	writeEvent := func(event contract.FileTransferEvent) bool {
		if !streamStarted {
			w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			output.WriteHeader(http.StatusOK)
			encoder = json.NewEncoder(output)
			streamStarted = true
		}
		if err := encoder.Encode(event); err != nil {
			cancel()
			return false
		}
		if err := responseController.Flush(); err != nil && !errors.Is(err, http.ErrNotSupported) {
			cancel()
			return false
		}
		return true
	}
	fail := func(code, detail string, loaded, total int64) {
		failure := cloneRemoteDownloadChange(change)
		failure["errorCode"] = code
		failure["bytes"] = loaded
		_ = s.audit(r, session.User.ID, "file.remote_download", "directory", input.TargetDirectory, "failure", failure)
		_ = writeEvent(contract.FileTransferEvent{
			State: "error", LoadedBytes: loaded, TotalBytes: total, Code: code, Detail: detail,
		})
	}

	if err := s.ensureFileTransferDirectory(transferContext, input.TargetDirectory, requestID(r)); err != nil {
		fail("target_unavailable", "目标目录不存在或不可写。", 0, 0)
		return
	}
	if s.remoteDownloadOpen == nil {
		fail("remote_download_unavailable", "远程下载服务不可用。", 0, 0)
		return
	}
	response, err := s.remoteDownloadOpen(transferContext, parsedURL.String())
	if err != nil {
		code, detail := fileRemoteDownloadError(err)
		fail(code, detail, 0, 0)
		return
	}
	if response == nil || response.Body == nil {
		fail("remote_download_unavailable", "远程下载服务不可用。", 0, 0)
		return
	}
	limited := &fileRemoteDownloadLimitReader{
		source: response.Body, remaining: filemanager.MaxUploadBytes,
	}
	var loaded atomic.Int64
	uploadBody := &fileRemoteDownloadUploadBody{
		source: limited, closer: response.Body, loaded: &loaded,
	}
	defer uploadBody.Close()
	if response.ContentLength > filemanager.MaxUploadBytes {
		fail("remote_download_too_large", "远程文件超过 512 MiB。", 0, response.ContentLength)
		return
	}
	totalBytes := response.ContentLength
	if totalBytes < 0 {
		totalBytes = 0
	}
	name := fileRemoteDownloadName(input.Name, response)
	name, err = s.uniqueFileTransferName(transferContext, input.TargetDirectory, name, requestID(r))
	if err != nil {
		fail("target_name_unavailable", "无法确定可用的保存名称。", 0, totalBytes)
		return
	}
	change["targetName"] = name
	if response.ContentLength >= 0 {
		change["expectedBytes"] = response.ContentLength
	}
	if !writeEvent(contract.FileTransferEvent{State: "connecting"}) {
		return
	}
	if !writeEvent(contract.FileTransferEvent{
		State: "transferring", TotalBytes: totalBytes, Name: name,
	}) {
		return
	}

	streamer, ok := s.agent.(agentStreamAPI)
	if !ok {
		fail("agent_stream_unavailable", "Agent 文件流不可用。", 0, totalBytes)
		return
	}
	query := url.Values{
		"path": []string{input.TargetDirectory}, "name": []string{name}, "overwrite": []string{"false"},
	}
	headers := make(http.Header)
	headers.Set("Content-Type", "application/octet-stream")
	agentContentLength := response.ContentLength
	if agentContentLength < 0 {
		agentContentLength = -1
	}
	type streamResult struct {
		response *http.Response
		err      error
	}
	resultChannel := make(chan streamResult)
	go func() {
		agentResponse, streamError := streamer.OpenStream(
			transferContext, http.MethodPost, "/v1/files/upload", query.Encode(), requestID(r),
			uploadBody, headers, agentContentLength,
		)
		result := streamResult{response: agentResponse, err: streamError}
		select {
		case resultChannel <- result:
		case <-transferContext.Done():
			if result.response != nil && result.response.Body != nil {
				_ = result.response.Body.Close()
			}
		}
	}()

	progressTicker := time.NewTicker(180 * time.Millisecond)
	defer progressTicker.Stop()
	reportedBytes := int64(0)
	lastProgressEvent := time.Now()
	var agentResponse *http.Response
	streamFinished := false
	for !streamFinished {
		select {
		case result := <-resultChannel:
			agentResponse, err = result.response, result.err
			streamFinished = true
			// An Agent may reject the upload from response headers before the
			// Transport has finished reading the request body. Closing the body
			// makes that read stop without letting it write stream events.
			_ = uploadBody.Close()
		case <-progressTicker.C:
			current := loaded.Load()
			if current != reportedBytes || time.Since(lastProgressEvent) >= 10*time.Second {
				if !writeEvent(contract.FileTransferEvent{
					State: "transferring", LoadedBytes: current, TotalBytes: totalBytes, Name: name,
				}) {
					_ = uploadBody.Close()
					return
				}
				reportedBytes = current
				lastProgressEvent = time.Now()
			}
		case <-transferContext.Done():
			_ = uploadBody.Close()
			current := loaded.Load()
			code, detail := fileRemoteDownloadError(transferContext.Err())
			fail(code, detail, current, totalBytes)
			return
		}
	}
	loadedBytes := loaded.Load()
	if err != nil {
		if limited.exceeded.Load() {
			fail("remote_download_too_large", "远程文件超过 512 MiB，传输已停止；请刷新目录确认结果。", loadedBytes, totalBytes)
		} else {
			code, detail := fileRemoteDownloadError(err)
			if code == "remote_download_unreachable" {
				code = "agent_write_interrupted"
				detail = "目标 Agent 写入中断；请刷新目录确认结果。"
			}
			fail(code, detail, loadedBytes, totalBytes)
		}
		return
	}
	if agentResponse == nil {
		fail("agent_write_failed", "目标文件写入结果未知；请刷新目录确认。", loadedBytes, totalBytes)
		return
	}
	defer agentResponse.Body.Close()
	if agentResponse.StatusCode < http.StatusOK || agentResponse.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(agentResponse.Body, 1<<20))
		if limited.exceeded.Load() || agentResponse.StatusCode == http.StatusRequestEntityTooLarge {
			fail("remote_download_too_large", "远程文件超过 512 MiB，传输已停止；请刷新目录确认结果。", loadedBytes, totalBytes)
		} else if agentResponse.StatusCode == http.StatusConflict {
			fail("target_conflict", "保存名称刚被占用，请重试。", loadedBytes, totalBytes)
		} else if agentResponse.StatusCode == http.StatusTooManyRequests {
			fail("agent_write_busy", "目标 Agent 当前繁忙，请稍后重试。", loadedBytes, totalBytes)
		} else if agentResponse.StatusCode == http.StatusForbidden {
			fail("target_permission_denied", "目标目录不再允许写入；请刷新目录确认结果。", loadedBytes, totalBytes)
		} else if agentResponse.StatusCode == http.StatusNotFound {
			fail("target_unavailable", "目标目录已不存在；请刷新目录确认结果。", loadedBytes, totalBytes)
		} else if agentResponse.StatusCode == http.StatusInsufficientStorage {
			fail("target_storage_full", "目标磁盘空间或配额不足；请刷新目录确认结果。", loadedBytes, totalBytes)
		} else {
			fail("agent_write_failed", "目标文件写入失败；请刷新目录确认结果。", loadedBytes, totalBytes)
		}
		return
	}
	if !writeEvent(contract.FileTransferEvent{
		State: "confirming", LoadedBytes: loadedBytes, TotalBytes: totalBytes, Name: name,
	}) {
		return
	}
	var entry contract.FileEntry
	decoder := json.NewDecoder(io.LimitReader(agentResponse.Body, 1<<20))
	decoder.DisallowUnknownFields()
	var extra any
	if err := decoder.Decode(&entry); err != nil ||
		!errors.Is(decoder.Decode(&extra), io.EOF) ||
		!validFileRemoteDownloadEntry(entry, input.TargetDirectory, name, loadedBytes) {
		fail("agent_result_invalid", "目标 Agent 返回无效结果，请刷新目录确认。", loadedBytes, totalBytes)
		return
	}
	success := cloneRemoteDownloadChange(change)
	success["bytes"] = loadedBytes
	_ = s.audit(r, session.User.ID, "file.remote_download", "file", entry.Path, "success", success)
	_ = writeEvent(contract.FileTransferEvent{
		State: "complete", LoadedBytes: loadedBytes, TotalBytes: totalBytes, Name: name, Entry: &entry,
	})
}

func validFileRemoteDownloadName(name string) bool {
	if name == "" || strings.TrimSpace(name) != name || name == "." || name == ".." ||
		len(name) > 255 || !utf8.ValidString(name) || strings.ContainsAny(name, `/\`) ||
		strings.HasPrefix(name, ".kpanel-") {
		return false
	}
	for _, character := range name {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func validFileRemoteDownloadEntry(
	entry contract.FileEntry,
	directory string,
	name string,
	loadedBytes int64,
) bool {
	return entry.Kind == "file" && entry.Name == name && entry.Path == path.Join(directory, name) &&
		entry.SizeBytes == loadedBytes && entry.ResourceVersion != ""
}

func fileRemoteDownloadName(requested string, response *http.Response) string {
	if validFileRemoteDownloadName(requested) {
		return requested
	}
	if disposition := response.Header.Get("Content-Disposition"); disposition != "" {
		if _, parameters, err := mime.ParseMediaType(disposition); err == nil {
			if name := strings.TrimSpace(parameters["filename"]); validFileRemoteDownloadName(name) {
				return name
			}
		}
	}
	// Never derive the saved name from the URL path. Signed download URLs often
	// carry bearer material in path segments, while names are persisted in audit
	// events and streamed status. A fixed fallback keeps the complete URL secret.
	return "download"
}

func fileRemoteDownloadError(err error) (string, string) {
	var statusError *remotedownload.StatusError
	switch {
	case errors.Is(err, context.Canceled):
		return "remote_download_cancelled", "停止请求已发送；文件可能已经完成保存，请刷新目标目录确认结果。"
	case errors.Is(err, context.DeadlineExceeded):
		return "remote_download_timeout", "远程下载超过 2 小时，已停止。"
	case errors.Is(err, remotedownload.ErrAddressBlocked):
		return "remote_download_address_blocked", "该地址不是允许访问的公开网络地址。"
	case errors.Is(err, remotedownload.ErrRedirectRejected):
		return "remote_download_redirect_rejected", "远程服务器跳转到了不允许的地址。"
	case errors.Is(err, remotedownload.ErrTLS):
		return "remote_download_tls_failed", "远程服务器证书校验失败。"
	case errors.Is(err, remotedownload.ErrEncoding):
		return "remote_download_encoding_unsupported", "远程服务器返回了不支持的内容编码。"
	case errors.Is(err, remotedownload.ErrPartialContent):
		return "remote_download_partial_unsupported", "远程服务器只返回了部分内容，未保存为完整文件。"
	case errors.Is(err, remotedownload.ErrIdleTimeout):
		return "remote_download_idle_timeout", "远程服务器 45 秒内没有返回数据。"
	case errors.As(err, &statusError):
		return "remote_download_upstream_status", "远程服务器返回 HTTP " + strconv.Itoa(statusError.StatusCode) + "。"
	default:
		return "remote_download_unreachable", "无法连接远程服务器，请检查地址后重试。"
	}
}

func cloneRemoteDownloadChange(source map[string]any) map[string]any {
	result := make(map[string]any, len(source)+2)
	for key, value := range source {
		result[key] = value
	}
	return result
}

type fileRemoteDownloadLimitReader struct {
	source    io.Reader
	remaining int64
	exceeded  atomic.Bool
}

func (r *fileRemoteDownloadLimitReader) Read(buffer []byte) (int, error) {
	if r.exceeded.Load() {
		return 0, filemanager.ErrTooLarge
	}
	limit := int64(len(buffer))
	if limit > r.remaining+1 {
		limit = r.remaining + 1
	}
	count, err := r.source.Read(buffer[:limit])
	if int64(count) > r.remaining {
		allowed := int(r.remaining)
		r.remaining = 0
		r.exceeded.Store(true)
		return allowed, filemanager.ErrTooLarge
	}
	r.remaining -= int64(count)
	return count, err
}

type fileRemoteDownloadUploadBody struct {
	source    io.Reader
	closer    io.Closer
	loaded    *atomic.Int64
	closeOnce sync.Once
	closeErr  error
}

func (body *fileRemoteDownloadUploadBody) Read(buffer []byte) (int, error) {
	count, err := body.source.Read(buffer)
	if count > 0 {
		body.loaded.Add(int64(count))
	}
	return count, err
}

func (body *fileRemoteDownloadUploadBody) Close() error {
	body.closeOnce.Do(func() {
		body.closeErr = body.closer.Close()
	})
	return body.closeErr
}
