package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/caveman/go-rt/internal/domain"
	"gopkg.in/yaml.v3"
)

// IssueSeverity indicates whether a validation finding blocks execution or is a warning.
type IssueSeverity string

const (
	IssueError   IssueSeverity = "error"
	IssueWarning IssueSeverity = "warning"
)

// ValidationIssue represents a single format, structure, or logic problem in config.yaml.
type ValidationIssue struct {
	Severity   IssueSeverity `json:"severity"`
	Code       string        `json:"code"`
	Field      string        `json:"field"`
	Message    string        `json:"message"`
	Suggestion string        `json:"suggestion,omitempty"`
}

// CatalogSnapshot holds valid IDs from SQLite for cross-checking config references.
type CatalogSnapshot struct {
	Plugins     map[string]bool
	Collections map[string]bool
	Strategies  map[string]bool
	Datasets    map[string]bool
}

// ValidationReport summarizes all format, structure, and logic findings for a YAML config.
type ValidationReport struct {
	Valid                       bool                         `json:"valid"`
	SyntaxError                 bool                         `json:"syntaxError"`
	NeedsAttackerResolution     bool                         `json:"needsAttackerResolution"`
	AttackerDependentStrategies []string                     `json:"attackerDependentStrategies,omitempty"`
	CanUseJudgeAsAttacker       bool                         `json:"canUseJudgeAsAttacker"`
	JudgeID                     string                       `json:"judgeId,omitempty"`
	Issues                      []ValidationIssue            `json:"issues"`
	Reachability                []ProviderReachabilityResult `json:"reachability,omitempty"`
}

var attackerRequiredStrategies = map[string]bool{
	"gort:strategy:crescendo": true,
	"gort:strategy:goat":      true,
	"gort:strategy:hydra":     true,
	"gort:strategy:retry":     true,
}

// ValidateYAML performs full format, structure, and logic validation on raw YAML bytes.
func ValidateYAML(raw []byte, cat *CatalogSnapshot) (ValidationReport, domain.Config) {
	report := ValidationReport{Valid: true, Issues: []ValidationIssue{}}
	expanded := os.ExpandEnv(string(raw))
	if err := checkStrictYAMLSyntax(expanded); err != nil {
		return buildSyntaxErrorReport(err), domain.Config{}
	}
	var cfg domain.Config
	_ = yaml.Unmarshal([]byte(expanded), &cfg)
	applyDefaults(&cfg)
	collectStructuralIssues(&report, cfg, cat)
	collectLogicIssues(&report, cfg)
	report.Valid = !hasBlockingErrors(report.Issues) && !report.NeedsAttackerResolution
	return report, cfg
}

func checkStrictYAMLSyntax(expanded string) error {
	if strings.TrimSpace(expanded) == "" {
		return errors.New("configuration yaml is empty")
	}
	dec := yaml.NewDecoder(bytes.NewReader([]byte(expanded)))
	dec.KnownFields(true)
	var cfg domain.Config
	return dec.Decode(&cfg)
}

func buildSyntaxErrorReport(err error) ValidationReport {
	issue := ValidationIssue{
		Severity:   IssueError,
		Code:       "YAML_FORMAT_ERROR",
		Field:      "root",
		Message:    err.Error(),
		Suggestion: "Check YAML indentation (use spaces, not tabs) and field names.",
	}
	return ValidationReport{Valid: false, SyntaxError: true, Issues: []ValidationIssue{issue}}
}

func collectStructuralIssues(rep *ValidationReport, cfg domain.Config, cat *CatalogSnapshot) {
	checkTargetsStructure(rep, cfg.Targets)
	checkJudgeStructure(rep, cfg.Judge)
	if cat != nil {
		checkPluginCatalogRefs(rep, cfg.Redteam.Plugins, cat)
		checkStrategyCatalogRefs(rep, cfg.Strategies, cat)
	}
}

func checkTargetsStructure(rep *ValidationReport, targets []domain.ProviderSpec) {
	if len(targets) == 0 {
		rep.Issues = append(rep.Issues, ValidationIssue{
			Severity:   IssueError,
			Code:       "MISSING_TARGETS",
			Field:      "targets",
			Message:    "At least one target provider must be configured.",
			Suggestion: "Add a target under 'targets:' (e.g., id: \"llamacpp:chat:Qwen2.5-7B-Instruct\").",
		})
		return
	}
	for i, t := range targets {
		checkSingleTarget(rep, i, t)
	}
}

