package web

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/caveman/go-rt/internal/config"
	"gopkg.in/yaml.v3"
)

func (s *Server) handleListConfigs(w http.ResponseWriter, r *http.Request) {
	files := listWorkspaceYAMLFiles(s.workDir)
	if files == nil {
		files = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

func listWorkspaceYAMLFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		lower := strings.ToLower(e.Name())
		if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") {
			out = append(out, e.Name())
		}
	}
	slices.Sort(out)
	if out == nil {
		out = []string{}
	}
	return out
}

func (s *Server) handleLoadConfig(w http.ResponseWriter, r *http.Request) {
	file := sanitizeConfigFilename(r.URL.Query().Get("file"))
	fullPath := filepath.Join(s.workDir, file)
	raw, err := os.ReadFile(fullPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "config file not found: "+file)
		return
	}
	cfg, err := config.ParseBytes(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"filename": file,
		"config":   cfg,
		"rawYaml":  string(raw),
	})
}

type saveConfigRequest struct {
	Filename string `json:"filename"`
	RawYAML  string `json:"rawYaml"`
}

func (s *Server) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	var req saveConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	file := sanitizeConfigFilename(req.Filename)
	rawBytes, err := validateAndNormalizeYAML(req.RawYAML)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	fullPath := filepath.Join(s.workDir, file)
	if err := os.WriteFile(fullPath, rawBytes, 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"saved": file, "bytes": len(rawBytes)})
}

func validateAndNormalizeYAML(raw string) ([]byte, error) {
	cfg, err := config.ParseBytes([]byte(raw))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) != "" {
		return []byte(raw), nil
	}
	return yaml.Marshal(cfg)
}

func sanitizeConfigFilename(raw string) string {
	clean := filepath.Base(strings.TrimSpace(raw))
	if clean == "" || clean == "." || clean == "/" {
		return "config.yaml"
	}
	lower := strings.ToLower(clean)
	if !strings.HasSuffix(lower, ".yaml") && !strings.HasSuffix(lower, ".yml") {
		clean += ".yaml"
	}
	return clean
}
