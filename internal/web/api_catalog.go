package web

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caveman/go-rt/internal/dataset"
	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/store"
)

// ProviderProtocolTemplate describes an LLM runtime provider format and default connection URL.
type ProviderProtocolTemplate struct {
	Type        string `json:"type"`
	Label       string `json:"label"`
	Prefix      string `json:"prefix"`
	DefaultURL  string `json:"defaultUrl"`
	Placeholder string `json:"placeholder"`
}

type catalogResponse struct {
	Plugins     []store.PluginRecord       `json:"plugins"`
	Collections map[string][]string        `json:"collections"`
	Datasets    []store.DatasetSummary     `json:"datasets"`
	Strategies  []store.StrategyRecord     `json:"strategies"`
	Providers   []ProviderProtocolTemplate `json:"providers"`
	LocalModels []string                   `json:"localModels"`
}

func (s *Server) handleGetCatalog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	plugins, _ := s.db.ListPlugins(ctx)
	cols, _ := s.db.CollectionMap(ctx)
	datasets, _ := s.db.ListDatasets(ctx)
	strategies, _ := s.db.ListStrategies(ctx)
	resp := catalogResponse{
		Plugins:     plugins,
		Collections: cols,
		Datasets:    datasets,
		Strategies:  strategies,
		Providers:   supportedProviderTemplates(),
		LocalModels: discoverLiveLocalModels(r.Context()),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetDatasetPrompts(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" {
		writeError(w, http.StatusBadRequest, "dataset name is required")
		return
	}
	recs, err := s.db.LoadDatasetPrompts(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dataset": name, "prompts": recs})
}

type customDatasetRequest struct {
	Name    string                 `json:"name"`
	Records []domain.DatasetRecord `json:"records"`
}

func (s *Server) handleSaveCustomDataset(w http.ResponseWriter, r *http.Request) {
	var req customDatasetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	name := sanitizeDatasetIdentifier(req.Name)
	if name == "" || len(req.Records) == 0 {
		writeError(w, http.StatusBadRequest, "dataset name and at least one prompt record are required")
		return
	}
	if err := s.persistCustomDataset(r, name, req.Records); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"saved": name, "count": len(req.Records)})
}

func (s *Server) persistCustomDataset(r *http.Request, name string, recs []domain.DatasetRecord) error {
	if err := s.db.SaveDatasetPrompts(r.Context(), name, recs); err != nil {
		return err
	}
	csvPath := filepath.Join(s.workDir, "prompts", name+".csv")
	_ = os.MkdirAll(filepath.Dir(csvPath), 0o755)
	return dataset.WriteCSV(csvPath, recs)
}

func sanitizeDatasetIdentifier(raw string) string {
	clean := strings.ToLower(strings.TrimSpace(raw))
	clean = strings.TrimSuffix(clean, ".csv")
	clean = strings.ReplaceAll(clean, " ", "_")
	return filepath.Base(clean)
}

func supportedProviderTemplates() []ProviderProtocolTemplate {
	return []ProviderProtocolTemplate{
		{
			Type:        "ollama",
			Label:       "Ollama (Local LLM Daemon)",
			Prefix:      "ollama:chat:",
			DefaultURL:  "http://localhost:11434",
			Placeholder: "ollama:chat:llama3.1:8b",
		},
		{
			Type:        "llamacpp",
			Label:       "llama.cpp Server / GGUF Runtime",
			Prefix:      "llamacpp:chat:",
			DefaultURL:  "http://localhost:8080",
			Placeholder: "llamacpp:chat:model",
		},
		{
			Type:        "vllm",
			Label:       "vLLM / LiteLLM / SGLang",
			Prefix:      "vllm:chat:",
			DefaultURL:  "http://localhost:8000/v1",
			Placeholder: "vllm:chat:Qwen/Qwen2.5-7B-Instruct",
		},
		{
			Type:        "localai",
			Label:       "LocalAI",
			Prefix:      "localai:chat:",
			DefaultURL:  "http://localhost:8090/v1",
			Placeholder: "localai:chat:model",
		},
		{
			Type:        "openai",
			Label:       "OpenAI / Azure / Cloud Hosted",
			Prefix:      "openai:chat:",
			DefaultURL:  "https://api.openai.com/v1",
			Placeholder: "openai:chat:gpt-4o-mini",
		},
		{
			Type:        "webhook",
			Label:       "Custom HTTP Webhook API",
			Prefix:      "webhook:",
			DefaultURL:  "http://localhost:9000/api/chat",
			Placeholder: "webhook:http://localhost:9000/api/chat",
		},
	}
}

func discoverLiveLocalModels(ctx context.Context) []string {
	var discovered []string
	client := &http.Client{Timeout: 800 * time.Millisecond}
	discovered = appendOllamaLiveModels(ctx, client, discovered)
	discovered = appendLlamaCppLiveModels(ctx, client, discovered)
	return discovered
}

func appendOllamaLiveModels(ctx context.Context, client *http.Client, out []string) []string {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:11434/api/tags", nil)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return out
	}
	defer resp.Body.Close()
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil {
		for _, m := range payload.Models {
			if m.Name != "" {
				out = append(out, "ollama:chat:"+m.Name)
			}
		}
	}
	return out
}

func appendLlamaCppLiveModels(ctx context.Context, client *http.Client, out []string) []string {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8080/v1/models", nil)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return out
	}
	defer resp.Body.Close()
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil {
		for _, d := range payload.Data {
			if d.ID != "" {
				out = append(out, "llamacpp:chat:"+d.ID)
			}
		}
	}
	return out
}
