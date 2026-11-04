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
			Choices: []chatChoice{
				{Message: chatMessage{Role: "assistant", Content: "I cannot fulfill this request."}},
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
	if resolveModelName("llamacpp:chat:Qwen2.5-7B-Instruct") != "Qwen2.5-7B-Instruct" {
		t.Errorf("failed resolving llamacpp model name")
	}
	if resolveModelName("localai:chat:qwen3.8-9b-q4") != "qwen3.8-9b-q4" {
		t.Errorf("failed resolving localai model name")
	}
	if resolveModelName("ollama:chat:qwen2.5:3b") != "qwen2.5:3b" {
		t.Errorf("failed resolving ollama model name")
	}
	if resolveModelName("openai:gpt-4o") != "gpt-4o" {
		t.Errorf("failed resolving openai model name")
	}
	if resolveBaseURL("llamacpp:chat:Qwen2.5-7B-Instruct", "") != "http://localhost:8080/v1" {
		t.Errorf("failed resolving default llamacpp base url")
	}
	if resolveBaseURL("localai:chat:qwen3.8-9b-q4", "") != "http://localhost:8090/v1" {
		t.Errorf("failed resolving default localai base url")
	}
	if resolveBaseURL("openai:chat:gpt-4o", "") != "https://api.openai.com/v1" {
		t.Errorf("failed resolving default openai base url")
	}
	if resolveBaseURL("ollama:llama3", "") != "http://localhost:11434/v1" {
		t.Errorf("failed resolving default ollama base url")
	}

	t.Setenv("LLAMACPP_BASE_URL", "http://127.0.0.1:8080/v1")
	if resolveBaseURL("llamacpp:chat:Llama-3.2-3B-Instruct", "") != "http://127.0.0.1:8080/v1" {
		t.Errorf("failed resolving env LLAMACPP_BASE_URL")
	}

	t.Setenv("LOCALAI_BASE_URL", "http://127.0.0.1:8090/v1")
	if resolveBaseURL("localai:chat:qwen3-4b", "") != "http://127.0.0.1:8090/v1" {
		t.Errorf("failed resolving env LOCALAI_BASE_URL")
	}

	t.Setenv("OLLAMA_BASE_URL", "http://gpu-rig.local:11434/v1")
	if resolveBaseURL("ollama:llama3", "") != "http://gpu-rig.local:11434/v1" {
		t.Errorf("failed resolving env OLLAMA_BASE_URL")
	}

	t.Setenv("OPENAI_BASE_URL", "https://custom-gateway.internal/v1")
	if resolveBaseURL("openai:chat:gpt-4o", "") != "https://custom-gateway.internal/v1" {
		t.Errorf("failed resolving env OPENAI_BASE_URL")
	}

	f, tr := false, true
	if applyThinkDirective("hello", nil) != "hello" {
		t.Errorf("expected nil think to leave prompt untouched")
	}
	if applyThinkDirective("hello", &f) != "/no_think\nhello" {
		t.Errorf("expected false think to prepend /no_think")
	}
	if applyThinkDirective("hello", &tr) != "/think\nhello" {
		t.Errorf("expected true think to prepend /think")
	}
}

func TestOpenAICompatibleMultiTurnMessages(t *testing.T) {
	var capturedMessages []domain.Message
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []domain.Message `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		capturedMessages = req.Messages
		resp := chatCompletionResponse{
			Choices: []chatChoice{
				{Message: chatMessage{Role: "assistant", Content: "Understood."}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOpenAICompatible(domain.ProviderSpec{
		ID:      "llamacpp:chat:llama-3.2-1b",
		BaseURL: server.URL,
	}, server.Client())

	req := domain.PromptRequest{
		Prompt: "What is my name?",
		Messages: []domain.Message{
			{Role: "user", Content: "My name is Alice."},
			{Role: "assistant", Content: "Hello Alice!"},
			{Role: "user", Content: "What is my name?"},
		},
	}
	_, err := p.Call(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected call error: %v", err)
	}
	if len(capturedMessages) != 3 {
		t.Fatalf("expected 3 structured messages, got %d", len(capturedMessages))
	}
	if capturedMessages[0].Content != "My name is Alice." || capturedMessages[1].Role != "assistant" {
		t.Errorf("messages not preserved correctly: %+v", capturedMessages)
	}
}
