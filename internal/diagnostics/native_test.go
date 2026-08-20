package diagnostics

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNativeCatalogIsIndependentFromScripts(t *testing.T) {
	catalog := nativeCatalog()
	if len(catalog.Categories) != 1 || catalog.Categories[0].ID != nativeCategoryID {
		t.Fatalf("native categories = %#v", catalog.Categories)
	}
	if len(catalog.Items) < 7 {
		t.Fatalf("native items = %#v", catalog.Items)
	}
	for _, item := range catalog.Items {
		if item.Provider != nativeProvider || !strings.HasPrefix(item.ID, "native-") {
			t.Fatalf("native item is not marked native: %#v", item)
		}
	}
}

func TestMergeCatalogKeepsNativeChecksBeforeOptionalScripts(t *testing.T) {
	merged := mergeCatalogs(nativeCatalog(), Catalog{
		Categories: []Category{{ID: "network", Name: "Network"}},
		Items:      []Check{{ID: "mtr", Category: "network", Name: "MTR", Provider: ""}},
	})
	if len(merged.Items) != len(nativeCatalog().Items)+1 {
		t.Fatalf("merged items = %#v", merged.Items)
	}
	if merged.Items[0].ID != nativeComprehensiveCheckID || merged.Items[len(merged.Items)-1].Provider != "script" {
		t.Fatalf("merged order/provider = %#v", merged.Items)
	}
}

func TestNativeSummaryOnlyContainsProbeMetrics(t *testing.T) {
	summary := nativeSummary([]nativeProbeResult{
		{Dimension: "performance", Metrics: []DiagnosticSummaryMetric{
			{Key: "cpu_score", Value: "123 KPS"},
		}},
		{Dimension: "ip", Metrics: []DiagnosticSummaryMetric{
			{Key: "public_ip", Value: "203.0.113.10"},
			{Key: "quality", Value: "基础信息已采集；未接入第三方信誉库"},
		}},
	})
	if summary == nil || summary.Parser != "kpanel-native-v1" {
		t.Fatalf("native summary = %#v", summary)
	}
	if got := summaryMetricValue(summary, "performance", "cpu_score"); got != "123 KPS" {
		t.Fatalf("cpu score = %q", got)
	}
	if got := summaryMetricValue(summary, "ip", "public_ip"); got != "203.0.113.10" {
		t.Fatalf("public ip = %q", got)
	}
}

func TestNativeLocalProbesReturnMeasuredMetrics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	workspace := t.TempDir()
	for _, checkID := range []string{nativeCPUCheckID, nativeMemoryCheckID, nativeDiskCheckID} {
		result, err := runNativeProbe(ctx, checkID, workspace)
		if err != nil {
			t.Fatalf("runNativeProbe(%s) error = %v", checkID, err)
		}
		if result.Dimension != "performance" || len(result.Metrics) == 0 || len(result.Lines) == 0 {
			t.Fatalf("runNativeProbe(%s) = %#v", checkID, result)
		}
	}
}
