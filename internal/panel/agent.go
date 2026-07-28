package panel

import (
	"bytes"
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
			// WordPress and declarative application installers may need to pull
			// a pinned package or image. The inbound request context still
			// cancels ordinary calls immediately.
			Timeout: 12 * time.Minute,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *AgentClient) Get(ctx context.Context, path, rawQuery, requestID string) (AgentResponse, error) {
	return c.Do(ctx, http.MethodGet, path, rawQuery, requestID, nil)
}

func (c *AgentClient) Open(ctx context.Context, method, path, rawQuery, requestID string) (*http.Response, error) {
	token, err := readSmallSecret(c.tokenFile)
	if err != nil {
		return nil, fmt.Errorf("read agent credential: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, method, "http://unix"+path, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create agent request: %w", err)
	}
	request.URL.RawQuery = rawQuery
	request.Header.Set("Authorization", "Bearer "+token)
	if requestID != "" {
		request.Header.Set("X-Request-ID", requestID)
	}
	streamClient := &http.Client{
		Transport: c.client.Transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := streamClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call agent: %w", err)
	}
	return response, nil
}

func (c *AgentClient) Do(ctx context.Context, method, path, rawQuery, requestID string, body []byte) (AgentResponse, error) {
	token, err := readSmallSecret(c.tokenFile)
	if err != nil {
		return AgentResponse{}, fmt.Errorf("read agent credential: %w", err)
	}
	var requestBody io.Reader = http.NoBody
	if len(body) > 0 {
		requestBody = bytes.NewReader(body)
	}
	request, err := http.NewRequestWithContext(ctx, method, "http://unix"+path, requestBody)
	if err != nil {
		return AgentResponse{}, fmt.Errorf("create agent request: %w", err)
	}
	request.URL.RawQuery = rawQuery
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	if requestID != "" {
		request.Header.Set("X-Request-ID", requestID)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return AgentResponse{}, fmt.Errorf("call agent: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, c.maxBytes+1))
	if err != nil {
		return AgentResponse{}, fmt.Errorf("read agent response: %w", err)
	}
	if int64(len(responseBody)) > c.maxBytes {
		return AgentResponse{}, errors.New("agent response exceeds configured limit")
	}
	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json; charset=utf-8"
	}
	return AgentResponse{
		StatusCode:  response.StatusCode,
		ContentType: contentType,
		Body:        responseBody,
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
