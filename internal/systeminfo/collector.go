package systeminfo

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

// Collector reads Linux kernel pseudo-files directly. It never invokes a shell.
type Collector struct {
	ProcRoot string
	EtcRoot  string
	Now      func() time.Time
}

func NewCollector() *Collector {
	return &Collector{ProcRoot: "/proc", EtcRoot: "/etc", Now: time.Now}
}

func (c *Collector) Collect(_ context.Context) (contract.SystemSummary, error) {
	if c.ProcRoot == "" {
		c.ProcRoot = "/proc"
	}
	if c.EtcRoot == "" {
		c.EtcRoot = "/etc"
	}
	if c.Now == nil {
		c.Now = time.Now
	}

	result := contract.SystemSummary{
		Architecture: runtime.GOARCH,
		CollectedAt:  c.Now().UTC(),
	}
	result.Hostname, _ = os.Hostname()
	result.OS = c.readOSName()
	result.Kernel = strings.TrimSpace(c.readOptional("sys/kernel/osrelease"))

	var errs []error
	if err := c.readLoad(&result.Load); err != nil {
		errs = append(errs, err)
	}
	if err := c.readCPU(&result.CPU); err != nil {
		errs = append(errs, err)
	}
	if err := c.readMemory(&result.Memory); err != nil {
		errs = append(errs, err)
	}
	if err := c.readUptime(&result.UptimeSeconds); err != nil {
		errs = append(errs, err)
	}
	if err := c.readNetwork(&result.Network); err != nil {
		errs = append(errs, err)
	}
	result.Disks = c.readDisks()
	return result, errors.Join(errs...)
}

func (c *Collector) readOSName() string {
	data, err := os.ReadFile(filepath.Join(c.EtcRoot, "os-release"))
	if err != nil {
		return runtime.GOOS
	}
	values := make(map[string]string)
	s := bufio.NewScanner(strings.NewReader(string(data)))
	for s.Scan() {
		k, v, ok := strings.Cut(s.Text(), "=")
		if !ok {
			continue
		}
		values[k] = strings.Trim(strings.TrimSpace(v), `"'`)
	}
	if values["PRETTY_NAME"] != "" {
		return values["PRETTY_NAME"]
	}
	return strings.TrimSpace(values["NAME"] + " " + values["VERSION"])
}

func (c *Collector) readLoad(out *contract.LoadSummary) error {
	fields := strings.Fields(c.readOptional("loadavg"))
	if len(fields) < 3 {
		return errors.New("read loadavg: invalid or unavailable")
	}
	var err error
	if out.One, err = strconv.ParseFloat(fields[0], 64); err != nil {
		return fmt.Errorf("read loadavg: %w", err)
	}
	out.Five, _ = strconv.ParseFloat(fields[1], 64)
	out.Fifteen, _ = strconv.ParseFloat(fields[2], 64)
	return nil
}

func (c *Collector) readCPU(out *contract.CPUSummary) error {
	stat := c.readOptional("stat")
	first, _, _ := strings.Cut(stat, "\n")
	fields := strings.Fields(first)
	if len(fields) < 5 || fields[0] != "cpu" {
		return errors.New("read cpu: invalid or unavailable /proc/stat")
	}
	var values []uint64
	for _, field := range fields[1:] {
		n, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return fmt.Errorf("read cpu: %w", err)
		}
		values = append(values, n)
	}
	var total uint64
	for _, n := range values {
		total += n
	}
	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	if total > 0 {
		out.UsagePercent = roundPercent(float64(total-idle) * 100 / float64(total))
	}

	cpuInfo := c.readOptional("cpuinfo")
	for _, line := range strings.Split(cpuInfo, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "processor":
			out.Cores++
		case "model name", "Hardware":
			if out.Model == "" {
				out.Model = strings.TrimSpace(value)
			}
		}
	}
	if out.Cores == 0 {
		out.Cores = runtime.NumCPU()
	}
	return nil
}

