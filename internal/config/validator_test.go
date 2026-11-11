package config

import (
	"testing"
)

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
