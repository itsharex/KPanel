package sites

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

var wordPressJobIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

type WordPressJob struct {
	ID        string                `json:"id"`
	Domain    string                `json:"domain"`
	Status    string                `json:"status"`
	Message   string                `json:"message,omitempty"`
	Site      *contract.SiteSummary `json:"site,omitempty"`
	CreatedAt time.Time             `json:"createdAt"`
	StartedAt *time.Time            `json:"startedAt,omitempty"`
	EndedAt   *time.Time            `json:"endedAt,omitempty"`
}

type wordPressJobRegistry struct {
	mu       sync.RWMutex
	stateDir string
	jobs     map[string]WordPressJob
}

func newWordPressJobRegistry(stateDir string) *wordPressJobRegistry {
	return &wordPressJobRegistry{stateDir: stateDir, jobs: make(map[string]WordPressJob)}
}

func (m *Manager) ConfigureWordPressJobState(stateDir string) error {
	stateDir = filepath.Clean(stateDir)
	if !filepath.IsAbs(stateDir) || stateDir == string(filepath.Separator) {
		return errors.New("WordPress job state requires a dedicated absolute directory")
	}
	info, err := os.Lstat(stateDir)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(stateDir, 0o750); err != nil {
			return err
		}
		info, err = os.Lstat(stateDir)
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("WordPress job state directory is unavailable or unsafe")
	}
	registry := newWordPressJobRegistry(stateDir)
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 ||
			!strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if !wordPressJobIDPattern.MatchString(id) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(stateDir, entry.Name()))
		if err != nil || len(data) > 256<<10 {
			continue
		}
		var job WordPressJob
		if json.Unmarshal(data, &job) != nil || job.ID != id {
			continue
		}
		if job.Status == "queued" || job.Status == "running" {
			now := time.Now().UTC()
			job.Status = "failed"
			job.Message = "Agent 在安装完成前重启，请核对实际产物后重新提交。"
			job.EndedAt = &now
		}
		registry.jobs[id] = job
		_ = registry.persist(job)
	}
	m.wordPressJobs = registry
	return nil
}

func (m *Manager) StartWordPress(ctx context.Context, input SiteInput) (WordPressJob, error) {
	spec, err := normalizeWordPressInput(input)
	if err != nil {
		return WordPressJob{}, err
	}
	if err := m.WordPressWritable(ctx); err != nil {
		return WordPressJob{}, err
	}
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return WordPressJob{}, fmt.Errorf("%w: create WordPress job identity", ErrUnavailable)
	}
	job := WordPressJob{
		ID: hex.EncodeToString(idBytes), Domain: spec.Primary,
		Status: "queued", Message: "安装任务已进入安全事务队列。",
		CreatedAt: time.Now().UTC(),
	}
	registry := m.wordPressJobs
	if registry == nil {
		registry = newWordPressJobRegistry("")
		m.wordPressJobs = registry
	}
	if err := registry.put(job); err != nil {
		return WordPressJob{}, fmt.Errorf("%w: persist WordPress job: %v", ErrUnavailable, err)
	}
	go m.runWordPressJob(job.ID, input)
	return job, nil
}

func (m *Manager) WordPressJob(id string) (WordPressJob, error) {
	if !wordPressJobIDPattern.MatchString(id) || m.wordPressJobs == nil {
		return WordPressJob{}, fmt.Errorf("%w: WordPress job does not exist", ErrConflict)
	}
	job, ok := m.wordPressJobs.get(id)
	if !ok {
		return WordPressJob{}, fmt.Errorf("%w: WordPress job does not exist", ErrConflict)
	}
	return job, nil
}

func (m *Manager) runWordPressJob(id string, input SiteInput) {
	registry := m.wordPressJobs
	job, ok := registry.get(id)
	if !ok {
		return
	}
	now := time.Now().UTC()
	job.Status = "running"
	job.Message = "正在准备 LDNMP、源码、数据库和 TLS 证书。"
	job.StartedAt = &now
	if err := registry.put(job); err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	site, err := m.installWordPress(ctx, input)
	ended := time.Now().UTC()
	job.EndedAt = &ended
	if err != nil {
		job.Status = "failed"
		job.Message = safeWordPressJobError(err)
	} else {
		job.Status = "succeeded"
		job.Message = "WordPress 源码、数据库、证书和 Nginx 产物已完成对账。"
		job.Site = &site
	}
	_ = registry.put(job)
}

func safeWordPressJobError(err error) string {
	message := strings.TrimSpace(err.Error())
	if len(message) > 1000 {
		message = message[:1000]
	}
	if message == "" {
		return "WordPress 安装失败，已尝试回滚本次新建产物。"
	}
	return message
}

func (registry *wordPressJobRegistry) put(job WordPressJob) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	previous, existed := registry.jobs[job.ID]
	if !existed && len(registry.jobs) >= 100 {
		hasTerminal := false
		for _, item := range registry.jobs {
			if item.Status == "succeeded" || item.Status == "failed" {
				hasTerminal = true
				break
			}
		}
		if !hasTerminal {
			return errors.New("too many active WordPress installation jobs")
		}
	}
	registry.jobs[job.ID] = job
	if err := registry.persist(job); err != nil {
		if existed {
			registry.jobs[job.ID] = previous
		} else {
			delete(registry.jobs, job.ID)
		}
		return err
	}
	if len(registry.jobs) > 100 {
		jobs := make([]WordPressJob, 0, len(registry.jobs))
		for _, item := range registry.jobs {
			if item.Status == "succeeded" || item.Status == "failed" {
				jobs = append(jobs, item)
			}
		}
		sort.Slice(jobs, func(i, j int) bool { return jobs[i].CreatedAt.Before(jobs[j].CreatedAt) })
		removedState := false
		removeCount := len(registry.jobs) - 100
		if removeCount > len(jobs) {
			removeCount = len(jobs)
		}
		for _, item := range jobs[:removeCount] {
			delete(registry.jobs, item.ID)
			if registry.stateDir != "" {
				if os.Remove(filepath.Join(registry.stateDir, item.ID+".json")) == nil {
					removedState = true
				}
			}
		}
		if removedState {
			_ = syncDirectory(registry.stateDir)
		}
	}
	return nil
}

func (registry *wordPressJobRegistry) get(id string) (WordPressJob, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	job, ok := registry.jobs[id]
	return job, ok
}

func (registry *wordPressJobRegistry) persist(job WordPressJob) error {
	if registry.stateDir == "" {
		return nil
	}
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(registry.stateDir, "."+job.ID+".*.tmp")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if err := file.Chmod(0o640); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	destination := filepath.Join(registry.stateDir, job.ID+".json")
	if err := os.Rename(temp, destination); err != nil {
		return err
	}
	return syncDirectory(registry.stateDir)
}
