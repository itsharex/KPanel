package appmarket

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func TestThirdPartyScriptAppUsesVerifiedMainContainerAndManagementProtocol(t *testing.T) {
	root := t.TempDir()
	configRoot := filepath.Join(root, "apps")
	if err := os.Mkdir(configRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "appno.txt"),
		[]byte("AIClient-2-API\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(configRoot, "AIClient-2-API.conf"),
		[]byte("docker_name=\"aiclient-2-api\"\ndocker_app_service=aiclient-web\ndocker_app_plus\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "aiclient-2-api_access.conf"),
		[]byte("domain_only\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	containerID := strings.Repeat("a", 64)
	service, err := New(&fakeDocker{containers: []contract.ContainerSummary{{
		ID: containerID, Name: "aiclient-web", Image: "example/aiclient:latest",
		State: "running", Status: "Up",
		Ports: []contract.PortBinding{{
			PrivatePort: 3000, PublicPort: 18081, IP: "0.0.0.0", Type: "tcp",
		}},
		ResourceVersion: "sha256:" + strings.Repeat("b", 64),
		AllowedActions:  []string{"stop", "restart"},
	}}}, root)
	if err != nil {
		t.Fatal(err)
	}
	service.scriptAppRoot = configRoot
	service.fileOwnerTrusted = func(os.FileInfo) bool { return true }
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
	service.scriptManageFinder = func() (string, error) { return "/usr/local/bin/k", nil }

	item, err := service.Find(context.Background(), "thirdparty-AIClient-2-API")
	if err != nil {
		t.Fatal(err)
	}
	if !item.Runtime.Installed || item.Runtime.ContainerName != "aiclient-web" ||
		item.Runtime.AccessMode != "domain_only" {
		t.Fatalf("verified runtime was not reconciled: %#v", item.Runtime)
	}
	for _, detector := range []string{"docker", "appno", "app_config", "access_state"} {
		if !containsString(item.Runtime.DetectedBy, detector) {
			t.Fatalf("missing detector %q: %#v", detector, item.Runtime.DetectedBy)
		}
	}
	for _, action := range []string{"update", "uninstall", "direct_access"} {
		if !item.Capabilities[action].Enabled {
			t.Fatalf("%s was not enabled: %#v", action, item.Capabilities[action])
		}
	}

	job, scriptBacked, err := service.StartScriptMutation(
		context.Background(),
		item.ID,
		"update",
		MutationInput{ResourceVersion: item.Runtime.ResourceVersion},
	)
	if err != nil || !scriptBacked {
		t.Fatalf("script update job = %#v script=%v err=%v", job, scriptBacked, err)
	}
	record, err := service.jobs.read(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.Action != "update" || record.Selector != "AIClient-2-API" ||
		record.ExpectedContainerID != containerID {
		t.Fatalf("unsafe script job record: %#v", record)
	}
	if _, scriptBacked, err := service.StartScriptMutation(
		context.Background(),
		item.ID,
		"uninstall",
		MutationInput{ResourceVersion: "stale"},
	); !scriptBacked || err == nil {
		t.Fatalf("stale mutation was accepted: script=%v err=%v", scriptBacked, err)
	}
}

func TestDynamicThirdPartyConfigDoesNotBecomeAManagementGuardrail(t *testing.T) {
	root := t.TempDir()
	configRoot := filepath.Join(root, "apps")
	if err := os.Mkdir(configRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "appno.txt"),
		[]byte("AIClient-2-API\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(configRoot, "AIClient-2-API.conf"),
		[]byte("docker_name=\"${APP_NAME}\"\ndocker_app_plus\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	containerID := strings.Repeat("c", 64)
	service, err := New(&fakeDocker{containers: []contract.ContainerSummary{{
		ID: containerID, Name: "aiclient-runtime", Image: "example/aiclient:latest",
		State: "running", Status: "Up",
		Labels: map[string]string{
			"com.docker.compose.project": "AIClient-2-API",
		},
		ResourceVersion: "sha256:" + strings.Repeat("d", 64),
		AllowedActions:  []string{"stop", "restart", "logs", "exec"},
	}}}, root)
	if err != nil {
		t.Fatal(err)
	}
	service.scriptAppRoot = configRoot
	service.fileOwnerTrusted = func(os.FileInfo) bool { return true }
	if err := service.configureJobs(
		filepath.Join(root, "jobs"),
		filepath.Join(root, "kejilion-agent"),
		&fakeJobRunner{},
	); err != nil {
		t.Fatal(err)
	}
	service.scriptFinder = func() (string, error) { return "/usr/local/bin/k", nil }
	service.scriptInteractiveFinder = func() (string, error) { return "/usr/local/bin/k", nil }
	service.scriptManageFinder = func() (string, error) { return "/usr/local/bin/k", nil }

	item, err := service.Find(context.Background(), "thirdparty-AIClient-2-API")
	if err != nil {
		t.Fatal(err)
	}
	if !item.Runtime.Installed || item.Runtime.ContainerID != containerID ||
		!containsString(item.Runtime.DetectedBy, "compose_label") {
		t.Fatalf("dynamic script product was not reconciled from Docker: %#v", item.Runtime)
	}
	for _, action := range []string{"update", "uninstall", "direct_access"} {
		if !item.Capabilities[action].Enabled {
			t.Fatalf("%s stayed disabled by config parsing: %#v", action, item.Capabilities[action])
		}
	}
}

func TestThirdPartyScriptConfigRejectsUnsafeOrDynamicContainerMetadata(t *testing.T) {
	root := t.TempDir()
	service, err := New(&fakeDocker{}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service.scriptAppRoot = root
	service.fileOwnerTrusted = func(os.FileInfo) bool { return true }

	tests := map[string]string{
		"dynamic":    "docker_name=\"${APP_NAME}\"\ndocker_app_plus\n",
		"duplicate":  "docker_name=one\ndocker_name=two\ndocker_app_plus\n",
		"adapter":    "docker_name=one\ndocker_app\ndocker_app_plus\n",
		"incomplete": "docker_app_plus\n",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, name+".conf")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := service.readThirdPartyScriptSpec(name); err == nil {
				t.Fatal("unsafe application configuration was accepted")
			}
		})
	}

	target := filepath.Join(root, "target.conf")
	if err := os.WriteFile(target, []byte("docker_name=one\ndocker_app_plus\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "linked.conf")); err == nil {
		if _, err := service.readThirdPartyScriptSpec("linked"); err == nil {
			t.Fatal("symlinked application configuration was accepted")
		}
	}
}

func TestManageCompatibilityRequiresExactNoninteractiveProtocol(t *testing.T) {
	installOnly := []byte(
		"KJ_APP_NONINTERACTIVE\nkpanel_run_docker_app_install\npermission_granted=\"true\"\n",
	)
	if !isKPanelCompatibleScript(installOnly) {
		t.Fatal("install-compatible script was rejected")
	}
	if isKPanelManageCompatibleScript(installOnly) {
		t.Fatal("install-only script enabled destructive management")
	}
	managed := append(
		append([]byte{}, installOnly...),
		[]byte("kpanel_run_docker_app_action\nKJ_APP_EXPECTED_CONTAINER_ID\n")...,
	)
	if !isKPanelManageCompatibleScript(managed) {
		t.Fatal("management-compatible script was rejected")
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
