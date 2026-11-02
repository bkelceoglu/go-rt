package web

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/caveman/go-rt/internal/config"
	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/plugin"
	"github.com/caveman/go-rt/internal/provider"
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
	cat := plugin.BuildCatalogSnapshot(r.Context(), s.db)
	rep, cfg := config.ValidateYAML(raw, cat)
	writeJSON(w, http.StatusOK, map[string]any{
		"filename": file,
		"config":   cfg,
		"rawYaml":  string(raw),
		"report":   rep,
	})
}

type validateConfigRequest struct {
	RawYAML       string `json:"rawYaml"`
	CheckEndpoint bool   `json:"checkEndpoint"`
}

func (s *Server) handleValidateConfig(w http.ResponseWriter, r *http.Request) {
	var req validateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	cat := plugin.BuildCatalogSnapshot(r.Context(), s.db)
	rep, cfg := config.ValidateYAML([]byte(req.RawYAML), cat)
	if req.CheckEndpoint {
		rep.Reachability = config.CheckProvidersReachability(r.Context(), cfg)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"report": rep,
		"config": cfg,
	})
}

type pingProvidersRequest struct {
	RawYAML string `json:"rawYaml"`
}

func (s *Server) handlePingProviders(w http.ResponseWriter, r *http.Request) {
	var req pingProvidersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	_, cfg := config.ValidateYAML([]byte(req.RawYAML), nil)
	results := config.CheckProvidersReachability(r.Context(), cfg)
	writeJSON(w, http.StatusOK, map[string]any{"reachability": results})
}

type autoFixConfigRequest struct {
	RawYAML            string                        `json:"rawYaml"`
	Engine             string                        `json:"engine"`
	AttackerResolution config.AttackerResolutionMode `json:"attackerResolution"`
	CustomAttackerID   string                        `json:"customAttackerId"`
	CustomAttackerURL  string                        `json:"customAttackerUrl"`
}

func (s *Server) handleAutoFixConfig(w http.ResponseWriter, r *http.Request) {
	var req autoFixConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	res, err := s.executeWebAutoFix(r, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) executeWebAutoFix(r *http.Request, req autoFixConfigRequest) (config.AutoFixResult, error) {
	cat := plugin.BuildCatalogSnapshot(r.Context(), s.db)
	opts := config.AutoFixOptions{
		AttackerResolution: req.AttackerResolution,
		CustomAttackerID:   req.CustomAttackerID,
		CustomAttackerURL:  req.CustomAttackerURL,
	}
	if strings.EqualFold(req.Engine, "judge") {
		return s.runJudgeWebAutoFix(r, req.RawYAML, cat, opts)
	}
	return config.AutoFixMiniEngine([]byte(req.RawYAML), cat, opts)
}

func (s *Server) runJudgeWebAutoFix(r *http.Request, rawYAML string, cat *config.CatalogSnapshot, opts config.AutoFixOptions) (config.AutoFixResult, error) {
	var partial domain.Config
	_ = yaml.Unmarshal([]byte(rawYAML), &partial)
	judge := provider.BuildJudge(partial.Judge)
	return config.AutoFixWithJudgeLLM(r.Context(), judge, []byte(rawYAML), cat, opts)
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
