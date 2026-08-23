package agent

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiskPartitionRoutesAreAuthenticatedExactAndParameterFree(t *testing.T) {
	server := testServer(t)
	token := "Bearer " + strings.Repeat("x", 32)

	unauthenticated := httptest.NewRequest(http.MethodGet, "/v1/system/disk-partitions", nil)
	unauthenticatedResponse := httptest.NewRecorder()
	server.ServeHTTP(unauthenticatedResponse, unauthenticated)
	if unauthenticatedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d body=%s", unauthenticatedResponse.Code, unauthenticatedResponse.Body.String())
	}

	tests := []struct {
		name       string
		method     string
		path       string
		rawPath    string
		body       string
		wantStatus int
		wantCode   string
	}{
		{
			name: "inspect wrong method", method: http.MethodPost,
			path: "/v1/system/disk-partitions", wantStatus: http.StatusMethodNotAllowed,
			wantCode: "method_not_allowed",
		},
		{
			name: "inspect query", method: http.MethodGet,
			path: "/v1/system/disk-partitions?refresh=1", wantStatus: http.StatusBadRequest,
			wantCode: "invalid_disk_partition_url",
		},
		{
			name: "inspect raw path", method: http.MethodGet,
			path: "/v1/system/disk-partitions", rawPath: "/v1/system/%64isk-partitions",
			wantStatus: http.StatusBadRequest, wantCode: "invalid_disk_partition_url",
		},
		{
			name: "inspect trailing slash", method: http.MethodGet,
			path: "/v1/system/disk-partitions/", wantStatus: http.StatusNotFound,
			wantCode: "not_found",
		},
		{
			name: "action wrong method", method: http.MethodGet,
			path: "/v1/system/disk-partition-actions", wantStatus: http.StatusMethodNotAllowed,
			wantCode: "method_not_allowed",
		},
		{
			name: "action query", method: http.MethodPost,
			path: "/v1/system/disk-partition-actions?force=1", body: `{}`,
			wantStatus: http.StatusBadRequest, wantCode: "invalid_disk_partition_action_url",
		},
		{
			name: "action raw path", method: http.MethodPost,
			path: "/v1/system/disk-partition-actions", rawPath: "/v1/system/disk-partition-%61ctions",
			body: `{}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_disk_partition_action_url",
		},
		{
			name: "action trailing slash", method: http.MethodPost,
			path: "/v1/system/disk-partition-actions/", body: `{}`,
			wantStatus: http.StatusNotFound, wantCode: "not_found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			request.Header.Set("Authorization", token)
			request.Header.Set("Content-Type", "application/json")
			if test.rawPath != "" {
				request.URL.RawPath = test.rawPath
			}
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), test.wantCode) {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestDiskPartitionActionKeepsStrictBodyAndValidationBoundary(t *testing.T) {
	server := testServer(t)
	token := "Bearer " + strings.Repeat("x", 32)
	deviceID := strings.Repeat("a", 64)
	version := strings.Repeat("b", 64)
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{
			name: "unknown field",
			body: `{"action":"check","deviceId":"` + deviceID + `","expectedResourceVersion":"` + version +
				`","unknown":true}`,
			wantStatus: http.StatusBadRequest, wantCode: "invalid_request",
		},
		{
			name: "duplicate field",
			body: `{"action":"check","action":"repair","deviceId":"` + deviceID +
				`","expectedResourceVersion":"` + version + `"}`,
			wantStatus: http.StatusBadRequest, wantCode: "invalid_request",
		},
		{
			name:       "multiple values",
			body:       `{"action":"check","deviceId":"` + deviceID + `","expectedResourceVersion":"` + version + `"} {}`,
			wantStatus: http.StatusBadRequest, wantCode: "invalid_request",
		},
		{
			name: "inapplicable zero field",
			body: `{"action":"check","deviceId":"` + deviceID + `","expectedResourceVersion":"` + version +
				`","persist":false}`,
			wantStatus: http.StatusUnprocessableEntity, wantCode: "invalid_disk_partition_action",
		},
		{
			name: "noncanonical mount point",
			body: `{"action":"mount","deviceId":"` + deviceID + `","expectedResourceVersion":"` + version +
				`","mountPoint":"/mnt/../data"}`,
			wantStatus: http.StatusUnprocessableEntity, wantCode: "invalid_disk_partition_action",
		},
		{
			name: "unsupported filesystem",
			body: `{"action":"format","deviceId":"` + deviceID + `","expectedResourceVersion":"` + version +
				`","filesystem":"btrfs"}`,
			wantStatus: http.StatusUnprocessableEntity, wantCode: "invalid_disk_partition_action",
		},
		{
			name:       "valid request reaches disabled boundary",
			body:       `{"action":"check","deviceId":"` + deviceID + `","expectedResourceVersion":"` + version + `"}`,
			wantStatus: http.StatusForbidden, wantCode: "disk_partition_write_disabled",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(
				http.MethodPost, "/v1/system/disk-partition-actions", strings.NewReader(test.body),
			)
			request.Header.Set("Authorization", token)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), test.wantCode) {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}
