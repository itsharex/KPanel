package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

const maxResponseBytes = 8 << 20

type Client struct {
	httpClient *http.Client
	token      string
}

func New(socketPath, tokenFile string) (*Client, error) {
	if strings.TrimSpace(socketPath) == "" {
		return nil, errors.New("agent socket path is required")
	}

	rawToken, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("read agent token: %w", err)
	}
	token := strings.TrimSpace(string(rawToken))
	if token == "" {
		return nil, errors.New("agent token is empty")
	}

	dialer := &net.Dialer{Timeout: 3 * time.Second}
	transport := &http.Transport{
		DisableCompression: true,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", socketPath)
		},
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second,
		},
		token: token,
	}, nil
}

func (c *Client) Get(ctx context.Context, path string, output any) error {
	return c.Do(ctx, http.MethodGet, path, nil, output)
}

func (c *Client) Do(ctx context.Context, method, path string, input, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode agent request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, "http://agent"+path, body)
	if err != nil {
		return fmt.Errorf("create agent request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Accept", "application/json")
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("agent request: %w", err)
	}
	defer response.Body.Close()

	reader := io.LimitReader(response.Body, maxResponseBytes+1)
	payload, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("read agent response: %w", err)
	}
	if len(payload) > maxResponseBytes {
		return errors.New("agent response exceeds size limit")
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var problem contract.Problem
		if json.Unmarshal(payload, &problem) == nil && problem.Code != "" {
			return &RemoteError{Status: response.StatusCode, Problem: problem}
		}
		return fmt.Errorf("agent returned HTTP %d", response.StatusCode)
	}

	if output == nil || len(payload) == 0 {
		return nil
	}
	if err := json.Unmarshal(payload, output); err != nil {
		return fmt.Errorf("decode agent response: %w", err)
	}
	return nil
}

type RemoteError struct {
	Status  int
	Problem contract.Problem
}

func (e *RemoteError) Error() string {
	if e.Problem.Detail != "" {
		return e.Problem.Code + ": " + e.Problem.Detail
	}
	return e.Problem.Code
}
