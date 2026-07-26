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
	if len(runner.calls) != 1 || runner.calls[0][0] != "systemd-run" ||
		!strings.Contains(strings.Join(runner.calls[0], " "), appJobUnitPrefix+job.ID) {
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

func TestKejilionScriptCompatibilityRequiresExplicitLicenseAcceptance(t *testing.T) {
	base := []byte("KJ_APP_NONINTERACTIVE\nkpanel_run_docker_app_install\n")
	if isKPanelCompatibleScript(append(base, []byte(`permission_granted="false"`)...)) {
		t.Fatal("unaccepted user license enabled background script execution")
	}
	if !isKPanelCompatibleScript(append(base, []byte(`permission_granted="true"`)...)) {
		t.Fatal("accepted compatible script was rejected")
	}
}
