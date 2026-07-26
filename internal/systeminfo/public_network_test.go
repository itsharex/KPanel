package systeminfo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func TestReadPublicNetworkCachesSuccessfulLookup(t *testing.T) {
	now := time.Date(2026, 7, 26, 8, 0, 0, 0, time.UTC)
	calls := 0
	collector := &Collector{
		Now:                   func() time.Time { return now },
		PublicNetworkCacheTTL: time.Hour,
		PublicNetworkLookup: func(context.Context) (contract.PublicNetworkSummary, error) {
			calls++
			return contract.PublicNetworkSummary{
				IPv4: "203.0.113.10", ISP: "AS64500 Example Network", Country: "CN", City: "Shanghai",
			}, nil
		},
	}

	first := collector.readPublicNetwork(context.Background())
	second := collector.readPublicNetwork(context.Background())
	if calls != 1 || first.IPv4 != "203.0.113.10" || second.IPv4 != first.IPv4 {
		t.Fatalf("unexpected cached lookup: calls=%d first=%#v second=%#v", calls, first, second)
	}
	if first.Source != "ipinfo.io" || first.UpdatedAt == nil || !first.UpdatedAt.Equal(now) {
		t.Fatalf("lookup metadata was not normalized: %#v", first)
	}

	now = now.Add(2 * time.Hour)
	_ = collector.readPublicNetwork(context.Background())
	if calls != 2 {
		t.Fatalf("expired cache was not refreshed: calls=%d", calls)
	}
}

func TestReadPublicNetworkCachesFailureBriefly(t *testing.T) {
	now := time.Date(2026, 7, 26, 8, 0, 0, 0, time.UTC)
	calls := 0
	collector := &Collector{
		Now:                   func() time.Time { return now },
		PublicNetworkCacheTTL: 30 * time.Minute,
		PublicNetworkLookup: func(context.Context) (contract.PublicNetworkSummary, error) {
			calls++
			return contract.PublicNetworkSummary{}, errors.New("offline")
		},
	}

	if got := collector.readPublicNetwork(context.Background()); got.Source != "" {
		t.Fatalf("failed lookup returned data: %#v", got)
	}
	_ = collector.readPublicNetwork(context.Background())
	if calls != 1 {
		t.Fatalf("failed lookup was repeated without backoff: calls=%d", calls)
	}

	now = now.Add(6 * time.Minute)
	_ = collector.readPublicNetwork(context.Background())
	if calls != 2 {
		t.Fatalf("failed lookup was not retried after backoff: calls=%d", calls)
	}
}

func TestQueryIPInfoValidatesAddressFamilyAndBoundsMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{
			"ip":"203.0.113.8",
			"city":"Shanghai",
			"region":"Shanghai",
			"country":"CN",
			"org":"AS64500 Example Network",
			"timezone":"Asia/Shanghai"
		}`))
	}))
	defer server.Close()

	info, err := queryIPInfo(context.Background(), server.Client(), server.URL, 4)
	if err != nil || info.IP != "203.0.113.8" || info.Org != "AS64500 Example Network" {
		t.Fatalf("queryIPInfo() = %#v, %v", info, err)
	}
	if _, err := queryIPInfo(context.Background(), server.Client(), server.URL, 6); err == nil {
		t.Fatal("IPv4 response was accepted as IPv6")
	}
	if got := cleanPublicField("  ISP\nName\x00  ", 16); got != "ISPName" {
		t.Fatalf("cleanPublicField() = %q", got)
	}
}

func TestQueryIPInfoRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write(make([]byte, (64<<10)+1))
	}))
	defer server.Close()

	if _, err := queryIPInfo(context.Background(), server.Client(), server.URL, 4); err == nil {
		t.Fatal("oversized response was accepted")
	}
}

func TestMergeIPInfoMetadataSupportsASNObject(t *testing.T) {
	var info ipInfoResponse
	info.IP = "2001:db8::1"
	info.CountryCode = "US"
	info.ASN.Name = "Example Transit"
	var summary contract.PublicNetworkSummary
	mergeIPInfoMetadata(&summary, info)
	if summary.ISP != "Example Transit" || summary.Country != "US" {
		t.Fatalf("unexpected merged metadata: %#v", summary)
	}
}
