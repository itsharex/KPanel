package auth

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
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
	t.Cleanup(func() { _ = storage.Close() })
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
	t.Cleanup(func() { _ = storage.Close() })
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

func TestLoginBoundsConcurrentPasswordHashes(t *testing.T) {
	directory := t.TempDir()
	storage, err := store.Open(filepath.Join(directory, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = storage.Close() })
	now := time.Now().UTC()
	if err := storage.CreateInitialAdmin(store.User{
		ID: "user-1", Username: "admin", PasswordHash: "stored",
		Role: "admin", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	hasher := &gatedHasher{started: make(chan struct{}, 1), release: make(chan struct{})}
	service, err := NewService(storage, hasher, Config{
		BootstrapTokenPath:  filepath.Join(directory, "bootstrap.token"),
		SessionTTL:          time.Hour,
		LoginWindow:         time.Minute,
		MaxLoginFailures:    5,
		MaxConcurrentHashes: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	firstResult := make(chan error, 1)
	go func() {
		_, loginErr := service.Login("192.0.2.1", "admin", "wrong")
		firstResult <- loginErr
	}()
	select {
	case <-hasher.started:
	case <-time.After(time.Second):
		t.Fatal("first password verification did not start")
	}

	if _, err := service.Login("192.0.2.2", "admin", "wrong"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected concurrent hash limit, got %v", err)
	}
	close(hasher.release)
	if err := <-firstResult; !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unexpected first login result: %v", err)
	}
}

func TestBootstrapSerializesPasswordHashing(t *testing.T) {
	directory := t.TempDir()
	storage, err := store.Open(filepath.Join(directory, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = storage.Close() })
	hasher := &bootstrapGatedHasher{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	defer hasher.unblock()
	service, err := NewService(storage, hasher, Config{
		BootstrapTokenPath:  filepath.Join(directory, "bootstrap.token"),
		SessionTTL:          time.Hour,
		LoginWindow:         time.Minute,
		MaxLoginFailures:    5,
		MaxConcurrentHashes: 1,
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

	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, bootstrapErr := service.Bootstrap(
				string(token),
				"admin",
				"a-strong-password",
			)
			results <- bootstrapErr
		}()
	}
	select {
	case <-hasher.started:
	case <-time.After(time.Second):
		t.Fatal("bootstrap password hash did not start")
	}
	time.Sleep(25 * time.Millisecond)
	if calls := hasher.calls(); calls != 2 {
		t.Fatalf("concurrent bootstrap started %d password hashes; want dummy + one bootstrap", calls)
	}
	hasher.unblock()

	first, second := <-results, <-results
	if !((first == nil && errors.Is(second, ErrBootstrapUnavailable)) ||
		(second == nil && errors.Is(first, ErrBootstrapUnavailable))) {
		t.Fatalf("unexpected concurrent bootstrap results: first=%v second=%v", first, second)
	}
}

type gatedHasher struct {
	started chan struct{}
	release chan struct{}
}

func (h *gatedHasher) Hash(string) (string, error) {
	return "dummy", nil
}

func (h *gatedHasher) Verify(string, string) (bool, error) {
	select {
	case h.started <- struct{}{}:
	default:
	}
	<-h.release
	return false, nil
}

type bootstrapGatedHasher struct {
	mu          sync.Mutex
	hashCalls   int
	started     chan struct{}
	release     chan struct{}
	releaseOnce sync.Once
}

func (h *bootstrapGatedHasher) Hash(string) (string, error) {
	h.mu.Lock()
	h.hashCalls++
	call := h.hashCalls
	h.mu.Unlock()
	if call == 1 {
		return "dummy-hash", nil
	}
	select {
	case h.started <- struct{}{}:
	default:
	}
	<-h.release
	return "bootstrap-hash", nil
}

func (h *bootstrapGatedHasher) Verify(string, string) (bool, error) {
	return false, nil
}

func (h *bootstrapGatedHasher) calls() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hashCalls
}

func (h *bootstrapGatedHasher) unblock() {
	h.releaseOnce.Do(func() { close(h.release) })
}
