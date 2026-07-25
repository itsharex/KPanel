package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStorePersistsIdentitySessionAndAudit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	storage, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	user := User{
		ID: "user-1", Username: "admin", PasswordHash: "secret-hash",
		Role: "admin", CreatedAt: now, UpdatedAt: now,
	}
	if err := storage.CreateInitialAdmin(user); err != nil {
		t.Fatal(err)
	}
	if err := storage.PutSession(Session{
		TokenHash: "token-hash", CSRFHash: "csrf-hash", UserID: user.ID,
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := storage.AppendAudit(AuditEvent{
		ID: "event-1", OccurredAt: now, ActorType: "user", ActorID: user.ID,
		Action: "auth.login", Result: "success", RequestID: "request-1",
	}, 100); err != nil {
		t.Fatal(err)
	}
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	if !reopened.IsInitialized() {
		t.Fatal("store lost initialized state")
	}
	gotUser, err := reopened.UserByUsername("ADMIN")
	if err != nil || gotUser.ID != user.ID {
		t.Fatalf("unexpected user %#v, err=%v", gotUser, err)
	}
	if _, err := reopened.SessionByTokenHash("token-hash", now); err != nil {
		t.Fatal(err)
	}
	events, next := reopened.ListAudit(10, "")
	if len(events) != 1 || events[0].ID != "event-1" || next != "" {
		t.Fatalf("unexpected audit page: %#v next=%q", events, next)
	}
}

func TestStoreRejectsOversizedState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maxStoreBytes + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path); err == nil {
		t.Fatal("oversized store was accepted")
	}
}

func TestStoreAllowsOnlyOneWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	first, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path); !errors.Is(err, ErrStoreLocked) {
		t.Fatalf("second writer was not rejected: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("store did not unlock after Close: %v", err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDirectorySyncFailureDoesNotTurnACommittedWriteIntoFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	storage, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	storage.syncDirectory = func(string) error {
		return errors.New("injected directory sync failure")
	}
	now := time.Now().UTC()
	err = storage.CreateInitialAdmin(User{
		ID: "user-1", Username: "admin", PasswordHash: "hash", Role: "admin",
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("committed write returned an ambiguous failure: %v", err)
	}
	if !storage.IsInitialized() {
		t.Fatal("committed in-memory state was rolled back")
	}
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if !reopened.IsInitialized() {
		t.Fatal("committed on-disk state was lost")
	}
}
