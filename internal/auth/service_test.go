package auth

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/store"
)

func TestBootstrapLoginSessionAndLogout(t *testing.T) {
	directory := t.TempDir()
	storage, err := store.Open(filepath.Join(directory, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(storage, testHasher(t), Config{
		BootstrapTokenPath: filepath.Join(directory, "bootstrap.token"),
		SessionTTL:         time.Hour,
		LoginWindow:        time.Minute,
		MaxLoginFailures:   3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.EnsureBootstrapToken(); err != nil {
		t.Fatal(err)
	}
	tokenBytes, err := os.ReadFile(filepath.Join(directory, "bootstrap.token"))
	if err != nil {
		t.Fatal(err)
	}
	credentials, err := service.Bootstrap(
		string(tokenBytes),
		"admin",
		"a-strong-password",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, "bootstrap.token")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("bootstrap token was not consumed: %v", err)
	}
	session, err := service.Authenticate(credentials.Token)
	if err != nil {
		t.Fatal(err)
	}
	if session.User.Username != "admin" {
		t.Fatalf("unexpected user: %#v", session.User)
	}
	if err := service.ValidateCSRF(session, credentials.CSRFToken); err != nil {
		t.Fatal(err)
	}
	if err := service.ValidateCSRF(session, "wrong"); !errors.Is(err, ErrInvalidCSRFToken) {
		t.Fatalf("expected CSRF failure, got %v", err)
	}
	if err := service.Logout(credentials.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(credentials.Token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("expected invalid session, got %v", err)
	}

	login, err := service.Login("127.0.0.1", "admin", "a-strong-password")
	if err != nil {
		t.Fatal(err)
	}
	if login.User.ID != credentials.User.ID {
		t.Fatalf("login user changed: %#v", login.User)
	}
}

func TestLoginRateLimit(t *testing.T) {
	directory := t.TempDir()
	storage, err := store.Open(filepath.Join(directory, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(storage, testHasher(t), Config{
		BootstrapTokenPath: filepath.Join(directory, "bootstrap.token"),
		SessionTTL:         time.Hour,
		LoginWindow:        time.Minute,
		MaxLoginFailures:   2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.EnsureBootstrapToken(); err != nil {
		t.Fatal(err)
	}
	token, err := os.ReadFile(filepath.Join(directory, "bootstrap.token"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Bootstrap(string(token), "admin", "a-strong-password"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := service.Login("192.0.2.1", "admin", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected invalid credentials, got %v", err)
		}
	}
	if _, err := service.Login("192.0.2.1", "admin", "a-strong-password"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected rate limit, got %v", err)
	}
}
