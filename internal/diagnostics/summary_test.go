package diagnostics

import (
	"strings"
	"testing"
)

func TestDiagnosticSummaryFromNodeQualityText(t *testing.T) {
	output := []byte(`
Basic System Information:
---------------------------------
Processor   : AMD EPYC 7B12 64-Core Processor
CPU cores   : 4
RAM         : 3.84 GiB
Disk        : 80.00 GiB

Geekbench 5 Benchmark Test:
Single Core | 1234
Multi Core  | 2345

Fio Disk Speed Tests (Mixed R/W 50/50):
read  | 2.50 GiB/s
write | 1.25 GiB/s
total | 3.75 GiB/s

Iperf3 Network Speed Tests:
Provider | Location | Send Speed | Recv Speed | Latency
Example  | Tokyo    | 800 Mbps   | 700 Mbps   | 25 ms

Report URL: https://nodequality.com/r/abc123
`)

	summary := diagnosticSummaryFromOutput("nodequality", "NodeQuality 综合测评", output)
	if summary == nil {
		t.Fatal("expected a summary")
	}
	if summary.ReportURL != "https://nodequality.com/r/abc123" {
		t.Fatalf("report url = %q", summary.ReportURL)
	}
	for _, check := range []struct {
		dimension string
		key       string
		want      string
	}{
		{"performance", "cpu_model", "AMD EPYC 7B12 64-Core Processor"},
		{"performance", "cpu_cores", "4"},
		{"performance", "memory", "3.84 GiB"},
		{"performance", "disk_read", "2.50 GiB/s"},
		{"performance", "geekbench_single", "1234"},
		{"speed", "upload", "800 Mbps"},
		{"speed", "download", "700 Mbps"},
		{"latency", "average", "25 ms"},
	} {
		if got := summaryMetricValue(summary, check.dimension, check.key); got != check.want {
			t.Errorf("%s.%s = %q, want %q", check.dimension, check.key, got, check.want)
		}
	}
}

func TestDiagnosticSummaryFromYABSJSON(t *testing.T) {
	output := []byte(`YABS JSON output:
{"cpu":{"model":"Intel Xeon Gold 6338","cores":8},"mem":{"ram":4096,"ram_units":"MiB","disk":80,"disk_units":"GB"},"net":{"ipv4":true,"ipv6":false},"ip_info":{"isp":"Example ISP","asn":"AS64500","org":"Example Network","country":"JP"},"fio":[{"speed_r":2500,"speed_w":1800,"speed_rw":4300,"speed_units":"KBps"}],"iperf":[{"send":"900 Mbps","recv":"850 Mbps","latency":"18 ms"}],"geekbench":[{"single":1234,"multi":2345,"url":"https://browser.geekbench.com/v6/cpu/12345"}]}`)

	summary := diagnosticSummaryFromOutput("yabs", "YABS 性能测试", output)
	if summary == nil {
		t.Fatal("expected a summary")
	}
	for _, check := range []struct {
		dimension string
		key       string
		want      string
	}{
		{"performance", "cpu_model", "Intel Xeon Gold 6338"},
		{"performance", "cpu_cores", "8"},
		{"performance", "memory", "4096 MiB"},
		{"performance", "disk", "80 GB"},
		{"performance", "disk_read", "2.56 MB/s"},
		{"performance", "disk_write", "1.84 MB/s"},
		{"ip", "isp", "Example ISP"},
		{"ip", "asn", "AS64500"},
		{"ip", "host", "Example Network"},
		{"ip", "ipv4_ipv6", "IPv4 已连接 · IPv6 不可用"},
		{"speed", "upload", "900 Mbps"},
		{"speed", "download", "850 Mbps"},
		{"latency", "average", "18 ms"},
		{"performance", "geekbench_single", "1234"},
		{"performance", "geekbench_multi", "2345"},
	} {
		if got := summaryMetricValue(summary, check.dimension, check.key); got != check.want {
			t.Errorf("%s.%s = %q, want %q", check.dimension, check.key, got, check.want)
		}
	}
	if summary.ReportURL != "https://browser.geekbench.com/v6/cpu/12345" {
		t.Fatalf("report url = %q", summary.ReportURL)
	}
}

func TestDiagnosticSummaryProtocolMarker(t *testing.T) {
	output := []byte("script output\nKPANEL_RESULT {\"parser\":\"net-quality\",\"dimensions\":{\"latency\":{\"metrics\":[{\"key\":\"average\",\"value\":\"42 ms\"}]}}}\n")

	summary := diagnosticSummaryFromOutput("net-quality", "网络质量体检", output)
	if summary == nil || summary.Parser != "protocol" {
		t.Fatalf("summary = %#v", summary)
	}
	if got := summaryMetricValue(summary, "latency", "average"); got != "42 ms" {
		t.Fatalf("latency.average = %q", got)
	}
}

