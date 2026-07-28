package panel

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/appmarket"
	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/store"
	"github.com/kejilion/kejilion-panel/internal/webenv"
)

type auditJobGroup struct {
	requestID  string
	action     string
	targetKind string
	targetID   string
	intent     *store.AuditEvent
	outcome    *store.AuditEvent
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/jobs" || r.URL.RawPath != "" {
		s.writeProblem(w, r, http.StatusNotFound, "route_not_found", "Route not found", "")
		return
	}
	_, _, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	limit, valid := jobsLimit(r)
	if !valid {
		s.writeValidationProblem(w, r, "limit", "limit must be between 1 and 100")
		return
	}
	events, _ := s.store.ListAudit(200, "")
	jobs := jobsFromAudit(events, limit)
	if response, err := s.agent.Get(r.Context(), "/v1/app-jobs", "", requestID(r)); err == nil &&
		response.StatusCode == http.StatusOK {
		var page contract.PageResult[appmarket.AppJob]
		if json.Unmarshal(response.Body, &page) == nil {
			jobs = append(jobs, jobsFromAppJobs(page.Items)...)
		}
	}
	if response, err := s.agent.Get(r.Context(), "/v1/web-environment/jobs", "", requestID(r)); err == nil &&
		response.StatusCode == http.StatusOK {
		var page contract.PageResult[webenv.Job]
		if json.Unmarshal(response.Body, &page) == nil {
			jobs = append(jobs, jobsFromWebEnvironment(page.Items)...)
		}
	}
	sort.SliceStable(jobs, func(left, right int) bool {
		return jobs[left].CreatedAt.After(jobs[right].CreatedAt)
	})
	if len(jobs) > limit {
		jobs = jobs[:limit]
	}
	s.writeJSON(w, http.StatusOK, contract.PageResult[contract.Job]{Items: jobs})
}

func jobsFromAppJobs(items []appmarket.AppJob) []contract.Job {
	jobs := make([]contract.Job, 0, len(items))
	for _, item := range items {
		state := contract.JobRunning
		switch item.Status {
		case "queued":
			state = contract.JobQueued
		case "succeeded":
			state = contract.JobSucceeded
		case "failed":
			state = contract.JobFailedNeedsAttention
		}
		job := contract.Job{
			ID: item.ID, Action: "app." + item.Action, Origin: contract.OriginWeb,
			State: state, Stage: item.Stage, Progress: item.Progress,
			TargetKind: "application", TargetID: item.AppID, TargetLabel: item.AppName,
			CreatedAt: item.CreatedAt, StartedAt: item.StartedAt, FinishedAt: item.FinishedAt,
		}
		if item.Status == "failed" {
			job.Error = &contract.Problem{
				Title: "应用安装失败", Code: "app_install_failed",
				Detail: item.Message, Retryable: true,
			}
		}
		jobs = append(jobs, job)
	}
	return jobs
}

func jobsFromWebEnvironment(items []webenv.Job) []contract.Job {
	jobs := make([]contract.Job, 0, len(items))
	for _, item := range items {
		state := contract.JobRunning
		switch item.Status {
		case "queued":
			state = contract.JobQueued
		case "succeeded":
			state = contract.JobSucceeded
		case "failed", "needs_attention":
			state = contract.JobFailedNeedsAttention
		}
		job := contract.Job{
			ID: item.ID, Action: "web.environment." + item.Action, Origin: contract.OriginWeb,
			State: state, Stage: item.Stage, Progress: item.Progress,
			TargetKind: "web_environment", TargetID: item.Target, TargetLabel: item.Target,
			CreatedAt: item.CreatedAt, StartedAt: item.StartedAt, FinishedAt: item.FinishedAt,
		}
		if item.Status == "failed" || item.Status == "needs_attention" {
			job.Error = &contract.Problem{
				Title: "LDNMP 环境任务未完成", Code: "web_environment_job_failed",
				Detail: item.Message, Retryable: item.Status == "failed",
			}
		}
		jobs = append(jobs, job)
	}
	return jobs
}

func jobsLimit(r *http.Request) (int, bool) {
	values := r.URL.Query()
	for key := range values {
		if key != "limit" {
			return 0, false
		}
	}
	rawValues, present := values["limit"]
	if !present {
		return 50, true
	}
	if len(rawValues) != 1 || rawValues[0] == "" {
		return 0, false
	}
	value, err := strconv.Atoi(rawValues[0])
	if err != nil || value < 1 || value > 100 {
		return 0, false
	}
	return value, true
}

func jobsFromAudit(events []store.AuditEvent, limit int) []contract.Job {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	groups := make(map[string]*auditJobGroup)
	for index := range events {
		event := &events[index]
		if !managementAuditAction(event.Action) || !jobAuditResult(event.Result) {
			continue
		}
		groupRequestID := event.RequestID
		if groupRequestID == "" {
			groupRequestID = event.ID
		}
		key := strings.Join(
			[]string{groupRequestID, event.Action, event.TargetKind, event.TargetID},
			"\x00",
		)
		group := groups[key]
		if group == nil {
			group = &auditJobGroup{
				requestID: groupRequestID, action: event.Action,
				targetKind: event.TargetKind, targetID: event.TargetID,
			}
			groups[key] = group
		}
		switch event.Result {
		case "intent":
			if group.intent == nil || event.OccurredAt.After(group.intent.OccurredAt) {
				group.intent = event
			}
		case "success", "failure", "denied":
			if group.outcome == nil || event.OccurredAt.After(group.outcome.OccurredAt) {
				group.outcome = event
			}
		}
	}

	jobs := make([]contract.Job, 0, len(groups))
	for _, group := range groups {
		createdAt := time.Time{}
		if group.intent != nil {
			createdAt = group.intent.OccurredAt
		} else if group.outcome != nil {
			createdAt = group.outcome.OccurredAt
		}
		job := contract.Job{
			ID: group.requestID, Action: group.action, Origin: contract.OriginWeb,
			State: contract.JobRunning, Stage: "running",
			TargetKind: group.targetKind, TargetID: group.targetID,
			TargetLabel: group.targetID, CreatedAt: createdAt,
		}
		if group.intent != nil {
			startedAt := group.intent.OccurredAt
			job.StartedAt = &startedAt
		}
		if group.outcome != nil {
			finishedAt := group.outcome.OccurredAt
			job.FinishedAt = &finishedAt
			job.Progress = 100
			switch group.outcome.Result {
			case "success":
				job.State = contract.JobSucceeded
				job.Stage = "completed"
			default:
				job.State = contract.JobFailedNeedsAttention
				job.Stage = "attention_required"
			}
		}
		jobs = append(jobs, job)
	}
	sort.SliceStable(jobs, func(left, right int) bool {
		return jobs[left].CreatedAt.After(jobs[right].CreatedAt)
	})
	if len(jobs) > limit {
		jobs = jobs[:limit]
	}
	return jobs
}

func managementAuditAction(action string) bool {
	return strings.HasPrefix(action, "docker.") ||
		strings.HasPrefix(action, "site.") ||
		strings.HasPrefix(action, "system.") ||
		strings.HasPrefix(action, "web.environment.")
}

func jobAuditResult(result string) bool {
	switch result {
	case "intent", "success", "failure", "denied":
		return true
	default:
		return false
	}
}
