package agent

import (
	"encoding/json"
	"errors"
	"fmt"
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

func TestFileListSupportsBoundedSearchAndPagination(t *testing.T) {
	server := testServer(t)
	root := t.TempDir()
	manager, err := filemanager.New(filemanager.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	server.files = manager
	for index := range 510 {
		name := fmt.Sprintf("item-%03d.txt", index)
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "wanted.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	response := fileRequest(
		server,
		http.MethodGet,
		"/v1/files?path=%2F&limit=100&offset=500",
		"",
	)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"offset":500`) {
		t.Fatalf("page status=%d body=%s", response.Code, response.Body.String())
	}
	response = fileRequest(server, http.MethodGet, "/v1/files?path=%2F&search=wanted", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"wanted.txt"`) {
		t.Fatalf("search status=%d body=%s", response.Code, response.Body.String())
	}
	response = fileRequest(server, http.MethodGet, "/v1/files?path=%2F&offset=20000", "")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid offset status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestFileBusyMapsToTooManyRequests(t *testing.T) {
	response := httptest.NewRecorder()
	writeFileProblem(response, "request-id", filemanager.ErrBusy)
	if response.Code != http.StatusTooManyRequests ||
		!strings.Contains(response.Body.String(), `"file_transfer_busy"`) {
		t.Fatalf("busy status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestInvalidArchiveMapsToUnprocessableEntity(t *testing.T) {
	response := httptest.NewRecorder()
	writeFileProblem(response, "request-id", filemanager.ErrInvalidArchive)
	if response.Code != http.StatusUnprocessableEntity ||
		!strings.Contains(response.Body.String(), `"file_archive_invalid"`) {
		t.Fatalf("invalid archive status=%d body=%s", response.Code, response.Body.String())
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

func TestFileArchiveActionsRoundTripThroughAgentAPI(t *testing.T) {
	server := testServer(t)
	root := t.TempDir()
	manager, err := filemanager.New(filemanager.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	server.files = manager
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	compressBody, _ := json.Marshal(contract.FileActionRequest{
		Action: "compress", Sources: []string{"/hello.txt"}, Target: "/",
		Name: "hello.tar.gz", Format: "tar.gz",
	})
	compressResponse := fileRequest(server, http.MethodPost, "/v1/files/actions", string(compressBody))
	if compressResponse.Code != http.StatusOK {
		t.Fatalf("compress status=%d body=%s", compressResponse.Code, compressResponse.Body.String())
	}
	var compressed contract.FileActionResult
	if err := json.Unmarshal(compressResponse.Body.Bytes(), &compressed); err != nil || len(compressed.Succeeded) != 1 {
		t.Fatalf("compress result=%#v err=%v", compressed, err)
	}
	extractBody, _ := json.Marshal(contract.FileActionRequest{
		Action: "extract", Sources: []string{"/hello.tar.gz"}, Target: "/",
		Name: "restored", Format: "tar.gz",
		ExpectedResourceVersion: compressed.Succeeded[0].ResourceVersion,
	})
	extractResponse := fileRequest(server, http.MethodPost, "/v1/files/actions", string(extractBody))
	if extractResponse.Code != http.StatusOK {
		t.Fatalf("extract status=%d body=%s", extractResponse.Code, extractResponse.Body.String())
	}
	content, err := os.ReadFile(filepath.Join(root, "restored", "hello.txt"))
	if err != nil || string(content) != "hello" {
		t.Fatalf("restored content=%q err=%v", content, err)
	}
}

func TestFileTrashEndpointSupportsRestoreAndPermanentDelete(t *testing.T) {
	server := testServer(t)
	root := t.TempDir()
	manager, err := filemanager.New(filemanager.Config{Root: root, TrashVirtual: "/state/file-trash"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	server.files = manager
	for _, name := range []string{"restore.txt", "delete.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
		content, _ := json.Marshal(contract.FileActionRequest{Action: "trash", Sources: []string{"/" + name}})
		response := fileRequest(server, http.MethodPost, "/v1/files/actions", string(content))
		if response.Code != http.StatusOK {
			t.Fatalf("trash %s status=%d body=%s", name, response.Code, response.Body.String())
		}
	}
	response := fileRequest(server, http.MethodGet, "/v1/files/trash", "")
	if response.Code != http.StatusOK {
		t.Fatalf("trash list status=%d body=%s", response.Code, response.Body.String())
	}
	var trash contract.FileTrashDirectory
	if err := json.Unmarshal(response.Body.Bytes(), &trash); err != nil || trash.Total != 2 {
		t.Fatalf("trash list=%#v err=%v", trash, err)
	}
	for _, entry := range trash.Entries {
		action := "trash_delete"
		if entry.Name == "restore.txt" {
			action = "trash_restore"
		}
		content, _ := json.Marshal(contract.FileActionRequest{
			Action: action, TrashIDs: []string{entry.ID},
			ExpectedResourceVersions: map[string]string{entry.ID: entry.ResourceVersion},
		})
		response = fileRequest(server, http.MethodPost, "/v1/files/actions", string(content))
		if response.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", action, response.Code, response.Body.String())
		}
	}
	if content, err := os.ReadFile(filepath.Join(root, "restore.txt")); err != nil || string(content) != "restore.txt" {
		t.Fatalf("restored content=%q err=%v", content, err)
	}
	if _, err := os.Stat(filepath.Join(root, "delete.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("permanently deleted file remains: %v", err)
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
