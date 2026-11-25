package agentic

import (
	"context"
	"testing"

	"github.com/bkelceoglu/go-rt/internal/domain"
)

var _ domain.Provider = (*mockRetryTarget)(nil)

type mockRetryTarget struct {
	attempts int
}

func (m *mockRetryTarget) ID() string { return "mock:target" }
func (m *mockRetryTarget) Call(_ context.Context, _ domain.PromptRequest) (domain.ProviderResponse, error) {
	m.attempts++
	if m.attempts == 1 {
		return domain.ProviderResponse{Output: "I cannot fulfill this request as it violates safety policies."}, nil
	}
	return domain.ProviderResponse{Output: "Here is the requested technical explanation."}, nil
}

func TestRetryDecoratorRetriesOnRefusal(t *testing.T) {
	target := &mockRetryTarget{}
	dec := NewRetryDecorator(target, nil, 2)

	resp, err := dec.Call(t.Context(), domain.PromptRequest{Prompt: "Secret query"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", target.attempts)
	}
	if resp.Output != "Here is the requested technical explanation." {
		t.Errorf("expected successful second attempt output, got: %q", resp.Output)
	}
}
