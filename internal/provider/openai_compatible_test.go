package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/caveman/go-rt/internal/domain"
)

func TestOpenAICompatibleRemoteSimulation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-sk-123" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if r.Header.Get("X-Custom-Env") != "production" {
			http.Error(w, `{"error":"missing header"}`, http.StatusBadRequest)
			return
		}
		resp := chatCompletionResponse{
			Choices: []struct {
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{Role: "assistant", Content: "I cannot fulfill this request."}},
			},
		}
		resp.Usage.PromptTokens = 12
		resp.Usage.CompletionTokens = 8
		resp.Usage.TotalTokens = 20
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	spec := domain.ProviderSpec{
		ID:      "openai:chat:gpt-4o-mini",
		BaseURL: server.URL,
		APIKey:  "test-sk-123",
		Headers: map[string]string{"X-Custom-Env": "production"},
	}
	p := NewOpenAICompatible(spec, server.Client())
	res, err := p.Call(context.Background(), domain.PromptRequest{Prompt: "Explain X"})
	if err != nil {
		t.Fatalf("unexpected call error: %v", err)
	}
	if res.Output != "I cannot fulfill this request." {
		t.Errorf("unexpected output: %s", res.Output)
	}
	if res.TokenUsage.Total != 20 {
		t.Errorf("expected 20 tokens, got %d", res.TokenUsage.Total)
	}
}

func TestOpenAICompatibleErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
	}))
	defer server.Close()

	spec := domain.ProviderSpec{
		ID:      "ollama:chat:llama3.1:8b",
		BaseURL: server.URL,
	}
	p := NewOpenAICompatible(spec, server.Client())
	_, err := p.Call(context.Background(), domain.PromptRequest{Prompt: "hello"})
	if err == nil {
		t.Fatal("expected error for 429 response, got nil")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("expected error message to contain 429, got: %v", err)
	}
}

func TestResolveModelNameAndBaseURL(t *testing.T) {
	if resolveModelName("ollama:chat:qwen2.5:3b") != "qwen2.5:3b" {
		t.Errorf("failed resolving ollama model name")
	}
	if resolveModelName("openai:gpt-4o") != "gpt-4o" {
		t.Errorf("failed resolving openai model name")
	}
	if resolveBaseURL("openai:chat:gpt-4o", "") != "https://api.openai.com/v1" {
		t.Errorf("failed resolving default openai base url")
	}
	if resolveBaseURL("ollama:llama3", "") != "http://localhost:11434/v1" {
		t.Errorf("failed resolving default ollama base url")
	}
}
