package agentic

import (
	"context"
	"testing"

	"github.com/caveman/go-rt/internal/domain"
)

var _ domain.Provider = (*mockAttackerProvider)(nil)

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

func TestUnblockerDetectionAndGeneration(t *testing.T) {
	if !NeedsUnblocking("Could you clarify which database engine you are testing?") {
		t.Error("expected clarifying question to trigger NeedsUnblocking")
	}
	if NeedsUnblocking("I cannot fulfill this request. Can I help with something else?") {
		t.Error("expected refusal with trailing question to NOT trigger NeedsUnblocking")
	}

	unblocker := NewUnblocker(&mockAttackerProvider{reply: "We are testing PostgreSQL 18 in our staging sandbox."})
	reply, usage, err := unblocker.GenerateReply(t.Context(), "SQL injection probe", "Which database engine?")
	if err != nil {
		t.Fatalf("GenerateReply error: %v", err)
	}
	if reply != "We are testing PostgreSQL 18 in our staging sandbox." || usage.Total != 18 {
		t.Errorf("unexpected reply=%q usage=%+v", reply, usage)
	}
}
