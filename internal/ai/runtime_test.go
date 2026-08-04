package ai

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type scriptedClient struct {
	mu    sync.Mutex
	calls int
	tool  string
}

func (c *scriptedClient) Stream(_ context.Context, _ Provider, _ string, _ CompletionRequest, emit func(CompletionEvent) error) error {
	c.mu.Lock()
	c.calls++
	call := c.calls
	c.mu.Unlock()
	if call == 1 {
		return emit(CompletionEvent{Done: true, ToolCalls: []ToolCall{{ID: "call_1", Name: c.tool, Arguments: json.RawMessage(`{"resourceVersion":"sha256:test"}`)}}})
	}
	if err := emit(CompletionEvent{Delta: "完成"}); err != nil {
		return err
	}
	return emit(CompletionEvent{Done: true})
}
func (*scriptedClient) Models(context.Context, Provider, string) ([]Model, error) { return nil, nil }

type fakeTools struct {
	mu         sync.Mutex
	executed   int
	readOnly   bool
	approval   *bool
	executeErr error
}

func (f *fakeTools) Definitions() []ToolDefinition {
	return []ToolDefinition{{Name: "host_action", Description: "test", Schema: json.RawMessage(`{"type":"object"}`), ReadOnly: f.readOnly}}
}
func (f *fakeTools) DryRun(string, json.RawMessage) error { return nil }
func (f *fakeTools) RequiresApproval(string, json.RawMessage) bool {
	if f.approval != nil {
		return *f.approval
	}
	return !f.readOnly
}
func (f *fakeTools) Execute(context.Context, ToolExecutionContext, string, json.RawMessage) (string, error) {
	f.mu.Lock()
	f.executed++
	f.mu.Unlock()
	return `{"status":"ok"}`, f.executeErr
}

