package ai

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

type syncModelsClient struct {
	models []Model
}

func (client *syncModelsClient) Stream(context.Context, Provider, string, CompletionRequest, func(CompletionEvent) error) error {
	return nil
}

func (client *syncModelsClient) Models(context.Context, Provider, string) ([]Model, error) {
	return append([]Model(nil), client.models...), nil
}

func TestSyncModelsDefaultsAllModelsToVision(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	secrets, err := OpenSecretBox(filepath.Join(dir, "ai-secrets.key"), false)
	if err != nil {
		t.Fatal(err)
	}
	providers, err := NewProviderService(store, secrets)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := store.SaveProvider(ctx, Provider{Name: "Mock", Protocol: ProtocolOpenAICompatible, BaseURL: "https://example.com/v1", EndpointScope: EndpointPublic, Enabled: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveModels(ctx, provider.ID, []Model{{ModelID: "existing", ContextWindow: 8192, Vision: false, Reasoning: true, Enabled: true}}); err != nil {
		t.Fatal(err)
	}
	service := &Service{Store: store, Providers: providers, client: &syncModelsClient{models: []Model{
		{ModelID: "existing", ContextWindow: 8192, Enabled: true},
		{ModelID: "plain-new-model", ContextWindow: 8192, Enabled: true},
	}}}
	models, err := service.SyncModels(ctx, provider.ID)
	if err != nil || len(models) != 2 {
		t.Fatalf("synced models=%#v err=%v", models, err)
	}
	for _, model := range models {
		if !model.Vision {
			t.Fatalf("synced model did not default to vision: %#v", model)
		}
		if model.ModelID == "existing" && !model.Reasoning {
			t.Fatalf("existing reasoning capability was not preserved: %#v", model)
		}
	}
}

func TestInferredReasoningIncludesDeepSeekV4(t *testing.T) {
	for _, modelID := range []string{"deepseek-v4-flash", "deepseek-v4-pro", "deepseek-r1"} {
		if !inferredReasoning(ProtocolOpenAICompatible, modelID) {
			t.Fatalf("model %q was not recognized as a reasoning model", modelID)
		}
	}
	if inferredReasoning(ProtocolOpenAICompatible, "deepseek-chat") {
		t.Fatal("non-reasoning DeepSeek chat model was inferred as reasoning")
	}
}

func TestDeleteSessionCancelsActiveRun(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(filepath.Join(t.TempDir(), "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	provider, err := store.SaveProvider(ctx, Provider{Name: "Mock", Protocol: ProtocolOpenAICompatible, BaseURL: "https://example.com/v1", EndpointScope: EndpointPublic, Enabled: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveModels(ctx, provider.ID, []Model{{ModelID: "model", ContextWindow: 8192, Enabled: true}}); err != nil {
		t.Fatal(err)
	}
	models, _ := store.ListModels(ctx, provider.ID)
	session, err := store.CreateSession(ctx, Session{UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: models[0].ID, ModelName: models[0].DisplayName})
	if err != nil {
		t.Fatal(err)
	}
	run, err := store.CreateRun(ctx, Run{SessionID: session.ID, UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: models[0].ID, ModelName: models[0].DisplayName})
	if err != nil {
		t.Fatal(err)
	}
	events := NewEventHub()
	eventChannel, unsubscribe := events.Subscribe(run.ID)
	defer unsubscribe()
	runtime := &NativeRuntime{store: store, events: events, cancels: make(map[string]context.CancelFunc)}
	service := &Service{Store: store, Runtime: runtime}
	if err := service.DeleteSession(ctx, "admin", session.ID); err != nil {
		t.Fatalf("delete active session: %v", err)
	}
	if _, err := store.Session(ctx, "admin", session.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted session lookup error=%v", err)
	}
	select {
	case event := <-eventChannel:
		if event.Type != "run.cancelled" {
			t.Fatalf("delete cancellation event=%#v", event)
		}
	default:
		t.Fatal("active session delete did not cancel its run")
	}
}

func TestSessionTitleFromMessage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "  查询 CPU 使用情况  ", want: "查询 CPU 使用情况"},
		{input: "检查服务器。然后修复问题", want: "检查服务器"},
		{input: "第一行\n第二行", want: "第一行 第二行"},
	}
	for _, test := range tests {
		if got := sessionTitleFromMessage(test.input); got != test.want {
			t.Errorf("sessionTitleFromMessage(%q)=%q want=%q", test.input, got, test.want)
		}
	}
	long := sessionTitleFromMessage(strings.Repeat("测", 40))
	if len([]rune(long)) != 37 || !strings.HasSuffix(long, "…") {
		t.Fatalf("long title was not safely truncated: %q", long)
	}
}
