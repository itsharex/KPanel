package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/systeminfo"
	"github.com/kejilion/kejilion-panel/internal/version"
)

const (
	lightTokenPrefix  = "kpl1."
	lightEnrollPath   = "/api/v3/federation/light/enroll"
	lightReportPath   = "/api/v3/federation/light/report"
	lightProtocol     = "light-v1"
	defaultConfigPath = "/etc/kejilion-node/node.json"
	maxResponseBytes  = int64(64 << 10)
)

type nodeConfig struct {
	SchemaVersion  int    `json:"schemaVersion"`
	Origin         string `json:"origin"`
	NodeID         string `json:"nodeId"`
	ReportingKey   string `json:"reportingKey"`
	ReportInterval int    `json:"reportIntervalSeconds"`
}

type tokenWire struct {
	Version   int    `json:"v"`
	Origin    string `json:"origin"`
	ID        string `json:"id"`
	Secret    string `json:"secret"`
	ExpiresAt int64  `json:"expiresAt"`
}

type enrollRequest struct {
	Token       string `json:"token"`
	Name        string `json:"name,omitempty"`
	NodeVersion string `json:"nodeVersion"`
}

type enrollResponse struct {
	NodeID         string `json:"nodeId"`
	ReportingKey   string `json:"reportingKey"`
	ReportInterval int    `json:"reportIntervalSeconds"`
}

type reportRequest struct {
	Telemetry contract.HostTelemetry `json:"telemetry"`
}

type reportResponse struct {
	NextReport int `json:"nextReportSeconds"`
}

var nodeHTTPClient = newHTTPClient()

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("kejilion-node stopped", "error", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) == 1 && arguments[0] == "version" {
		fmt.Printf("%s %s\n", version.Version, lightProtocol)
		return nil
	}
	if len(arguments) == 0 {
		return errors.New("expected enroll, run, or version")
	}
	switch arguments[0] {
	case "enroll":
		return runEnroll(arguments[1:])
	case "run":
		return runNode(arguments[1:])
	default:
		return errors.New("unsupported kejilion-node command")
	}
}

func runEnroll(arguments []string) error {
	flags := flag.NewFlagSet("kejilion-node enroll", flag.ContinueOnError)
	token := flags.String("token", "", "one-time enrollment token")
	name := flags.String("name", "", "optional display name")
	configPath := flags.String("config", defaultConfigPath, "configuration path")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected enrollment argument")
	}
	origin, err := originFromToken(*token)
	if err != nil {
		return err
	}
	request := enrollRequest{Token: strings.TrimSpace(*token), Name: strings.TrimSpace(*name), NodeVersion: version.Version}
	var response enrollResponse
	if err := postJSON(context.Background(), origin+lightEnrollPath, request, nil, &response); err != nil {
		return fmt.Errorf("enroll lightweight node: %w", err)
	}
	if !validHexID(response.NodeID) || response.ReportInterval < 20 || response.ReportInterval > 600 {
		return errors.New("enrollment response is invalid")
	}
	key, err := base64.RawURLEncoding.DecodeString(response.ReportingKey)
	if err != nil || len(key) != 32 {
		return errors.New("enrollment credential is invalid")
	}
	config := nodeConfig{
		SchemaVersion: 1, Origin: origin, NodeID: response.NodeID,
		ReportingKey: response.ReportingKey, ReportInterval: response.ReportInterval,
	}
	if err := writeConfigAtomic(*configPath, config); err != nil {
		return err
	}
	fmt.Printf("KPanel lightweight node enrolled: %s\n", response.NodeID)
	return nil
}

func runNode(arguments []string) error {
	flags := flag.NewFlagSet("kejilion-node run", flag.ContinueOnError)
	configPath := flags.String("config", defaultConfigPath, "configuration path")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected run argument")
	}
	config, secret, err := readConfig(*configPath)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	collector := systeminfo.NewCollector()
	collector.PublicNetworkCacheTTL = 30 * time.Minute
	interval := time.Duration(config.ReportInterval) * time.Second
	backoff := time.Second
	for {
		err := collectAndReport(ctx, collector, config, secret)
		if err == nil {
			backoff = time.Second
			if !waitContext(ctx, interval) {
				return nil
			}
			continue
		}
		slog.Warn("lightweight report failed", "error", err)
		if !waitContext(ctx, backoff) {
			return nil
		}
		if backoff < interval {
			backoff *= 2
			if backoff > interval {
				backoff = interval
			}
		}
	}
}

