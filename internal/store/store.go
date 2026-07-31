package store

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxStoreBytes int64 = 32 << 20

var (
	ErrAlreadyInitialized = errors.New("store already initialized")
	ErrConflict           = errors.New("record changed")
	ErrNotFound           = errors.New("record not found")
	ErrStoreLocked        = errors.New("store is already open by another process")
)

type processLock interface {
	Close() error
}

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"passwordHash"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Session struct {
	TokenHash string    `json:"tokenHash"`
	CSRFHash  string    `json:"csrfHash"`
	UserID    string    `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type AuditEvent struct {
	ID         string         `json:"id"`
	OccurredAt time.Time      `json:"occurredAt"`
	ActorType  string         `json:"actorType"`
	ActorID    string         `json:"actorId,omitempty"`
	SourceIP   string         `json:"sourceIp,omitempty"`
	Action     string         `json:"action"`
	TargetKind string         `json:"targetKind,omitempty"`
	TargetID   string         `json:"targetId,omitempty"`
	Result     string         `json:"result"`
	RequestID  string         `json:"requestId"`
	Change     map[string]any `json:"change,omitempty"`
}

type LoginAttempt struct {
	Key        string    `json:"key"`
	OccurredAt time.Time `json:"occurredAt"`
	Success    bool      `json:"success"`
}

type SecurityEntrance struct {
	Enabled   bool      `json:"enabled"`
	Path      string    `json:"path,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

type diskState struct {
	SchemaVersion    int              `json:"schemaVersion"`
	Users            []User           `json:"users"`
	Sessions         []Session        `json:"sessions"`
	Audit            []AuditEvent     `json:"audit"`
	LoginAttempts    []LoginAttempt   `json:"loginAttempts"`
	SecurityEntrance SecurityEntrance `json:"securityEntrance,omitempty"`
}

// Store is a small, single-node persistence layer. It deliberately stores only
// panel identity/session/audit and panel-local security settings; host resources
// remain owned by the Agent.
type Store struct {
	mu            sync.RWMutex
	path          string
	data          diskState
	processLock   processLock
	syncDirectory func(string) error
}

func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("store path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create store directory: %w", err)
	}
	lock, err := acquireProcessLock(path + ".lock")
	if err != nil {
		return nil, err
	}
	opened := false
	defer func() {
		if !opened {
			_ = lock.Close()
		}
	}()

	s := &Store{
		path:          path,
		data:          diskState{SchemaVersion: 1},
		processLock:   lock,
		syncDirectory: syncDirectory,
	}
	if info, err := os.Stat(path); err == nil && info.Size() > maxStoreBytes {
		return nil, fmt.Errorf("store file exceeds %d bytes", maxStoreBytes)
	}
	content, err := os.ReadFile(path)
	switch {
	case err == nil:
		if len(content) == 0 {
			return nil, errors.New("store file is empty")
		}
		if err := json.Unmarshal(content, &s.data); err != nil {
			return nil, fmt.Errorf("decode store: %w", err)
		}
		if s.data.SchemaVersion != 1 {
			return nil, fmt.Errorf("unsupported store schema version %d", s.data.SchemaVersion)
		}
	case errors.Is(err, os.ErrNotExist):
		if err := s.persistLocked(); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("read store: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return nil, fmt.Errorf("protect store: %w", err)
	}

	opened = true
	return s, nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.processLock == nil {
		return nil
	}
	err := s.processLock.Close()
	s.processLock = nil
	return err
}

func (s *Store) IsInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data.Users) > 0
}

func (s *Store) CreateInitialAdmin(user User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.data.Users) != 0 {
		return ErrAlreadyInitialized
	}
	previous := cloneDiskState(s.data)
	s.data.Users = append(s.data.Users, user)
	if err := s.persistLocked(); err != nil {
		s.data = previous
		return err
	}
	return nil
}

