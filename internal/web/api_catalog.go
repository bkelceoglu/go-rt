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

// StrategyMeta describes a built-in strategy for the Web UI Config Wizard.
type StrategyMeta struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type catalogResponse struct {
	Plugins     []store.PluginRecord   `json:"plugins"`
	Collections map[string][]string    `json:"collections"`
	Datasets    []store.DatasetSummary `json:"datasets"`
	Strategies  []StrategyMeta         `json:"strategies"`
	LocalModels []string               `json:"localModels"`
}

func (s *Server) handleGetCatalog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	plugins, _ := s.db.ListPlugins(ctx)
	cols, _ := s.db.CollectionMap(ctx)
	datasets, _ := s.db.ListDatasets(ctx)
	resp := catalogResponse{
		Plugins:     plugins,
		Collections: cols,
		Datasets:    datasets,
		Strategies:  builtinStrategyCatalog(),
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

func builtinStrategyCatalog() []StrategyMeta {
	return []StrategyMeta{
		{ID: "gort:strategy:basic", Kind: "static", Label: "Basic (Unmutated)", Description: "Direct pass-through of seed prompt without encoding"},
		{ID: "gort:strategy:base64", Kind: "static", Label: "Base64 Encoding", Description: "Encodes payload in Base64 and asks target to decode and execute"},
		{ID: "gort:strategy:rot13", Kind: "static", Label: "ROT13 Cipher", Description: "Obfuscates payload with ROT13 character substitution"},
		{ID: "gort:strategy:hex", Kind: "static", Label: "Hexadecimal Encoding", Description: "Encodes payload as hexadecimal ASCII byte sequence"},
		{ID: "gort:strategy:leetspeak", Kind: "static", Label: "Leetspeak Substitution", Description: "Replaces ASCII characters with leetspeak symbols (4, 3, 1, 0, 5, 7)"},
		{ID: "gort:strategy:homoglyph", Kind: "static", Label: "Unicode Homoglyphs", Description: "Substitutes Latin letters with visually identical Cyrillic characters"},
		{ID: "gort:strategy:morse", Kind: "static", Label: "Morse Code Encoding", Description: "Converts payload into Morse code dots and dashes"},
		{ID: "gort:strategy:piglatin", Kind: "static", Label: "Pig Latin Translation", Description: "Obfuscates payload into Pig Latin linguistic shifts"},
		{ID: "gort:strategy:camelcase", Kind: "static", Label: "camelCase Concatenation", Description: "Removes whitespace and joins words in camelCase to confuse token boundaries"},
		{ID: "gort:strategy:emoji", Kind: "static", Label: "Emoji Insertion", Description: "Intersperse evasion emojis between words to defeat naive keyword filters"},
		{ID: "gort:strategy:jailbreak-templates", Kind: "framing", Label: "Jailbreak Framing Templates", Description: "Wraps payload in Sandbox Audit, Academic Simulation, or Red-Team framing"},
		{ID: "gort:strategy:math-prompt", Kind: "framing", Label: "Mathematical Problem Pretext", Description: "Disguises payload inside an algebra problem or logic optimization hypothesis"},
		{ID: "gort:strategy:prompt-injection", Kind: "framing", Label: "Direct Prompt Injection Override", Description: "Prepends system administrative directive and role-override commands"},
		{ID: "gort:strategy:citation", Kind: "framing", Label: "Academic Citation Pretext", Description: "Frames prompt as a peer-reviewed scientific research study on safety"},
		{ID: "gort:strategy:authoritative-markup-injection", Kind: "framing", Label: "Authoritative Markup Injection", Description: "Injects XML/JSON tags mimicking root administrative system commands"},
		{ID: "gort:strategy:mischievous-user", Kind: "framing", Label: "Mischievous User Persona", Description: "Adopts a rule-bending, playful persona exploring unrestricted knowledge"},
		{ID: "gort:strategy:jailbreak", Kind: "framing", Label: "DAN Master Jailbreak", Description: "Classic Do-Anything-Now persona escaping alignment guidelines"},
		{ID: "gort:strategy:jailbreak:composite", Kind: "framing", Label: "Composite Jailbreak", Description: "Multi-layered framing combining simulation protocol and rule bypass"},
		{ID: "gort:strategy:jailbreak:likert", Kind: "framing", Label: "Likert Scale Survey Jailbreak", Description: "Scientific evaluation pretext requesting unredacted output for scoring"},
		{ID: "gort:strategy:jailbreak:tree", Kind: "framing", Label: "Tree-of-Thought Jailbreak", Description: "Branching conceptual logic exploring unrestricted implementation branches"},
		{ID: "gort:strategy:jailbreak:meta", Kind: "framing", Label: "Meta-Prompting Jailbreak", Description: "Reflective meta-analysis examining unhooked output generation"},
		{ID: "gort:strategy:jailbreak:goblin", Kind: "framing", Label: "GOBLIN Persona Jailbreak", Description: "Unrestricted persona answering directly without moralizing or refusal"},
		{ID: "gort:strategy:best-of-n", Kind: "framing", Label: "Best-of-N Adversarial Variations", Description: "Generates multiple high-entropy adversarial framing variants"},
		{ID: "gort:strategy:retry", Kind: "agentic", Label: "Automatic Refusal Retry", Description: "Detects refusal and re-attempts with attacker-rephrased prompts"},
		{ID: "gort:strategy:layer", Kind: "layer", Label: "Layered Strategy Chain", Description: "Chains multiple strategies sequentially (e.g. jailbreak-templates -> base64)"},
		{ID: "gort:strategy:crescendo", Kind: "multiturn", Label: "Crescendo Multi-Turn", Description: "Gradual multi-turn escalation with refusal RollbackLastTurn() and Unblocker"},
		{ID: "gort:strategy:goat", Kind: "multiturn", Label: "GOAT Offensive Agent", Description: "Generative Offensive Agent Tester adaptive multi-turn reasoning"},
		{ID: "gort:strategy:hydra", Kind: "multiturn", Label: "Hydra Multi-Head Pivot", Description: "Switches to orthogonal technical/compliance angles upon target refusal"},
	}
}
