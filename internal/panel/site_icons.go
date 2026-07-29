package panel

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
)

const (
	siteIconBrowserCache        = "private, max-age=604800, stale-if-error=604800"
	siteIconMissingBrowserCache = "private, max-age=3600"
)

func (s *Server) handleSiteIcon(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	agentPath, _, allowed := allowedSiteIconPath(r.URL.Path)
	if !allowed {
		s.writeProblem(w, r, http.StatusNotFound, "route_not_found", "Route not found", "")
		return
	}
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		s.writeProblem(w, r, http.StatusBadRequest, "invalid_site_icon_request", "Site icon URL is invalid", "")
		return
	}
	response, err := s.agent.Get(r.Context(), agentPath, "", requestID(r))
	if err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Agent unavailable", "")
		return
	}
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusNotFound {
			w.Header().Set("Cache-Control", siteIconMissingBrowserCache)
		}
		s.writeAgentResponse(w, r, response)
		return
	}
	contentType, valid := normalizedSiteIconContentType(response.ContentType)
	if !valid || len(response.Body) == 0 || len(response.Body) > 256<<10 ||
		!siteIconBodyMatchesContentType(response.Body, contentType) {
		s.writeProblem(w, r, http.StatusBadGateway, "invalid_site_icon", "Agent returned an invalid site icon", "")
		return
	}

	sum := sha256.Sum256(response.Body)
	etag := `"` + hex.EncodeToString(sum[:]) + `"`
	w.Header().Set("Cache-Control", siteIconBrowserCache)
	w.Header().Set("ETag", etag)
	if requestMatchesETag(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(response.Body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response.Body)
}

func allowedSiteIconPath(publicPath string) (agentPath, siteID string, allowed bool) {
	const prefix = "/api/v1/sites/"
	const suffix = "/icon"
	if !strings.HasPrefix(publicPath, prefix) || !strings.HasSuffix(publicPath, suffix) {
		return "", "", false
	}
	siteID = strings.TrimSuffix(strings.TrimPrefix(publicPath, prefix), suffix)
	if strings.Contains(siteID, "/") || !siteIDPattern.MatchString(siteID) {
		return "", "", false
	}
	return "/v1/sites/" + siteID + "/icon", siteID, true
}

func normalizedSiteIconContentType(value string) (string, bool) {
	value = strings.TrimSpace(strings.SplitN(value, ";", 2)[0])
	switch value {
	case "image/png", "image/jpeg", "image/gif", "image/vnd.microsoft.icon", "image/webp":
		return value, true
	default:
		return "", false
	}
}

func siteIconBodyMatchesContentType(body []byte, contentType string) bool {
	switch contentType {
	case "image/png":
		return bytes.HasPrefix(body, []byte("\x89PNG\r\n\x1a\n"))
	case "image/jpeg":
		return bytes.HasPrefix(body, []byte("\xff\xd8\xff"))
	case "image/gif":
		return bytes.HasPrefix(body, []byte("GIF87a")) || bytes.HasPrefix(body, []byte("GIF89a"))
	case "image/vnd.microsoft.icon":
		return len(body) >= 4 && bytes.Equal(body[:4], []byte{0, 0, 1, 0})
	case "image/webp":
		return len(body) >= 12 &&
			bytes.Equal(body[:4], []byte("RIFF")) &&
			bytes.Equal(body[8:12], []byte("WEBP"))
	default:
		return false
	}
}

func requestMatchesETag(header, etag string) bool {
	for _, value := range strings.Split(header, ",") {
		value = strings.TrimSpace(value)
		if value == "*" || value == etag || strings.TrimPrefix(value, "W/") == etag {
			return true
		}
	}
	return false
}
