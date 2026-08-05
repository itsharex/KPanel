package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/monitoring"
)

type fakeMonitoringProvider struct {
	rangeValue string
	result     contract.MonitoringHistory
	err        error
}

func (provider *fakeMonitoringProvider) History(
	_ context.Context,
	rangeValue string,
) (contract.MonitoringHistory, error) {
	provider.rangeValue = rangeValue
	return provider.result, provider.err
}

func TestMonitoringHistoryRequiresAuthAndStrictRange(t *testing.T) {
	server := testServer(t)
	provider := &fakeMonitoringProvider{
		result: contract.MonitoringHistory{Range: "6h"},
	}
	server.monitoring = provider

	request := httptest.NewRequest(http.MethodGet, "/v1/monitoring/history?range=6h", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", response.Code)
	}

	for _, path := range []string{
		"/v1/monitoring/history?range=31d",
		"/v1/monitoring/history?range=6h&range=24h",
		"/v1/monitoring/history?range=6h&unknown=1",
	} {
		request = httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
		response = httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusUnprocessableEntity {
			t.Fatalf("invalid query %q status = %d body=%s", path, response.Code, response.Body.String())
		}
	}

	request = httptest.NewRequest(http.MethodGet, "/v1/monitoring/history?range=6h", nil)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || provider.rangeValue != "6h" {
		t.Fatalf("history status=%d range=%q body=%s", response.Code, provider.rangeValue, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/v1/monitoring/history?range=30d", nil)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || provider.rangeValue != "30d" {
		t.Fatalf("30-day history status=%d range=%q body=%s", response.Code, provider.rangeValue, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/v1/monitoring/history?range=12m", nil)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || provider.rangeValue != "12m" {
		t.Fatalf("12-month history status=%d range=%q body=%s", response.Code, provider.rangeValue, response.Body.String())
	}
}

func TestMonitoringHistoryMapsBusyAndUnavailable(t *testing.T) {
	server := testServer(t)
	provider := &fakeMonitoringProvider{err: monitoring.ErrBusy}
	server.monitoring = provider
	request := httptest.NewRequest(http.MethodGet, "/v1/monitoring/history", nil)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("busy status = %d body=%s", response.Code, response.Body.String())
	}

	server.monitoring = nil
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable status = %d body=%s", response.Code, response.Body.String())
	}
}
