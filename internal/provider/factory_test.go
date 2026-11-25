package provider

import (
	"testing"

	"github.com/bkelceoglu/go-rt/internal/domain"
)

func TestBuildFromSpecAndBuildJudge(t *testing.T) {
	t.Run("openai_compatible_provider", func(t *testing.T) {
		p := BuildFromSpec(domain.ProviderSpec{ID: "llamacpp:chat:qwen"})
		if p == nil || p.ID() != "llamacpp:chat:qwen" {
			t.Fatalf("expected llamacpp provider, got %v", p)
		}
	})

	t.Run("webhook_provider", func(t *testing.T) {
		p := BuildFromSpec(domain.ProviderSpec{ID: "webhook:http://localhost:9000/api"})
		if p == nil || p.ID() != "webhook:http://localhost:9000/api" {
			t.Fatalf("expected webhook provider, got %v", p)
		}
	})

	t.Run("empty_judge_returns_nil", func(t *testing.T) {
		if j := BuildJudge(domain.JudgeSpec{}); j != nil {
			t.Fatalf("expected nil judge for empty ID, got %v", j)
		}
	})

	t.Run("valid_judge_returns_provider", func(t *testing.T) {
		j := BuildJudge(domain.JudgeSpec{ID: "openai:chat:gpt-4o"})
		if j == nil || j.ID() != "openai:chat:gpt-4o" {
			t.Fatalf("expected judge provider, got %v", j)
		}
	})
}
