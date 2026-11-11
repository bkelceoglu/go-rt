package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInteractiveValidationResolvesSequentialJudge(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "config.yaml")
	raw := `version: "1.0"
targets:
  - id: ollama:llama3:8b
judge:
  id: ollama:qwen3:8b
strategies:
  - id: gort:strategy:crescendo
`
	if err := os.WriteFile(tmp, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	userInput := strings.NewReader("3\ny\n")
	cfg, err := runInteractiveValidation(tmp, nil, userInput)
	if err != nil {
		t.Fatalf("runInteractiveValidation error: %v", err)
	}
	if !cfg.SequentialSharedLLM || cfg.Attacker.ID != "ollama:qwen3:8b" || cfg.MaxConcurrency != 1 {
		t.Fatalf("expected SequentialSharedLLM=true, Attacker=ollama:qwen3:8b, MaxConcurrency=1; got %+v", cfg)
	}
}
