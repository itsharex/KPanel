package ai

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"testing"
)

func TestValidateProviderURL(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		scope EndpointScope
		ok    bool
	}{
		{"public HTTPS", "https://api.example.com/v1/", EndpointPublic, true},
		{"public HTTP", "http://api.example.com/v1", EndpointPublic, false},
		{"private HTTP", "http://192.168.1.2:11434/v1", EndpointPrivate, true},
		{"public loopback", "https://127.0.0.1/v1", EndpointPublic, false},
		{"userinfo", "https://user@example.com/v1", EndpointPublic, false},
		{"query", "https://api.example.com/v1?x=1", EndpointPublic, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ValidateProviderURL(test.url, test.scope)
			if (err == nil) != test.ok {
				t.Fatalf("error = %v, want ok=%v", err, test.ok)
			}
		})
	}
	if err := ValidateResolvedAddresses(EndpointPublic, []net.IPAddr{{IP: net.ParseIP("10.0.0.1")}}); err == nil {
		t.Fatal("resolved private address must be rejected")
	}
}

func TestProviderServiceEncryptsAPIKey(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	box, err := OpenSecretBox(filepath.Join(dir, "ai-secrets.key"), false)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewProviderService(store, box)
	if err != nil {
		t.Fatal(err)
	}
	key := "sk-example-1234"
	provider, err := service.Save(ctx, "", ProviderInput{Name: "OpenAI", Protocol: ProtocolOpenAICompatible, BaseURL: "https://api.openai.com/v1", EndpointScope: EndpointPublic, APIKey: &key, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if !provider.APIKeySet || provider.APIKeyHint != "1234" {
		t.Fatalf("unexpected public secret metadata: %#v", provider)
	}
	if provider.APIMode != OpenAIChatCompletions {
		t.Fatalf("legacy OpenAI-compatible providers must default to Chat Completions, got %q", provider.APIMode)
	}
	stored, err := store.Provider(ctx, provider.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored.EncryptedKey) == key {
		t.Fatal("API key was stored in plaintext")
	}
	plain, err := service.APIKey(stored)
	if err != nil || plain != key {
		t.Fatalf("decrypted key = %q, %v", plain, err)
	}
}

func TestProviderServiceValidatesOpenAIAPIMode(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	box, err := OpenSecretBox(filepath.Join(dir, "ai-secrets.key"), false)
	if err != nil {
		t.Fatal(err)
	}
	service, _ := NewProviderService(store, box)
	provider, err := service.Save(context.Background(), "", ProviderInput{Name: "Responses", Protocol: ProtocolOpenAICompatible, APIMode: OpenAIResponses, BaseURL: "https://api.example.com/v1", EndpointScope: EndpointPublic, Enabled: true})
	if err != nil || provider.APIMode != OpenAIResponses {
		t.Fatalf("provider=%#v err=%v", provider, err)
	}
	_, err = service.Save(context.Background(), "", ProviderInput{Name: "Invalid", Protocol: ProtocolOpenAICompatible, APIMode: "realtime", BaseURL: "https://api.example.com/v1", EndpointScope: EndpointPublic, Enabled: true})
	if err == nil {
		t.Fatal("unsupported OpenAI API mode was accepted")
	}
}

func TestPrivateProviderRequiresExplicitConfirmation(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	box, err := OpenSecretBox(filepath.Join(t.TempDir(), "ai-secrets.key"), false)
	if err != nil {
		t.Fatal(err)
	}
	service, _ := NewProviderService(store, box)
	_, err = service.Save(context.Background(), "", ProviderInput{Name: "local", Protocol: ProtocolOpenAICompatible, BaseURL: "http://127.0.0.1:11434/v1", EndpointScope: EndpointPrivate, Enabled: true})
	if err == nil {
		t.Fatal("private provider was accepted without explicit confirmation")
	}
}

func TestProviderUpdateIsFrozenDuringActiveRun(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	box, _ := OpenSecretBox(filepath.Join(dir, "ai-secrets.key"), false)
	service, _ := NewProviderService(store, box)
	key := "key"
	provider, err := service.Save(ctx, "", ProviderInput{Name: "provider", Protocol: ProtocolOpenAICompatible, BaseURL: "https://example.com/v1", EndpointScope: EndpointPublic, APIKey: &key, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveModels(ctx, provider.ID, []Model{{ModelID: "model", DisplayName: "model", ContextWindow: 8192, Enabled: true}}); err != nil {
		t.Fatal(err)
	}
	models, _ := store.ListModels(ctx, provider.ID)
	session, _ := store.CreateSession(ctx, Session{UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: models[0].ID, ModelName: models[0].DisplayName})
	_, _ = store.CreateRun(ctx, Run{SessionID: session.ID, UserID: "admin", ProviderID: provider.ID, ModelID: models[0].ID})
	_, err = service.Save(ctx, provider.ID, ProviderInput{Name: "changed", Protocol: provider.Protocol, BaseURL: provider.BaseURL, EndpointScope: provider.EndpointScope, Enabled: true, ExpectedVersion: provider.Version})
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("provider changed during active run: %v", err)
	}
}
