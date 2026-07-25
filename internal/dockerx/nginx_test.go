package dockerx

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	pathpkg "path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNginxReadyRequiresManagedRunningSafeContainer(t *testing.T) {
	id := strings.Repeat("a", 64)
	tests := []struct {
		name    string
		inspect containerInspect
		wantErr error
	}{
		{
			name: "compose working directory",
			inspect: nginxInspect(id, map[string]string{
				"com.docker.compose.project.working_dir": "/home/web",
			}),
		},
		{
			name: "explicit marker",
			inspect: nginxInspect(id, map[string]string{
				"io.kejilion.panel.managed": "true",
			}),
		},
		{
			name: "external",
			inspect: nginxInspect(id, map[string]string{
				"com.docker.compose.project.working_dir": "/srv/external",
			}),
			wantErr: ErrReadOnlyContainer,
		},
		{
			name: "stopped",
			inspect: func() containerInspect {
				raw := nginxInspect(id, map[string]string{"io.kejilion.panel.managed": "true"})
				raw.State.Running = false
				raw.State.Status = "exited"
				return raw
			}(),
			wantErr: ErrNginxNotReady,
		},
		{
			name: "unsafe",
			inspect: func() containerInspect {
				raw := nginxInspect(id, map[string]string{"io.kejilion.panel.managed": "true"})
				raw.HostConfig.Privileged = true
				return raw
			}(),
			wantErr: ErrUnsafeOrInvalidAction,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			webRoot := prepareNginxWebRoot(t)
			inspect := test.inspect
			addRequiredNginxMounts(&inspect, webRoot)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/containers/nginx/json" {
					http.NotFound(w, r)
					return
				}
				_ = json.NewEncoder(w).Encode(inspect)
			}))
			defer server.Close()

			client := testHTTPClient(server)
			client.webRoot = webRoot
			err := client.NginxReady(context.Background())
			if test.wantErr == nil && err != nil {
				t.Fatalf("NginxReady() error = %v, want nil", err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("NginxReady() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestNginxReadyDoesNotSocketActivateDocker(t *testing.T) {
	client := New(filepath.Join(t.TempDir(), "docker.sock"), "/home/web", t.TempDir())
	client.ConfigureDaemonAccess(filepath.Join(t.TempDir(), "missing.pid"), false)

	err := client.NginxReady(context.Background())
	if !errors.Is(err, ErrDockerNotRunning) {
		t.Fatalf("NginxReady() error = %v, want ErrDockerNotRunning", err)
	}
	if !errors.Is(err, ErrNginxNotReady) {
		t.Fatalf("NginxReady() error = %v, want ErrNginxNotReady", err)
	}
}

func TestNginxReadyAcceptsLegacyHostNetworkFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "legacy_nginx_inspect.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw containerInspect
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw.HostConfig.NetworkMode != "host" {
		t.Fatalf("legacy fixture NetworkMode = %q, want host", raw.HostConfig.NetworkMode)
	}
	if raw.Config.Labels["com.docker.compose.project.working_dir"] != "/home/web" {
		t.Fatalf("legacy fixture does not preserve /home/web ownership: %#v", raw.Config.Labels)
	}

	webRoot := t.TempDir()
	for index := range raw.Mounts {
		mount := &raw.Mounts[index]
		if mount.Type != "bind" {
			continue
		}
		cleanSource := pathpkg.Clean(mount.Source)
		if !strings.HasPrefix(cleanSource, nginxComposeWorkingDir+"/") {
			t.Fatalf("legacy fixture bind escaped /home/web: %q", mount.Source)
		}
		relative := strings.TrimPrefix(cleanSource, nginxComposeWorkingDir+"/")
		source := filepath.Join(webRoot, filepath.FromSlash(relative))
		if pathpkg.Base(mount.Source) == "nginx.conf" {
			if err := os.MkdirAll(filepath.Dir(source), 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(source, []byte("events {}\nhttp {\n    include /etc/nginx/conf.d/*.conf;\n}\n"), 0o640); err != nil {
				t.Fatal(err)
			}
		} else if err := os.MkdirAll(source, 0o750); err != nil {
			t.Fatal(err)
		}
		mount.Source = filepath.ToSlash(source)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/containers/nginx/json" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(raw)
	}))
	defer server.Close()

	client := testHTTPClient(server)
	client.webRoot = filepath.ToSlash(webRoot)
	if err := client.NginxReady(context.Background()); err != nil {
		t.Fatalf("NginxReady() rejected the legacy host-network fixture: %v", err)
	}
}