func (c *Collector) readMemory(out *contract.MemorySummary) error {
	data := c.readOptional("meminfo")
	if data == "" {
		return errors.New("read memory: unavailable /proc/meminfo")
	}
	values := make(map[string]uint64)
	for _, line := range strings.Split(data, "\n") {
		key, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		n, _ := strconv.ParseUint(fields[0], 10, 64)
		values[key] = n * 1024
	}
	out.TotalBytes = values["MemTotal"]
	out.AvailableBytes = values["MemAvailable"]
	if out.AvailableBytes == 0 {
		out.AvailableBytes = values["MemFree"] + values["Buffers"] + values["Cached"]
	}
	if out.TotalBytes >= out.AvailableBytes {
		out.UsedBytes = out.TotalBytes - out.AvailableBytes
	}
	if out.TotalBytes > 0 {
		out.UsagePercent = roundPercent(float64(out.UsedBytes) * 100 / float64(out.TotalBytes))
	}
	out.SwapTotalBytes = values["SwapTotal"]
	if out.SwapTotalBytes >= values["SwapFree"] {
		out.SwapUsedBytes = out.SwapTotalBytes - values["SwapFree"]
	}
	if out.TotalBytes == 0 {
		return errors.New("read memory: MemTotal missing")
	}
	return nil
}

func (c *Collector) readUptime(out *uint64) error {
	fields := strings.Fields(c.readOptional("uptime"))
	if len(fields) == 0 {
		return errors.New("read uptime: invalid or unavailable")
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return fmt.Errorf("read uptime: %w", err)
	}
	if value > 0 {
		*out = uint64(value)
	}
	return nil
}

func (c *Collector) readNetwork(out *contract.NetworkSummary) error {
	data := c.readOptional("net/dev")
	if data == "" {
		return errors.New("read network: unavailable /proc/net/dev")
	}
	for _, line := range strings.Split(data, "\n") {
		_, values, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields := strings.Fields(values)
		if len(fields) < 16 {
			continue
		}
		rx, _ := strconv.ParseUint(fields[0], 10, 64)
		tx, _ := strconv.ParseUint(fields[8], 10, 64)
		out.ReceivedBytes += rx
		out.SentBytes += tx
	}
	out.TCPConnections = c.connectionCount("net/tcp") + c.connectionCount("net/tcp6")
	out.UDPConnections = c.connectionCount("net/udp") + c.connectionCount("net/udp6")
	return nil
}

func (c *Collector) connectionCount(name string) int {
	data := strings.TrimSpace(c.readOptional(name))
	if data == "" {
		return 0
	}
	lines := strings.Split(data, "\n")
	if len(lines) <= 1 {
		return 0
	}
	return len(lines) - 1
}

func (c *Collector) readDisks() []contract.DiskSummary {
	data := c.readOptional("self/mounts")
	seen := make(map[string]bool)
	var result []contract.DiskSummary
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		mountPoint := unescapeMount(fields[1])
		if seen[mountPoint] || !meaningfulMount(mountPoint, fields[2]) {
			continue
		}
		seen[mountPoint] = true
		total, free, ok := diskUsage(mountPoint)
		if !ok || total == 0 {
			continue
		}
		used := total - free
		result = append(result, contract.DiskSummary{
			Device:       unescapeMount(fields[0]),
			MountPoint:   mountPoint,
			FileSystem:   fields[2],
			TotalBytes:   total,
			UsedBytes:    used,
			UsagePercent: roundPercent(float64(used) * 100 / float64(total)),
		})
	}
	return result
}

func (c *Collector) readOptional(name string) string {
	data, _ := os.ReadFile(filepath.Join(c.ProcRoot, filepath.FromSlash(name)))
	return string(data)
}

func roundPercent(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

func unescapeMount(value string) string {
	replacer := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\134`, `\`)
	return replacer.Replace(value)
}

func meaningfulMount(path, fs string) bool {
	switch fs {
	case "proc", "sysfs", "devtmpfs", "devpts", "tmpfs", "cgroup", "cgroup2",
		"overlay", "squashfs", "nsfs", "mqueue", "securityfs", "pstore", "tracefs",
		"debugfs", "configfs", "fusectl", "hugetlbfs", "rpc_pipefs":
		return false
	}
	return path == "/" || path == "/home" || strings.HasPrefix(path, "/mnt/") || strings.HasPrefix(path, "/data")
}
