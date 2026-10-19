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
	"github.com/caveman/go-rt/internal/engine"
	"github.com/caveman/go-rt/internal/plugin"
	"github.com/caveman/go-rt/internal/provider"
	"github.com/caveman/go-rt/internal/store"
	"github.com/caveman/go-rt/internal/strategy"
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
		return executeAttackerSynthesisFlow(&cfg, db, in)
	}
	return executeDatasetFlow(configPath, &cfg, db, datasetArgs)
}

func executeAttackerSynthesisFlow(cfg *domain.Config, db *store.SQLiteStore, in io.Reader) error {
	confirmed, err := promptAttackerConfirmation(cfg, in)
	if err != nil || !confirmed {
		fmt.Printf("%s Operation canceled by user.\n", ui.PrefixGort())
		return err
	}
	ctx := context.Background()
	allCases, err := synthesizeAllCases(ctx, cfg, db)
	if err != nil {
		return err
	}
	fmt.Printf("%s Synthesized %s attack prompts. Starting assessment...\n",
		ui.PrefixGort(), ui.BoldGreen(fmt.Sprintf("%d", len(allCases))))
	return runTargetsEvaluation(ctx, cfg, db, "synthetic_redteam", allCases)
}

func synthesizeAllCases(ctx context.Context, cfg *domain.Config, db *store.SQLiteStore) ([]domain.TestCase, error) {
	recMap, err := loadPluginRecordMap(ctx, db)
	if err != nil {
		return nil, err
	}
	attacker := buildProviderFromSpec(cfg.Attacker)
	syn := plugin.NewSynthesizer(attacker)
	fmt.Printf("\n%s Synthesizing dynamic prompts via Attacker LLM (%s)...\n",
		ui.PrefixGort(), ui.Bold(attacker.ID()))
	return collectSynthesizedCases(ctx, syn, cfg, recMap)
}

func collectSynthesizedCases(ctx context.Context, syn *plugin.Synthesizer, cfg *domain.Config, recMap map[string]store.PluginRecord) ([]domain.TestCase, error) {
	var allCases []domain.TestCase
	for _, spec := range cfg.Redteam.Plugins {
		rec := resolvePluginRecord(spec, recMap)
		cases, err := syn.SynthesizeBatch(ctx, rec, spec.NumTests, cfg.Purpose, cfg.Redteam.Languages)
		if err != nil {
			fmt.Printf("  %s Failed synthesizing for %s: %v\n", ui.Badge("FAIL"), spec.ID, err)
			continue
		}
		allCases = append(allCases, cases...)
	}
	if len(allCases) == 0 {
		return nil, fmt.Errorf("no test cases could be synthesized by attacker LLM")
	}
	return allCases, nil
}

func loadPluginRecordMap(ctx context.Context, db *store.SQLiteStore) (map[string]store.PluginRecord, error) {
	allRecs, err := plugin.All(ctx, db)
	if err != nil {
		return nil, err
	}
	recMap := make(map[string]store.PluginRecord, len(allRecs))
	for _, r := range allRecs {
		recMap[r.ID] = r
	}
	return recMap, nil
}

func resolvePluginRecord(spec domain.PluginSpec, recMap map[string]store.PluginRecord) store.PluginRecord {
	if rec, ok := recMap[spec.ID]; ok {
		return rec
	}
	return store.PluginRecord{ID: spec.ID, Category: "redteam", Severity: spec.Severity, Description: spec.ID}
}

func runTargetsEvaluation(ctx context.Context, cfg *domain.Config, db *store.SQLiteStore, setName string, cases []domain.TestCase) error {
	judge := buildJudgeProvider(cfg.Judge)
	attacker := buildAttackerProvider(cfg.Attacker)
	exemptMap := loadStrategyExemptMap(ctx, db)
	injectVar := template.ResolveInjectVar(cfg.InjectVar, cfg.PromptTemplate)
	evaluator := engine.NewEvaluator(db, judge, cfg.Judge.Rubric, cfg.MaxConcurrency)

	for _, targetSpec := range cfg.Targets {
		if err := evaluateSingleTarget(ctx, cfg, evaluator, targetSpec, attacker, exemptMap, injectVar, setName, cases); err != nil {
			return err
		}
	}
	return nil
}

