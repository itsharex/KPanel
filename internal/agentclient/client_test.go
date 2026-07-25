package agentclient

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRejectsEmptyToken(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte(" \n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := New("/tmp/agent.sock", tokenFile); err == nil {
		t.Fatal("expected empty token to be rejected")
	}
}

func TestNewRequiresSocket(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("test-token"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := New("", tokenFile); err == nil {
		t.Fatal("expected empty socket path to be rejected")
	}
}