func TestNginxReadyRequiresExactArtifactBindingsAndInclude(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string, *containerInspect)
	}{
		{
			name: "missing main config bind",
			mutate: func(_ *testing.T, _ string, raw *containerInspect) {
				raw.Mounts = raw.Mounts[1:]
			},
		},
		{
			name: "missing conf directory bind",
			mutate: func(_ *testing.T, _ string, raw *containerInspect) {
				raw.Mounts = append(raw.Mounts[:1], raw.Mounts[2:]...)
			},
		},
		{
			name: "missing html directory bind",
			mutate: func(_ *testing.T, _ string, raw *containerInspect) {
				raw.Mounts = raw.Mounts[:2]
			},
		},
		{
			name: "wrong conf destination",
			mutate: func(_ *testing.T, _ string, raw *containerInspect) {
				raw.Mounts[1].Destination = "/etc/nginx/sites-enabled"
			},
		},
		{
			name: "symlink conf source",
			mutate: func(t *testing.T, root string, _ *containerInspect) {
				confPath := filepath.FromSlash(pathpkg.Join(root, "conf.d"))
				if err := os.Remove(confPath); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.FromSlash(pathpkg.Join(root, "html")), confPath); err != nil {
					t.Skipf("symlink is not available on this platform: %v", err)
				}
			},
		},
		{
			name: "main config does not include conf directory",
			mutate: func(t *testing.T, root string, _ *containerInspect) {
				if err := os.WriteFile(
					filepath.FromSlash(pathpkg.Join(root, "nginx.conf")),
					[]byte("events {}\nhttp {\n    # include /etc/nginx/conf.d/*.conf;\n}\n"),
					0o640,
				); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			webRoot := prepareNginxWebRoot(t)
			raw := nginxInspect(
				strings.Repeat("7", 64),
				map[string]string{"io.kejilion.panel.managed": "true"},
			)
			addRequiredNginxMounts(&raw, webRoot)
			test.mutate(t, webRoot, &raw)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/containers/nginx/json" {
					http.NotFound(w, r)
					return
				}
				_ = json.NewEncoder(w).Encode(raw)
			}))
			defer server.Close()

			client := testHTTPClient(server)
			client.webRoot = webRoot
			err := client.NginxReady(context.Background())
			if !errors.Is(err, ErrReadOnlyContainer) {
				t.Fatalf("NginxReady() error = %v, want ErrReadOnlyContainer", err)
			}
		})
	}
}

func TestNginxSafetyPolicyRejectsDangerousConfiguration(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	client := &Client{
		webRoot:   filepath.ToSlash(root),
		appRoot:   filepath.ToSlash(filepath.Join(root, "apps")),
		stateRoot: filepath.ToSlash(filepath.Join(root, "state")),
	}
	tests := []struct {
		name   string
		mutate func(*containerInspect)
	}{
		{
			name: "privileged",
			mutate: func(raw *containerInspect) {
				raw.HostConfig.Privileged = true
			},
		},
		{
			name: "host pid",
			mutate: func(raw *containerInspect) {
				raw.HostConfig.PidMode = "host"
			},
		},
		{
			name: "host ipc",
			mutate: func(raw *containerInspect) {
				raw.HostConfig.IpcMode = "host"
			},
		},
		{
			name: "host uts",
			mutate: func(raw *containerInspect) {
				raw.HostConfig.UTSMode = "host"
			},
		},
		{
			name: "host userns",
			mutate: func(raw *containerInspect) {
				raw.HostConfig.UsernsMode = "host"
			},
		},
		{
			name: "added capability",
			mutate: func(raw *containerInspect) {
				raw.HostConfig.CapAdd = []string{"SYS_ADMIN"}
			},
		},
		{
			name: "host device",
			mutate: func(raw *containerInspect) {
				raw.HostConfig.Devices = append(raw.HostConfig.Devices, struct {
					PathOnHost string `json:"PathOnHost"`
				}{PathOnHost: "/dev/sda"})
			},
		},
		{
			name: "unconfined security",
			mutate: func(raw *containerInspect) {
				raw.HostConfig.SecurityOpt = []string{"seccomp=unconfined"}
			},
		},
		{
			name: "disabled security",
			mutate: func(raw *containerInspect) {
				raw.HostConfig.SecurityOpt = []string{"label=disable"}
			},
		},
		{
			name: "Docker Socket",
			mutate: func(raw *containerInspect) {
				raw.Mounts = []dockerMount{{
					Type:        "volume",
					Name:        "socket",
					Source:      "/var/lib/docker/volumes/socket/_data",
					Destination: "/var/run/docker.sock",
				}}
			},
		},
		{
			name: "out of bounds bind",
			mutate: func(raw *containerInspect) {
				raw.Mounts = []dockerMount{{
					Type:        "bind",
					Source:      filepath.ToSlash(outside),
					Destination: "/etc/nginx/conf.d",
				}}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw := nginxInspect(
				strings.Repeat("9", 64),
				map[string]string{"io.kejilion.panel.managed": "true"},
			)
			raw.HostConfig.NetworkMode = "host"
			test.mutate(&raw)
			if reason := client.unsafeNginxReason(raw); reason == "" {
				t.Fatal("unsafeNginxReason() accepted a dangerous container")
			}
		})
	}
}

