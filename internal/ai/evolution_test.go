package ai

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvolutionProposalRequiresApprovalAndRegisteredTools(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	memory, err := store.SaveProposal(ctx, EvolutionProposal{UserID: "admin", Type: EvolutionMemory, Title: "偏好", Content: "以后只先检查，不自动修改。"})
	if err != nil {
		t.Fatal(err)
	}
	if items, _ := store.Memories(ctx, "admin"); len(items) != 0 {
		t.Fatal("pending memory became active")
	}
	if err := store.DecideProposal(ctx, "admin", memory.ID, true, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if items, _ := store.Memories(ctx, "admin"); len(items) != 1 || !items[0].Enabled {
		t.Fatalf("memories=%#v", items)
	}
	payload, _ := json.Marshal(map[string]any{"condition": "容器异常时", "steps": []ProcedureStep{{Tool: "host_docker_containers", Arguments: json.RawMessage(`{}`)}}})
	procedure, err := store.SaveProposal(ctx, EvolutionProposal{UserID: "admin", Type: EvolutionProcedure, Title: "容器巡检", Content: "读取容器状态", Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DecideProposal(ctx, "admin", procedure.ID, true, map[string]bool{"other": true}); err == nil {
		t.Fatal("unknown tool must reject approval")
	}
	if err := store.DecideProposal(ctx, "admin", procedure.ID, true, map[string]bool{"host_docker_containers": true}); err != nil {
		t.Fatal(err)
	}
	procedures, _ := store.Procedures(ctx, "admin")
	if len(procedures) != 1 || !strings.Contains(string(procedures[0].Steps), "host_docker_containers") {
		t.Fatalf("procedures=%#v", procedures)
	}
	revisedSteps, _ := json.Marshal([]ProcedureStep{{Tool: "host_docker_containers", Arguments: json.RawMessage(`{"all":true}`)}})
	revised, err := store.ReviseProcedure(ctx, "admin", procedures[0].ID, "容器深度巡检", "需要完整检查时", revisedSteps, 0)
	if err != nil || revised.Version != 2 {
		t.Fatalf("revised=%#v err=%v", revised, err)
	}
	rolled, err := store.ReviseProcedure(ctx, "admin", procedures[0].ID, "", "", nil, 1)
	if err != nil || rolled.Version != 3 || rolled.Title != "容器巡检" {
		t.Fatalf("rolled=%#v err=%v", rolled, err)
	}
}

func TestContextMessagesCompactsOldHistoryAtSeventyPercent(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	provider, _ := store.SaveProvider(ctx, Provider{Name: "mock", Protocol: ProtocolOpenAICompatible, BaseURL: "https://example.com/v1", EndpointScope: EndpointPublic, Enabled: true}, 0)
	session, _ := store.CreateSession(ctx, Session{UserID: "admin", ProviderID: provider.ID, ModelID: "model", ProviderName: "mock", ModelName: "mock"})
	for index := 0; index < 20; index++ {
		_, err = store.AddMessage(ctx, Message{SessionID: session.ID, Role: RoleUser, Content: strings.Repeat("x", 300)})
		if err != nil {
			t.Fatal(err)
		}
	}
	items, summary, err := store.ContextMessages(ctx, session.ID, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if summary == "" || len(items) >= 20 {
		t.Fatalf("summary=%q remaining=%d", summary, len(items))
	}
	loaded, _ := store.Session(ctx, "admin", session.ID)
	if loaded.Summary == "" {
		t.Fatal("summary was not persisted")
	}
	page, err := store.MessagesPage(ctx, session.ID, 50, "")
	if err != nil || len(page.Items) != 20 {
		t.Fatalf("history was deleted during compaction: items=%d err=%v", len(page.Items), err)
	}
	itemsAgain, summaryAgain, err := store.ContextMessages(ctx, session.ID, 1024)
	if err != nil || summaryAgain != summary || len(itemsAgain) != len(items) {
		t.Fatalf("repeated compaction changed stable context: items=%d/%d err=%v", len(itemsAgain), len(items), err)
	}
}

func TestEvolutionRetirementPreservesRecords(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	proposal, err := store.SaveProposal(ctx, EvolutionProposal{UserID: "admin", Type: EvolutionMemory, Title: "preference", Content: "keep concise"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DecideProposal(ctx, "admin", proposal.ID, true, nil); err != nil {
		t.Fatal(err)
	}
	memories, _ := store.Memories(ctx, "admin")
	if err := store.DeleteMemory(ctx, "admin", memories[0].ID); err != nil {
		t.Fatal(err)
	}
	memories, _ = store.Memories(ctx, "admin")
	if len(memories) != 1 || !memories[0].Retired || memories[0].Enabled {
		t.Fatalf("retired memory was not preserved: %#v", memories)
	}
}
