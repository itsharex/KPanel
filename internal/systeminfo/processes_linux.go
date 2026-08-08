//go:build linux

package systeminfo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

type processSample struct {
	pid, parentPID int
	name, state    string
	ticks          uint64
	startTicks     uint64
	threads, nice  int
}

func (c *Collector) collectProcesses(ctx context.Context, query ProcessQuery) (ProcessSnapshot, error) {
	started := time.Now()
	beforeTotal, err := c.readCPUTimes()
	if err != nil {
		return ProcessSnapshot{}, err
	}
	before, truncated, err := readProcessSamples(c.ProcRoot)
	if err != nil {
		return ProcessSnapshot{}, err
	}
	timer := time.NewTimer(c.ProcessSampleInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ProcessSnapshot{}, ctx.Err()
	case <-timer.C:
	}
	afterTotal, err := c.readCPUTimes()
	if err != nil {
		return ProcessSnapshot{}, err
	}
	after, afterTruncated, err := readProcessSamples(c.ProcRoot)
	if err != nil {
		return ProcessSnapshot{}, err
	}
	totalDelta := afterTotal.total - beforeTotal.total
	users := c.readProcessUsers()
	items := make([]ProcessMetric, 0, len(after))
	for pid, current := range after {
		previous, ok := before[pid]
		if !ok || previous.startTicks != current.startTicks || current.ticks < previous.ticks {
			continue
		}
		metric := ProcessMetric{
			PID: current.pid, ParentPID: current.parentPID, Name: current.name,
			State: current.state, StartTimeTicks: current.startTicks,
			Threads: current.threads, Nice: current.nice,
		}
		if totalDelta > 0 {
			metric.CPUPercent = float64(current.ticks-previous.ticks) / float64(totalDelta) * float64(runtime.NumCPU()) * 100
		}
		metric.MemoryBytes, metric.UserID = readProcessStatus(c.ProcRoot, pid)
		metric.User = users[metric.UserID]
		items = append(items, metric)
	}
	summary := processSummary(items, beforeTotal, afterTotal)
	var memory contract.MemorySummary
	if err := c.readMemory(&memory); err == nil {
		summary.MemoryUsedBytes = memory.UsedBytes
		summary.MemoryTotalBytes = memory.TotalBytes
	}
	var topCPU, topMemory []ProcessMetric
	var queryItems []ProcessMetric
	total := 0
	if query.Limit > 0 {
		queryItems = filterAndSortProcesses(items, query)
		total = len(queryItems)
		if len(queryItems) > query.Limit {
			queryItems = queryItems[:query.Limit]
			truncated = true
		}
	} else {
		sort.Slice(items, func(i, j int) bool {
			if items[i].CPUPercent != items[j].CPUPercent {
				return items[i].CPUPercent > items[j].CPUPercent
			}
			if items[i].MemoryBytes != items[j].MemoryBytes {
				return items[i].MemoryBytes > items[j].MemoryBytes
			}
			return items[i].PID < items[j].PID
		})
		topCPU = append([]ProcessMetric(nil), items...)
		if len(topCPU) > maxProcessItems {
			topCPU = topCPU[:maxProcessItems]
			truncated = true
		}
		topMemory = append([]ProcessMetric(nil), items...)
		sort.Slice(topMemory, func(i, j int) bool {
			if topMemory[i].MemoryBytes != topMemory[j].MemoryBytes {
				return topMemory[i].MemoryBytes > topMemory[j].MemoryBytes
			}
			if topMemory[i].CPUPercent != topMemory[j].CPUPercent {
				return topMemory[i].CPUPercent > topMemory[j].CPUPercent
			}
			return topMemory[i].PID < topMemory[j].PID
		})
		if len(topMemory) > maxProcessItems {
			topMemory = topMemory[:maxProcessItems]
			truncated = true
		}
	}
	return ProcessSnapshot{
		TopCPU: topCPU, TopMemory: topMemory, Items: queryItems, Total: total, Summary: summary,
		Scanned: len(after), Truncated: truncated || afterTruncated,
		SampleDuration: time.Since(started), CollectedAt: c.Now().UTC(),
	}, nil
}

func (c *Collector) readProcessUsers() map[int]string {
	users := make(map[int]string)
	for _, line := range strings.Split(readFileLimited(filepath.Join(c.EtcRoot, "passwd")), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) < 3 || len(fields[0]) > 64 {
			continue
		}
		uid, err := strconv.Atoi(fields[2])
		if err == nil && uid >= 0 {
			users[uid] = fields[0]
		}
	}
	return users
}

