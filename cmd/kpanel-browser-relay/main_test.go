package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/browsercore"
)

func TestReadSecretRequiresBoundedStrongFile(t *testing.T) {
	directory := t.TempDir()
	validPath := filepath.Join(directory, "valid")
	if err := os.WriteFile(validPath, []byte("0123456789abcdef0123456789abcdef\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secret, err := browsercore.LoadSecretFile(validPath)
	if err != nil || string(secret) != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("valid secret = %q, %v", secret, err)
	}

	shortPath := filepath.Join(directory, "short")
	if err := os.WriteFile(shortPath, []byte("short"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := browsercore.LoadSecretFile(shortPath); err == nil {
		t.Fatal("short secret accepted")
	}
	if _, err := browsercore.LoadSecretFile(""); err == nil {
		t.Fatal("missing path accepted")
	}
}
