package panel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type AgentClient struct {
	socketPath string
	tokenFile  string
	maxBytes   int64
	client     *http.Client
}

type AgentResponse struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

func NewAgentClient(socketPath, tokenFile string, maxBytes int64) *AgentClient {
	transport := &http.Transport{
		DisableCompression: true,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			dialer := net.Dialer{Timeout: 3 * time.Second}
			return dialer.DialContext(ctx, "unix", socketPath)
		},
	}
	return &AgentClient{
		socketPath: socketPath,
		tokenFile:  tokenFile,
		maxBytes:   maxBytes,
		client: &http.Client{
			Transport: transport,
			Timeout:   20 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *AgentClient) Get(ctx context.Context, path, rawQuery, requestID string) (AgentResponse, error) {
	token, err := readSmallSecret(c.tokenFile)
	if err != nil {
		return AgentResponse{}, fmt.Errorf("read agent credential: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix"+path, nil)
	if err != nil {
		return AgentResponse{}, fmt.Errorf("create agent request: %w", err)
	}
	request.URL.RawQuery = rawQuery
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	if requestID != "" {
		request.Header.Set("X-Request-ID", requestID)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return AgentResponse{}, fmt.Errorf("call agent: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, c.maxBytes+1))
	if err != nil {
		return AgentResponse{}, fmt.Errorf("read agent response: %w", err)
	}
	if int64(len(body)) > c.maxBytes {
		return AgentResponse{}, errors.New("agent response exceeds configured limit")
	}
	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json; charset=utf-8"
	}
	return AgentResponse{
		StatusCode:  response.StatusCode,
		ContentType: contentType,
		Body:        body,
	}, nil
}

func readSmallSecret(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, 4097))
	if err != nil {
		return "", err
	}
	if len(content) > 4096 {
		return "", errors.New("secret file exceeds 4 KiB")
	}
	value := strings.TrimSpace(string(content))
	if value == "" {
		return "", errors.New("secret file is empty")
	}
	return value, nil
}