func collectAndReport(parent context.Context, collector *systeminfo.Collector, config nodeConfig, secret []byte) error {
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	summary, collectErr := collector.Collect(ctx)
	if collectErr != nil && summary.Hostname == "" {
		return fmt.Errorf("collect host telemetry: %w", collectErr)
	}
	disk := contract.DiskCapacitySummary{}
	if len(summary.Disks) > 0 {
		selected := summary.Disks[0]
		for _, candidate := range summary.Disks {
			if candidate.MountPoint == "/" || (selected.MountPoint != "/" && candidate.TotalBytes > selected.TotalBytes) {
				selected = candidate
			}
		}
		disk = contract.DiskCapacitySummary{TotalBytes: selected.TotalBytes, UsedBytes: selected.UsedBytes, UsagePercent: selected.UsagePercent}
	}
	payload := reportRequest{Telemetry: contract.HostTelemetry{
		AgentVersion: version.Version, AgentProtocolVersion: lightProtocol,
		Hostname: summary.Hostname, OS: summary.OS, OSID: summary.OSID, OSLike: summary.OSLike,
		Kernel: summary.Kernel, Architecture: summary.Architecture, UptimeSeconds: summary.UptimeSeconds,
		Load: summary.Load, CPU: summary.CPU, Memory: summary.Memory, Disk: disk,
		Network: summary.Network, PublicNetwork: summary.PublicNetwork, CollectedAt: summary.CollectedAt,
	}}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	timestamp := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	requestID, err := randomHex(16)
	if err != nil {
		return err
	}
	bodyHash := sha256.Sum256(body)
	material := strings.Join([]string{"POST", lightReportPath, config.NodeID, timestamp, requestID, hex.EncodeToString(bodyHash[:])}, "\n")
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(material))
	headers := map[string]string{
		"X-KPanel-Light-Node-ID": config.NodeID,
		"X-KPanel-Timestamp":     timestamp,
		"X-KPanel-Request-ID":    requestID,
		"X-KPanel-Signature":     base64.RawURLEncoding.EncodeToString(mac.Sum(nil)),
	}
	var response reportResponse
	return postRawJSON(ctx, config.Origin+lightReportPath, body, headers, &response)
}

func originFromToken(token string) (string, error) {
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, lightTokenPrefix) || len(token) > 2048 {
		return "", errors.New("enrollment token is invalid")
	}
	content, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(token, lightTokenPrefix))
	if err != nil || len(content) > 1536 {
		return "", errors.New("enrollment token is invalid")
	}
	var wire tokenWire
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return "", errors.New("enrollment token is invalid")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return "", errors.New("enrollment token is invalid")
	}
	now := time.Now().UTC()
	expiresAt := time.Unix(wire.ExpiresAt, 0).UTC()
	if wire.Version != 1 || !validHexID(wire.ID) || !expiresAt.After(now) || expiresAt.After(now.Add(10*time.Minute)) {
		return "", errors.New("enrollment token is invalid or expired")
	}
	secret, err := base64.RawURLEncoding.DecodeString(wire.Secret)
	if err != nil || len(secret) != 32 {
		return "", errors.New("enrollment token is invalid")
	}
	return validateHTTPSOrigin(wire.Origin)
}

func readConfig(path string) (nodeConfig, []byte, error) {
	if !filepath.IsAbs(path) {
		return nodeConfig{}, nil, errors.New("configuration path must be absolute")
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() {
		return nodeConfig{}, nil, errors.New("configuration file is unavailable")
	}
	file, err := os.Open(path)
	if err != nil {
		return nodeConfig{}, nil, err
	}
	defer file.Close()
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) || after.Size() > 8192 {
		return nodeConfig{}, nil, errors.New("configuration file is invalid")
	}
	content, err := io.ReadAll(io.LimitReader(file, 8193))
	if err != nil || len(content) > 8192 {
		return nodeConfig{}, nil, errors.New("configuration file is invalid")
	}
	var config nodeConfig
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return nodeConfig{}, nil, errors.New("configuration file is invalid")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nodeConfig{}, nil, errors.New("configuration file is invalid")
	}
	origin, err := validateHTTPSOrigin(config.Origin)
	key, keyErr := base64.RawURLEncoding.DecodeString(config.ReportingKey)
	if err != nil || keyErr != nil || origin != config.Origin || config.SchemaVersion != 1 ||
		!validHexID(config.NodeID) || len(key) != 32 || config.ReportInterval < 20 || config.ReportInterval > 600 {
		return nodeConfig{}, nil, errors.New("configuration file is invalid")
	}
	return config, key, nil
}

func writeConfigAtomic(path string, config nodeConfig) error {
	if !filepath.IsAbs(path) {
		return errors.New("configuration path must be absolute")
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return errors.New("configuration target is not a regular file")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return err
	}
	content, err := json.Marshal(config)
	if err != nil {
		return err
	}
	temporary, err := os.OpenFile(filepath.Join(directory, ".node.json.tmp-"+strconv.FormatInt(time.Now().UnixNano(), 10)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := temporary.Write(append(content, '\n')); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		directoryHandle, err := os.Open(directory)
		if err != nil {
			return err
		}
		defer directoryHandle.Close()
		if err := directoryHandle.Sync(); err != nil {
			return err
		}
	}
	return nil
}

func postJSON(ctx context.Context, endpoint string, input any, headers map[string]string, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return postRawJSON(ctx, endpoint, body, headers, output)
}

func postRawJSON(ctx context.Context, endpoint string, body []byte, headers map[string]string, output any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "KPanel-Light-Node/"+version.Version)
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := nodeHTTPClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	content, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil || int64(len(content)) > maxResponseBytes {
		return errors.New("server response is invalid")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("server returned HTTP %d", response.StatusCode)
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return errors.New("server response is invalid")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("server response is invalid")
	}
	return nil
}

func newHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	transport.MaxIdleConns = 4
	transport.MaxIdleConnsPerHost = 2
	return &http.Client{
		Transport: transport, Timeout: 20 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func validateHTTPSOrigin(value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Hostname() == "" ||
		parsed.User != nil || parsed.Opaque != "" || parsed.Path != "" || parsed.RawPath != "" ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.ForceQuery || strings.ContainsAny(parsed.Host, "\r\n\t ") {
		return "", errors.New("KPanel origin must be an HTTPS origin")
	}
	return value, nil
}

func validHexID(value string) bool {
	if len(value) != 32 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 16
}

func randomHex(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func waitContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
