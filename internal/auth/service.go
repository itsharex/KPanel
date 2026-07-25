package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/store"
)

var (
	ErrBootstrapUnavailable  = errors.New("bootstrap is unavailable")
	ErrInvalidBootstrapToken = errors.New("invalid bootstrap token")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrInvalidSession        = errors.New("invalid session")
	ErrInvalidCSRFToken      = errors.New("invalid csrf token")
	ErrRateLimited           = errors.New("too many login attempts")
	ErrInvalidUsername       = errors.New("username must contain 3-32 letters, numbers, dots, underscores, or hyphens")
	ErrWeakPassword          = errors.New("password must contain 12-256 bytes")
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{2,31}$`)

type Config struct {
	BootstrapTokenPath string
	SessionTTL         time.Duration
	LoginWindow        time.Duration
	MaxLoginFailures   int
}

type Service struct {
	store     *store.Store
	hasher    PasswordHasher
	config    Config
	now       func() time.Time
	dummyHash string
}

type PublicUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type Credentials struct {
	Token     string
	CSRFToken string
	ExpiresAt time.Time
	User      PublicUser
}

type Session struct {
	TokenHash string
	CSRFHash  string
	ExpiresAt time.Time
	User      PublicUser
}

type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return ErrRateLimited.Error()
}

func (e *RateLimitError) Unwrap() error {
	return ErrRateLimited
}

func NewService(storage *store.Store, hasher PasswordHasher, config Config) (*Service, error) {
	if storage == nil {
		return nil, errors.New("auth store is required")
	}
	if hasher == nil {
		return nil, errors.New("password hasher is required")
	}
	if config.BootstrapTokenPath == "" {
		return nil, errors.New("bootstrap token path is required")
	}
	if config.SessionTTL <= 0 {
		config.SessionTTL = 12 * time.Hour
	}
	if config.LoginWindow <= 0 {
		config.LoginWindow = 15 * time.Minute
	}
	if config.MaxLoginFailures <= 0 {
		config.MaxLoginFailures = 5
	}

	dummyHash, err := hasher.Hash("kejilion-panel-invalid-password")
	if err != nil {
		return nil, fmt.Errorf("prepare password verifier: %w", err)
	}
	return &Service{
		store:     storage,
		hasher:    hasher,
		config:    config,
		now:       func() time.Time { return time.Now().UTC() },
		dummyHash: dummyHash,
	}, nil
}

func (s *Service) IsInitialized() bool {
	return s.store.IsInitialized()
}

// EnsureBootstrapToken creates a secret once without ever returning or logging
// its value. An installer with local filesystem access reads the 0600 file.
func (s *Service) EnsureBootstrapToken() error {
	path := s.config.BootstrapTokenPath
	if s.store.IsInitialized() {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove obsolete bootstrap token: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create bootstrap token directory: %w", err)
	}
	if content, err := os.ReadFile(path); err == nil {
		if strings.TrimSpace(string(content)) == "" {
			return errors.New("bootstrap token file is empty")
		}
		return os.Chmod(path, 0o600)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read bootstrap token: %w", err)
	}

	token, err := randomToken(32)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("create bootstrap token: %w", err)
	}
	if _, err := file.WriteString(token + "\n"); err != nil {
		file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write bootstrap token: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("sync bootstrap token: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close bootstrap token: %w", err)
	}
	return nil
}

func (s *Service) Bootstrap(token, username, password string) (Credentials, error) {
	if s.store.IsInitialized() {
		return Credentials{}, ErrBootstrapUnavailable
	}
	expected, err := os.ReadFile(s.config.BootstrapTokenPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Credentials{}, ErrBootstrapUnavailable
		}
		return Credentials{}, fmt.Errorf("read bootstrap token: %w", err)
	}
	if !secureEqual(strings.TrimSpace(string(expected)), strings.TrimSpace(token)) {
		return Credentials{}, ErrInvalidBootstrapToken
	}
	if err := validateUsername(username); err != nil {
		return Credentials{}, err
	}
	if err := validatePassword(password); err != nil {
		return Credentials{}, err
	}

	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return Credentials{}, fmt.Errorf("hash password: %w", err)
	}
	now := s.now()
	userID, err := randomToken(18)
	if err != nil {
		return Credentials{}, err
	}
	user := store.User{
		ID:           userID,
		Username:     username,
		PasswordHash: passwordHash,
		Role:         "admin",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.store.CreateInitialAdmin(user); err != nil {
		if errors.Is(err, store.ErrAlreadyInitialized) {
			return Credentials{}, ErrBootstrapUnavailable
		}
		return Credentials{}, err
	}
	s.consumeBootstrapToken()
	return s.createSession(user)
}

func (s *Service) Login(ip, username, password string) (Credentials, error) {
	now := s.now()
	ipKey, accountKey := loginKeys(ip, username)
	since := now.Add(-s.config.LoginWindow)
	if s.store.FailedLoginCount(ipKey, since) >= s.config.MaxLoginFailures ||
		s.store.FailedLoginCount(accountKey, since) >= s.config.MaxLoginFailures {
		return Credentials{}, &RateLimitError{RetryAfter: s.config.LoginWindow}
	}

	user, userErr := s.store.UserByUsername(username)
	hash := s.dummyHash
	if userErr == nil {
		hash = user.PasswordHash
	}
	passwordToVerify := password
	inputInvalid := len(password) < 1 || len(password) > 256 || len(username) > 64
	if inputInvalid {
		passwordToVerify = "invalid-input"
		hash = s.dummyHash
	}
	valid, verifyErr := s.hasher.Verify(passwordToVerify, hash)
	if inputInvalid || userErr != nil || verifyErr != nil || !valid {
		if err := s.recordLoginAttempt(ipKey, accountKey, now, false); err != nil {
			return Credentials{}, err
		}
		return Credentials{}, ErrInvalidCredentials
	}
	if err := s.recordLoginAttempt(ipKey, accountKey, now, true); err != nil {
		return Credentials{}, err
	}
	return s.createSession(user)
}

func (s *Service) Authenticate(token string) (Session, error) {
	if token == "" {
		return Session{}, ErrInvalidSession
	}
	tokenHash := hashSecret(token)
	session, err := s.store.SessionByTokenHash(tokenHash, s.now())
	if err != nil {
		return Session{}, ErrInvalidSession
	}
	user, err := s.store.UserByID(session.UserID)
	if err != nil {
		return Session{}, ErrInvalidSession
	}
	return Session{
		TokenHash: tokenHash,
		CSRFHash:  session.CSRFHash,
		ExpiresAt: session.ExpiresAt,
		User:      publicUser(user),
	}, nil
}

func (s *Service) ValidateCSRF(session Session, token string) error {
	if token == "" || !secureEqual(session.CSRFHash, hashSecret(token)) {
		return ErrInvalidCSRFToken
	}
	return nil
}

func (s *Service) Logout(sessionToken string) error {
	if sessionToken == "" {
		return nil
	}
	return s.store.DeleteSession(hashSecret(sessionToken))
}

func (s *Service) createSession(user store.User) (Credentials, error) {
	token, err := randomToken(32)
	if err != nil {
		return Credentials{}, err
	}
	csrfToken, err := randomToken(32)
	if err != nil {
		return Credentials{}, err
	}
	now := s.now()
	expiresAt := now.Add(s.config.SessionTTL)
	if err := s.store.PutSession(store.Session{
		TokenHash: hashSecret(token),
		CSRFHash:  hashSecret(csrfToken),
		UserID:    user.ID,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}); err != nil {
		return Credentials{}, err
	}
	return Credentials{
		Token:     token,
		CSRFToken: csrfToken,
		ExpiresAt: expiresAt,
		User:      publicUser(user),
	}, nil
}

func (s *Service) recordLoginAttempt(ipKey, accountKey string, now time.Time, success bool) error {
	retainSince := now.Add(-24 * time.Hour)
	for _, key := range []string{ipKey, accountKey} {
		if err := s.store.RecordLoginAttempt(store.LoginAttempt{
			Key:        key,
			OccurredAt: now,
			Success:    success,
		}, retainSince); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) consumeBootstrapToken() {
	if err := os.Remove(s.config.BootstrapTokenPath); err == nil || errors.Is(err, os.ErrNotExist) {
		return
	}
	_ = os.WriteFile(s.config.BootstrapTokenPath, nil, 0o600)
}

func validateUsername(username string) error {
	if !usernamePattern.MatchString(username) {
		return ErrInvalidUsername
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 12 || len(password) > 256 {
		return ErrWeakPassword
	}
	return nil
}

func loginKeys(ip, username string) (string, string) {
	ip = strings.TrimSpace(strings.ToLower(ip))
	if len(ip) > 128 {
		ip = ip[:128]
	}
	username = strings.TrimSpace(strings.ToLower(username))
	if len(username) > 64 {
		username = username[:64]
	}
	return "ip:" + ip, "account:" + username
}

func publicUser(user store.User) PublicUser {
	return PublicUser{ID: user.ID, Username: user.Username, Role: user.Role}
}

func randomToken(bytes int) (string, error) {
	value := make([]byte, bytes)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func hashSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func secureEqual(left, right string) bool {
	leftHash := sha256.Sum256([]byte(left))
	rightHash := sha256.Sum256([]byte(right))
	return subtle.ConstantTimeCompare(leftHash[:], rightHash[:]) == 1
}
