package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/filemanager"
)

func TestFileEndpointsListWriteUploadAndRejectProtectedPaths(t *testing.T) {
	server := testServer(t)
	root := t.TempDir()
	manager, err := filemanager.New(filemanager.Config{
		Root: root, ProtectedVirtual: []string{"/docker/kpanel", "/.kpanel-trash"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	server.files = manager
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	response := fileRequest(server, http.MethodGet, "/v1/files?path=%2F", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"hello.txt"`) {
		t.Fatalf("list status=%d body=%s", response.Code, response.Body.String())
	}
	var listing contract.FileDirectory
	if err := json.Unmarshal(response.Body.Bytes(), &listing); err != nil {
		t.Fatal(err)
	}
	version := listing.Entries[0].ResourceVersion

	body := `{"content":"new","expectedResourceVersion":"` + version + `"}`
	response = fileRequest(server, http.MethodPut, "/v1/files/content?path=%2Fhello.txt", body)
	if response.Code != http.StatusOK {
		t.Fatalf("write status=%d body=%s", response.Code, response.Body.String())
	}
	response = fileRequest(
		server, http.MethodGet,
		"/v1/files/content?path=%2Fhello.txt&disposition=inline&mode=text", "",
	)
	if response.Code != http.StatusOK || response.Body.String() != "new" {
		t.Fatalf("text read status=%d body=%q", response.Code, response.Body.String())
	}

	response = fileRequest(server, http.MethodPost, "/v1/files/upload?path=%2F&name=upload.txt", "uploaded")
	if response.Code != http.StatusCreated {
		t.Fatalf("upload status=%d body=%s", response.Code, response.Body.String())
	}

	response = fileRequest(server, http.MethodGet, "/v1/files?path=%2Fdocker%2Fkpanel", "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("protected status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestFileTextModeRejectsBinaryContentAndMapsBodyLimit(t *testing.T) {
	server := testServer(t)
	root := t.TempDir()
	manager, err := filemanager.New(filemanager.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	server.files = manager
	if err := os.WriteFile(filepath.Join(root, "binary.txt"), []byte{0xff, 0x00, 0xfe}, 0644); err != nil {
		t.Fatal(err)
	}
	response := fileRequest(
		server, http.MethodGet,
		"/v1/files/content?path=%2Fbinary.txt&disposition=inline&mode=text", "",
	)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("binary text status=%d body=%s", response.Code, response.Body.String())
	}

	limited := httptest.NewRecorder()
	writeFileProblem(limited, "request-id", &http.MaxBytesError{Limit: filemanager.MaxUploadBytes})
	if limited.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("body limit status=%d body=%s", limited.Code, limited.Body.String())
	}
}

func TestFileEndpointsRejectUnknownQueryAndOversizedBatch(t *testing.T) {
	server := testServer(t)
	root := t.TempDir()
	manager, err := filemanager.New(filemanager.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	server.files = manager
	response := fileRequest(server, http.MethodGet, "/v1/files?path=%2F&extra=1", "")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("query status=%d body=%s", response.Code, response.Body.String())
	}
	sources := make([]string, filemanager.MaxBatchItems+1)
	for index := range sources {
		sources[index] = "/file"
	}
	content, _ := json.Marshal(contract.FileActionRequest{Action: "trash", Sources: sources})
	response = fileRequest(server, http.MethodPost, "/v1/files/actions", string(content))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("batch status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestFileActionReturnsMultiStatusForPartialResult(t *testing.T) {
	server := testServer(t)
	root := t.TempDir()
	manager, err := filemanager.New(filemanager.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	server.files = manager
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	content, _ := json.Marshal(contract.FileActionRequest{
		Action: "trash", Sources: []string{"/keep.txt", "/missing.txt"},
	})
	response := fileRequest(server, http.MethodPost, "/v1/files/actions", string(content))
	if response.Code != http.StatusMultiStatus {
		t.Fatalf("partial action status=%d body=%s", response.Code, response.Body.String())
	}
	var result contract.FileActionResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Succeeded) != 1 || len(result.Failed) != 1 {
		t.Fatalf("partial result=%#v", result)
	}
}

func fileRequest(server *Server, method, target, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	switch target {
	case "/v1/files/upload?path=%2F&name=upload.txt":
		request.Header.Set("Content-Type", "application/octet-stream")
	default:
		if body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
	}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	return response
}
