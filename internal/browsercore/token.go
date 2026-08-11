package browsercore

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	tokenAudience = "kpanel-browser-relay"
	maxTokenBytes = 2 << 10
	minTokenTTL   = time.Minute
	maxTokenTTL   = 15 * time.Minute
)

var ErrInvalidToken = errors.New("invalid browser relay token")

type Claims struct {
	Version   int    `json:"v"`
	SessionID string `json:"sid"`
	Subject   string `json:"sub"`
	Audience  string `json:"aud"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type TokenCodec struct {
	secret []byte
	now    func() time.Time
}

func NewTokenCodec(secret []byte) (*TokenCodec, error) {
	if len(secret) < 32 {
		return nil, errors.New("browser relay secret must contain at least 32 bytes")
	}
	return &TokenCodec{secret: append([]byte(nil), secret...), now: time.Now}, nil
}

func (c *TokenCodec) Issue(subject string, ttl time.Duration) (string, Claims, error) {
	if !validTokenText(subject, 128) || ttl < minTokenTTL || ttl > maxTokenTTL {
		return "", Claims{}, ErrInvalidToken
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", Claims{}, fmt.Errorf("generate browser session id: %w", err)
	}
	now := c.now().UTC()
	claims := Claims{
		Version:   1,
		SessionID: base64.RawURLEncoding.EncodeToString(random),
		Subject:   subject,
		Audience:  tokenAudience,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
	}
	token, err := c.sign(claims)
	return token, claims, err
}

func (c *TokenCodec) Verify(token string) (Claims, error) {
	if token == "" || len(token) > maxTokenBytes || strings.TrimSpace(token) != token {
		return Claims{}, ErrInvalidToken
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Claims{}, ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(payload) > 1024 {
		return Claims{}, ErrInvalidToken
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(signature) != sha256.Size {
		return Claims{}, ErrInvalidToken
	}
	expected := c.signature(parts[0])
	if !hmac.Equal(signature, expected) {
		return Claims{}, ErrInvalidToken
	}
	var claims Claims
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&claims) != nil || claims.Version != 1 || claims.Audience != tokenAudience ||
		!validTokenText(claims.SessionID, 64) || !validTokenText(claims.Subject, 128) {
		return Claims{}, ErrInvalidToken
	}
	now := c.now().UTC().Unix()
	if claims.IssuedAt > now+30 || claims.ExpiresAt <= now || claims.ExpiresAt-claims.IssuedAt > int64(maxTokenTTL/time.Second) {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}

func (c *TokenCodec) sign(claims Claims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	signature := base64.RawURLEncoding.EncodeToString(c.signature(encoded))
	return encoded + "." + signature, nil
}

func (c *TokenCodec) signature(payload string) []byte {
	mac := hmac.New(sha256.New, c.secret)
	_, _ = mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func validTokenText(value string, limit int) bool {
	return value != "" && len(value) <= limit && strings.TrimSpace(value) == value &&
		!strings.ContainsAny(value, "\x00\r\n\t")
}
