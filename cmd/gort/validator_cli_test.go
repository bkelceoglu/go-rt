package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInteractiveValidationResolvesSequentialJudge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "config.yaml")
	raw := fmt.Sprintf(`version: "1.0"
targets:
  - id: ollama:llama3:8b
    baseUrl: %s
judge:
  id: ollama:qwen3:8b
  baseUrl: %s
strategies:
  - id: gort:strategy:crescendo
`, srv.URL, srv.URL)
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

func TestRunInteractiveValidationFailsWhenEndpointUnreachable(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "offline.yaml")
	raw := `version: "1.0"
targets:
  - id: llamacpp:chat:llama-3.2-1b-instruct
    baseUrl: http://127.0.0.1:59997
judge:
  id: llamacpp:chat:llama-3.2-1b-instruct
  baseUrl: http://127.0.0.1:59997
`
	if err := os.WriteFile(tmp, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := runInteractiveValidation(tmp, nil, strings.NewReader(""))
	if err == nil {
		t.Fatal("expected runInteractiveValidation to fail when LLM endpoint is unreachable")
	}
}
