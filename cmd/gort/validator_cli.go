package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/caveman/go-rt/internal/config"
	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/plugin"
	"github.com/caveman/go-rt/internal/store"
	"github.com/caveman/go-rt/internal/ui"
)

func runInteractiveValidation(configPath string, db *store.SQLiteStore, in io.Reader) (domain.Config, error) {
	cat := plugin.BuildCatalogSnapshot(context.Background(), db)
	reader := bufio.NewReader(in)
	for {
		cfg, done, err := validateSingleIteration(configPath, cat, reader)
		if err != nil || done {
			return cfg, err
		}
	}
}

func validateSingleIteration(configPath string, cat *config.CatalogSnapshot, r *bufio.Reader) (domain.Config, bool, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return domain.Config{}, false, fmt.Errorf("read config %s: %w", configPath, err)
	}
	rep, cfg := config.ValidateYAML(raw, cat)
	printValidationReport(configPath, rep)
	if !rep.Valid || len(rep.Issues) > 0 || rep.NeedsAttackerResolution {
		return handleInteractiveRepair(configPath, raw, rep, cfg, cat, r)
	}
	checkAndPrintEndpointsHealth(cfg)
	return cfg, true, nil
}

func checkAndPrintEndpointsHealth(cfg domain.Config) {
	results := config.CheckProvidersReachability(context.Background(), cfg)
	if len(results) == 0 {
		return
	}
	fmt.Printf("\n%s Pre-flight LLM Endpoint Reachability:\n", ui.PrefixGort())
	for _, res := range results {
		badge := ui.Badge("PASS")
		if !res.Reachable {
			badge = ui.Badge("FAIL")
		}
		fmt.Printf("  %s %-12s | %-32s | %s (%s)\n",
			badge, ui.Bold(res.Role), ui.Cyan(res.ID), ui.Dim(res.BaseURL), res.Message)
	}
	fmt.Println()
}

func printValidationReport(configPath string, rep config.ValidationReport) {
	if rep.Valid && len(rep.Issues) == 0 {
		fmt.Printf("%s Configuration %s validated: %s\n", ui.PrefixGort(), ui.Bold(configPath), ui.Badge("PASS"))
		return
	}
	fmt.Printf("\n%s Configuration Validation Report for %s:\n", ui.PrefixGort(), ui.Bold(configPath))
	for i, iss := range rep.Issues {
		badge := ui.Badge("WARN")
		if iss.Severity == config.IssueError {
			badge = ui.Badge("FAIL")
		}
		fmt.Printf("  %2d. %s [%s] %s: %s\n", i+1, badge, ui.Cyan(iss.Code), ui.Bold(iss.Field), iss.Message)
		if iss.Suggestion != "" {
			fmt.Printf("      %s %s\n", ui.Dim("Suggestion:"), ui.Yellow(iss.Suggestion))
		}
	}
}

func handleInteractiveRepair(
	configPath string,
	raw []byte,
	rep config.ValidationReport,
	cfg domain.Config,
	cat *config.CatalogSnapshot,
	r *bufio.Reader,
) (domain.Config, bool, error) {
	opts := config.AutoFixOptions{}
	if rep.NeedsAttackerResolution {
		opts = promptCLIAttackerResolution(r, rep.AttackerDependentStrategies)
	}
	res, ok, err := executeSelectedAutoFix(raw, rep, cfg, cat, opts, r)
	if err != nil || !ok {
		return promptManualRevalidate(configPath, rep, cfg, r)
	}
	return confirmAndPersistFix(configPath, res, r)
}

func promptCLIAttackerResolution(r *bufio.Reader, strategies []string) config.AutoFixOptions {
	fmt.Printf("\n%s %s Strategies %v require an Attacker LLM, which is not configured.\n",
		ui.PrefixGort(), ui.BoldYellow("[LOGIC CONTROL]"), strategies)
	fmt.Println("  [1] Enter a dedicated Attacker LLM ID (parallel execution)")
	fmt.Println("  [2] No Attacker LLM -> Remove Attacker-dependent strategies from config")
	fmt.Println("  [3] No Attacker LLM -> Reuse Judge LLM sequentially (Attack first, then Judge)")
	fmt.Print("Select resolution option [1/2/3] (default 3): ")
	choice := readTrimmedLine(r)
	switch choice {
	case "1":
		return promptCustomAttackerSpec(r)
	case "2":
		return config.AutoFixOptions{AttackerResolution: config.ResolveRemoveStrategies}
	default:
		return config.AutoFixOptions{AttackerResolution: config.ResolveUseJudgeSequential}
	}
}

