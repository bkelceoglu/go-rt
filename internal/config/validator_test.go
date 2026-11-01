package config

import (
	"context"
	"strings"
	"testing"

	"github.com/caveman/go-rt/internal/domain"
)

type stubJudgeProvider struct {
	id  string
	out string
}

func (s *stubJudgeProvider) ID() string { return s.id }

func (s *stubJudgeProvider) Call(_ context.Context, _ domain.PromptRequest) (domain.ProviderResponse, error) {
	return domain.ProviderResponse{Output: s.out}, nil
}

func sampleCatalog() *CatalogSnapshot {
	return &CatalogSnapshot{
		Plugins: map[string]bool{
			"gort:plugin:sql-injection": true,
			"gort:plugin:rag:poisoning": true,
		},
		Collections: map[string]bool{
			"gort:collection:default": true,
		},
		Strategies: map[string]bool{
			"gort:strategy:basic":     true,
			"gort:strategy:base64":    true,
			"gort:strategy:crescendo": true,
			"gort:strategy:retry":     true,
		},
		Datasets: map[string]bool{
			"lakera_gandalf": true,
		},
	}
}

func TestValidateYAMLDetectsMissingAttackerAndTypos(t *testing.T) {
	raw := []byte(`
version: "1.0"
targets:
  - id: "llamacpp:chat:Qwen2.5-7B-Instruct"
judge:
  id: "llamacpp:chat:Qwen2.5-7B-Instruct"
redteam:
  plugins:
    - id: "rag-poisoning"
strategies:
  - id: "crescendo"
`)
	rep, _ := ValidateYAML(raw, sampleCatalog())
	if rep.Valid {
		t.Fatalf("expected invalid report when attacker is missing for crescendo and plugin id has typo")
	}
	if !rep.NeedsAttackerResolution {
		t.Fatalf("expected NeedsAttackerResolution=true")
	}
	if !rep.CanUseJudgeAsAttacker {
		t.Fatalf("expected CanUseJudgeAsAttacker=true when judge.id is present")
	}
}

func TestAutoFixMiniEngineRepairsIDsAndAppliesJudgeSequential(t *testing.T) {
	raw := []byte("target:\n\t- id: \"llamacpp:chat:Qwen2.5-7B-Instruct\"\njudge:\n  id: \"llamacpp:chat:Qwen2.5-7B-Instruct\"\nredteam:\n  plugin:\n    - id: \"rag-poisoning\"\nstrategy:\n  - id: \"crescendo\"\n")
	res, err := AutoFixMiniEngine(raw, sampleCatalog(), AutoFixOptions{
		AttackerResolution: ResolveUseJudgeSequential,
	})
	if err != nil {
		t.Fatalf("AutoFixMiniEngine failed: %v", err)
	}
	if !res.Report.Valid {
		t.Fatalf("expected valid report after AutoFixMiniEngine, got issues: %+v", res.Report.Issues)
	}
	if !res.Config.SequentialSharedLLM || res.Config.MaxConcurrency != 1 {
		t.Fatalf("expected SequentialSharedLLM=true and MaxConcurrency=1, got %+v", res.Config)
	}
	if res.Config.Attacker.ID != "llamacpp:chat:Qwen2.5-7B-Instruct" {
		t.Fatalf("expected Attacker.ID to match Judge.ID, got %q", res.Config.Attacker.ID)
	}
	if res.Config.Redteam.Plugins[0].ID != "gort:plugin:rag:poisoning" {
		t.Fatalf("expected normalized plugin ID gort:plugin:rag:poisoning, got %q", res.Config.Redteam.Plugins[0].ID)
	}
}

func TestAutoFixMiniEngineRemovesStrategiesOption(t *testing.T) {
	raw := []byte(`
version: "1.0"
targets:
  - id: "llamacpp:chat:Qwen2.5-7B-Instruct"
judge:
  id: "llamacpp:chat:Qwen2.5-7B-Instruct"
strategies:
  - id: "gort:strategy:basic"
  - id: "gort:strategy:crescendo"
  - id: "gort:strategy:retry"
`)
	res, err := AutoFixMiniEngine(raw, sampleCatalog(), AutoFixOptions{
		AttackerResolution: ResolveRemoveStrategies,
	})
	if err != nil {
		t.Fatalf("AutoFixMiniEngine failed: %v", err)
	}
	if !res.Report.Valid || len(res.Config.Strategies) != 1 || res.Config.Strategies[0].ID != "gort:strategy:basic" {
		t.Fatalf("expected only gort:strategy:basic to remain, got %+v", res.Config.Strategies)
	}
}

func TestAutoFixWithJudgeLLMRepairsBrokenSyntax(t *testing.T) {
	stub := &stubJudgeProvider{
		id:  "llamacpp:chat:Judge",
		out: "```yaml\nversion: \"1.0\"\ntargets:\n  - id: \"llamacpp:chat:Qwen2.5-7B\"\njudge:\n  id: \"llamacpp:chat:Judge\"\nstrategies:\n  - id: \"gort:strategy:basic\"\n```",
	}
	res, err := AutoFixWithJudgeLLM(context.Background(), stub, []byte(":::broken yaml:::"), sampleCatalog(), AutoFixOptions{})
	if err != nil {
		t.Fatalf("AutoFixWithJudgeLLM failed: %v", err)
	}
	if !res.Report.Valid || !strings.Contains(res.EngineUsed, "judge-llm") {
		t.Fatalf("expected valid Judge LLM repair result, got %+v", res)
	}
}