func evaluateSingleTarget(ctx context.Context, cfg *domain.Config, evaluator *engine.Evaluator, targetSpec domain.ProviderSpec, attacker domain.Provider, exemptMap map[string]bool, injectVar, setName string, cases []domain.TestCase) error {
	target := buildProviderFromSpec(targetSpec)
	expanded, err := strategy.ExpandForTarget(ctx, cases, cfg.Strategies, exemptMap, injectVar, target, attacker)
	if err != nil {
		return err
	}
	fmt.Printf("       Target: %s | Test Cases (after strategies): %s\n",
		ui.Bold(target.ID()), ui.BoldGreen(fmt.Sprintf("%d", len(expanded))))
	run, err := evaluator.ExecuteRun(ctx, cfg, expanded, target)
	if err != nil {
		return err
	}
	printRunCompletionCard(setName, run)
	return nil
}

func loadStrategyExemptMap(ctx context.Context, db *store.SQLiteStore) map[string]bool {
	recs, _ := plugin.All(ctx, db)
	out := make(map[string]bool, len(recs))
	for _, r := range recs {
		if r.StrategyExempt {
			out[r.ID] = true
		}
	}
	return out
}

func buildAttackerProvider(spec domain.ProviderSpec) domain.Provider {
	if spec.ID == "" {
		return nil
	}
	return buildProviderFromSpec(spec)
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
	fmt.Printf("%s Loaded Config   : %s | Targets: %d | Judge: %s | Active Plugins: %d | Strategies: %d (%s)\n",
		ui.PrefixGort(), configPath, len(cfg.Targets), cfg.Judge.ID, len(cfg.Redteam.Plugins), len(cfg.Strategies), joinStrategyIDs(cfg.Strategies))

	for i, set := range sets {
		if err := runSingleNamedSet(i+1, len(sets), cfg, db, set); err != nil {
			fmt.Printf("%s Error executing prompt set %s: %v\n", ui.PrefixGort(), set.Name, err)
		}
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

func runSingleNamedSet(idx, total int, cfg *domain.Config, db *store.SQLiteStore, set dataset.NamedDataset) error {
	injectVar := template.ResolveInjectVar(cfg.InjectVar, cfg.PromptTemplate)
	cases := dataset.RecordsToTestCases(set.Records, injectVar)
	fmt.Printf("\n%s [%d/%d] Running Prompt Set %s : %s seed prompts (via SQLite prompt_bank)\n",
		ui.PrefixGort(), idx, total, ui.BoldCyan(set.Name), ui.BoldGreen(fmt.Sprintf("%d", len(cases))))
	return runTargetsEvaluation(context.Background(), cfg, db, set.Name, cases)
}

func printRunCompletionCard(setName string, run domain.EvalRun) {
	sep := strings.Repeat("-", 88)
	fmt.Printf("\n%s\n", ui.Dim(sep))
	fmt.Printf("%s Assessment Complete [%s] | Pass: %s | Fail: %s | Total: %d | Duration: %dms\n",
		ui.PrefixGort(), ui.BoldCyan(setName),
		ui.BoldGreen(fmt.Sprintf("%d", run.PassedTests)),
		ui.BoldRed(fmt.Sprintf("%d", run.FailedTests)),
		run.TotalTests, run.DurationMs)
	fmt.Printf("%s TraceID: %s | Stored in SQLite: %s\n",
		ui.PrefixGort(), ui.Dim(run.TraceID), ui.Dim(".gort/gort.db (eval_runs & eval_results)"))
	fmt.Printf("%s\n\n", ui.Dim(sep))
}

func buildProviderFromSpec(spec domain.ProviderSpec) domain.Provider {
	var base domain.Provider
	if strings.HasPrefix(spec.ID, "webhook:") || spec.BodyTmpl != "" {
		base = provider.NewWebhook(spec, nil)
	} else {
		base = provider.NewOpenAICompatible(spec, nil)
	}
	return provider.NewRateLimitedProvider(base, 0, 0, 4)
}

func buildJudgeProvider(spec domain.JudgeSpec) domain.Provider {
	if spec.ID == "" {
		return nil
	}
	pSpec := domain.ProviderSpec{
		ID:      spec.ID,
		BaseURL: spec.BaseURL,
		APIKey:  spec.APIKey,
		Temp:    spec.Temp,
		MaxToks: spec.MaxToks,
	}
	return buildProviderFromSpec(pSpec)
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
