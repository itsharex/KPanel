package systemmanage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

const maxProcessStatBytes = 4096

func (m *Manager) signalProcess(ctx context.Context, input contract.SystemActionRequest) (bool, string, error) {
	if input.PID <= 0 || input.StartTimeTicks == 0 || (input.Signal != "term" && input.Signal != "kill") {
		return false, "", fmt.Errorf("%w: process identity and signal are required", ErrInvalidInput)
	}
	if err := ctx.Err(); err != nil {
		return false, "", err
	}
	currentStart, err := readProcessStartTime(filepath.Join(m.procRoot, strconv.Itoa(input.PID), "stat"), input.PID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, "", fmt.Errorf("%w: process no longer exists", ErrConflict)
		}
		return false, "", fmt.Errorf("%w: read process identity: %v", ErrNeedsAttention, err)
	}
	if currentStart != input.StartTimeTicks {
		return false, "", fmt.Errorf("%w: process identity changed", ErrConflict)
	}
	if err := m.processSignaler(input.PID, input.Signal); err != nil {
		return false, "", fmt.Errorf("%w: deliver process signal: %v", ErrNeedsAttention, err)
	}
	return true, fmt.Sprintf("已向 PID %d 发送 SIG%s；进程状态将从宿主机重新读取", input.PID, strings.ToUpper(input.Signal)), nil
}

func readProcessStartTime(path string, expectedPID int) (uint64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxProcessStatBytes+1))
	if err != nil {
		return 0, err
	}
	if len(data) > maxProcessStatBytes {
		return 0, errors.New("process stat exceeds limit")
	}
	value := strings.TrimSpace(string(data))
	open := strings.IndexByte(value, '(')
	close := strings.LastIndex(value, ") ")
	if open <= 0 || close <= open {
		return 0, errors.New("invalid process stat")
	}
	pid, err := strconv.Atoi(strings.TrimSpace(value[:open]))
	if err != nil || pid != expectedPID {
		return 0, errors.New("process stat PID mismatch")
	}
	fields := strings.Fields(value[close+2:])
	if len(fields) < 20 {
		return 0, errors.New("incomplete process stat")
	}
	start, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil || start == 0 {
		return 0, errors.New("invalid process start time")
	}
	return start, nil
}
