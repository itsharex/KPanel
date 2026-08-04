package ai

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

type Service struct {
	Store     *Store
	Providers *ProviderService
	Runtime   AgentRuntime
	Events    *EventHub
	client    ModelClient
}

func Open(dataDir string, tools ToolExecutor) (*Service, error) {
	store, err := OpenStore(filepath.Join(dataDir, "ai.db"))
	if err != nil {
		return nil, err
	}
	fail := true
	defer func() {
		if fail {
			_ = store.Close()
		}
	}()
	count, err := store.EncryptedSecretCount(context.Background())
	if err != nil {
		return nil, err
	}
	secrets, err := OpenSecretBox(filepath.Join(dataDir, "ai-secrets.key"), count > 0)
	if err != nil {
		return nil, err
	}
	providers, err := NewProviderService(store, secrets)
	if err != nil {
		return nil, err
	}
	events := NewEventHub()
	client := NewHTTPModelClient()
	runtime, err := NewNativeRuntime(store, providers, client, tools, events)
	if err != nil {
		return nil, err
	}
	fail = false
	return &Service{Store: store, Providers: providers, Runtime: runtime, Events: events, client: client}, nil
}

func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	if closer, ok := s.Runtime.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
	return s.Store.Close()
}

func (s *Service) SyncModels(ctx context.Context, providerID string) ([]Model, error) {
	provider, err := s.Store.Provider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	key, err := s.Providers.APIKey(provider)
	if err != nil {
		return nil, err
	}
	items, err := s.client.Models(ctx, provider, key)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index].ProviderID = provider.ID
	}
	if err := s.Store.SaveModels(ctx, provider.ID, items); err != nil {
		return nil, err
	}
	return s.Store.ListModels(ctx, provider.ID)
}

func (s *Service) CreateSession(ctx context.Context, userID, providerID, modelID, title string) (Session, error) {
	title = strings.TrimSpace(title)
	if len(title) > 120 || strings.IndexFunc(title, func(r rune) bool { return r < 0x20 }) >= 0 {
		return Session{}, errors.New("session title is invalid")
	}
	provider, err := s.Store.Provider(ctx, providerID)
	if err != nil || !provider.Enabled {
		return Session{}, errors.New("provider is unavailable")
	}
	model, err := s.Store.Model(ctx, modelID)
	if err != nil || model.ProviderID != providerID || !model.Enabled {
		return Session{}, errors.New("model is unavailable")
	}
	return s.Store.CreateSession(ctx, Session{UserID: userID, Title: title, ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
}

func (s *Service) Send(ctx context.Context, userID, sessionID, content string) (Run, error) {
	content = strings.TrimSpace(content)
	if content == "" || len(content) > MaxUserMessageBytes {
		return Run{}, errors.New("message must be between 1 byte and 16 KiB")
	}
	session, err := s.Store.Session(ctx, userID, sessionID)
	if err != nil {
		return Run{}, err
	}
	if !session.ModelAvailable {
		return Run{}, errors.New("session model is unavailable")
	}
	if session.Title == "新会话" {
		if err := s.Store.SetInitialSessionTitle(ctx, userID, session.ID, sessionTitleFromMessage(content)); err != nil {
			return Run{}, err
		}
	}
	active, activeErr := s.Store.ActiveRun(ctx, session.ID, userID)
	if activeErr == nil {
		if _, err := s.Store.AddMessage(ctx, Message{SessionID: session.ID, RunID: active.ID, Role: RoleUser, Content: content}); err != nil {
			return Run{}, err
		}
		return active, nil
	}
	if !errors.Is(activeErr, ErrNotFound) {
		return Run{}, activeErr
	}
	run, err := s.Store.CreateRun(ctx, Run{SessionID: session.ID, UserID: userID, ProviderID: session.ProviderID, ProviderName: session.ProviderName, ModelID: session.ModelID, ModelName: session.ModelName, ApprovalMode: session.ApprovalMode})
	if err != nil {
		return Run{}, err
	}
	if _, err := s.Store.AddMessage(ctx, Message{SessionID: session.ID, RunID: run.ID, Role: RoleUser, Content: content}); err != nil {
		run.Status, run.ErrorCode, run.ErrorMessage = RunFailed, "message_store_failed", err.Error()
		_ = s.Store.UpdateRun(ctx, run)
		return Run{}, err
	}
	go func() {
		if err := s.Runtime.Run(context.Background(), run.ID); err != nil && !errors.Is(err, context.Canceled) {
			_ = err
		}
	}()
	return run, nil
}

func sessionTitleFromMessage(content string) string {
	content = strings.Map(func(value rune) rune {
		if unicode.IsControl(value) {
			return ' '
		}
		return value
	}, strings.TrimSpace(content))
	content = strings.Join(strings.Fields(content), " ")
	for _, separator := range []string{"。", "！", "？", "\n", ". ", "! ", "? "} {
		if index := strings.Index(content, separator); index > 0 {
			content = strings.TrimSpace(content[:index])
		}
	}
	characters := []rune(content)
	if len(characters) > 36 {
		content = strings.TrimSpace(string(characters[:36])) + "…"
	}
	if content == "" {
		return "新会话"
	}
	return content
}

func (s *Service) Resume(runID string, decision Decision) {
	go func() { _ = s.Runtime.Resume(context.Background(), runID, decision) }()
}

func (s *Service) Retry(ctx context.Context, userID, runID string) (Run, error) {
	previous, err := s.Store.Run(ctx, userID, runID)
	if err != nil {
		return Run{}, err
	}
	if previous.Status != RunInterrupted && previous.Status != RunFailed {
		return Run{}, ErrConflict
	}
	run, err := s.Store.CreateRun(ctx, Run{SessionID: previous.SessionID, UserID: userID, ProviderID: previous.ProviderID, ProviderName: previous.ProviderName, ModelID: previous.ModelID, ModelName: previous.ModelName, ApprovalMode: previous.ApprovalMode})
	if err != nil {
		return Run{}, err
	}
	go func() { _ = s.Runtime.Run(context.Background(), run.ID) }()
	return run, nil
}

func (s *Service) Propose(ctx context.Context, userID, runID string) error {
	run, err := s.Store.Run(ctx, userID, runID)
	if err != nil {
		return err
	}
	runtime, ok := s.Runtime.(interface {
		Propose(context.Context, string) error
	})
	if !ok {
		return errors.New("runtime does not support evolution proposals")
	}
	return runtime.Propose(ctx, run.ID)
}

func (s *Service) TestProvider(ctx context.Context, id string) error {
	provider, err := s.Store.Provider(ctx, id)
	if err != nil {
		return err
	}
	key, err := s.Providers.APIKey(provider)
	if err != nil {
		return err
	}
	if provider.Protocol != ProtocolAnthropic {
		_, err = s.client.Models(ctx, provider, key)
		return err
	}
	models, err := s.Store.ListModels(ctx, provider.ID)
	if err != nil {
		return err
	}
	if len(models) == 0 {
		return errors.New("add an Anthropic model before testing")
	}
	var got bool
	err = s.client.Stream(ctx, provider, key, CompletionRequest{Model: models[0].ModelID, System: "Reply OK.", Messages: []ChatMessage{{Role: "user", Content: "OK"}}}, func(event CompletionEvent) error { got = got || event.Delta != ""; return nil })
	if err != nil {
		return err
	}
	if !got {
		return fmt.Errorf("provider returned no content")
	}
	return nil
}
