package ai

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSecretBoxRoundTripAndMissingKeyProtection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ai-secrets.key")
	box, err := OpenSecretBox(path, false)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal("provider-1", "secret-value")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := box.Open("provider-1", sealed)
	if err != nil || plain != "secret-value" {
		t.Fatalf("round trip = %q, %v", plain, err)
	}
	if _, err := box.Open("provider-2", sealed); err == nil {
		t.Fatal("expected provider-bound AAD failure")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("key mode = %o", info.Mode().Perm())
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenSecretBox(path, true); err == nil {
		t.Fatal("missing key must fail closed when encrypted secrets exist")
	}
}

func TestSecretBoxRejectsInvalidKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-secrets.key")
	if err := os.WriteFile(path, []byte("short"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenSecretBox(path, false); err == nil {
		t.Fatal("expected invalid key length error")
	}
}
