package httpstream

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type deadlineWriter struct {
	header         http.Header
	body           strings.Builder
	readDeadlines  int
	writeDeadlines int
}

func (writer *deadlineWriter) Header() http.Header {
	if writer.header == nil {
		writer.header = make(http.Header)
	}
	return writer.header
}

func (writer *deadlineWriter) WriteHeader(int) {}

func (writer *deadlineWriter) Write(buffer []byte) (int, error) {
	return writer.body.Write(buffer)
}

func (writer *deadlineWriter) SetReadDeadline(time.Time) error {
	writer.readDeadlines++
	return nil
}

func (writer *deadlineWriter) SetWriteDeadline(time.Time) error {
	writer.writeDeadlines++
	return nil
}

func TestIdleReaderAndWriterRefreshDeadlines(t *testing.T) {
	writer := &deadlineWriter{}
	reader := NewIdleReader(context.Background(), writer, strings.NewReader("payload"), time.Second)
	content, err := io.ReadAll(reader)
	if err != nil || string(content) != "payload" {
		t.Fatalf("read = %q, err=%v", content, err)
	}
	response := NewIdleResponseWriter(context.Background(), writer, time.Second)
	if _, err := response.Write([]byte("response")); err != nil {
		t.Fatal(err)
	}
	if writer.readDeadlines < 1 || writer.writeDeadlines != 1 {
		t.Fatalf(
			"deadline refreshes read=%d write=%d",
			writer.readDeadlines,
			writer.writeDeadlines,
		)
	}
}

type slowBody struct {
	mu     sync.Mutex
	chunks int
}

func (body *slowBody) Read(buffer []byte) (int, error) {
	body.mu.Lock()
	defer body.mu.Unlock()
	if body.chunks == 0 {
		return 0, io.EOF
	}
	time.Sleep(20 * time.Millisecond)
	body.chunks--
	return copy(buffer, "chunk"), nil
}

func TestIdleReaderOverridesShortServerReadTimeout(t *testing.T) {
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		content, err := io.ReadAll(NewIdleReader(
			request.Context(),
			writer,
			request.Body,
			80*time.Millisecond,
		))
		if err != nil {
			http.Error(writer, err.Error(), http.StatusRequestTimeout)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
		if len(content) != 5*5 {
			t.Errorf("received %d bytes", len(content))
		}
	})
	server := httptest.NewUnstartedServer(handler)
	server.Config.ReadTimeout = 30 * time.Millisecond
	server.Start()
	defer server.Close()

	request, err := http.NewRequest(http.MethodPost, server.URL, &slowBody{chunks: 5})
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		content, _ := io.ReadAll(response.Body)
		t.Fatalf("status=%d body=%s", response.StatusCode, content)
	}
}

func TestIdleWriterOverridesShortServerWriteTimeout(t *testing.T) {
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		stream := NewIdleResponseWriter(request.Context(), writer, 80*time.Millisecond)
		for range 5 {
			time.Sleep(20 * time.Millisecond)
			if _, err := stream.Write([]byte("chunk")); err != nil {
				return
			}
		}
	})
	server := httptest.NewUnstartedServer(handler)
	server.Config.WriteTimeout = 30 * time.Millisecond
	server.Start()
	defer server.Close()

	response, err := server.Client().Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	content, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || len(content) != 5*5 {
		t.Fatalf("status=%d bytes=%d", response.StatusCode, len(content))
	}
}
