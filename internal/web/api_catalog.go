package web

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/caveman/go-rt/internal/dataset"
	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/store"
)

type catalogResponse struct {
	Plugins     []store.PluginRecord   `json:"plugins"`
	Collections map[string][]string    `json:"collections"`
	Datasets    []store.DatasetSummary `json:"datasets"`
	Strategies  []store.StrategyRecord `json:"strategies"`
	LocalModels []string               `json:"localModels"`
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
		LocalModels: discoverLocalModels(),
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

func discoverLocalModels() []string {
	models := []string{
		"llamacpp:chat:Qwen2.5-7B-Instruct",
		"llamacpp:chat:Llama-3.2-3B-Instruct",
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return models
	}
	return appendDiscoveredGGUFs(models, filepath.Join(home, ".models"))
}

func appendDiscoveredGGUFs(existing []string, dir string) []string {
	matches, _ := filepath.Glob(filepath.Join(dir, "*.gguf"))
	seen := make(map[string]bool, len(existing))
	for _, m := range existing {
		seen[m] = true
	}
	for _, p := range matches {
		base := strings.TrimSuffix(filepath.Base(p), ".gguf")
		id := "llamacpp:chat:" + base
		if !seen[id] {
			seen[id] = true
			existing = append(existing, id)
		}
	}
	return existing
}
