package appmarket

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeJobRunner struct {
	mu    sync.Mutex
	calls [][]string
}

func (runner *fakeJobRunner) Run(_ context.Context, name string, arguments ...string) ([]byte, error) {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	call := append([]string{name}, arguments...)
	runner.calls = append(runner.calls, call)
	return nil, nil
}

func (*fakeJobRunner) LookPath(name string) (string, error) {
	if name != "systemd-run" {
		return "", errors.New("not found")
	}
	return "/usr/bin/systemd-run", nil
}

func TestDeclarativeInstallRunsAsPersistentBackgroundJob(t *testing.T) {
	root := t.TempDir()
	service, err := New(&fakeDocker{}, root)
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeJobRunner{}
	if err := service.configureJobs(
		filepath.Join(root, "jobs"),
		filepath.Join(root, "kejilion-agent"),
		runner,
	); err != nil {
		t.Fatal(err)
	}

	job, err := service.StartInstall(context.Background(), "builtin-28", InstallInput{
		HostPort:   18028,
		AccessMode: "domain_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "queued" || job.Progress != 0 {
		t.Fatalf("initial job = %#v", job)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		job, err = service.AppJob(job.ID)
		if err != nil {
			t.Fatal(err)
		}
		if job.Status == "succeeded" {
			break
		}
		if job.Status == "failed" || time.Now().After(deadline) {
			record, _ := service.jobs.read(job.ID)
			t.Fatalf("background job did not succeed: public=%#v record=%#v launches=%#v", job, record, runner.calls)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if job.Progress != 100 || job.Stage != "completed" {
		t.Fatalf("completed job = %#v", job)
	}
	if len(service.AppJobs()) != 1 {
		t.Fatalf("job history count = %d", len(service.AppJobs()))
	}
}

func TestKejilionStandardAppsBecomeDirectlyInstallable(t *testing.T) {
	root := t.TempDir()
	service, err := New(&fakeDocker{}, root)
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeJobRunner{}
	if err := service.configureJobs(
		filepath.Join(root, "jobs"),
		filepath.Join(root, "kejilion-agent"),
		runner,
	); err != nil {
		t.Fatal(err)
	}
	service.scriptFinder = func() (string, error) {
		return "/usr/local/bin/k", nil
	}
	service.scriptInteractiveFinder = func() (string, error) {
		return "/usr/local/bin/k", nil
	}

	item, err := service.Find(context.Background(), "builtin-4")
	if err != nil {
		t.Fatal(err)
	}
	if item.Installer != "kejilion" || !item.Capabilities["install"].Enabled {
		t.Fatalf("standard app is not directly installable: %#v", item)
	}
	job, err := service.StartInstall(context.Background(), item.ID, InstallInput{HostPort: 18081})
	if err != nil {
		t.Fatal(err)
	}
	record, err := service.jobs.read(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.Selector != "4" || record.Adapter != "kejilion" || record.HostPort != 18081 {
		t.Fatalf("unsafe or incomplete script job record: %#v", record)
	}
	if !record.Interactive {
		t.Fatal("kejilion.sh install was not routed through the interactive terminal")
	}
	if len(runner.calls) != 1 || runner.calls[0][0] != "systemd-run" ||
		!strings.Contains(strings.Join(runner.calls[0], " "), appJobUnitPrefix+job.ID) ||
		!strings.Contains(strings.Join(runner.calls[0], " "), "app-pty-run") {
		t.Fatalf("background launch = %#v", runner.calls)
	}
	if _, err := service.StartInstall(context.Background(), "builtin-28", InstallInput{}); !errors.Is(err, ErrConflict) {
		t.Fatalf("parallel install error = %v, want conflict", err)
	}

	restarted, err := New(&fakeDocker{}, root)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.configureJobs(
		filepath.Join(root, "jobs"),
		filepath.Join(root, "kejilion-agent"),
		runner,
	); err != nil {
		t.Fatal(err)
	}
	recovered, err := restarted.AppJob(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Status != "failed" || recovered.Stage != "interrupted" {
		t.Fatalf("interrupted background job was not recovered: %#v", recovered)
	}
}

func TestInactiveScriptJobAutomaticallyReleasesTaskLock(t *testing.T) {
	root := t.TempDir()
	service, err := New(&fakeDocker{}, root)
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeJobRunner{}
	if err := service.configureJobs(
		filepath.Join(root, "jobs"),
		filepath.Join(root, "kejilion-agent"),
		runner,
	); err != nil {
		t.Fatal(err)
	}
	service.scriptFinder = func() (string, error) { return "/usr/local/bin/k", nil }
	service.scriptInteractiveFinder = func() (string, error) { return "/usr/local/bin/k", nil }

	job, err := service.StartInstall(context.Background(), "builtin-4", InstallInput{})
	if err != nil {
		t.Fatal(err)
	}
	record, err := service.jobs.read(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	record.CreatedAt = service.now().Add(-appJobLaunchGrace - time.Second)
	if err := service.jobs.put(record); err != nil {
		t.Fatal(err)
	}

	history := service.AppJobs()
	if len(history) != 1 || history[0].Status != "failed" ||
		history[0].Stage != "interrupted" || history[0].InputOpen {
		t.Fatalf("stale task lock was not released: %#v", history)
	}
	if _, err := service.StartInstall(context.Background(), "builtin-28", InstallInput{}); err != nil {
		t.Fatalf("stale task still blocked a new install: %v", err)
	}
}

func TestKejilionScriptCompatibilityRequiresExplicitLicenseAcceptance(t *testing.T) {
	base := []byte("KJ_APP_NONINTERACTIVE\nkpanel_run_docker_app_install\n")
	unaccepted := append(
		append([]byte{}, base...),
		[]byte("permission_granted=\"false\"\nsed -i 's/permission_granted=\"false\"/permission_granted=\"true\"/' /usr/local/bin/k\n")...,
	)
	if isKPanelCompatibleScript(unaccepted) {
		t.Fatal("unaccepted user license enabled background script execution")
	}
	accepted := append(append([]byte{}, base...), []byte("permission_granted=\"true\"\n")...)
	if !isKPanelCompatibleScript(accepted) {
		t.Fatal("accepted compatible script was rejected")
	}
}

func TestKejilionInteractiveCompatibilityIsExplicit(t *testing.T) {
	base := []byte(
		"KJ_APP_NONINTERACTIVE\nkpanel_run_docker_app_install\n" +
			"permission_granted=\"true\"\n",
	)
	if isKPanelInteractiveCompatibleScript(base) {
		t.Fatal("legacy script unexpectedly enabled interactive terminal jobs")
	}
	compatible := append(
		append([]byte{}, base...),
		[]byte("KJ_APP_INTERACTIVE\nkpanel_app_interactive_choice\n")...,
	)
	if !isKPanelInteractiveCompatibleScript(compatible) {
		t.Fatal("interactive script protocol was rejected")
	}
}
