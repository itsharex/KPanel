package diagnostics

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// DiagnosticSummary contains only values explicitly found in script output.
// It deliberately has no derived overall score.
type DiagnosticSummary struct {
	Parser     string                               `json:"parser,omitempty"`
	ReportURL  string                               `json:"reportUrl,omitempty"`
	Dimensions map[string]DiagnosticSummaryDimension `json:"dimensions,omitempty"`
}

type DiagnosticSummaryDimension struct {
	Metrics []DiagnosticSummaryMetric `json:"metrics,omitempty"`
}

type DiagnosticSummaryMetric struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var (
	summaryReportURLPattern   = regexp.MustCompile(`(?i)https://(?:www\.)?(?:nodequality\.com/r/[A-Za-z0-9_-]+|browser\.geekbench\.com/[A-Za-z0-9./_?=&%#+:-]+|report\.check\.place/[A-Za-z0-9_-]+)`)
	summaryMeasurementPattern = regexp.MustCompile(`(?i)\b\d+(?:\.\d+)?\s*(?:[kmgt]?i?b(?:/s|ps)?|[kmgt]?bits?/s(?:ec)?|[kmgt]?bps|ms|%|scores?|[k])\b`)
	summaryPipeScorePattern   = regexp.MustCompile(`(?i)^\s*(single core|multi core)\s*\|\s*([^|\s]+)`)
	summaryJSONMarkerPattern  = regexp.MustCompile(`^\s*KPANEL_RESULT(?:\s+|\t)(\{.*\})\s*$`)
	summaryUpArrowPattern     = regexp.MustCompile(`↑\s*([0-9]+(?:\.[0-9]+)?(?:\s*[A-Za-z/]+)?)`)
	summaryDownArrowPattern   = regexp.MustCompile(`↓\s*([0-9]+(?:\.[0-9]+)?(?:\s*[A-Za-z/]+)?)`)
	summaryTrailingNumber     = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)\s*(?:ms|毫秒)?\s*$`)
)

var summaryDimensionIDs = map[string]bool{
	"performance": true,
	"route":       true,
	"latency":     true,
	"speed":       true,
	"ip":          true,
}

var summaryMetricKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

type summaryBuilder struct {
	parser     string
	reportURL  string
	dimensions map[string][]DiagnosticSummaryMetric
	indexes    map[string]map[string]bool
}

func newSummaryBuilder(parser string) *summaryBuilder {
	return &summaryBuilder{
		parser:     parser,
		dimensions: make(map[string][]DiagnosticSummaryMetric),
		indexes:    make(map[string]map[string]bool),
	}
}

func (builder *summaryBuilder) add(dimension, key, value string) {
	dimension = strings.TrimSpace(dimension)
	key = strings.TrimSpace(key)
	value = cleanSummaryValue(value)
	if !summaryDimensionIDs[dimension] || !summaryMetricKeyPattern.MatchString(key) || value == "" {
		return
	}
	if strings.EqualFold(value, "unknown") || strings.EqualFold(value, "n/a") || strings.EqualFold(value, "not available") {
		return
	}
	if len([]rune(value)) > 160 {
		value = string([]rune(value)[:160]) + "…"
	}
	if builder.indexes[dimension] == nil {
		builder.indexes[dimension] = make(map[string]bool)
	}
	if builder.indexes[dimension][key] {
		return
	}
	builder.indexes[dimension][key] = true
	builder.dimensions[dimension] = append(builder.dimensions[dimension], DiagnosticSummaryMetric{
		Key: key, Value: value,
	})
}

func (builder *summaryBuilder) setReportURL(value string) {
	if builder.reportURL == "" {
		builder.reportURL = summaryReportURLPattern.FindString(value)
	}
}

func (builder *summaryBuilder) merge(summary DiagnosticSummary) {
	if builder.parser == "" && summary.Parser != "" {
		builder.parser = summary.Parser
	}
	builder.setReportURL(summary.ReportURL)
	for dimension, item := range summary.Dimensions {
		for _, metric := range item.Metrics {
			builder.add(dimension, metric.Key, metric.Value)
		}
	}
}

func (builder *summaryBuilder) build() *DiagnosticSummary {
	if len(builder.dimensions) == 0 && builder.reportURL == "" {
		return nil
	}
	dimensions := make(map[string]DiagnosticSummaryDimension, len(builder.dimensions))
	for dimension, metrics := range builder.dimensions {
		dimensions[dimension] = DiagnosticSummaryDimension{Metrics: metrics}
	}
	return &DiagnosticSummary{
		Parser:     builder.parser,
		ReportURL:  builder.reportURL,
		Dimensions: dimensions,
	}
}

func diagnosticSummaryFromOutput(checkID, checkName string, output []byte) *DiagnosticSummary {
	text := stripControls(string(output))
	parser := "text"
	switch {
	case strings.EqualFold(checkID, "nodequality") || strings.Contains(strings.ToLower(checkName), "nodequality"):
		parser = "nodequality-text"
	case strings.EqualFold(checkID, "yabs"):
		parser = "yabs-text"
	}
	builder := newSummaryBuilder(parser)
	builder.setReportURL(text)
	parseSummaryMarkers(text, builder)
	parseEmbeddedJSON(text, builder)
	parseTextSummary(text, builder)
	return builder.build()
}

func parseSummaryMarkers(text string, builder *summaryBuilder) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 4096), 256<<10)
	for scanner.Scan() {
		match := summaryJSONMarkerPattern.FindStringSubmatch(scanner.Text())
		if len(match) != 2 {
			continue
		}
		var summary DiagnosticSummary
		if json.Unmarshal([]byte(match[1]), &summary) == nil {
			builder.merge(summary)
			builder.parser = "protocol"
		}
	}
}

func parseEmbeddedJSON(text string, builder *summaryBuilder) {
	for offset := 0; offset < len(text); {
		index := strings.IndexByte(text[offset:], '{')
		if index < 0 {
			return
		}
		index += offset
		decoder := json.NewDecoder(strings.NewReader(text[index:]))
		var value map[string]any
		if err := decoder.Decode(&value); err == nil && looksLikeBenchmarkJSON(value) {
			parseBenchmarkJSON(value, builder)
			if builder.parser != "protocol" {
				builder.parser = "json"
			}
		}
		offset = index + 1
	}
}

func looksLikeBenchmarkJSON(value map[string]any) bool {
	for _, key := range []string{"cpu", "mem", "fio", "iperf", "ip_info", "net", "geekbench", "dimensions", "Info", "Score", "BGP", "Delay", "Speedtest", "Transfer"} {
		if _, ok := value[key]; ok {
			return true
		}
	}
	return false
}

func parseBenchmarkJSON(value map[string]any, builder *summaryBuilder) {
	parseIPQualityJSON(value, builder)
	parseNetQualityJSON(value, builder)

	if dimensions, ok := value["dimensions"].(map[string]any); ok {
		for dimension, raw := range dimensions {
			object, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			metrics, ok := object["metrics"].([]any)
			if !ok {
				continue
			}
			for _, rawMetric := range metrics {
				metric, ok := rawMetric.(map[string]any)
				if !ok {
					continue
				}
				key, _ := metric["key"].(string)
				value, _ := metric["value"].(string)
				builder.add(dimension, key, value)
			}
		}
	}

	if cpu, ok := value["cpu"].(map[string]any); ok {
		builder.add("performance", "cpu_model", jsonString(cpu["model"]))
		builder.add("performance", "cpu_cores", jsonString(cpu["cores"]))
	}
	if memory, ok := value["mem"].(map[string]any); ok {
		builder.add("performance", "memory", jsonQuantity(memory["ram"], memory["ram_units"]))
		builder.add("performance", "disk", jsonQuantity(memory["disk"], memory["disk_units"]))
	}
	if info, ok := value["ip_info"].(map[string]any); ok {
		builder.add("ip", "isp", jsonString(info["isp"]))
		builder.add("ip", "asn", jsonString(info["asn"]))
		builder.add("ip", "host", jsonString(info["org"]))
		builder.add("ip", "country", jsonString(info["country"]))
	}
	if network, ok := value["net"].(map[string]any); ok {
		statuses := make([]string, 0, 2)
		if connected, present := jsonBool(network["ipv4"]); present {
			statuses = append(statuses, networkStatus("IPv4", connected))
		}
		if connected, present := jsonBool(network["ipv6"]); present {
			statuses = append(statuses, networkStatus("IPv6", connected))
		}
		builder.add("ip", "ipv4_ipv6", strings.Join(statuses, " · "))
	}
	if fio, ok := value["fio"].([]any); ok {
		for _, raw := range fio {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			builder.add("performance", "disk_read", jsonQuantity(item["speed_r"], item["speed_units"]))
			builder.add("performance", "disk_write", jsonQuantity(item["speed_w"], item["speed_units"]))
			builder.add("performance", "disk_total", jsonQuantity(item["speed_rw"], item["speed_units"]))
			break
		}
	}
	if iperf, ok := value["iperf"].([]any); ok {
		for _, raw := range iperf {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			builder.add("speed", "upload", jsonString(item["send"]))
			builder.add("speed", "download", jsonString(item["recv"]))
			builder.add("latency", "average", jsonString(item["latency"]))
			break
		}
	}
	if geekbench, ok := value["geekbench"].([]any); ok {
		for _, raw := range geekbench {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			builder.add("performance", "geekbench_single", jsonString(item["single"]))
			builder.add("performance", "geekbench_multi", jsonString(item["multi"]))
			builder.setReportURL(jsonString(item["url"]))
		}
	}
}

func parseIPQualityJSON(value map[string]any, builder *summaryBuilder) {
	info, ok := value["Info"].(map[string]any)
	if !ok {
		return
	}
	builder.add("ip", "asn", jsonString(info["ASN"]))
	builder.add("ip", "host", jsonString(info["Organization"]))
	if region, ok := info["Region"].(map[string]any); ok {
		builder.add("ip", "country", jsonString(region["Name"]))
	}
	if city, ok := info["City"].(map[string]any); ok {
		builder.add("ip", "location", jsonString(city["Name"]))
	}
	builder.add("ip", "quality", summarizeJSONMap(value["Score"]))
	builder.add("ip", "unlock", summarizeJSONStatuses(value["Media"]))
}

func parseNetQualityJSON(value map[string]any, builder *summaryBuilder) {
	if bgp, ok := value["BGP"].(map[string]any); ok {
		parts := make([]string, 0, 2)
		if asn := jsonString(bgp["ASN"]); asn != "" {
			parts = append(parts, asn)
		}
		if organization := jsonString(bgp["Organization"]); organization != "" {
			parts = append(parts, organization)
		}
		builder.add("route", "path", strings.Join(parts, " · "))
	}
	if connectivity, ok := value["Connectivity"].([]any); ok {
		parts := make([]string, 0, 3)
		for _, raw := range connectivity {
			item, itemOK := raw.(map[string]any)
			if !itemOK {
				continue
			}
			asn := jsonString(item["ASN"])
			organization := jsonString(item["Org"])
			if asn != "" && organization != "" {
				parts = append(parts, asn+" "+organization)
			} else if asn != "" {
				parts = append(parts, asn)
			}
			if len(parts) == 3 {
				break
			}
		}
		if len(parts) > 0 {
			builder.add("route", "path", strings.Join(parts, " · "))
		}
	}
	if delay, ok := value["Delay"].([]any); ok {
		for _, raw := range delay {
			item, itemOK := raw.(map[string]any)
			if !itemOK {
				continue
			}
			for _, itemKey := range []struct {
				key   string
				label string
			}{
				{"average", "CT"},
				{"cu_average", "CU"},
				{"cm_average", "CM"},
			} {
				if metric, metricOK := item[itemKey.label].(map[string]any); metricOK {
					value := normalizeLatencyValue(jsonString(metric["Average"]))
					if itemKey.key != "average" && value != "" {
						value = itemKey.label + " " + value
					}
					builder.add("latency", itemKey.key, value)
				}
			}
			break
		}
	}
	if speedtest, ok := value["Speedtest"].([]any); ok {
		for _, raw := range speedtest {
			item, itemOK := raw.(map[string]any)
			if !itemOK {
				continue
			}
			builder.add("speed", "upload", normalizeSpeedValue(jsonString(item["SendSpeed"])))
			builder.add("speed", "download", normalizeSpeedValue(jsonString(item["ReceiveSpeed"])))
			break
		}
	}
	if transfer, ok := value["Transfer"].([]any); ok {
		for _, raw := range transfer {
			item, itemOK := raw.(map[string]any)
			if !itemOK {
				continue
			}
			builder.add("speed", "upload", normalizeSpeedValue(jsonString(item["SendSpeed"])))
			builder.add("speed", "download", normalizeSpeedValue(jsonString(item["ReceiveSpeed"])))
			if average, averageOK := item["Delay"].(map[string]any); averageOK {
				builder.add("latency", "average", normalizeLatencyValue(jsonString(average["Average"])))
			}
			break
		}
	}
}

func summarizeJSONMap(value any) string {
	object, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		text := jsonString(object[key])
		if text == "" || strings.EqualFold(text, "unknown") || strings.EqualFold(text, "null") {
			continue
		}
		parts = append(parts, key+": "+text)
	}
	return strings.Join(parts, " · ")
}

func summarizeJSONStatuses(value any) string {
	object, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		item, itemOK := object[key].(map[string]any)
		if !itemOK {
			continue
		}
		status := jsonString(item["Status"])
		if status == "" {
			status = jsonString(item["Type"])
		}
		if status != "" {
			parts = append(parts, key+": "+status)
		}
	}
	return strings.Join(parts, " · ")
}

func parseTextSummary(text string, builder *summaryBuilder) {
	inDiskTable := false
	inIperfTable := false
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 4096), 256<<10)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "fio disk speed tests") || strings.Contains(lower, "dd sequential disk speed tests") {
			inDiskTable = true
			inIperfTable = false
			continue
		}
		if strings.Contains(lower, "iperf3 network speed tests") {
			inIperfTable = true
			inDiskTable = false
			continue
		}
		if inDiskTable {
			parseDiskTableRow(line, builder)
		}
		if inIperfTable {
			parseIperfTableRow(line, builder)
		}
		parseSuperSpeedRow(line, builder)
		lowerLine := strings.ToLower(line)
		if strings.Contains(line, "->") && (strings.Contains(line, "电信") || strings.Contains(line, "联通") || strings.Contains(line, "移动") || strings.Contains(lowerLine, "traceroute")) {
			builder.add("route", "path", line)
		}
		if match := summaryPipeScorePattern.FindStringSubmatch(line); len(match) == 3 {
			key := "geekbench_single"
			if strings.EqualFold(match[1], "multi core") {
				key = "geekbench_multi"
			}
			builder.add("performance", key, match[2])
		}
		parseKeyValueLine(line, builder)
		parseInlineScore(line, builder)
	}
}

func parseKeyValueLine(line string, builder *summaryBuilder) {
	separator := strings.IndexAny(line, ":：")
	if separator <= 0 || strings.Contains(line[:separator], "|") {
		return
	}
	key := strings.TrimSpace(line[:separator])
	value := strings.TrimSpace(line[separator+1:])
	if key == "" || value == "" {
		return
	}
	normalized := strings.ToLower(strings.Join(strings.Fields(key), " "))
	switch {
	case normalized == "processor" || normalized == "cpu model" || strings.Contains(normalized, "处理器"):
		builder.add("performance", "cpu_model", value)
	case normalized == "cpu cores" || normalized == "cpu core" || strings.Contains(normalized, "cpu核心") || strings.Contains(normalized, "cpu 核心"):
		builder.add("performance", "cpu_cores", value)
	case strings.Contains(normalized, "cpu") && strings.Contains(normalized, "score"):
		builder.add("performance", "cpu_score", value)
	case normalized == "ram" || normalized == "memory" || strings.Contains(normalized, "内存"):
		builder.add("performance", "memory", value)
	case normalized == "disk" || strings.Contains(normalized, "磁盘容量"):
		builder.add("performance", "disk", value)
	case strings.Contains(normalized, "disk") && strings.Contains(normalized, "read"):
		builder.add("performance", "disk_read", value)
	case strings.Contains(normalized, "disk") && strings.Contains(normalized, "write"):
		builder.add("performance", "disk_write", value)
	case normalized == "single core" || (strings.Contains(normalized, "单核") && strings.Contains(normalized, "分")):
		builder.add("performance", "geekbench_single", value)
	case normalized == "multi core" || (strings.Contains(normalized, "多核") && strings.Contains(normalized, "分")):
		builder.add("performance", "geekbench_multi", value)
	case strings.Contains(normalized, "memory") && strings.Contains(normalized, "score"):
		builder.add("performance", "memory_score", value)
	case strings.Contains(normalized, "单线程读") || strings.Contains(normalized, "memory read"):
		builder.add("performance", "memory_read", value)
	case strings.Contains(normalized, "单线程写") || strings.Contains(normalized, "memory write"):
		builder.add("performance", "memory_write", value)
	case normalized == "isp" || normalized == "asn" || normalized == "host" || normalized == "country" || normalized == "ipv4/ipv6":
		builder.add("ip", strings.ReplaceAll(normalized, "/", "_"), value)
	case normalized == "organization" || strings.Contains(normalized, "组织"):
		builder.add("ip", "host", value)
	case normalized == "location" || normalized == "city" || strings.Contains(normalized, "位置") || strings.Contains(normalized, "城市"):
		builder.add("ip", "location", value)
	case strings.Contains(normalized, "ip quality") || strings.Contains(normalized, "ip质量") || strings.Contains(normalized, "信誉") || strings.Contains(normalized, "风险") || strings.Contains(normalized, "risk") || strings.Contains(normalized, "解锁") || strings.Contains(normalized, "streaming"):
		builder.add("ip", "quality", value)
	case strings.Contains(normalized, "download") || strings.Contains(normalized, "recv speed") || strings.Contains(normalized, "下行") || strings.Contains(normalized, "下载"):
		builder.add("speed", "download", value)
	case strings.Contains(normalized, "upload") || strings.Contains(normalized, "send speed") || strings.Contains(normalized, "上行") || strings.Contains(normalized, "上传"):
		builder.add("speed", "upload", value)
	case strings.Contains(normalized, "jitter") || strings.Contains(normalized, "抖动"):
		builder.add("latency", "jitter", value)
	case strings.Contains(normalized, "loss") || strings.Contains(normalized, "丢包"):
		builder.add("latency", "loss", value)
	case strings.Contains(normalized, "latency") || strings.Contains(normalized, "ping") || strings.Contains(normalized, "rtt") || strings.Contains(normalized, "延迟"):
		builder.add("latency", "average", value)
	case strings.Contains(normalized, "route") || strings.Contains(normalized, "path") || strings.Contains(normalized, "路由") || strings.Contains(normalized, "线路") || strings.Contains(normalized, "回程"):
		builder.add("route", "path", value)
	}
}

func parseInlineScore(line string, builder *summaryBuilder) {
	if !strings.Contains(line, "得分") || !strings.Contains(line, "线程") {
		return
	}
	value := firstSummaryMeasurement(line)
	if value == "" {
		return
	}
	if strings.Contains(line, "内存") {
		builder.add("performance", "memory_score", value)
		return
	}
	builder.add("performance", "cpu_score", value)
}

func parseDiskTableRow(line string, builder *summaryBuilder) {
	fields := strings.Split(line, "|")
	if len(fields) < 2 {
		return
	}
	name := strings.ToLower(strings.TrimSpace(fields[0]))
	value := firstSummaryMeasurement(strings.Join(fields[1:], " "))
	if value == "" {
		return
	}
	switch name {
	case "read", "读取":
		builder.add("performance", "disk_read", value)
	case "write", "写入":
		builder.add("performance", "disk_write", value)
	case "total", "总计":
		builder.add("performance", "disk_total", value)
	}
}

func parseIperfTableRow(line string, builder *summaryBuilder) {
	if strings.Contains(strings.ToLower(line), "provider") || strings.Contains(line, "-----") {
		return
	}
	fields := strings.Split(line, "|")
	if len(fields) < 5 {
		return
	}
	if send := firstSummaryMeasurement(fields[2]); send != "" {
		builder.add("speed", "upload", send)
	}
	if receive := firstSummaryMeasurement(fields[3]); receive != "" {
		builder.add("speed", "download", receive)
	}
	if latency := firstSummaryMeasurement(fields[4]); latency != "" {
		builder.add("latency", "average", latency)
	}
}

func parseSuperSpeedRow(line string, builder *summaryBuilder) {
	if !strings.Contains(line, "↑") || !strings.Contains(line, "↓") {
		return
	}
	upload := summaryUpArrowPattern.FindStringSubmatch(line)
	download := summaryDownArrowPattern.FindStringSubmatch(line)
	if len(upload) == 2 {
		builder.add("speed", "upload", normalizeSpeedValue(upload[1]))
	}
	if len(download) == 2 {
		builder.add("speed", "download", normalizeSpeedValue(download[1]))
	}
	if latency := summaryTrailingNumber.FindStringSubmatch(line); len(latency) == 2 {
		builder.add("latency", "average", normalizeLatencyValue(latency[1]))
	}
}

func normalizeSpeedValue(value string) string {
	value = cleanSummaryValue(value)
	if strings.Contains(strings.ToLower(value), "bit") || strings.Contains(strings.ToLower(value), "bps") {
		return value
	}
	if value == "" {
		return ""
	}
	return value + " Mbps"
}

func normalizeLatencyValue(value string) string {
	value = cleanSummaryValue(value)
	if value == "" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(value), "ms") || strings.HasSuffix(value, "毫秒") {
		return value
	}
	return value + " ms"
}

func firstSummaryMeasurement(value string) string {
	return cleanSummaryValue(summaryMeasurementPattern.FindString(value))
}

func jsonString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case float64:
		return formatSummaryNumber(typed)
	default:
		return fmt.Sprint(typed)
	}
}

func jsonBool(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		parsed, err := strconv.ParseBool(typed)
		return parsed, err == nil
	default:
		return false, false
	}
}

func networkStatus(label string, connected bool) string {
	if connected {
		return label + " 已连接"
	}
	return label + " 不可用"
}

func jsonQuantity(value any, unit any) string {
	if value == nil {
		return ""
	}
	if stringValue, ok := value.(string); ok {
		return stringValue
	}
	number, ok := value.(float64)
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
		return jsonString(value)
	}
	unitValue := jsonString(unit)
	switch strings.ToLower(unitValue) {
	case "kib":
		return formatBinaryQuantity(number, []string{"KiB", "MiB", "GiB", "TiB"}, 1024)
	case "kb":
		return formatBinaryQuantity(number, []string{"KB", "MB", "GB", "TB"}, 1000)
	case "kbps":
		// YABS stores fio speeds as KiB/s while labelling the JSON unit KBps.
		return formatBinaryQuantity(number*1024, []string{"B/s", "KB/s", "MB/s", "GB/s", "TB/s"}, 1000)
	default:
		if unitValue == "" {
			return formatSummaryNumber(number)
		}
		return formatSummaryNumber(number) + " " + unitValue
	}
}

func formatBinaryQuantity(value float64, units []string, base float64) string {
	unitIndex := 0
	for value >= base && unitIndex < len(units)-1 {
		value /= base
		unitIndex++
	}
	return formatSummaryNumber(value) + " " + units[unitIndex]
}

func formatSummaryNumber(value float64) string {
	if math.Trunc(value) == value {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func cleanSummaryValue(value string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}
