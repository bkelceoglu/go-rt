package main

import (
	"bufio"
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/caveman/go-rt/internal/config"
	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/plugin"
	"github.com/caveman/go-rt/internal/store"
	"github.com/caveman/go-rt/internal/ui"
)

type repairContext struct {
	configPath string
	raw        []byte
	rep        config.ValidationReport
	cfg        domain.Config
	cat        *config.CatalogSnapshot
}

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
	if !rep.Valid || rep.NeedsAttackerResolution {
		rc := &repairContext{configPath: configPath, raw: raw, rep: rep, cfg: cfg, cat: cat}
		return rc.handleInteractiveRepair(r)
	}
	if err := checkAndPrintEndpointsHealth(configPath, cfg, r); err != nil {
		return domain.Config{}, true, err
	}
	return cfg, true, nil
}

func checkAndPrintEndpointsHealth(configPath string, cfg domain.Config, r *bufio.Reader) error {
	for {
		results := config.CheckProvidersReachability(context.Background(), cfg)
		if len(results) == 0 {
			return nil
		}
		printReachabilityRows(results)
		bad, hasOffline := config.FirstUnreachableProvider(results)
		if !hasOffline {
			return nil
		}
		if err := promptEndpointRetryOrAbort(configPath, bad, r); err != nil {
			return err
		}
	}
}

func promptEndpointRetryOrAbort(configPath string, bad config.ProviderReachabilityResult, r *bufio.Reader) error {
	fmt.Printf("%s %s Validation Failed: LLM endpoint(s) unreachable.\n", ui.PrefixGort(), ui.Badge("FAIL"))
	fmt.Print("Ensure the LLM service is running and press [y] to re-check, or [N] to abort [y/N]: ")
	ans := strings.ToLower(readTrimmedLine(r))
	if ans != "y" && ans != "yes" {
		return fmt.Errorf("validation failed for %s: %s '%s' (%s) is unreachable (%s)",
			configPath, bad.Role, bad.ID, bad.BaseURL, bad.Message)
	}
	fmt.Printf("%s Re-checking LLM endpoint reachability...\n", ui.PrefixGort())
	return nil
}

func printReachabilityRows(results []config.ProviderReachabilityResult) {
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
		printSingleValidationIssue(i+1, iss)
	}
}

func printSingleValidationIssue(idx int, iss config.ValidationIssue) {
	badge := ui.Badge("WARN")
	if iss.Severity == config.IssueError {
		badge = ui.Badge("FAIL")
	}
	fmt.Printf("  %2d. %s [%s] %s: %s\n", idx, badge, ui.Cyan(iss.Code), ui.Bold(iss.Field), iss.Message)
	if iss.Suggestion != "" {
		fmt.Printf("      %s %s\n", ui.Dim("Suggestion:"), ui.Yellow(iss.Suggestion))
	}
}

func (rc *repairContext) handleInteractiveRepair(r *bufio.Reader) (domain.Config, bool, error) {
	var opts config.AutoFixOptions
	if rc.rep.NeedsAttackerResolution {
		opts = promptCLIAttackerResolution(r, rc.rep.AttackerDependentStrategies)
	}
	res, ok, err := rc.executeSelectedAutoFix(opts, r)
	if err != nil || !ok {
		return rc.promptManualRevalidate(r)
	}
	return confirmAndPersistFix(rc.configPath, res, r)
}

