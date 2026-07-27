//go:build linux

package sites

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

type fakeWordPressRuntime struct {
	database              string
	databaseRolledBack    bool
	certificate           string
	certificateRolledBack bool
}

func (runtime *fakeWordPressRuntime) WordPressReady(context.Context) error { return nil }

func (runtime *fakeWordPressRuntime) PrepareWordPressDatabase(
	_ context.Context,
	database string,
) (WordPressDatabaseCredentials, error) {
	runtime.database = database
	return WordPressDatabaseCredentials{
		Name: database, User: "kejilion", Password: "fixed-test-password",
	}, nil
}

func (runtime *fakeWordPressRuntime) RollbackWordPressDatabase(_ context.Context, database string) error {
	if database != runtime.database {
		return errors.New("unexpected database rollback")
	}
	runtime.databaseRolledBack = true
	return nil
}

func (runtime *fakeWordPressRuntime) PrepareWordPressCertificate(_ context.Context, domain string) error {
	runtime.certificate = domain
	return nil
}

func (runtime *fakeWordPressRuntime) RollbackWordPressCertificate(_ context.Context, domain string) error {
	if domain != runtime.certificate {
		return errors.New("unexpected certificate rollback")
	}
	runtime.certificateRolledBack = true
	return nil
}

func TestWordPressInstallerProducesKejilionCompatibleArtifacts(t *testing.T) {
	manager, nginx, root := newTestManager(t)
	runtime := &fakeWordPressRuntime{}
	manager.wordPressRuntime = runtime
	manager.archiveLoader = func(context.Context) ([]byte, error) {
		return testWordPressArchive(t), nil
	}
	enabled := true
	created, err := manager.Create(context.Background(), SiteInput{
		PrimaryDomain: "blog.example.com", Type: "wordpress", Enabled: &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Kind != contract.SiteWordPress || created.Origin != contract.OriginWeb ||
		created.DocumentRoot != filepath.Join(root, "html", "blog.example.com", "wordpress") ||
		!containsString(created.AllowedActions, "delete") ||
		containsString(created.AllowedActions, "update") {
		t.Fatalf("unexpected WordPress summary: %#v", created)
	}
	if runtime.database != "blog_example_com" || runtime.databaseRolledBack {
		t.Fatalf("unexpected database transaction: %#v", runtime)
	}
	config, err := os.ReadFile(filepath.Join(root, "conf.d", "blog.example.com.conf"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"listen 443 ssl;", "/var/www/html/blog.example.com/wordpress",
		"/etc/nginx/certs/blog.example.com_cert.pem", "fastcgi_pass unix:/run/php/php-fpm.sock;",
		"aio threads;",
	} {
		if !bytes.Contains(config, []byte(expected)) {
			t.Fatalf("WordPress Nginx config is missing %q:\n%s", expected, config)
		}
	}
	wpConfig, err := os.ReadFile(filepath.Join(
		root, "html", "blog.example.com", "wordpress", "wp-config.php",
	))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"'blog_example_com'", "'kejilion'", "'fixed-test-password'", "'mysql'",
		"WP_REDIS_HOST", "https://blog.example.com",
	} {
		if !bytes.Contains(wpConfig, []byte(expected)) {
			t.Fatalf("wp-config.php is missing %q", expected)
		}
	}
	if nginx.tests != 3 || nginx.reloads != 2 {
		t.Fatalf("nginx calls tests=%d reloads=%d, want 3/2", nginx.tests, nginx.reloads)
	}
}

func TestWordPressInstallerRollsBackEveryCreatedArtifact(t *testing.T) {
	manager, nginx, root := newTestManager(t)
	runtime := &fakeWordPressRuntime{}
	manager.wordPressRuntime = runtime
	manager.archiveLoader = func(context.Context) ([]byte, error) {
		return testWordPressArchive(t), nil
	}
	nginx.reloadErrs = []error{nil, errors.New("final reload failed")}
	_, err := manager.Create(context.Background(), SiteInput{
		PrimaryDomain: "rollback.example.com", Type: "wordpress",
	})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Create() error = %v, want ErrUnavailable", err)
	}
	for _, path := range []string{
		filepath.Join(root, "conf.d", "rollback.example.com.conf"),
		filepath.Join(root, "html", "rollback.example.com"),
	} {
		if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("rollback left artifact %s: %v", path, statErr)
		}
	}
	if !runtime.databaseRolledBack || !runtime.certificateRolledBack {
		t.Fatalf("external artifacts were not rolled back: %#v", runtime)
	}
	if nginx.reloads != 3 {
		t.Fatalf("nginx reload count = %d, want bootstrap, failed final, recovery", nginx.reloads)
	}
}

