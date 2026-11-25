package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bkelceoglu/go-rt/internal/config"
)

func TestParseBytesAndDefaults(t *testing.T) {
	t.Setenv("TEST_API_KEY", "secret-token-123")
	yamlInput := []byte(`
version: "1.0"
purpose: "Test assistant"
targets:
  - id: "ollama:chat:llama3.1:8b"
    apiKey: "${TEST_API_KEY}"
`)
	cfg, err := config.ParseBytes(yamlInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PromptsDir != "prompts" || cfg.InjectVar != "prompt" {
		t.Fatalf("expected defaults applied, got dir=%q var=%q", cfg.PromptsDir, cfg.InjectVar)
	}
	if cfg.Targets[0].APIKey != "secret-token-123" {
		t.Fatalf("expected env var expansion, got %q", cfg.Targets[0].APIKey)
	}
	if cfg.Storage.SQLitePath != ".gort/gort.db" {
		t.Fatalf("expected default sqlite path, got %q", cfg.Storage.SQLitePath)
	}
}

func TestValidationRejectsEmptyTargets(t *testing.T) {
	_, err := config.ParseBytes([]byte(`version: "1.0"`))
	if err == nil {
		t.Fatal("expected error when targets are empty")
	}
}

func TestLoadRootConfigFile(t *testing.T) {
	rootConfig := filepath.Join("..", "..", "config.yaml")
	if _, err := os.Stat(rootConfig); err != nil {
		t.Skip("root config.yaml not found")
	}
	cfg, err := config.Load(rootConfig)
	if err != nil {
		t.Fatalf("failed to load root config.yaml: %v", err)
	}
	if len(cfg.Targets) == 0 || cfg.Judge.ID == "" {
		t.Fatalf("expected targets and judge in root config.yaml")
	}
}
