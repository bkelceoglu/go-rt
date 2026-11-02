package config

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caveman/go-rt/internal/domain"
)

func TestCheckProvidersReachability(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[{"id":"mock-model"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cfg := domain.Config{
		Targets: []domain.ProviderSpec{
			{ID: "llamacpp:chat:test", BaseURL: ts.URL},
			{ID: "ollama:chat:unreachable", BaseURL: "http://127.0.0.1:54321"},
		},
		Judge: domain.JudgeSpec{
			ID:      "openai:chat:gpt-test",
			BaseURL: ts.URL,
		},
	}

	results := CheckProvidersReachability(context.Background(), cfg)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if !results[0].Reachable || results[0].StatusCode != http.StatusOK {
		t.Fatalf("expected target[0] to be reachable, got %+v", results[0])
	}
	if results[1].Reachable {
		t.Fatalf("expected target[1] to be unreachable on 54321, got %+v", results[1])
	}
	if !results[2].Reachable {
		t.Fatalf("expected judge to be reachable, got %+v", results[2])
	}
}