func TestNginxExecUsesOnlyFixedCommands(t *testing.T) {
	containerID := strings.Repeat("b", 64)
	execID := strings.Repeat("c", 64)
	tests := []struct {
		name        string
		wantCommand []string
		exitCode    int
		run         func(*Client, context.Context) error
	}{
		{
			name:        "test",
			wantCommand: []string{"nginx", "-t"},
			exitCode:    1,
			run:         (*Client).NginxTest,
		},
		{
			name:        "reload",
			wantCommand: []string{"nginx", "-s", "reload"},
			exitCode:    0,
			run:         (*Client).NginxReload,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var sequence []string
			webRoot := prepareNginxWebRoot(t)
			inspect := nginxInspect(
				containerID,
				map[string]string{"io.kejilion.panel.managed": "true"},
			)
			addRequiredNginxMounts(&inspect, webRoot)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sequence = append(sequence, r.Method+" "+r.URL.Path)
				switch r.Method + " " + r.URL.Path {
				case "GET /containers/nginx/json":
					_ = json.NewEncoder(w).Encode(inspect)
				case "POST /containers/" + containerID + "/exec":
					var request struct {
						AttachStdout bool     `json:"AttachStdout"`
						AttachStderr bool     `json:"AttachStderr"`
						Tty          bool     `json:"Tty"`
						Cmd          []string `json:"Cmd"`
					}
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Errorf("decode exec create request: %v", err)
					}
					if !request.AttachStdout || !request.AttachStderr || request.Tty {
						t.Errorf("unsafe exec create settings: %#v", request)
					}
					if !reflect.DeepEqual(request.Cmd, test.wantCommand) {
						t.Errorf("exec command = %#v, want %#v", request.Cmd, test.wantCommand)
					}
					_ = json.NewEncoder(w).Encode(map[string]string{"Id": execID})
				case "POST /exec/" + execID + "/start":
					var request struct {
						Detach bool `json:"Detach"`
						Tty    bool `json:"Tty"`
					}
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Errorf("decode exec start request: %v", err)
					}
					if request.Detach || request.Tty {
						t.Errorf("unsafe exec start settings: %#v", request)
					}
					_, _ = w.Write(dockerMux([]byte("password=super-secret\nnginx command output\n")))
				case "GET /exec/" + execID + "/json":
					_ = json.NewEncoder(w).Encode(nginxExecState{
						ID:       execID,
						Running:  false,
						ExitCode: test.exitCode,
					})
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			client := testHTTPClient(server)
			client.webRoot = webRoot
			err := test.run(client, context.Background())
			if test.exitCode == 0 {
				if err != nil {
					t.Fatalf("%s error = %v, want nil", test.name, err)
				}
			} else {
				var execErr *NginxExecError
				if !errors.As(err, &execErr) {
					t.Fatalf("%s error = %v, want *NginxExecError", test.name, err)
				}
				if !errors.Is(err, ErrNginxCommandFailed) {
					t.Fatalf("%s error = %v, want ErrNginxCommandFailed", test.name, err)
				}
				if execErr.Operation != test.name || execErr.ExitCode != test.exitCode {
					t.Fatalf("structured error = %#v", execErr)
				}
				if strings.Contains(execErr.Output, "super-secret") ||
					!strings.Contains(execErr.Output, "password=[REDACTED]") {
					t.Fatalf("exec output was not redacted: %q", execErr.Output)
				}
			}
			wantSequence := []string{
				"GET /containers/nginx/json",
				"POST /containers/" + containerID + "/exec",
				"POST /exec/" + execID + "/start",
				"GET /exec/" + execID + "/json",
			}
			if !reflect.DeepEqual(sequence, wantSequence) {
				t.Fatalf("Docker API sequence = %#v, want %#v", sequence, wantSequence)
			}
		})
	}
}

