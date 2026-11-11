package agentic

import (
	"strings"
	"testing"
)

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