func TestNativeRuntimeReplansAfterResourceVersionConflict(t *testing.T) {
	store, providerService, provider, model := runtimeFixture(t)
	defer store.Close()
	session, _ := store.CreateSession(context.Background(), Session{UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	_, _ = store.AddMessage(context.Background(), Message{SessionID: session.ID, Role: RoleUser, Content: "执行"})
	run, _ := store.CreateRun(context.Background(), Run{SessionID: session.ID, UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	runtime, _ := NewNativeRuntime(store, providerService, &scriptedClient{tool: "host_action"}, &fakeTools{executeErr: ErrToolConflict}, NewEventHub())
	if err := runtime.Run(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	}
	calls, _ := store.ToolCalls(context.Background(), run.ID)
	if err := runtime.Resume(context.Background(), run.ID, Decision{ToolCallID: calls[0].ID, Approve: true}); err != nil {
		t.Fatal(err)
	}
	loaded, _ := store.Run(context.Background(), "admin", run.ID)
	if loaded.Status != RunCompleted {
		t.Fatalf("conflict should be returned to the model for replanning, status=%s", loaded.Status)
	}
	messages, _ := store.Messages(context.Background(), session.ID, 50)
	found := false
	for _, message := range messages {
		found = found || strings.Contains(message.Content, "重新读取状态")
	}
	if !found {
		t.Fatal("resource-version conflict was not added to model context")
	}
}

func TestNativeRuntimeApprovalAndResume(t *testing.T) {
	store, providerService, provider, model := runtimeFixture(t)
	defer store.Close()
	session, _ := store.CreateSession(context.Background(), Session{UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	_, _ = store.AddMessage(context.Background(), Message{SessionID: session.ID, Role: RoleUser, Content: "执行"})
	run, _ := store.CreateRun(context.Background(), Run{SessionID: session.ID, UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	tools := &fakeTools{}
	runtime, err := NewNativeRuntime(store, providerService, &scriptedClient{tool: "host_action"}, tools, NewEventHub())
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Run(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	}
	loaded, _ := store.Run(context.Background(), "admin", run.ID)
	if loaded.Status != RunPendingApproval || tools.executed != 0 {
		t.Fatalf("status=%s executed=%d", loaded.Status, tools.executed)
	}
	calls, _ := store.ToolCalls(context.Background(), run.ID)
	if err := runtime.Resume(context.Background(), run.ID, Decision{ToolCallID: calls[0].ID, Approve: true}); err != nil {
		t.Fatal(err)
	}
	loaded, _ = store.Run(context.Background(), "admin", run.ID)
	if loaded.Status != RunCompleted || tools.executed != 1 {
		t.Fatalf("status=%s executed=%d", loaded.Status, tools.executed)
	}
}

func TestNativeRuntimeReadOnlyToolRunsWithoutApproval(t *testing.T) {
	store, providerService, provider, model := runtimeFixture(t)
	defer store.Close()
	session, _ := store.CreateSession(context.Background(), Session{UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	_, _ = store.AddMessage(context.Background(), Message{SessionID: session.ID, Role: RoleUser, Content: "读取"})
	run, _ := store.CreateRun(context.Background(), Run{SessionID: session.ID, UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	tools := &fakeTools{readOnly: true}
	runtime, _ := NewNativeRuntime(store, providerService, &scriptedClient{tool: "host_action"}, tools, NewEventHub())
	if err := runtime.Run(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	}
	loaded, _ := store.Run(context.Background(), "admin", run.ID)
	if loaded.Status != RunCompleted || tools.executed != 1 {
		t.Fatalf("status=%s executed=%d", loaded.Status, tools.executed)
	}
}

func TestNativeRuntimeApprovedClassifiedWriteRunsWithoutApproval(t *testing.T) {
	store, providerService, provider, model := runtimeFixture(t)
	defer store.Close()
	session, _ := store.CreateSession(context.Background(), Session{UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	_, _ = store.AddMessage(context.Background(), Message{SessionID: session.ID, Role: RoleUser, Content: "执行常规操作"})
	run, _ := store.CreateRun(context.Background(), Run{SessionID: session.ID, UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	requiresApproval := false
	tools := &fakeTools{approval: &requiresApproval}
	runtime, _ := NewNativeRuntime(store, providerService, &scriptedClient{tool: "host_action"}, tools, NewEventHub())
	if err := runtime.Run(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	}
	loaded, _ := store.Run(context.Background(), "admin", run.ID)
	if loaded.Status != RunCompleted || tools.executed != 1 {
		t.Fatalf("status=%s executed=%d", loaded.Status, tools.executed)
	}
	calls, _ := store.ToolCalls(context.Background(), run.ID)
	if len(calls) != 1 || calls[0].RequiresApproval {
		t.Fatalf("classified write unexpectedly required approval: %#v", calls)
	}
}

type queuedClient struct {
	started chan struct{}
	release chan struct{}
	mu      sync.Mutex
	calls   int
}

func (c *queuedClient) Stream(_ context.Context, _ Provider, _ string, _ CompletionRequest, emit func(CompletionEvent) error) error {
	c.mu.Lock()
	c.calls++
	call := c.calls
	c.mu.Unlock()
	if call == 1 {
		close(c.started)
		<-c.release
	}
	if err := emit(CompletionEvent{Delta: "answer"}); err != nil {
		return err
	}
	return emit(CompletionEvent{Done: true})
}

func (*queuedClient) Models(context.Context, Provider, string) ([]Model, error) { return nil, nil }

func TestServiceQueuesMessageIntoActiveSessionRun(t *testing.T) {
	store, providers, provider, model := runtimeFixture(t)
	defer store.Close()
	client := &queuedClient{started: make(chan struct{}), release: make(chan struct{})}
	runtime, err := NewNativeRuntime(store, providers, client, &fakeTools{readOnly: true}, NewEventHub())
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{Store: store, Providers: providers, Runtime: runtime}
	session, err := service.CreateSession(context.Background(), "admin", provider.ID, model.ID, "queue")
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Send(context.Background(), "admin", session.ID, "first")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-client.started:
	case <-time.After(2 * time.Second):
		t.Fatal("first model request did not start")
	}
	second, err := service.Send(context.Background(), "admin", session.ID, "second")
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("queued message created another run: %s != %s", second.ID, first.ID)
	}
	close(client.release)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		loaded, loadErr := store.Run(context.Background(), "admin", first.ID)
		if loadErr == nil && loaded.Status == RunCompleted {
			client.mu.Lock()
			calls := client.calls
			client.mu.Unlock()
			if calls != 2 {
				t.Fatalf("model calls=%d, want 2", calls)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("queued run did not complete")
}

type concurrencyClient struct {
	mu          sync.Mutex
	active, max int
	started     chan struct{}
	release     chan struct{}
}

func (c *concurrencyClient) Stream(_ context.Context, _ Provider, _ string, _ CompletionRequest, emit func(CompletionEvent) error) error {
	c.mu.Lock()
	c.active++
	if c.active > c.max {
		c.max = c.active
	}
	c.mu.Unlock()
	c.started <- struct{}{}
	<-c.release
	c.mu.Lock()
	c.active--
	c.mu.Unlock()
	return emit(CompletionEvent{Delta: "ok", Done: true})
}

func (*concurrencyClient) Models(context.Context, Provider, string) ([]Model, error) { return nil, nil }

func TestNativeRuntimeLimitsGlobalConcurrencyToTwo(t *testing.T) {
	store, providers, provider, model := runtimeFixture(t)
	defer store.Close()
	client := &concurrencyClient{started: make(chan struct{}, 3), release: make(chan struct{})}
	runtime, err := NewNativeRuntime(store, providers, client, &fakeTools{readOnly: true}, NewEventHub())
	if err != nil {
		t.Fatal(err)
	}
	runs := make([]Run, 0, 3)
	for index := 0; index < 3; index++ {
		session, createErr := store.CreateSession(context.Background(), Session{UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
		if createErr != nil {
			t.Fatal(createErr)
		}
		run, createErr := store.CreateRun(context.Background(), Run{SessionID: session.ID, UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
		if createErr != nil {
			t.Fatal(createErr)
		}
		_, _ = store.AddMessage(context.Background(), Message{SessionID: session.ID, RunID: run.ID, Role: RoleUser, Content: "run"})
		runs = append(runs, run)
	}
	var wait sync.WaitGroup
	errorsOut := make(chan error, 3)
	for _, run := range runs {
		wait.Add(1)
		go func(id string) {
			defer wait.Done()
			errorsOut <- runtime.Run(context.Background(), id)
		}(run.ID)
	}
	for index := 0; index < 2; index++ {
		select {
		case <-client.started:
		case <-time.After(2 * time.Second):
			t.Fatal("two runs did not start")
		}
	}
	select {
	case <-client.started:
		t.Fatal("third run bypassed the global concurrency limit")
	case <-time.After(100 * time.Millisecond):
	}
	close(client.release)
	wait.Wait()
	close(errorsOut)
	for runErr := range errorsOut {
		if runErr != nil {
			t.Fatal(runErr)
		}
	}
	client.mu.Lock()
	maximum := client.max
	client.mu.Unlock()
	if maximum != 2 {
		t.Fatalf("maximum concurrent requests=%d, want 2", maximum)
	}
}

type recordingRuntime struct{ started chan string }

func (r *recordingRuntime) Run(_ context.Context, id string) error       { r.started <- id; return nil }
func (*recordingRuntime) Resume(context.Context, string, Decision) error { return nil }
func (*recordingRuntime) Cancel(context.Context, string) error           { return nil }

func TestServiceRetriesInterruptedRunWithOriginalSnapshot(t *testing.T) {
	store, _, provider, model := runtimeFixture(t)
	defer store.Close()
	session, _ := store.CreateSession(context.Background(), Session{UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	previous, _ := store.CreateRun(context.Background(), Run{SessionID: session.ID, UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	previous.Status = RunInterrupted
	if err := store.UpdateRun(context.Background(), previous); err != nil {
		t.Fatal(err)
	}
	runtime := &recordingRuntime{started: make(chan string, 1)}
	service := &Service{Store: store, Runtime: runtime}
	retry, err := service.Retry(context.Background(), "admin", previous.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retry.ID == previous.ID || retry.ProviderID != previous.ProviderID || retry.ModelID != previous.ModelID {
		t.Fatalf("retry did not preserve immutable provider/model snapshot: %#v", retry)
	}
	select {
	case started := <-runtime.started:
		if started != retry.ID {
			t.Fatalf("started run=%s want=%s", started, retry.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("retry runtime did not start")
	}
}

type retryClient struct {
	mu         sync.Mutex
	calls      int
	emitBefore bool
	failures   int
}

func (c *retryClient) Stream(_ context.Context, _ Provider, _ string, _ CompletionRequest, emit func(CompletionEvent) error) error {
	c.mu.Lock()
	c.calls++
	call := c.calls
	c.mu.Unlock()
	if c.emitBefore {
		if err := emit(CompletionEvent{Delta: "partial"}); err != nil {
			return err
		}
	}
	if call <= c.failures {
		return &ProviderError{Status: 503, Retryable: true, Message: "temporary"}
	}
	return emit(CompletionEvent{Delta: "done", Done: true})
}
func (*retryClient) Models(context.Context, Provider, string) ([]Model, error) { return nil, nil }

type learningClient struct{ output string }

func (c *learningClient) Stream(_ context.Context, _ Provider, _ string, _ CompletionRequest, emit func(CompletionEvent) error) error {
	if err := emit(CompletionEvent{Delta: c.output}); err != nil {
		return err
	}
	return emit(CompletionEvent{Done: true})
}
func (*learningClient) Models(context.Context, Provider, string) ([]Model, error) { return nil, nil }

func TestNativeRuntimeSilentlyLearnsReusableProcedureAcrossSessions(t *testing.T) {
	store, providers, provider, model := runtimeFixture(t)
	defer store.Close()
	ctx := context.Background()
	session, _ := store.CreateSession(ctx, Session{UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	_, _ = store.AddMessage(ctx, Message{SessionID: session.ID, Role: RoleUser, Content: "检查并恢复应用状态"})
	run, _ := store.CreateRun(ctx, Run{SessionID: session.ID, UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	_, err := store.SaveToolCall(ctx, ToolCall{ID: "learn-1", RunID: run.ID, SessionID: session.ID, Name: "host_action", Arguments: json.RawMessage(`{"resourceVersion":"sha256:test"}`), Status: ToolCompleted})
	if err != nil {
		t.Fatal(err)
	}
	output := `{"decision":"procedure","confidence":0.96,"title":"应用状态恢复","content":"检查并恢复应用状态","condition":"应用异常且需要恢复时","steps":[{"tool":"host_action","arguments":{"resourceVersion":"sha256:test"}}]}`
	requiresApproval := false
	runtime, _ := NewNativeRuntime(store, providers, &learningClient{output: output}, &fakeTools{approval: &requiresApproval}, NewEventHub())
	history, _, _ := store.ContextMessages(ctx, session.ID, model.ContextWindow)
	if err := runtime.generateProposal(ctx, run, provider, "test-key", model, history, false); err != nil {
		t.Fatal(err)
	}
	procedures, _ := store.Procedures(ctx, "admin")
	pending, _ := store.Proposals(ctx, "admin", EvolutionPending)
	if len(procedures) != 1 || !procedures[0].Enabled || len(pending) != 0 {
		t.Fatalf("procedures=%#v pending=%#v", procedures, pending)
	}
	if prompt := runtime.systemPrompt(ctx, "admin"); !strings.Contains(prompt, "应用状态恢复") {
		t.Fatalf("learned procedure was not available to another session: %s", prompt)
	}
	if err := runtime.generateProposal(ctx, run, provider, "test-key", model, history, false); err != nil {
		t.Fatal(err)
	}
	procedures, _ = store.Procedures(ctx, "admin")
	if len(procedures) != 1 {
		t.Fatalf("duplicate learning created %d procedures", len(procedures))
	}
}

func TestNativeRuntimeSkipsLowConfidenceLearning(t *testing.T) {
	store, providers, provider, model := runtimeFixture(t)
	defer store.Close()
	ctx := context.Background()
	session, _ := store.CreateSession(ctx, Session{UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	_, _ = store.AddMessage(ctx, Message{SessionID: session.ID, Role: RoleUser, Content: "记住当前 CPU 数值"})
	run, _ := store.CreateRun(ctx, Run{SessionID: session.ID, UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	runtime, _ := NewNativeRuntime(store, providers, &learningClient{output: `{"decision":"memory","confidence":0.4,"title":"CPU","content":"当前 CPU 为 10%"}`}, &fakeTools{}, NewEventHub())
	history, _, _ := store.ContextMessages(ctx, session.ID, model.ContextWindow)
	if err := runtime.generateProposal(ctx, run, provider, "test-key", model, history, false); err != nil {
		t.Fatal(err)
	}
	memories, _ := store.Memories(ctx, "admin")
	if len(memories) != 0 {
		t.Fatalf("low-confidence transient memory became active: %#v", memories)
	}
}

func TestNativeRuntimeDoesNotSilentlyLearnProtectedProcedure(t *testing.T) {
	store, providers, provider, model := runtimeFixture(t)
	defer store.Close()
	ctx := context.Background()
	session, _ := store.CreateSession(ctx, Session{UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	_, _ = store.AddMessage(ctx, Message{SessionID: session.ID, Role: RoleUser, Content: "执行核心维护"})
	run, _ := store.CreateRun(ctx, Run{SessionID: session.ID, UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
	for _, id := range []string{"protected-1", "protected-2"} {
		_, _ = store.SaveToolCall(ctx, ToolCall{ID: id, RunID: run.ID, SessionID: session.ID, Name: "host_action", Arguments: json.RawMessage(`{}`), Status: ToolCompleted})
	}
	output := `{"decision":"procedure","confidence":0.99,"title":"核心维护","content":"执行维护","condition":"需要维护时","steps":[{"tool":"host_action","arguments":{}}]}`
	runtime, _ := NewNativeRuntime(store, providers, &learningClient{output: output}, &fakeTools{}, NewEventHub())
	history, _, _ := store.ContextMessages(ctx, session.ID, model.ContextWindow)
	if err := runtime.generateProposal(ctx, run, provider, "test-key", model, history, false); err != nil {
		t.Fatal(err)
	}
	procedures, _ := store.Procedures(ctx, "admin")
	pending, _ := store.Proposals(ctx, "admin", EvolutionPending)
	if len(procedures) != 0 || len(pending) != 0 {
		t.Fatalf("protected workflow was learned: procedures=%#v pending=%#v", procedures, pending)
	}
}

func TestRuntimeRetriesOnlyBeforeStreamOutput(t *testing.T) {
	for _, test := range []struct {
		name, wantStatus string
		emitBefore       bool
		failures, calls  int
	}{
		{name: "retry before output", wantStatus: string(RunCompleted), failures: 2, calls: 3},
		{name: "no replay after output", wantStatus: string(RunFailed), emitBefore: true, failures: 2, calls: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, providers, provider, model := runtimeFixture(t)
			defer store.Close()
			session, _ := store.CreateSession(context.Background(), Session{UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
			run, _ := store.CreateRun(context.Background(), Run{SessionID: session.ID, UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: model.ID, ModelName: model.DisplayName})
			_, _ = store.AddMessage(context.Background(), Message{SessionID: session.ID, RunID: run.ID, Role: RoleUser, Content: "test"})
			client := &retryClient{emitBefore: test.emitBefore, failures: test.failures}
			runtime, _ := NewNativeRuntime(store, providers, client, &fakeTools{readOnly: true}, NewEventHub())
			_ = runtime.Run(context.Background(), run.ID)
			loaded, _ := store.Run(context.Background(), "admin", run.ID)
			if string(loaded.Status) != test.wantStatus || client.calls != test.calls {
				t.Fatalf("status=%s calls=%d", loaded.Status, client.calls)
			}
		})
	}
}

func runtimeFixture(t *testing.T) (*Store, *ProviderService, Provider, Model) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	box, err := OpenSecretBox(filepath.Join(dir, "ai-secrets.key"), false)
	if err != nil {
		t.Fatal(err)
	}
	service, _ := NewProviderService(store, box)
	key := "test-key"
	provider, err := service.Save(ctx, "", ProviderInput{Name: "mock", Protocol: ProtocolOpenAICompatible, BaseURL: "https://example.com/v1", EndpointScope: EndpointPublic, APIKey: &key, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveModels(ctx, provider.ID, []Model{{ModelID: "mock-model", DisplayName: "Mock", ContextWindow: 8000, ToolCalling: true, Enabled: true}}); err != nil {
		t.Fatal(err)
	}
	models, _ := store.ListModels(ctx, provider.ID)
	return store, service, provider, models[0]
}