func promptCustomAttackerSpec(r *bufio.Reader) config.AutoFixOptions {
	fmt.Print("  Enter Attacker LLM ID (e.g. ollama:qwen3:8b): ")
	id := readTrimmedLine(r)
	fmt.Print("  Enter Attacker Base URL (optional, press Enter for default): ")
	url := readTrimmedLine(r)
	return config.AutoFixOptions{
		AttackerResolution: config.ResolveSetAttacker,
		CustomAttackerID:   id,
		CustomAttackerURL:  url,
	}
}

func executeSelectedAutoFix(
	raw []byte,
	rep config.ValidationReport,
	cfg domain.Config,
	cat *config.CatalogSnapshot,
	opts config.AutoFixOptions,
	r *bufio.Reader,
) (config.AutoFixResult, bool, error) {
	if !hasFormatOrCatalogIssues(rep) && opts.AttackerResolution != config.ResolveNone {
		res, err := config.AutoFixMiniEngine(raw, cat, opts)
		return res, err == nil, err
	}
	engineChoice := promptAutoFixEngineChoice(r)
	if engineChoice == "n" || engineChoice == "skip" {
		return config.AutoFixResult{}, false, nil
	}
	if engineChoice == "2" {
		judge := buildJudgeProvider(cfg.Judge)
		res, err := config.AutoFixWithJudgeLLM(context.Background(), judge, raw, cat, opts)
		return res, err == nil, err
	}
	res, err := config.AutoFixMiniEngine(raw, cat, opts)
	return res, err == nil, err
}

func hasFormatOrCatalogIssues(rep config.ValidationReport) bool {
	for _, iss := range rep.Issues {
		if iss.Code != "LOGIC_MISSING_ATTACKER" {
			return true
		}
	}
	return false
}

func promptAutoFixEngineChoice(r *bufio.Reader) string {
	fmt.Printf("\n%s %s Select an Auto-Fix Engine to repair YAML issues:\n",
		ui.PrefixGort(), ui.BoldCyan("[AUTO-FIX]"))
	fmt.Println("  [1] Mini Engine (Instant <1ms deterministic syntax & SQLite catalog repair) [Recommended]")
	fmt.Println("  [2] Judge LLM Repair (AI syntax repair via configured Judge LLM + Mini Engine check)")
	fmt.Println("  [N] Skip auto-fix and edit file manually")
	fmt.Print("Choice [1/2/N] (default 1): ")
	ans := strings.ToLower(readTrimmedLine(r))
	if ans == "" {
		return "1"
	}
	return ans
}

func confirmAndPersistFix(configPath string, res config.AutoFixResult, r *bufio.Reader) (domain.Config, bool, error) {
	fmt.Printf("\n%s Proposed Fixes (%s):\n", ui.PrefixGort(), ui.Bold(res.EngineUsed))
	for i, f := range res.AppliedFixes {
		fmt.Printf("  %d. %s\n", i+1, ui.BoldGreen(f))
	}
	fmt.Printf("Confirm & apply these fixes to %s? [y/N]: ", configPath)
	ans := strings.ToLower(readTrimmedLine(r))
	if ans != "y" && ans != "yes" {
		return res.Config, true, nil
	}
	if err := os.WriteFile(configPath, []byte(res.FixedYAML), 0o644); err != nil {
		return domain.Config{}, false, fmt.Errorf("write fixed config %s: %w", configPath, err)
	}
	fmt.Printf("%s Updated %s and re-validating...\n", ui.PrefixGort(), ui.Bold(configPath))
	return domain.Config{}, false, nil
}

func promptManualRevalidate(configPath string, rep config.ValidationReport, cfg domain.Config, r *bufio.Reader) (domain.Config, bool, error) {
	fmt.Printf("Edit %s manually and press [y] to re-validate, or [N] to abort [y/N]: ", configPath)
	ans := strings.ToLower(readTrimmedLine(r))
	if ans == "y" || ans == "yes" {
		return domain.Config{}, false, nil
	}
	if rep.Valid {
		return cfg, true, nil
	}
	return domain.Config{}, true, fmt.Errorf("configuration %s failed validation", configPath)
}

func readTrimmedLine(r *bufio.Reader) string {
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}
