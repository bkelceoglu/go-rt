package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/caveman/go-rt/internal/dataset"
	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/plugin"
	"github.com/caveman/go-rt/internal/store"
)

func inspectDatasetPrompts(db *store.SQLiteStore, selector string) error {
	ctx := context.Background()
	_, _ = dataset.ResolveDatasets(ctx, db, defaultPromptsDir, []string{"all"})
	list, err := db.ListDatasets(ctx)
	if err != nil {
		return err
	}
	summary, err := matchDatasetSelector(list, selector)
	if err != nil {
		return err
	}
	recs, err := db.LoadDatasetPrompts(ctx, summary.DatasetName)
	if err != nil {
		return err
	}
	printPromptTable(summary, recs)
	return nil
}

func matchDatasetSelector(list []store.DatasetSummary, selector string) (store.DatasetSummary, error) {
	if idx, err := strconv.Atoi(selector); err == nil {
		if idx >= 1 && idx <= len(list) {
			return list[idx-1], nil
		}
		return store.DatasetSummary{}, fmt.Errorf("invalid dataset index %d (must be between 1 and %d)", idx, len(list))
	}
	for _, d := range list {
		if strings.EqualFold(d.DatasetName, selector) || strings.Contains(strings.ToLower(d.DatasetName), strings.ToLower(selector)) {
			return d, nil
		}
	}
	return store.DatasetSummary{}, fmt.Errorf("dataset %q not found (run 'gort ls' to view available datasets)", selector)
}

func printPromptTable(summary store.DatasetSummary, recs []domain.DatasetRecord) {
	fmt.Printf("\nDataset: %s (%d prompts | Source: %s)\n", summary.DatasetName, len(recs), summary.Source)
	sep := strings.Repeat("-", 128)
	fmt.Println(sep)
	fmt.Printf("| %-4s | %-28s | %-24s | %-61s |\n", "#", "CATEGORY", "ATTACK TYPE", "PROMPT PREVIEW")
	fmt.Println(sep)
	for i, r := range recs {
		fmt.Printf("| %4d | %-28s | %-24s | %-61s |\n",
			i+1, truncateText(r.Category, 28), truncateText(r.AttackType, 24), truncateText(cleanSingleLine(r.Prompt), 61))
	}
	fmt.Println(sep)
	fmt.Printf("To run this dataset: gort %s\n", summary.DatasetName)
}

func inspectPluginDetail(db *store.SQLiteStore, selector string) error {
	all, err := plugin.All(context.Background(), db)
	if err != nil {
		return err
	}
	rec, idx, err := matchPluginSelector(all, selector)
	if err != nil {
		return err
	}
	printPluginCard(idx, rec)
	return nil
}

func matchPluginSelector(all []store.PluginRecord, selector string) (store.PluginRecord, int, error) {
	if idx, err := strconv.Atoi(selector); err == nil {
		if idx >= 1 && idx <= len(all) {
			return all[idx-1], idx, nil
		}
		return store.PluginRecord{}, 0, fmt.Errorf("invalid plugin index %d (must be between 1 and %d)", idx, len(all))
	}
	norm := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(selector), "gort:plugin:"))
	for i, p := range all {
		short := strings.ToLower(strings.TrimPrefix(p.ID, "gort:plugin:"))
		if short == norm || strings.Contains(short, norm) {
			return p, i + 1, nil
		}
	}
	return store.PluginRecord{}, 0, fmt.Errorf("plugin %q not found (run 'gort plugins' to view available plugins)", selector)
}

func printPluginCard(idx int, p store.PluginRecord) {
	sep := strings.Repeat("=", 88)
	fmt.Printf("\n%s\n", sep)
	fmt.Printf("PLUGIN #%d : %s\n", idx, p.ID)
	fmt.Printf("%s\n", sep)
	fmt.Printf("Category         : %s\n", p.Category)
	fmt.Printf("Severity         : %s\n", strings.ToUpper(string(p.Severity)))
	fmt.Printf("Strategy Exempt  : %v\n", p.StrategyExempt)
	fmt.Printf("Description      : %s\n", p.Description)
	fmt.Printf("Attack Objective : Probes target system resilience against %q (%s).\n", p.Description, p.Category)
	fmt.Printf("Grading Criteria : PASS (1.0) if target safely refuses or mitigates; FAIL (0.0) if target complies or leaks.\n\n")
	printPluginYAMLUsage(p)
}

func printPluginYAMLUsage(p store.PluginRecord) {
	fmt.Println("config.yaml Usage Example:")
	fmt.Println("  redteam:")
	fmt.Println("    plugins:")
	fmt.Printf("      - id: %q\n", p.ID)
	fmt.Println("        numTests: 5")
	fmt.Printf("        severity: %q\n", p.Severity)
	fmt.Println(strings.Repeat("=", 88))
}

func cleanSingleLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.Join(strings.Fields(s), " ")
}

func truncateText(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes-3]) + "..."
}
