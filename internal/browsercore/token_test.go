package browsercore

import (
	"errors"
	"testing"
	"time"
)

func TestTokenCodecIssuesAndVerifiesShortLivedSession(t *testing.T) {
	codec, err := NewTokenCodec([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0).UTC()
	codec.now = func() time.Time { return now }
	token, issued, err := codec.Issue("admin", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := codec.Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	if verified != issued || verified.Subject != "admin" || verified.SessionID == "" || verified.Scope != TokenScopeBeta {
		t.Fatalf("verified claims = %#v", verified)
	}
}

func TestTokenCodecSeparatesReaderAndBetaScopes(t *testing.T) {
	codec, err := NewTokenCodec([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	token, issued, err := codec.IssueScoped("admin", TokenScopeReader, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := codec.Verify(token)
	if err != nil || verified != issued || verified.Scope != TokenScopeReader || verified.Version != 2 {
		t.Fatalf("reader claims = %#v, %v", verified, err)
	}
	if _, _, err := codec.IssueScoped("admin", "write", 10*time.Minute); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("unsupported token scope error = %v", err)
	}
}

func TestTokenCodecRejectsTamperingAndExpiry(t *testing.T) {
	codec, err := NewTokenCodec([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0).UTC()
	codec.now = func() time.Time { return now }
	token, _, err := codec.Issue("admin", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := codec.Verify(token + "x"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("tampered token error = %v", err)
	}
	codec.now = func() time.Time { return now.Add(2 * time.Minute) }
	if _, err := codec.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expired token error = %v", err)
	}
}

func TestTokenCodecEnforcesSecretAndTTLBounds(t *testing.T) {
	if _, err := NewTokenCodec([]byte("short")); err == nil {
		t.Fatal("short secret accepted")
	}
	codec, _ := NewTokenCodec([]byte("0123456789abcdef0123456789abcdef"))
	if _, _, err := codec.Issue("admin", 30*time.Second); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("short TTL error = %v", err)
	}
	if _, _, err := codec.Issue("admin", 16*time.Minute); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("long TTL error = %v", err)
	}
}