func checkSingleTarget(rep *ValidationReport, idx int, t domain.ProviderSpec) {
	if strings.TrimSpace(t.ID) == "" {
		rep.Issues = append(rep.Issues, ValidationIssue{
			Severity:   IssueError,
			Code:       "EMPTY_TARGET_ID",
			Field:      fmt.Sprintf("targets[%d].id", idx),
			Message:    fmt.Sprintf("Target #%d is missing a provider ID.", idx+1),
			Suggestion: "Specify provider format 'llamacpp:chat:<model>', 'openai:chat:<model>', or 'webhook:http...'.",
		})
	}
}

func checkJudgeStructure(rep *ValidationReport, j domain.JudgeSpec) {
	if strings.TrimSpace(j.ID) == "" {
		rep.Issues = append(rep.Issues, ValidationIssue{
			Severity:   IssueWarning,
			Code:       "MISSING_JUDGE_ID",
			Field:      "judge.id",
			Message:    "No Judge LLM ID configured; evaluation will rely only on basic refusal heuristics.",
			Suggestion: "Configure 'judge.id' for accurate LLM-as-a-Judge rubric scoring.",
		})
	}
}

func checkPluginCatalogRefs(rep *ValidationReport, plugins []domain.PluginSpec, cat *CatalogSnapshot) {
	for i, p := range plugins {
		id := strings.TrimSpace(p.ID)
		if cat.Plugins[id] || cat.Collections[id] {
			continue
		}
		sug := suggestCatalogID(id, cat.Plugins, "gort:plugin:")
		rep.Issues = append(rep.Issues, ValidationIssue{
			Severity:   IssueError,
			Code:       "UNKNOWN_PLUGIN_ID",
			Field:      fmt.Sprintf("redteam.plugins[%d].id", i),
			Message:    fmt.Sprintf("Unknown plugin or collection ID %q.", id),
			Suggestion: sug,
		})
	}
}

func checkStrategyCatalogRefs(rep *ValidationReport, strategies []domain.StrategySpec, cat *CatalogSnapshot) {
	for i, s := range strategies {
		id := strings.TrimSpace(s.ID)
		if cat.Strategies[id] {
			continue
		}
		sug := suggestCatalogID(id, cat.Strategies, "gort:strategy:")
		rep.Issues = append(rep.Issues, ValidationIssue{
			Severity:   IssueError,
			Code:       "UNKNOWN_STRATEGY_ID",
			Field:      fmt.Sprintf("strategies[%d].id", i),
			Message:    fmt.Sprintf("Unknown strategy ID %q.", id),
			Suggestion: sug,
		})
	}
}

func collectLogicIssues(rep *ValidationReport, cfg domain.Config) {
	dep := FindAttackerDependentStrategies(cfg.Strategies)
	hasAttacker := strings.TrimSpace(cfg.Attacker.ID) != ""
	rep.JudgeID = strings.TrimSpace(cfg.Judge.ID)
	rep.CanUseJudgeAsAttacker = rep.JudgeID != ""
	if len(dep) > 0 && !hasAttacker {
		rep.NeedsAttackerResolution = true
		rep.AttackerDependentStrategies = dep
		rep.Issues = append(rep.Issues, buildMissingAttackerIssue(dep, rep.JudgeID))
	}
}

func buildMissingAttackerIssue(dep []string, judgeID string) ValidationIssue {
	sug := "1) Enter an Attacker LLM, 2) Remove these strategies, or 3) Reuse Judge LLM sequentially."
	if judgeID != "" {
		sug = fmt.Sprintf("1) Enter an Attacker LLM, 2) Remove %v, or 3) Reuse Judge LLM (%s) sequentially.", dep, judgeID)
	}
	return ValidationIssue{
		Severity:   IssueError,
		Code:       "LOGIC_MISSING_ATTACKER",
		Field:      "attacker.id",
		Message:    fmt.Sprintf("Strategies %v require an Attacker LLM, but 'attacker.id' is not configured.", dep),
		Suggestion: sug,
	}
}

// FindAttackerDependentStrategies returns all configured strategies that require an Attacker LLM.
func FindAttackerDependentStrategies(specs []domain.StrategySpec) []string {
	out := []string{}
	for _, s := range specs {
		norm := normalizeStrategyPrefix(s.ID)
		if attackerRequiredStrategies[norm] {
			out = append(out, s.ID)
		}
	}
	return out
}

func normalizeStrategyPrefix(id string) string {
	trimmed := strings.ToLower(strings.TrimSpace(id))
	return "gort:strategy:" + strings.TrimPrefix(trimmed, "gort:strategy:")
}

func hasBlockingErrors(issues []ValidationIssue) bool {
	for _, iss := range issues {
		if iss.Severity == IssueError {
			return true
		}
	}
	return false
}
