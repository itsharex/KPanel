//go:build linux

package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/systeminfo"
)

func TestSystemProcessesReturnsBoundedLiveQuery(t *testing.T) {
	server := testServer(t)
	request := httptest.NewRequest(http.MethodGet, "/v1/system/processes?sort=pid&order=asc&limit=1", nil)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var snapshot systeminfo.ProcessSnapshot
	if err := json.Unmarshal(response.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Items) != 1 || snapshot.Total < 1 || snapshot.Items[0].PID < 1 ||
		snapshot.Items[0].StartTimeTicks == 0 {
		t.Fatalf("unexpected live process snapshot: %#v", snapshot)
	}
}
