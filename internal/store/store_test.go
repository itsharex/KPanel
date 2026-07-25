package store

import (
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

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
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