func (s *Store) UserByUsername(username string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, user := range s.data.Users {
		if strings.EqualFold(user.Username, username) {
			return user, nil
		}
	}
	return User{}, ErrNotFound
}

func (s *Store) UserByID(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, user := range s.data.Users {
		if user.ID == id {
			return user, nil
		}
	}
	return User{}, ErrNotFound
}

// ReplaceUserPassword atomically updates a user's password hash and revokes all
// of their sessions. expectedHash prevents concurrent password changes from
// overwriting one another.
func (s *Store) ReplaceUserPassword(userID, expectedHash, newHash string, updatedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userIndex := -1
	for index := range s.data.Users {
		if s.data.Users[index].ID == userID {
			userIndex = index
			break
		}
	}
	if userIndex < 0 {
		return ErrNotFound
	}
	if s.data.Users[userIndex].PasswordHash != expectedHash {
		return ErrConflict
	}

	previous := cloneDiskState(s.data)
	s.data.Users[userIndex].PasswordHash = newHash
	s.data.Users[userIndex].UpdatedAt = updatedAt
	sessions := make([]Session, 0, len(s.data.Sessions))
	for _, session := range s.data.Sessions {
		if session.UserID != userID {
			sessions = append(sessions, session)
		}
	}
	s.data.Sessions = sessions
	if err := s.persistLocked(); err != nil {
		s.data = previous
		return err
	}
	return nil
}

func (s *Store) PutSession(session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := cloneDiskState(s.data)
	filtered := s.data.Sessions[:0]
	for _, item := range s.data.Sessions {
		if item.TokenHash != session.TokenHash && item.ExpiresAt.After(time.Now().UTC()) {
			filtered = append(filtered, item)
		}
	}
	s.data.Sessions = append(filtered, session)
	if err := s.persistLocked(); err != nil {
		s.data = previous
		return err
	}
	return nil
}

func (s *Store) SessionByTokenHash(tokenHash string, now time.Time) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, session := range s.data.Sessions {
		if session.TokenHash == tokenHash && session.ExpiresAt.After(now) {
			return session, nil
		}
	}
	return Session{}, ErrNotFound
}

func (s *Store) DeleteSession(tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := cloneDiskState(s.data)
	filtered := s.data.Sessions[:0]
	found := false
	for _, session := range s.data.Sessions {
		if session.TokenHash == tokenHash {
			found = true
			continue
		}
		filtered = append(filtered, session)
	}
	s.data.Sessions = filtered
	if !found {
		return nil
	}
	if err := s.persistLocked(); err != nil {
		s.data = previous
		return err
	}
	return nil
}

func (s *Store) AppendAudit(event AuditEvent, maxEntries int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := cloneDiskState(s.data)
	s.data.Audit = append(s.data.Audit, event)
	if maxEntries > 0 && len(s.data.Audit) > maxEntries {
		s.data.Audit = append([]AuditEvent(nil), s.data.Audit[len(s.data.Audit)-maxEntries:]...)
	}
	if err := s.persistLocked(); err != nil {
		s.data = previous
		return err
	}
	return nil
}

// ListAudit returns newest-first records. Cursor is the last event ID received.
func (s *Store) ListAudit(limit int, cursor string) ([]AuditEvent, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	items := append([]AuditEvent(nil), s.data.Audit...)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].OccurredAt.After(items[j].OccurredAt)
	})

	start := 0
	if cursor != "" {
		for i, event := range items {
			if event.ID == cursor {
				start = i + 1
				break
			}
		}
	}
	if start >= len(items) {
		return []AuditEvent{}, ""
	}
	end := min(start+limit, len(items))
	next := ""
	if end < len(items) {
		next = items[end-1].ID
	}
	return items[start:end], next
}

func (s *Store) RecordLoginAttempt(attempt LoginAttempt, retainSince time.Time) error {
	return s.RecordLoginAttempts([]LoginAttempt{attempt}, retainSince)
}