func TestWordPressInstallerRejectsAliasesAndGenericSettings(t *testing.T) {
	invalid := []SiteInput{
		{PrimaryDomain: "blog.example.com", Type: "wordpress", Aliases: []string{"www.example.com"}},
		{PrimaryDomain: "blog.example.com", Type: "wordpress", PHPVersion: "latest"},
		{PrimaryDomain: "blog.example.com", Type: "wordpress", Upstream: "http://127.0.0.1"},
	}
	for _, input := range invalid {
		if _, err := normalizeWordPressInput(input); !errors.Is(err, ErrUnprocessable) {
			t.Errorf("normalizeWordPressInput(%#v) error = %v, want ErrUnprocessable", input, err)
		}
	}
}

func TestWordPressJobRunsAsynchronouslyAndPersistsResult(t *testing.T) {
	manager, _, _ := newTestManager(t)
	manager.wordPressRuntime = &fakeWordPressRuntime{}
	manager.archiveLoader = func(context.Context) ([]byte, error) {
		return testWordPressArchive(t), nil
	}
	stateDir := filepath.Join(t.TempDir(), "wordpress-jobs")
	if err := manager.ConfigureWordPressJobState(stateDir); err != nil {
		t.Fatal(err)
	}

	job, err := manager.StartWordPress(context.Background(), SiteInput{
		PrimaryDomain: "async.example.com", Type: "wordpress",
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "queued" || job.Stage != "queued" || job.Progress != 0 ||
		!wordPressJobIDPattern.MatchString(job.ID) {
		t.Fatalf("unexpected queued job: %#v", job)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		job, err = manager.WordPressJob(job.ID)
		if err != nil {
			t.Fatal(err)
		}
		if job.Status == "succeeded" {
			break
		}
		if job.Status == "failed" {
			t.Fatalf("WordPress job failed: %s", job.Message)
		}
		if time.Now().After(deadline) {
			t.Fatalf("WordPress job remained %q", job.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if job.Site == nil || job.Site.Kind != contract.SiteWordPress ||
		job.Site.PrimaryDomain != "async.example.com" ||
		job.Stage != "completed" || job.Progress != 100 {
		t.Fatalf("unexpected job result: %#v", job)
	}
	if _, err := os.Stat(filepath.Join(stateDir, job.ID+".json")); err != nil {
		t.Fatalf("persisted job result is missing: %v", err)
	}
}

func TestWordPressJobRecoveryFailsInterruptedWork(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "wordpress-jobs")
	manager, _, _ := newTestManager(t)
	if err := manager.ConfigureWordPressJobState(stateDir); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	interrupted := WordPressJob{
		ID:        strings.Repeat("a", 32),
		Domain:    "interrupted.example.com",
		Status:    "running",
		Message:   "installing",
		CreatedAt: now.Add(-time.Minute),
		StartedAt: &now,
	}
	if err := manager.wordPressJobs.put(interrupted); err != nil {
		t.Fatal(err)
	}

	restarted, _, _ := newTestManager(t)
	if err := restarted.ConfigureWordPressJobState(stateDir); err != nil {
		t.Fatal(err)
	}
	recovered, err := restarted.WordPressJob(interrupted.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Status != "failed" || recovered.Stage != "interrupted" ||
		recovered.Progress != 100 || recovered.EndedAt == nil ||
		!strings.Contains(recovered.Message, "Agent") {
		t.Fatalf("unexpected recovered job: %#v", recovered)
	}
}

func testWordPressArchive(t *testing.T) []byte {
	t.Helper()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	files := map[string]string{
		"wordpress/index.php": "<?php echo 'wordpress';\n",
		"wordpress/wp-config-sample.php": "<?php\n" +
			"define('DB_NAME', 'database_name_here');\n" +
			"define('DB_USER', 'username_here');\n" +
			"define('DB_PASSWORD', 'password_here');\n" +
			"define('DB_HOST', 'localhost');\n" +
			strings.Repeat("define('AUTH_KEY', 'put your unique phrase here');\n", 8) +
			"/* That's all, stop editing! Happy publishing. */\n",
	}
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}
