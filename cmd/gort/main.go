package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/caveman/go-rt/internal/config"
	"github.com/caveman/go-rt/internal/dataset"
	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/plugin"
	"github.com/caveman/go-rt/internal/store"
	"github.com/caveman/go-rt/internal/template"
	"github.com/caveman/go-rt/internal/ui"
)

const (
	version           = "0.1.0-dev"
	defaultPromptsDir = "prompts"
	defaultSQLitePath = ".gort/gort.db"
)

func main() {
	if err := run(os.Args[1:], os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "gort error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, in io.Reader) error {
	parsed := parseCLIArgs(args)
	if parsed.Kind == cmdHelp {
		printUsage()
		return nil
	}
	if parsed.Kind == cmdVersion {
		fmt.Printf("gort v%s (Go 1.27+ Enterprise AI Red Teaming Engine)\n", version)
		return nil
	}
	db, err := store.OpenSQLite(defaultSQLitePath)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := bootstrapAndSummarizeDB(db); err != nil {
		return err
	}
	return dispatchParsedCommand(parsed, db, in)
}

func bootstrapAndSummarizeDB(db *store.SQLiteStore) error {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if _, err := dataset.EnsureBootstrapped(ctx, db, defaultPromptsDir); err != nil {
		return err
	}
	plugins, _ := db.ListPlugins(ctx)
	cols, _ := db.CollectionMap(ctx)
	sets, _ := db.ListDatasets(ctx)
	fmt.Printf("%s SQLite DB Ready (%s): %s | %s | %s (%s total prompts)\n",
		ui.PrefixGort(), ui.Dim(defaultSQLitePath),
		ui.Bold(fmt.Sprintf("%d plugins", len(plugins))),
		ui.Bold(fmt.Sprintf("%d collections", len(cols))),
		ui.Bold(fmt.Sprintf("%d datasets", len(sets))),
		ui.BoldGreen(fmt.Sprintf("%d", sumPromptCounts(sets))))
	return nil
}

func sumPromptCounts(sets []store.DatasetSummary) int {
	total := 0
	for _, s := range sets {
		total += s.PromptCount
	}
	return total
}

func dispatchParsedCommand(parsed parsedCLI, db *store.SQLiteStore, in io.Reader) error {
	switch parsed.Kind {
	case cmdListDatasets:
		if parsed.TargetArg != "" {
			return inspectDatasetPrompts(db, parsed.TargetArg)
		}
		return printDatasetCatalog(db)
	case cmdListPlugins:
		if parsed.TargetArg != "" {
			return inspectPluginDetail(db, parsed.TargetArg, parsed.AIGenerate, parsed.ConfigPath)
		}
		return printPluginCatalog(db)
	case cmdPullDatasets:
		return runPullDatasets(db)
	default:
		return executeAssessment(parsed.ConfigPath, parsed.DatasetArgs, db, in)
	}
}

func printPluginCatalog(db *store.SQLiteStore) error {
	ctx := context.Background()
	all, err := plugin.All(ctx, db)
	if err != nil {
		return err
	}
	cols, _ := db.CollectionMap(ctx)
	fmt.Printf("\n%s (%s + %s):\n\n",
		ui.BoldCyan("GORT SQLite Redteam Plugin Catalog"),
		ui.Bold(fmt.Sprintf("%d plugins", len(all))),
		ui.Bold(fmt.Sprintf("%d collections", len(cols))))
	for i, d := range all {
		badge := ui.PadRightVisible(ui.Badge(string(d.Severity)), 11)
		fmt.Printf("  %3d. %-48s | %-12s | %s | %s\n",
			i+1, ui.Bold(d.ID), ui.Cyan(d.Category), badge, ui.Dim(d.Description))
	}
	return nil
}

func printDatasetCatalog(db *store.SQLiteStore) error {
	ctx := context.Background()
	_, _ = dataset.ResolveDatasets(ctx, db, defaultPromptsDir, []string{"all"})
	list, err := db.ListDatasets(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("\n%s (%s):\n\n",
		ui.BoldCyan("GORT SQLite Prompt Bank"),
		ui.Bold(fmt.Sprintf("%d named prompt sets", len(list))))
	for i, d := range list {
		fmt.Printf("  %2d. %-32s | %s prompts | Source: %s\n",
			i+1, ui.Bold(d.DatasetName), ui.BoldGreen(fmt.Sprintf("%6d", d.PromptCount)), ui.Dim(d.Source))
	}
	return nil
}

func runPullDatasets(db *store.SQLiteStore) error {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if _, err := dataset.PullAll(ctx, db, defaultPromptsDir); err != nil {
		return err
	}
	return printDatasetCatalog(db)
}

func executeAssessment(configPath string, datasetArgs []string, db *store.SQLiteStore, in io.Reader) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	expanded, err := plugin.ExpandSpecs(context.Background(), db, cfg.Redteam.Plugins, cfg.Redteam.NumTestsPerPlugin)
	if err != nil {
		return err
	}
	cfg.Redteam.Plugins = expanded
	if len(datasetArgs) == 0 {
		return executeAttackerSynthesisFlow(&cfg, in)
	}
	return executeDatasetFlow(configPath, &cfg, db, datasetArgs)
}

func executeAttackerSynthesisFlow(cfg *domain.Config, in io.Reader) error {
	confirmed, err := promptAttackerConfirmation(cfg, in)
	if err != nil || !confirmed {
		fmt.Println("[gort] Operation canceled by user.")
		return err
	}
	fmt.Printf("[gort] Starting Attacker LLM (%s) prompt synthesis for %d plugins (Phase 4)...\n",
		cfg.Attacker.ID, len(cfg.Redteam.Plugins))
	return nil
}

func promptAttackerConfirmation(cfg *domain.Config, in io.Reader) (bool, error) {
	fmt.Printf("[gort] No prompt dataset specified. Attacker LLM (%s) will generate prompts for %d plugins:\n",
		cfg.Attacker.ID, len(cfg.Redteam.Plugins))
	for i, p := range cfg.Redteam.Plugins {
		fmt.Printf("  %2d. %-45s (numTests: %d)\n", i+1, p.ID, p.NumTests)
	}
	fmt.Print("Proceed with prompt generation? [y/N]: ")
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes", nil
}

func executeDatasetFlow(configPath string, cfg *domain.Config, db *store.SQLiteStore, datasetArgs []string) error {
	sets, err := dataset.ResolveDatasets(context.Background(), db, cfg.PromptsDir, datasetArgs)
	if err != nil {
		return err
	}
	injectVar := template.ResolveInjectVar(cfg.InjectVar, cfg.PromptTemplate)
	fmt.Printf("[gort] Loaded Config   : %s | Targets: %d | Judge: %s | Active Plugins: %d | Strategies: %d (%s)\n",
		configPath, len(cfg.Targets), cfg.Judge.ID, len(cfg.Redteam.Plugins), len(cfg.Strategies), joinStrategyIDs(cfg.Strategies))
	for i, set := range sets {
		runSingleNamedSet(i+1, len(sets), cfg, injectVar, set)
	}
	return nil
}

func joinStrategyIDs(strategies []domain.StrategySpec) string {
	ids := make([]string, 0, len(strategies))
	for _, s := range strategies {
		ids = append(ids, s.ID)
	}
	return strings.Join(ids, ", ")
}

func runSingleNamedSet(idx, total int, cfg *domain.Config, injectVar string, set dataset.NamedDataset) {
	cases := dataset.RecordsToTestCases(set.Records, injectVar)
	renderer := template.NewSafeRenderer(injectVar)
	fmt.Printf("[gort] [%d/%d] Running Prompt Set %q : %d prompts (from SQLite prompt_bank)\n",
		idx, total, set.Name, len(cases))
	if len(cases) > 0 {
		sample := renderer.Render(cfg.PromptTemplate, cases[0].Vars)
		fmt.Printf("       Sample Rendered: %.85s...\n", sample)
	}
}

func printUsage() {
	fmt.Printf("GORT v%s — Enterprise AI Red Teaming & Evaluation Engine (SQLite-Backed)\n\n", version)
	fmt.Println("Usage / Commands:")
	fmt.Println("  gort [config.yaml] <dataset_name>    # Run a single prompt set (e.g. gort lakera_gandalf)")
	fmt.Println("  gort [config.yaml] all               # Run ALL prompt sets in SQLite prompt_bank sequentially")
	fmt.Println("  gort config.yaml                     # Prompt [y/N] confirmation & synthesize via Attacker LLM")
	fmt.Println("  gort ls [index|dataset_name]         # List all prompt sets or inspect prompts as a table")
	fmt.Println("  gort plugins [index|plugin_id]       # List all 155 plugins or inspect a plugin usage card")
	fmt.Println("  gort pull | sync | --pull-datasets   # Pull and sync all remote datasets into SQLite & prompts/")
}
