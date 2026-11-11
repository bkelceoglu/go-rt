package provider

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/caveman/go-rt/internal/domain"
)

func TestWebhookProviderWithCustomBodyAndPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"query":"test attack"`) {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"replies":[{"text":"Access denied by security gateway."}]}}`))
	}))
	defer server.Close()

	spec := domain.ProviderSpec{
		ID:       "webhook:" + server.URL,
		APIKey:   "secret-token",
		BodyTmpl: `{"query":"{{prompt}}","session":"redteam"}`,
		RespPath: "data.replies.0.text",
	}
	p := NewWebhook(spec, server.Client())
	res, err := p.Call(t.Context(), domain.PromptRequest{Prompt: "test attack"})
	if err != nil {
		t.Fatalf("unexpected call error: %v", err)
	}
	if res.Output != "Access denied by security gateway." {
		t.Errorf("unexpected extracted output: %s", res.Output)
	}
	if res.TokenUsage.Requests != 1 {
		t.Errorf("expected 1 request recorded, got %d", res.TokenUsage.Requests)
	}
}

func TestWebhookFallbackAndErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"reply":"Hello world"}`))
	}))
	defer server.Close()

	spec := domain.ProviderSpec{
		ID:      "webhook:test",
		BaseURL: server.URL,
	}
	p := NewWebhook(spec, server.Client())
	res, err := p.Call(t.Context(), domain.PromptRequest{Prompt: "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Output != "Hello world" {
		t.Errorf("expected fallback to 'reply' key, got: %s", res.Output)
	}
}

func TestTraverseJSONPathErrors(t *testing.T) {
	data := []byte(`{"status":"ok"}`)
	_, err := extractResponseText(data, "non.existent.path")
	if err == nil {
		t.Fatal("expected error for non-existent path, got nil")
	}
}
