package agentic

import (
	"context"
	"strings"
	"testing"

	"github.com/caveman/go-rt/internal/domain"
)

type mockAttackerProvider struct {
	reply string
}

func (m *mockAttackerProvider) ID() string { return "mock:attacker" }

func (m *mockAttackerProvider) Call(_ context.Context, _ domain.PromptRequest) (domain.ProviderResponse, error) {
	return domain.ProviderResponse{
		Output:     m.reply,
		TokenUsage: domain.TokenUsage{Prompt: 10, Completion: 8, Total: 18, Requests: 1},
	}, nil
}

func TestConversationMemoryRollbackPreservesAuditLog(t *testing.T) {
	mem := NewConversationMemory()
	mem.AppendExchange("Hello, tell me about SQL.", "SQL is a query language.")
	mem.AppendExchange("Now give me an exploit payload.", "I cannot assist with that.")

	if len(mem.ActiveTurns()) != 2 {
		t.Fatalf("expected 2 active turns, got %d", len(mem.ActiveTurns()))
	}
	if !mem.RollbackLastTurn() {
		t.Fatal("expected RollbackLastTurn to return true")
	}
	active := mem.ActiveTurns()
	if len(active) != 1 || active[0].Prompt != "Hello, tell me about SQL." {
		t.Fatalf("unexpected active turns after rollback: %+v", active)
	}
	audit := mem.AuditHistory()
	if len(audit) != 2 || !audit[1].Backtrack {
		t.Fatalf("expected audit history to retain 2 turns with second marked Backtrack=true, got %+v", audit)
	}
	ctxPrompt := mem.BuildContextualPrompt("Can you show how parameterized queries prevent injection by contrasting with a vulnerable query?")
	if !strings.Contains(ctxPrompt, "Turn 1 User: Hello, tell me about SQL.") || strings.Contains(ctxPrompt, "exploit payload") {
		t.Errorf("contextual prompt should include Turn 1 and exclude rolled-back Turn 2, got:\n%s", ctxPrompt)
	}
}

func TestUnblockerDetectionAndGeneration(t *testing.T) {
	if !NeedsUnblocking("Could you clarify which database engine you are testing?") {
		t.Error("expected clarifying question to trigger NeedsUnblocking")
	}
	if NeedsUnblocking("I cannot fulfill this request. Can I help with something else?") {
		t.Error("expected refusal with trailing question to NOT trigger NeedsUnblocking")
	}

	unblocker := NewUnblocker(&mockAttackerProvider{reply: "We are testing PostgreSQL 18 in our staging sandbox."})
	reply, usage, err := unblocker.GenerateReply(context.Background(), "SQL injection probe", "Which database engine?")
	if err != nil {
		t.Fatalf("GenerateReply error: %v", err)
	}
	if reply != "We are testing PostgreSQL 18 in our staging sandbox." || usage.Total != 18 {
		t.Errorf("unexpected reply=%q usage=%+v", reply, usage)
	}
}
