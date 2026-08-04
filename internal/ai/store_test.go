package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreProviderSessionRunLifecycle(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(filepath.Join(t.TempDir(), "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	provider, err := store.SaveProvider(ctx, Provider{Name: "Example", Protocol: ProtocolOpenAICompatible, BaseURL: "https://example.com/v1", EndpointScope: EndpointPublic, Enabled: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveModels(ctx, provider.ID, []Model{
		{ModelID: "model-a", DisplayName: "Model A", ContextWindow: 8192, ToolCalling: true, Enabled: true, IsDefault: true},
		{ModelID: "model-b", DisplayName: "Model B", ContextWindow: 32768, ToolCalling: true, Enabled: true},
	}); err != nil {
		t.Fatal(err)
	}
	models, err := store.ListModels(ctx, provider.ID)
	if err != nil || len(models) != 2 {
		t.Fatalf("models = %#v, %v", models, err)
	}
	session, err := store.CreateSession(ctx, Session{UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: models[0].ID, ModelName: models[0].DisplayName})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMessage(ctx, Message{SessionID: session.ID, Role: RoleUser, Content: "hello"}); err != nil {
		t.Fatal(err)
	}
	run, err := store.CreateRun(ctx, Run{SessionID: session.ID, UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: models[0].ID, ModelName: models[0].DisplayName})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateRun(ctx, Run{SessionID: session.ID, UserID: "admin"}); !errors.Is(err, ErrBusy) {
		t.Fatalf("second run error = %v", err)
	}
	providerData := json.RawMessage(`{"providerId":"provider","items":[{"type":"reasoning","encrypted_content":"opaque"}]}`)
	call, err := store.SaveToolCall(ctx, ToolCall{ID: "call-1", RunID: run.ID, SessionID: session.ID, Name: "host_read", Arguments: json.RawMessage(`{"x":1}`), ProviderData: providerData, Status: ToolRunning})
	if err != nil {
		t.Fatal(err)
	}
	loadedCall, err := store.ToolCall(ctx, run.ID, call.ID)
	if err != nil || string(loadedCall.ProviderData) != string(providerData) {
		t.Fatalf("provider-native tool context was not persisted: %s, %v", loadedCall.ProviderData, err)
	}
	if err := store.SaveModels(ctx, provider.ID, []Model{{ModelID: "model-a", DisplayName: "changed", ContextWindow: 8192, Enabled: true}}); !errors.Is(err, ErrBusy) {
		t.Fatalf("active run allowed model mutation: %v", err)
	}
	activeSession, err := store.Session(ctx, "admin", session.ID)
	if err != nil || !activeSession.Running || activeSession.ActiveRunID != run.ID {
		t.Fatalf("active session=%#v err=%v", activeSession, err)
	}
	updatedSession, err := store.UpdateSession(ctx, "admin", session.ID, "", provider.ID, models[1].ID, provider.Name, models[1].DisplayName, nil, nil)
	if err != nil || updatedSession.ModelID != models[1].ID {
		t.Fatalf("next-turn model update=%#v err=%v", updatedSession, err)
	}
	loadedRun, err := store.Run(ctx, "admin", run.ID)
	if err != nil || loadedRun.ModelID != models[0].ID {
		t.Fatalf("active run model snapshot changed=%#v err=%v", loadedRun, err)
	}
	if err := store.DeleteProvider(ctx, provider.ID); !errors.Is(err, ErrBusy) {
		t.Fatalf("active provider delete error = %v", err)
	}
	run.Status = RunCompleted
	if err := store.UpdateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteProvider(ctx, provider.ID); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Session(ctx, "admin", session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ModelAvailable {
		t.Fatal("deleted provider must leave session history but make model unavailable")
	}
}

func TestOpenStoreInterruptsRunningRunAndPreservesApproval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	provider, _ := store.SaveProvider(ctx, Provider{Name: "test", Protocol: ProtocolOpenAICompatible, BaseURL: "https://example.com/v1", EndpointScope: EndpointPublic, Enabled: true}, 0)
	session, _ := store.CreateSession(ctx, Session{UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: "model", ModelName: "model"})
	running, _ := store.CreateRun(ctx, Run{SessionID: session.ID, UserID: "admin", ProviderID: provider.ID, ModelID: "model"})
	running.Status = RunRunning
	if err := store.UpdateRun(ctx, running); err != nil {
		t.Fatal(err)
	}
	approvalSession, _ := store.CreateSession(ctx, Session{UserID: "admin", ProviderID: provider.ID, ProviderName: provider.Name, ModelID: "model", ModelName: "model"})
	approval, _ := store.CreateRun(ctx, Run{SessionID: approvalSession.ID, UserID: "admin", ProviderID: provider.ID, ModelID: "model"})
	approval.Status = RunPendingApproval
	if err := store.UpdateRun(ctx, approval); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	interrupted, _ := store.Run(ctx, "admin", running.ID)
	pending, _ := store.Run(ctx, "admin", approval.ID)
	if interrupted.Status != RunInterrupted || pending.Status != RunPendingApproval {
		t.Fatalf("running=%s approval=%s", interrupted.Status, pending.Status)
	}
	loadedSession, err := store.Session(ctx, "admin", session.ID)
	if err != nil || loadedSession.LastRunID != running.ID || loadedSession.LastRunStatus != RunInterrupted {
		t.Fatalf("restart session snapshot=%#v err=%v", loadedSession, err)
	}
}

func TestOpenStoreRejectsCorruptDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai.db")
	if err := os.WriteFile(path, []byte("not sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}
	if store, err := OpenStore(path); err == nil {
		store.Close()
		t.Fatal("corrupt database must fail to open")
	}
}

func TestOpenStoreMigratesPreSummaryCursorSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE providers (id TEXT PRIMARY KEY,name TEXT NOT NULL,protocol TEXT NOT NULL,base_url TEXT NOT NULL,endpoint_scope TEXT NOT NULL,enabled INTEGER NOT NULL,encrypted_key BLOB,api_key_hint TEXT NOT NULL DEFAULT '',version INTEGER NOT NULL,created_at INTEGER NOT NULL,updated_at INTEGER NOT NULL)`,
		`CREATE TABLE sessions (id TEXT PRIMARY KEY,user_id TEXT NOT NULL,title TEXT NOT NULL,provider_id TEXT NOT NULL,model_id TEXT NOT NULL,provider_name TEXT NOT NULL,model_name TEXT NOT NULL,summary TEXT NOT NULL DEFAULT '',pinned INTEGER NOT NULL,archived INTEGER NOT NULL,created_at INTEGER NOT NULL,updated_at INTEGER NOT NULL,last_message_at INTEGER NOT NULL)`,
		`CREATE TABLE tool_calls (id TEXT PRIMARY KEY,run_id TEXT NOT NULL,session_id TEXT NOT NULL,name TEXT NOT NULL,arguments BLOB NOT NULL,arguments_preview TEXT NOT NULL,result_preview TEXT NOT NULL DEFAULT '',status TEXT NOT NULL,requires_approval INTEGER NOT NULL,created_at INTEGER NOT NULL,updated_at INTEGER NOT NULL)`,
		`CREATE TABLE memories (id TEXT PRIMARY KEY,user_id TEXT NOT NULL,title TEXT NOT NULL,content TEXT NOT NULL,enabled INTEGER NOT NULL,version INTEGER NOT NULL,created_at INTEGER NOT NULL,updated_at INTEGER NOT NULL)`,
		`CREATE TABLE procedures (id TEXT PRIMARY KEY,user_id TEXT NOT NULL,title TEXT NOT NULL,condition_text TEXT NOT NULL,steps BLOB NOT NULL,enabled INTEGER NOT NULL,version INTEGER NOT NULL,created_at INTEGER NOT NULL,updated_at INTEGER NOT NULL)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for table, column := range map[string]string{"sessions": "summary_cursor", "memories": "retired", "procedures": "retired", "providers": "api_mode", "tool_calls": "provider_data"} {
		rows, err := store.db.Query(`PRAGMA table_info(` + table + `)`)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for rows.Next() {
			var cid, notNull, primaryKey int
			var name, columnType string
			var defaultValue any
			if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
				t.Fatal(err)
			}
			found = found || name == column
		}
		rows.Close()
		if !found {
			t.Fatalf("migration did not add %s.%s", table, column)
		}
	}
}