func processSummary(items []ProcessMetric, before, after cpuTimes) ProcessSummary {
	summary := ProcessSummary{Total: len(items), CPUPercent: cpuUsagePercent(before, after)}
	for _, item := range items {
		switch item.State {
		case "R":
			summary.Running++
		case "T", "t":
			summary.Stopped++
		case "Z":
			summary.Zombie++
		default:
			summary.Sleeping++
		}
	}
	return summary
}

func filterAndSortProcesses(items []ProcessMetric, query ProcessQuery) []ProcessMetric {
	search := strings.ToLower(query.Search)
	result := make([]ProcessMetric, 0, len(items))
	for _, item := range items {
		if search != "" && !strings.Contains(strings.ToLower(item.Name), search) &&
			!strings.Contains(strings.ToLower(item.User), search) &&
			!strings.Contains(strconv.Itoa(item.PID), search) {
			continue
		}
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		comparison := compareProcess(result[i], result[j], query.Sort)
		if comparison == 0 {
			return result[i].PID < result[j].PID
		}
		if query.Order == "asc" {
			return comparison < 0
		}
		return comparison > 0
	})
	return result
}

func compareProcess(left, right ProcessMetric, field string) int {
	switch field {
	case "memory":
		return compareUint64(left.MemoryBytes, right.MemoryBytes)
	case "pid":
		return left.PID - right.PID
	case "name":
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	case "user":
		return strings.Compare(strings.ToLower(left.User), strings.ToLower(right.User))
	case "state":
		return strings.Compare(left.State, right.State)
	case "threads":
		return left.Threads - right.Threads
	default:
		if left.CPUPercent < right.CPUPercent {
			return -1
		}
		if left.CPUPercent > right.CPUPercent {
			return 1
		}
		return 0
	}
}

func compareUint64(left, right uint64) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func readProcessSamples(root string) (map[int]processSample, bool, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, false, err
	}
	pids := make([]int, 0, len(entries))
	for _, entry := range entries {
		pid, parseErr := strconv.Atoi(entry.Name())
		if parseErr == nil && pid > 0 && entry.IsDir() {
			pids = append(pids, pid)
		}
	}
	sort.Ints(pids)
	truncated := len(pids) > maxProcessScan
	if truncated {
		pids = pids[:maxProcessScan]
	}
	result := make(map[int]processSample, len(pids))
	for _, pid := range pids {
		data, readErr := os.ReadFile(filepath.Join(root, strconv.Itoa(pid), "stat"))
		if readErr != nil {
			continue
		}
		value, parseErr := parseProcessStat(strings.TrimSpace(string(data)))
		if parseErr == nil {
			result[pid] = value
		}
	}
	return result, truncated, nil
}

func parseProcessStat(value string) (processSample, error) {
	open := strings.IndexByte(value, '(')
	close := strings.LastIndex(value, ") ")
	if open <= 0 || close <= open {
		return processSample{}, errors.New("invalid process stat")
	}
	pid, err := strconv.Atoi(strings.TrimSpace(value[:open]))
	if err != nil {
		return processSample{}, err
	}
	fields := strings.Fields(value[close+2:])
	if len(fields) < 20 {
		return processSample{}, errors.New("incomplete process stat")
	}
	parentPID, err := strconv.Atoi(fields[1])
	if err != nil {
		return processSample{}, err
	}
	userTicks, err := strconv.ParseUint(fields[11], 10, 64)
	if err != nil {
		return processSample{}, err
	}
	systemTicks, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return processSample{}, err
	}
	startTicks, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return processSample{}, err
	}
	niceValue, err := strconv.Atoi(fields[16])
	if err != nil {
		return processSample{}, err
	}
	threads, err := strconv.Atoi(fields[17])
	if err != nil || threads < 0 {
		return processSample{}, errors.New("invalid process thread count")
	}
	return processSample{
		pid: pid, parentPID: parentPID, name: value[open+1 : close], state: fields[0],
		ticks: userTicks + systemTicks, startTicks: startTicks, nice: niceValue, threads: threads,
	}, nil
}

func readProcessStatus(root string, pid int) (uint64, int) {
	data, err := os.ReadFile(filepath.Join(root, strconv.Itoa(pid), "status"))
	if err != nil {
		return 0, 0
	}
	var memory uint64
	userID := 0
	for _, line := range strings.Split(string(data), "\n") {
		switch {
		case strings.HasPrefix(line, "VmRSS:"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				value, _ := strconv.ParseUint(fields[1], 10, 64)
				memory = value * 1024
			}
		case strings.HasPrefix(line, "Uid:"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				userID, _ = strconv.Atoi(fields[1])
			}
		}
	}
	return memory, userID
}