func promptCLIAttackerResolution(r *bufio.Reader, strategies []string) config.AutoFixOptions {
	fmt.Printf("\n%s %s Strategies %v require an Attacker LLM, which is not configured.\n",
		ui.PrefixGort(), ui.BoldYellow("[LOGIC CONTROL]"), strategies)
	fmt.Println("  [1] Enter a dedicated Attacker LLM ID (parallel execution)")
	fmt.Println("  [2] No Attacker LLM -> Remove Attacker-dependent strategies from config")
	fmt.Println("  [3] No Attacker LLM -> Reuse Judge LLM sequentially (Attack first, then Judge)")
	fmt.Print("Select resolution option [1/2/3] (default 3): ")
	switch readTrimmedLine(r) {
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

func (rc *repairContext) executeSelectedAutoFix(opts config.AutoFixOptions, r *bufio.Reader) (config.AutoFixResult, bool, error) {
	if !hasFormatOrCatalogIssues(rc.rep) && opts.AttackerResolution != config.ResolveNone {
		res, err := config.AutoFixMiniEngine(rc.raw, rc.cat, opts)
		return res, err == nil, err
	}
	engineChoice := promptAutoFixEngineChoice(r)
	if engineChoice == "n" || engineChoice == "skip" {
		return config.AutoFixResult{}, false, nil
	}
	if engineChoice == "2" {
		judge := buildJudgeProvider(rc.cfg.Judge)
		res, err := config.AutoFixWithJudgeLLM(context.Background(), judge, rc.raw, rc.cat, opts)
		return res, err == nil, err
	}
	res, err := config.AutoFixMiniEngine(rc.raw, rc.cat, opts)
	return res, err == nil, err
}

func hasFormatOrCatalogIssues(rep config.ValidationReport) bool {
	return slices.ContainsFunc(rep.Issues, func(iss config.ValidationIssue) bool {
		return iss.Code != "LOGIC_MISSING_ATTACKER"
	})
}

func promptAutoFixEngineChoice(r *bufio.Reader) string {
	fmt.Printf("\n%s %s Select an Auto-Fix Engine to repair YAML issues:\n",
		ui.PrefixGort(), ui.BoldCyan("[AUTO-FIX]"))
	fmt.Println("  [1] Mini Engine (Instant <1ms deterministic syntax & SQLite catalog repair) [Recommended]")
	fmt.Println("  [2] Judge LLM Repair (AI syntax repair via configured Judge LLM + Mini Engine check)")
	fmt.Println("  [N] Skip auto-fix and edit file manually")
	fmt.Print("Choice [1/2/N] (default 1): ")
	return cmp.Or(strings.ToLower(readTrimmedLine(r)), "1")
}

func confirmAndPersistFix(configPath string, res config.AutoFixResult, r *bufio.Reader) (domain.Config, bool, error) {
	fmt.Printf("\n%s Proposed Fixes (%s):\n", ui.PrefixGort(), ui.Bold(res.EngineUsed))
	for i, f := range res.AppliedFixes {
		fmt.Printf("  %d. %s\n", i+1, ui.BoldGreen(f))
	}
	fmt.Printf("Confirm & apply these fixes to %s? [y/N]: ", configPath)
	ans := strings.ToLower(readTrimmedLine(r))
	if ans != "y" && ans != "yes" {
		return domain.Config{}, true, fmt.Errorf("validation incomplete for %s: proposed fixes were not applied. Please complete validation before running", configPath)
	}
	if err := os.WriteFile(configPath, []byte(res.FixedYAML), 0o644); err != nil {
		return domain.Config{}, false, fmt.Errorf("write fixed config %s: %w", configPath, err)
	}
	fmt.Printf("%s Updated %s and re-validating...\n", ui.PrefixGort(), ui.Bold(configPath))
	return domain.Config{}, false, nil
}

func (rc *repairContext) promptManualRevalidate(r *bufio.Reader) (domain.Config, bool, error) {
	fmt.Printf("Edit %s manually and press [y] to re-validate, or [N] to abort [y/N]: ", rc.configPath)
	ans := strings.ToLower(readTrimmedLine(r))
	if ans == "y" || ans == "yes" {
		return domain.Config{}, false, nil
	}
	return domain.Config{}, true, fmt.Errorf("validation incomplete for %s: please resolve validation issues before running", rc.configPath)
}

func readTrimmedLine(r *bufio.Reader) string {
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}
