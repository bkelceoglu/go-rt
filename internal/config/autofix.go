package config

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/bkelceoglu/go-rt/internal/domain"
	"gopkg.in/yaml.v3"
)

// AttackerResolutionMode defines how to resolve a missing Attacker LLM when required by strategies.
type AttackerResolutionMode string

const (
	ResolveNone               AttackerResolutionMode = ""
	ResolveSetAttacker        AttackerResolutionMode = "set_attacker"
	ResolveRemoveStrategies   AttackerResolutionMode = "remove_strategies"
	ResolveUseJudgeSequential AttackerResolutionMode = "use_judge_sequential"
)

// AutoFixOptions configures how the Mini Engine or Judge LLM repairs a configuration.
type AutoFixOptions struct {
	AttackerResolution AttackerResolutionMode `json:"attackerResolution,omitempty"`
	CustomAttackerID   string                 `json:"customAttackerId,omitempty"`
	CustomAttackerURL  string                 `json:"customAttackerUrl,omitempty"`
}

// AutoFixResult holds the repaired YAML, parsed Config, and human-readable list of applied fixes.
type AutoFixResult struct {
	FixedYAML    string           `json:"fixedYaml"`
	Config       domain.Config    `json:"config"`
	AppliedFixes []string         `json:"appliedFixes"`
	EngineUsed   string           `json:"engineUsed"`
	Report       ValidationReport `json:"report"`
}

// AutoFixMiniEngine deterministically repairs YAML syntax, key typos, catalog IDs, and Attacker logic.
func AutoFixMiniEngine(raw []byte, cat *CatalogSnapshot, opts AutoFixOptions) (AutoFixResult, error) {
	fixes := []string{}
	cleaned := sanitizeRawYAMLText(string(raw), &fixes)
	var cfg domain.Config
	if err := yaml.Unmarshal([]byte(cleaned), &cfg); err != nil {
		return AutoFixResult{}, fmt.Errorf("mini-engine could not parse yaml: %w", err)
	}
	return finalizeMiniEngineRepair(&cfg, cat, opts, fixes)
}

func finalizeMiniEngineRepair(cfg *domain.Config, cat *CatalogSnapshot, opts AutoFixOptions, fixes []string) (AutoFixResult, error) {
	applyDefaults(cfg)
	repairStructuralFields(cfg, cat, &fixes)
	applyAttackerResolution(cfg, opts, &fixes)
	outBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return AutoFixResult{}, fmt.Errorf("marshal repaired yaml: %w", err)
	}
	rep, parsed := ValidateYAML(outBytes, cat)
	return AutoFixResult{FixedYAML: string(outBytes), Config: parsed, AppliedFixes: fixes, EngineUsed: "mini-engine", Report: rep}, nil
}

// AutoFixWithJudgeLLM uses the Judge LLM to repair broken YAML structure and then runs MiniEngine validation.
func AutoFixWithJudgeLLM(ctx context.Context, judge domain.Provider, raw []byte, cat *CatalogSnapshot, opts AutoFixOptions) (AutoFixResult, error) {
	if judge == nil {
		return AutoFixResult{}, errors.New("judge llm provider is not available for ai auto-fix")
	}
	prompt := buildJudgeYAMLRepairPrompt(string(raw))
	resp, err := judge.Call(ctx, domain.PromptRequest{Prompt: prompt})
	if err != nil {
		return AutoFixResult{}, fmt.Errorf("judge llm auto-fix call failed: %w", err)
	}
	return runPostJudgeMiniEngine(judge.ID(), resp.Output, cat, opts)
}

func runPostJudgeMiniEngine(judgeID, output string, cat *CatalogSnapshot, opts AutoFixOptions) (AutoFixResult, error) {
	extracted := extractYAMLBlock(output)
	res, err := AutoFixMiniEngine([]byte(extracted), cat, opts)
	if err != nil {
		return AutoFixResult{}, fmt.Errorf("post-judge mini-engine repair: %w", err)
	}
	res.EngineUsed = "judge-llm (" + judgeID + ") + mini-engine"
	res.AppliedFixes = append([]string{"Repaired YAML structure via Judge LLM (" + judgeID + ")"}, res.AppliedFixes...)
	return res, nil
}

func buildJudgeYAMLRepairPrompt(brokenYAML string) string {
	return "You are a strict YAML configuration repair engine for GORT (AI Red Teaming Engine).\n" +
		"Fix any YAML indentation, syntax, or key naming errors in the following configuration.\n" +
		"Valid top-level keys are: version, description, purpose, promptsDir, promptTemplate, injectVar, datasets, targets, attacker, judge, redteam, strategies, storage, maxConcurrency, sequentialSharedLlm.\n" +
		"Output ONLY valid raw YAML without markdown fences or explanations:\n\n" + brokenYAML
}