func TestDiagnosticSummaryFromIPQualityJSON(t *testing.T) {
	output := []byte(`IPQuality report:
{"Head":{"Version":"v2026"},"Info":{"ASN":"AS64500","Organization":"Example Network","Region":{"Name":"Japan"},"City":{"Name":"Tokyo"}},"Score":{"IP2LOCATION":"Low","SCAMALYTICS":"VeryLow"},"Media":{"ChatGPT":{"Status":"Yes"},"Netflix":{"Status":"Native"}}}
Report Link: https://Report.Check.Place/abc123
`)

	summary := diagnosticSummaryFromOutput("ip-quality", "IP 质量体检", output)
	if summary == nil {
		t.Fatal("expected a summary")
	}
	for _, check := range []struct {
		dimension string
		key       string
		want      string
	}{
		{"ip", "asn", "AS64500"},
		{"ip", "host", "Example Network"},
		{"ip", "country", "Japan"},
		{"ip", "location", "Tokyo"},
		{"ip", "unlock", "ChatGPT: Yes · Netflix: Native"},
	} {
		if got := summaryMetricValue(summary, check.dimension, check.key); got != check.want {
			t.Errorf("%s.%s = %q, want %q", check.dimension, check.key, got, check.want)
		}
	}
	if quality := summaryMetricValue(summary, "ip", "quality"); !strings.Contains(quality, "IP2LOCATION: Low") || !strings.Contains(quality, "SCAMALYTICS: VeryLow") {
		t.Fatalf("quality = %q", quality)
	}
	if summary.ReportURL != "https://Report.Check.Place/abc123" {
		t.Fatalf("report url = %q", summary.ReportURL)
	}
}

func TestDiagnosticSummaryFromNetQualityJSON(t *testing.T) {
	output := []byte(`NetQuality report:
{"BGP":{"ASN":"AS64500","Organization":"Example Carrier"},"Connectivity":[{"ASN":"AS64500","Org":"Example Carrier"}],"Delay":[{"Code":"CN","Name":"China","CT":{"Average":"42"},"CU":{"Average":"53"},"CM":{"Average":"61"}}],"Speedtest":[{"City":"Shanghai","SendSpeed":"120","ReceiveSpeed":"680"}],"Transfer":[{"SendSpeed":"100 Mbps","ReceiveSpeed":"900 Mbps","Delay":{"Average":"18 ms"}}]}
Report Link: https://report.check.place/net456
`)

	summary := diagnosticSummaryFromOutput("net-quality", "网络质量体检", output)
	if summary == nil {
		t.Fatal("expected a summary")
	}
	for _, check := range []struct {
		dimension string
		key       string
		want      string
	}{
		{"route", "path", "AS64500 · Example Carrier"},
		{"latency", "average", "42 ms"},
		{"latency", "cu_average", "CU 53 ms"},
		{"speed", "upload", "120 Mbps"},
		{"speed", "download", "680 Mbps"},
	} {
		if got := summaryMetricValue(summary, check.dimension, check.key); got != check.want {
			t.Errorf("%s.%s = %q, want %q", check.dimension, check.key, got, check.want)
		}
	}
	if summary.ReportURL != "https://report.check.place/net456" {
		t.Fatalf("report url = %q", summary.ReportURL)
	}
}

func TestDiagnosticSummaryFromSuperSpeedText(t *testing.T) {
	output := []byte("ID    测速服务器信息       上传/Mbps   下载/Mbps   延迟/ms\n27377 电信 北京 ↑ 123.45 ↓ 678.90 20.00\n")

	summary := diagnosticSummaryFromOutput("superspeed", "SuperSpeed 三网测速", output)
	if summary == nil {
		t.Fatal("expected a summary")
	}
	for _, check := range []struct {
		dimension string
		key       string
		want      string
	}{
		{"speed", "upload", "123.45 Mbps"},
		{"speed", "download", "678.90 Mbps"},
		{"latency", "average", "20.00 ms"},
	} {
		if got := summaryMetricValue(summary, check.dimension, check.key); got != check.want {
			t.Errorf("%s.%s = %q, want %q", check.dimension, check.key, got, check.want)
		}
	}
}

func TestDiagnosticSummaryDoesNotInventValues(t *testing.T) {
	if summary := diagnosticSummaryFromOutput("ip-quality", "IP 质量体检", []byte("script completed successfully\n")); summary != nil {
		t.Fatalf("unexpected summary from output without measurements: %#v", summary)
	}
}

func summaryMetricValue(summary *DiagnosticSummary, dimension, key string) string {
	if summary == nil {
		return ""
	}
	for _, metric := range summary.Dimensions[dimension].Metrics {
		if metric.Key == key {
			return metric.Value
		}
	}
	return ""
}
