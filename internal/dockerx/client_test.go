package dockerx

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"strings"
	"testing"
)

func TestOwnershipAndSafety(t *testing.T) {
	web := t.TempDir()
	apps := t.TempDir()
	state := t.TempDir()
	c := &Client{
		webRoot:   filepath.ToSlash(web),
		appRoot:   filepath.ToSlash(apps),
		stateRoot: filepath.ToSlash(state),
	}
	var raw containerInspect
	raw.ID = strings.Repeat("a", 64)
	raw.Name = "/nginx"
	raw.Config.Image = "nginx:stable"
	raw.Config.Labels = map[string]string{
		"com.docker.compose.project.working_dir": filepath.ToSlash(web),
	}
	raw.State.Status = "running"
	raw.Mounts = []dockerMount{{
		Type: "bind", Source: filepath.ToSlash(filepath.Join(web, "conf.d")),
		Destination: "/etc/nginx/conf.d", RW: true,
	}}
	if err := ensureDir(filepath.Join(web, "conf.d")); err != nil {
		t.Fatal(err)
	}
	got := c.summaryFromInspect(raw)
	if got.Ownership != "kejilion" || !contains(got.AllowedActions, "restart") {
		t.Fatalf("expected safely managed container, got %#v", got)
	}

	raw.HostConfig.Privileged = true
	got = c.summaryFromInspect(raw)
	if len(got.AllowedActions) != 0 {
		t.Fatalf("privileged container must be read-only: %#v", got)
	}
}

func TestExplicitOwnershipStillRejectsDockerSocket(t *testing.T) {
	root := t.TempDir()
	c := &Client{webRoot: filepath.ToSlash(root), appRoot: "/home/docker", stateRoot: "/var/lib/kejilion-panel"}
	var raw containerInspect
	raw.Config.Labels = map[string]string{"io.kejilion.panel.managed": "true"}
	raw.State.Status = "running"
	raw.Mounts = []dockerMount{{Type: "bind", Source: "/var/run/docker.sock", Destination: "/var/run/docker.sock"}}
	if got := c.summaryFromInspect(raw); len(got.AllowedActions) != 0 {
		t.Fatalf("Docker Socket mount must be read-only: %#v", got)
	}
}

func TestDemuxAndRedactLogs(t *testing.T) {
	payload := []byte("token=super-secret\nhttps://user:pass@example.test/path\n")
	var stream bytes.Buffer
	header := make([]byte, 8)
	header[0] = 1
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	stream.Write(header)
	stream.Write(payload)
	lines := redactLines(demuxDockerStream(stream.Bytes()), 20)
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "super-secret") || strings.Contains(joined, "user:pass") {
		t.Fatalf("secret was not redacted: %s", joined)
	}
}

func ensureDir(path string) error {
	return osMkdirAll(path)
}