func extractYAMLBlock(output string) string {
	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		if len(lines) >= 2 {
			trimmed = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	return strings.TrimSpace(trimmed)
}

func sanitizeRawYAMLText(raw string, fixes *[]string) string {
	if strings.Contains(raw, "\t") {
		raw = strings.ReplaceAll(raw, "\t", "  ")
		*fixes = append(*fixes, "Replaced tab indentation characters with 2 spaces")
	}
	return replaceMisspelledYAMLKeys(raw, fixes)
}

func replaceMisspelledYAMLKeys(raw string, fixes *[]string) string {
	padded := "\n" + raw
	replacements := [][2]string{
		{"\ntarget:", "\ntargets:"},
		{"\nstrategy:", "\nstrategies:"},
		{"  plugin:", "  plugins:"},
	}
	for _, r := range replacements {
		if strings.Contains(padded, r[0]) {
			padded = strings.ReplaceAll(padded, r[0], r[1])
			*fixes = append(*fixes, fmt.Sprintf("Renamed misspelled YAML key %q to %q", strings.TrimSpace(r[0]), strings.TrimSpace(r[1])))
		}
	}
	return strings.TrimPrefix(padded, "\n")
}

func repairStructuralFields(cfg *domain.Config, cat *CatalogSnapshot, fixes *[]string) {
	if cfg.Version == "" {
		cfg.Version = "1.0"
		*fixes = append(*fixes, "Set default version to \"1.0\"")
	}
	if cat == nil {
		return
	}
	repairPluginIDs(cfg, cat, fixes)
	repairStrategyIDs(cfg, cat, fixes)
}

func repairPluginIDs(cfg *domain.Config, cat *CatalogSnapshot, fixes *[]string) {
	for i, p := range cfg.Redteam.Plugins {
		id := strings.TrimSpace(p.ID)
		if cat.Plugins[id] || cat.Collections[id] {
			continue
		}
		if matched := matchClosestCatalogID(id, cat.Plugins, "gort:plugin:"); matched != "" {
			cfg.Redteam.Plugins[i].ID = matched
			*fixes = append(*fixes, fmt.Sprintf("Normalized plugin ID %q -> %q", id, matched))
		}
	}
}

func repairStrategyIDs(cfg *domain.Config, cat *CatalogSnapshot, fixes *[]string) {
	for i, s := range cfg.Strategies {
		id := strings.TrimSpace(s.ID)
		if cat.Strategies[id] {
			continue
		}
		if matched := matchClosestCatalogID(id, cat.Strategies, "gort:strategy:"); matched != "" {
			cfg.Strategies[i].ID = matched
			*fixes = append(*fixes, fmt.Sprintf("Normalized strategy ID %q -> %q", id, matched))
		}
	}
}

func applyAttackerResolution(cfg *domain.Config, opts AutoFixOptions, fixes *[]string) {
	switch opts.AttackerResolution {
	case ResolveSetAttacker:
		applyCustomAttacker(cfg, opts, fixes)
	case ResolveRemoveStrategies:
		removeAttackerStrategies(cfg, fixes)
	case ResolveUseJudgeSequential:
		reuseJudgeAsSequentialAttacker(cfg, fixes)
	}
}

func applyCustomAttacker(cfg *domain.Config, opts AutoFixOptions, fixes *[]string) {
	id := strings.TrimSpace(opts.CustomAttackerID)
	if id == "" {
		return
	}
	cfg.Attacker.ID = id
	if strings.TrimSpace(opts.CustomAttackerURL) != "" {
		cfg.Attacker.BaseURL = strings.TrimSpace(opts.CustomAttackerURL)
	}
	*fixes = append(*fixes, fmt.Sprintf("Configured dedicated Attacker LLM: %s", id))
}

func removeAttackerStrategies(cfg *domain.Config, fixes *[]string) {
	var kept []domain.StrategySpec
	var removed []string
	for _, s := range cfg.Strategies {
		if attackerRequiredStrategies[normalizeStrategyPrefix(s.ID)] {
			removed = append(removed, s.ID)
			continue
		}
		kept = append(kept, s)
	}
	if len(removed) > 0 {
		cfg.Strategies = kept
		*fixes = append(*fixes, fmt.Sprintf("Removed Attacker-dependent strategies: %v", removed))
	}
}

func reuseJudgeAsSequentialAttacker(cfg *domain.Config, fixes *[]string) {
	if strings.TrimSpace(cfg.Judge.ID) == "" {
		return
	}
	cfg.Attacker = domain.ProviderSpec{
		ID:      cfg.Judge.ID,
		BaseURL: cfg.Judge.BaseURL,
		APIKey:  cfg.Judge.APIKey,
		Temp:    0.7,
		MaxToks: cfg.Judge.MaxToks,
		Think:   cfg.Judge.Think,
	}
	cfg.SequentialSharedLLM = true
	cfg.MaxConcurrency = 1
	*fixes = append(*fixes, fmt.Sprintf("Assigned Judge LLM (%s) as Attacker LLM in strict sequential mode (maxConcurrency=1)", cfg.Judge.ID))
}

func suggestCatalogID(raw string, catalog map[string]bool, prefix string) string {
	if matched := matchClosestCatalogID(raw, catalog, prefix); matched != "" {
		return fmt.Sprintf("Did you mean %q?", matched)
	}
	return fmt.Sprintf("Run 'gort %s' to view valid IDs.", strings.TrimSuffix(strings.TrimPrefix(prefix, "gort:"), ":"))
}

func matchClosestCatalogID(raw string, catalog map[string]bool, prefix string) string {
	clean := strings.ToLower(strings.TrimSpace(raw))
	withPrefix := prefix + strings.TrimPrefix(clean, prefix)
	if catalog[withPrefix] {
		return withPrefix
	}
	colonNorm := strings.ReplaceAll(withPrefix, "-", ":")
	if catalog[colonNorm] {
		return colonNorm
	}
	return findSubstringCatalogMatch(clean, catalog, prefix)
}

func findSubstringCatalogMatch(clean string, catalog map[string]bool, prefix string) string {
	short := strings.TrimPrefix(clean, prefix)
	for _, k := range slices.Sorted(maps.Keys(catalog)) {
		kShort := strings.TrimPrefix(k, prefix)
		if strings.ReplaceAll(kShort, ":", "-") == strings.ReplaceAll(short, ":", "-") || strings.Contains(kShort, short) {
			return k
		}
	}
	return ""
}
