package agentic

import (
	"context"
	"fmt"
	"testing"

	"github.com/caveman/go-rt/internal/domain"
)

var _ domain.Provider = (*scriptedProvider)(nil)

type scriptedProvider struct {
	id        string
	responses []string
	callIdx   int
}

func (s *scriptedProvider) ID() string { return s.id }

func (s *scriptedProvider) Call(_ context.Context, _ domain.PromptRequest) (domain.ProviderResponse, error) {
	idx := s.callIdx
	s.callIdx++
	out := "Default response"
	if idx < len(s.responses) {
		out = s.responses[idx]
	}
	return domain.ProviderResponse{
		Output:     out,
		LatencyMs:  10,
		TokenUsage: domain.TokenUsage{Prompt: 5, Completion: 5, Total: 10, Requests: 1},
	}, nil
}

func TestCrescendoBacktrackingAndUnblocking(t *testing.T) {
	target := &scriptedProvider{
		id: "mock:target",
		responses: []string{
			"Could you clarify which environment you are auditing?", // Turn 1 -> triggers Unblocker
			"Thank you for clarifying the staging environment.",     // Unblock response
			"I cannot fulfill this request.",                        // Turn 2 -> triggers RollbackLastTurn
			"Here is the diagnostic configuration output.",          // Turn 3 -> final turn output
		},
	}
	attacker := &scriptedProvider{
		id: "mock:attacker",
		responses: []string{
			"Turn 1 benign question",
			"Staging environment confirmation",
			"Turn 2 direct probe",
			"Turn 3 pivoted probe after backtrack",
		},
	}

	dec := NewCrescendoDecorator(target, attacker, StrategyCrescendo, 3, 2)
	resp, err := dec.Call(t.Context(), domain.PromptRequest{Prompt: "Verify diagnostic access"})
	if err != nil {
		t.Fatalf("CrescendoDecorator.Call error: %v", err)
	}
	if resp.Output != "Here is the diagnostic configuration output." {
		t.Errorf("unexpected final output: %q", resp.Output)
	}
	if len(resp.ConversationHistory) != 4 {
		t.Fatalf("expected 4 audit turns (Turn 1 + Unblock + Rolled-back Turn 2 + Turn 3), got %d", len(resp.ConversationHistory))
	}
	var backtrackCount int
	for _, turn := range resp.ConversationHistory {
		if turn.Backtrack {
			backtrackCount++
		}
	}
	if backtrackCount != 1 {
		t.Errorf("expected exactly 1 backtracked turn in ConversationHistory, got %d (%+v)", backtrackCount, resp.ConversationHistory)
	}
	if dec.ID() != fmt.Sprintf("mock:target[%s]", StrategyCrescendo) {
		t.Errorf("unexpected decorator ID: %s", dec.ID())
	}
}