func TestNginxExecRejectsInvalidExecIdentity(t *testing.T) {
	containerID := strings.Repeat("d", 64)
	webRoot := prepareNginxWebRoot(t)
	inspect := nginxInspect(
		containerID,
		map[string]string{"io.kejilion.panel.managed": "true"},
	)
	addRequiredNginxMounts(&inspect, webRoot)
	startRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /containers/nginx/json":
			_ = json.NewEncoder(w).Encode(inspect)
		case "POST /containers/" + containerID + "/exec":
			_, _ = w.Write([]byte(`{"Id":"../../containers/external"}`))
		default:
			if strings.Contains(r.URL.Path, "/start") {
				startRequests++
			}
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := testHTTPClient(server)
	client.webRoot = webRoot
	err := client.NginxTest(context.Background())
	if !errors.Is(err, ErrInvalidDockerExec) {
		t.Fatalf("NginxTest() error = %v, want ErrInvalidDockerExec", err)
	}
	if startRequests != 0 {
		t.Fatalf("invalid exec identity triggered %d start requests", startRequests)
	}
}

func TestNginxExecFailureOutputIsBounded(t *testing.T) {
	containerID := strings.Repeat("e", 64)
	execID := strings.Repeat("f", 64)
	webRoot := prepareNginxWebRoot(t)
	inspect := nginxInspect(
		containerID,
		map[string]string{"io.kejilion.panel.managed": "true"},
	)
	addRequiredNginxMounts(&inspect, webRoot)
	largeOutput := bytes.Repeat([]byte("password=large-secret\n"), 10_000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /containers/nginx/json":
			_ = json.NewEncoder(w).Encode(inspect)
		case "POST /containers/" + containerID + "/exec":
			_ = json.NewEncoder(w).Encode(map[string]string{"Id": execID})
		case "POST /exec/" + execID + "/start":
			_, _ = w.Write(dockerMux(largeOutput))
		case "GET /exec/" + execID + "/json":
			_ = json.NewEncoder(w).Encode(nginxExecState{ID: execID, ExitCode: 1})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := testHTTPClient(server)
	client.webRoot = webRoot
	err := client.NginxTest(context.Background())
	var execErr *NginxExecError
	if !errors.As(err, &execErr) {
		t.Fatalf("NginxTest() error = %v, want *NginxExecError", err)
	}
	if !execErr.Truncated {
		t.Fatal("NginxExecError.Truncated = false, want true")
	}
	if len(execErr.Output) > maxNginxExecResponseBytes {
		t.Fatalf("NginxExecError.Output length = %d, limit = %d", len(execErr.Output), maxNginxExecResponseBytes)
	}
	if strings.Contains(execErr.Output, "large-secret") {
		t.Fatalf("NginxExecError.Output contains a secret: %q", execErr.Output)
	}
}

func nginxInspect(id string, labels map[string]string) containerInspect {
	raw := managedInspect(id, "2026-07-25T00:00:00Z", 0)
	raw.Name = "/nginx"
	raw.Config.Labels = labels
	return raw
}

func dockerMux(payload []byte) []byte {
	var stream bytes.Buffer
	header := make([]byte, 8)
	header[0] = 2
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	stream.Write(header)
	stream.Write(payload)
	return stream.Bytes()
}

func prepareNginxWebRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, directory := range []string{"conf.d", "html"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(
		filepath.Join(root, "nginx.conf"),
		[]byte("events {}\nhttp {\n    include /etc/nginx/conf.d/*.conf;\n}\n"),
		0o640,
	); err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(root)
}

func addRequiredNginxMounts(raw *containerInspect, webRoot string) {
	raw.Mounts = append(raw.Mounts,
		dockerMount{
			Type: "bind", Source: pathpkg.Join(webRoot, "nginx.conf"),
			Destination: nginxMainConfigPath, RW: true,
		},
		dockerMount{
			Type: "bind", Source: pathpkg.Join(webRoot, "conf.d"),
			Destination: nginxConfDirectoryPath, RW: true,
		},
		dockerMount{
			Type: "bind", Source: pathpkg.Join(webRoot, "html"),
			Destination: nginxHTMLDirectoryPath, RW: true,
		},
	)
}