func (s *Store) RecordLoginAttempts(attempts []LoginAttempt, retainSince time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := cloneDiskState(s.data)
	filtered := s.data.LoginAttempts[:0]
	for _, item := range s.data.LoginAttempts {
		if item.OccurredAt.After(retainSince) {
			filtered = append(filtered, item)
		}
	}
	s.data.LoginAttempts = append(filtered, attempts...)
	if err := s.persistLocked(); err != nil {
		s.data = previous
		return err
	}
	return nil
}

func (s *Store) FailedLoginCount(key string, since time.Time) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	latestSuccess := since
	for _, item := range s.data.LoginAttempts {
		if item.Key == key && item.Success && item.OccurredAt.After(latestSuccess) {
			latestSuccess = item.OccurredAt
		}
	}
	count := 0
	for _, item := range s.data.LoginAttempts {
		if item.Key == key && !item.Success && item.OccurredAt.After(latestSuccess) {
			count++
		}
	}
	return count
}

func (s *Store) SecurityEntrance() (SecurityEntrance, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value := s.data.SecurityEntrance
	return value, SecurityEntranceResourceVersion(value)
}

func (s *Store) ReplaceSecurityEntrance(expectedResourceVersion string, value SecurityEntrance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if expectedResourceVersion != SecurityEntranceResourceVersion(s.data.SecurityEntrance) {
		return ErrConflict
	}
	previous := cloneDiskState(s.data)
	s.data.SecurityEntrance = value
	if err := s.persistLocked(); err != nil {
		s.data = previous
		return err
	}
	return nil
}

func SecurityEntranceResourceVersion(value SecurityEntrance) string {
	payload := fmt.Sprintf("%t\x00%s\x00%s", value.Enabled, value.Path, value.UpdatedAt.UTC().Format(time.RFC3339Nano))
	digest := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("sha256:%x", digest[:])
}

func (s *Store) persistLocked() error {
	content, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("encode store: %w", err)
	}
	content = append(content, '\n')
	if int64(len(content)) > maxStoreBytes {
		return fmt.Errorf("encoded store exceeds %d bytes", maxStoreBytes)
	}

	dir := filepath.Dir(s.path)
	temp, err := os.CreateTemp(dir, ".panel-store-*")
	if err != nil {
		return fmt.Errorf("create temporary store: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return fmt.Errorf("protect temporary store: %w", err)
	}
	if _, err := temp.Write(content); err != nil {
		temp.Close()
		return fmt.Errorf("write temporary store: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf("sync temporary store: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary store: %w", err)
	}

	// os.Rename atomically replaces the target on Linux, which is the production
	// platform. The backup fallback keeps local Windows development functional.
	if err := os.Rename(tempPath, s.path); err == nil {
		// The rename is the logical commit point. A directory fsync strengthens
		// crash durability, but failure must not make callers retry an operation
		// that is already visible in memory and on disk.
		_ = s.syncDirectory(dir)
		return nil
	}

	backupPath := s.path + ".previous"
	_ = os.Remove(backupPath)
	if err := os.Rename(s.path, backupPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("prepare store replacement: %w", err)
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		_ = os.Rename(backupPath, s.path)
		return fmt.Errorf("replace store: %w", err)
	}
	_ = os.Remove(backupPath)
	_ = s.syncDirectory(dir)
	return nil
}

func cloneDiskState(source diskState) diskState {
	return diskState{
		SchemaVersion:    source.SchemaVersion,
		Users:            append([]User(nil), source.Users...),
		Sessions:         append([]Session(nil), source.Sessions...),
		Audit:            append([]AuditEvent(nil), source.Audit...),
		LoginAttempts:    append([]LoginAttempt(nil), source.LoginAttempts...),
		SecurityEntrance: source.SecurityEntrance,
	}
}

func syncDirectory(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open store directory: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync store directory: %w", err)
	}
	return nil
}
