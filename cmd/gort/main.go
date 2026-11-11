package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/caveman/go-rt/internal/dataset"
	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/engine"
	"github.com/caveman/go-rt/internal/plugin"
	"github.com/caveman/go-rt/internal/provider"
	"github.com/caveman/go-rt/internal/store"
	"github.com/caveman/go-rt/internal/strategy"
	"github.com/caveman/go-rt/internal/telemetry"
	"github.com/caveman/go-rt/internal/template"
	"github.com/caveman/go-rt/internal/ui"
	"github.com/caveman/go-rt/internal/web"
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
	if handleInfoCommand(parsed.Kind) {
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

func handleInfoCommand(kind cliCommandKind) bool {
	switch kind {
	case cmdHelp:
		printUsage()
		return true
	case cmdVersion:
		fmt.Printf("gort v%s (Go 1.27+ Enterprise AI Red Teaming Engine)\n", version)
		return true
	default:
		return false
	}
}

func bootstrapAndSummarizeDB(db *store.SQLiteStore) error {
	_ = telemetry.EnableLiveLog(telemetry.DefaultLogPath)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if _, err := dataset.EnsureBootstrapped(ctx, db, defaultPromptsDir); err != nil {
		return err
	}
	printDBBootstrapBanner(ctx, db)
	return nil
}

func printDBBootstrapBanner(ctx context.Context, db *store.SQLiteStore) {
	plugins, _ := db.ListPlugins(ctx)
	cols, _ := db.CollectionMap(ctx)
	strats, _ := db.ListStrategies(ctx)
	sets, _ := db.ListDatasets(ctx)
	fmt.Printf("%s SQLite DB Ready (%s): %s | %s | %s | %s (%s total prompts)\n",
		ui.PrefixGort(), ui.Dim(defaultSQLitePath),
		ui.Bold(fmt.Sprintf("%d plugins", len(plugins))),
		ui.Bold(fmt.Sprintf("%d collections", len(cols))),
		ui.Bold(fmt.Sprintf("%d strategies", len(strats))),
		ui.Bold(fmt.Sprintf("%d datasets", len(sets))),
		ui.BoldGreen(fmt.Sprintf("%d", sumPromptCounts(sets))))
}

func sumPromptCounts(sets []store.DatasetSummary) int {
	var total int
	for _, s := range sets {
		total += s.PromptCount
	}
	return total
}

func dispatchParsedCommand(parsed parsedCLI, db *store.SQLiteStore, in io.Reader) error {
	if handled, err := dispatchCatalogInspect(parsed, db); handled {
		return err
	}
	return dispatchOperationalCommand(parsed, db, in)
}

func dispatchCatalogInspect(parsed parsedCLI, db *store.SQLiteStore) (bool, error) {
	switch parsed.Kind {
	case cmdListDatasets:
		if parsed.TargetArg != "" {
			return true, inspectDatasetPrompts(db, parsed.TargetArg)
		}
		return true, printDatasetCatalog(db)
	case cmdListPlugins:
		if parsed.TargetArg != "" {
			return true, inspectPluginDetail(db, parsed.TargetArg, parsed.AIGenerate, parsed.ConfigPath)
		}
		return true, printPluginCatalog(db)
	case cmdListStrategies:
		if parsed.TargetArg != "" {
			return true, inspectStrategyDetail(db, parsed.TargetArg)
		}
		return true, printStrategyCatalog(db)
	default:
		return false, nil
	}
}

func dispatchOperationalCommand(parsed parsedCLI, db *store.SQLiteStore, in io.Reader) error {
	switch parsed.Kind {
	case cmdListResults:
		if parsed.TargetArg != "" {
			return inspectEvalRunResults(db, parsed.TargetArg, parsed.Verbose)
		}
		return printEvalRuns(db)
	case cmdWebUI:
		return runWebDashboard(db, parsed.TargetArg)
	case cmdValidate:
		_, err := runInteractiveValidation(parsed.ConfigPath, db, in)
		return err
	case cmdPullDatasets:
		return runPullDatasets(db)
	default:
		return executeAssessment(parsed, db, in)
	}
}

func runWebDashboard(db *store.SQLiteStore, portArg string) error {
	addr := ":8585"
	if portArg != "" {
		addr = ":" + strings.TrimPrefix(portArg, ":")
	}
	fmt.Printf("\n%s Web Command Center & Config Wizard listening at %s\n",
		ui.PrefixGort(), ui.BoldGreen("http://localhost"+addr))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	srv := web.NewServer(db, ".")
	return srv.Start(ctx, addr)
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
	printPluginRows(all)
	return nil
}

func printPluginRows(all []store.PluginRecord) {
	for i, d := range all {
		badge := ui.PadRightVisible(ui.Badge(string(d.Severity)), 11)
		fmt.Printf("  %3d. %-48s | %-12s | %s | %s\n",
			i+1, ui.Bold(d.ID), ui.Cyan(d.Category), badge, ui.Dim(d.Description))
	}
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

type assessmentContext struct {
	configPath string
	cfg        *domain.Config
	db         *store.SQLiteStore
	isVerbose  bool
}

func executeAssessment(parsed parsedCLI, db *store.SQLiteStore, in io.Reader) error {
	cfg, err := runInteractiveValidation(parsed.ConfigPath, db, in)
	if err != nil {
		return err
	}
	expanded, err := plugin.ExpandSpecs(context.Background(), db, cfg.Redteam.Plugins, cfg.Redteam.NumTestsPerPlugin)
	if err != nil {
		return err
	}
	cfg.Redteam.Plugins = expanded
	ac := &assessmentContext{configPath: parsed.ConfigPath, cfg: &cfg, db: db, isVerbose: parsed.Verbose}
	if len(parsed.DatasetArgs) == 0 {
		return ac.executeAttackerSynthesisFlow(in)
	}
	return ac.executeDatasetFlow(parsed.DatasetArgs)
}

func (ac *assessmentContext) executeAttackerSynthesisFlow(in io.Reader) error {
	confirmed, err := promptAttackerConfirmation(ac.cfg, in)
	if err != nil || !confirmed {
		fmt.Printf("%s Operation canceled by user.\n", ui.PrefixGort())
		return err
	}
	ctx := context.Background()
	allCases, err := ac.synthesizeAllCases(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("%s Synthesized %s attack prompts. Starting assessment...\n",
		ui.PrefixGort(), ui.BoldGreen(fmt.Sprintf("%d", len(allCases))))
	return ac.runTargetsEvaluation(ctx, "synthetic_redteam", allCases)
}

func (ac *assessmentContext) synthesizeAllCases(ctx context.Context) ([]domain.TestCase, error) {
	recMap, err := loadPluginRecordMap(ctx, ac.db)
	if err != nil {
		return nil, err
	}
	attacker := buildProviderFromSpec(ac.cfg.Attacker)
	syn := plugin.NewSynthesizer(attacker)
	fmt.Printf("\n%s Synthesizing dynamic prompts via Attacker LLM (%s)...\n",
		ui.PrefixGort(), ui.Bold(attacker.ID()))
	return ac.collectSynthesizedCases(ctx, syn, recMap)
}

func (ac *assessmentContext) collectSynthesizedCases(ctx context.Context, syn *plugin.Synthesizer, recMap map[string]store.PluginRecord) ([]domain.TestCase, error) {
	var allCases []domain.TestCase
	for _, spec := range ac.cfg.Redteam.Plugins {
		cases, err := ac.synthesizePluginSpec(ctx, syn, spec, recMap)
		if err != nil {
			fmt.Printf("  %s Failed synthesizing for %s: %v\n", ui.Badge("FAIL"), spec.ID, err)
			continue
		}
		allCases = append(allCases, cases...)
	}
	if len(allCases) == 0 {
		return nil, errors.New("no test cases could be synthesized by attacker llm")
	}
	return allCases, nil
}

func (ac *assessmentContext) synthesizePluginSpec(ctx context.Context, syn *plugin.Synthesizer, spec domain.PluginSpec, recMap map[string]store.PluginRecord) ([]domain.TestCase, error) {
	rec := resolvePluginRecord(spec, recMap)
	cases, err := syn.SynthesizeBatch(ctx, rec, spec.NumTests, ac.cfg.Purpose, ac.cfg.Redteam.Languages)
	if err != nil {
		return nil, err
	}
	telemetry.LogSynthesizedCases(spec.ID, cases)
	if ac.isVerbose {
		printVerboseSynthesizedCases(spec.ID, cases)
	}
	return cases, nil
}

func printVerboseSynthesizedCases(pluginID string, cases []domain.TestCase) {
	for i, tc := range cases {
		fmt.Printf("  %s [%s #%d] %s\n",
			ui.BoldCyan("[SYNTHESIZED]"), pluginID, i+1, ui.Yellow(tc.Vars["prompt"]))
	}
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
		if spec.Severity != "" {
			rec.Severity = spec.Severity
		}
		return rec
	}
	return store.PluginRecord{ID: spec.ID, Category: "redteam", Severity: spec.Severity, Description: spec.ID}
}

type targetBatchRunner struct {
	cfg       *domain.Config
	evaluator *engine.Evaluator
	attacker  domain.Provider
	exemptMap map[string]bool
	injectVar string
}

func (ac *assessmentContext) runTargetsEvaluation(ctx context.Context, setName string, cases []domain.TestCase) error {
	judge := buildJudgeProvider(ac.cfg.Judge)
	attacker := buildAttackerProvider(ac.cfg.Attacker)
	attacker, judge = provider.WrapSharedSequential(attacker, judge, ac.cfg.SequentialSharedLLM)
	evaluator := engine.NewEvaluator(ac.db, judge, ac.cfg.Judge.Rubric, ac.cfg.MaxConcurrency)
	evaluator.SetVerbose(ac.isVerbose)
	runner := &targetBatchRunner{
		cfg:       ac.cfg,
		evaluator: evaluator,
		attacker:  attacker,
		exemptMap: loadStrategyExemptMap(ctx, ac.db),
		injectVar: template.ResolveInjectVar(ac.cfg.InjectVar, ac.cfg.PromptTemplate),
	}
	return runner.runAll(ctx, setName, cases)
}

func (r *targetBatchRunner) runAll(ctx context.Context, setName string, cases []domain.TestCase) error {
	for _, targetSpec := range r.cfg.Targets {
		if err := r.evaluateSingleTarget(ctx, targetSpec, setName, cases); err != nil {
			return err
		}
	}
	return nil
}

func (r *targetBatchRunner) evaluateSingleTarget(ctx context.Context, targetSpec domain.ProviderSpec, setName string, cases []domain.TestCase) error {
	target := buildProviderFromSpec(targetSpec)
	expanded, err := strategy.ExpandForTarget(ctx, cases, r.cfg.Strategies, r.exemptMap, r.injectVar, target, r.attacker)
	if err != nil {
		return err
	}
	fmt.Printf("       Target: %s | Test Cases (after strategies): %s | Live Log: %s\n",
		ui.Bold(target.ID()), ui.BoldGreen(fmt.Sprintf("%d", len(expanded))), ui.Cyan("tail -f "+telemetry.DefaultLogPath))
	run, err := r.evaluator.ExecuteRun(ctx, r.cfg, expanded, target)
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
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes", nil
}

func (ac *assessmentContext) executeDatasetFlow(datasetArgs []string) error {
	sets, err := dataset.ResolveDatasets(context.Background(), ac.db, ac.cfg.PromptsDir, datasetArgs)
	if err != nil {
		return err
	}
	fmt.Printf("%s Loaded Config   : %s | Targets: %d | Judge: %s | Active Plugins: %d | Strategies: %d (%s)\n",
		ui.PrefixGort(), ac.configPath, len(ac.cfg.Targets), ac.cfg.Judge.ID, len(ac.cfg.Redteam.Plugins), len(ac.cfg.Strategies), joinStrategyIDs(ac.cfg.Strategies))

	for i, set := range sets {
		if err := ac.runSingleNamedSet(i+1, len(sets), set); err != nil {
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

func (ac *assessmentContext) runSingleNamedSet(idx, total int, set dataset.NamedDataset) error {
	injectVar := template.ResolveInjectVar(ac.cfg.InjectVar, ac.cfg.PromptTemplate)
	cases := dataset.RecordsToTestCases(set.Records, injectVar)
	fmt.Printf("\n%s [%d/%d] Running Prompt Set %s : %s seed prompts (via SQLite prompt_bank)\n",
		ui.PrefixGort(), idx, total, ui.BoldCyan(set.Name), ui.BoldGreen(fmt.Sprintf("%d", len(cases))))
	return ac.runTargetsEvaluation(context.Background(), set.Name, cases)
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
	return provider.BuildFromSpec(spec)
}

func buildJudgeProvider(spec domain.JudgeSpec) domain.Provider {
	return provider.BuildJudge(spec)
}

func printUsage() {
	fmt.Printf("GORT v%s — Enterprise AI Red Teaming & Evaluation Engine (SQLite-Backed)\n\n", version)
	fmt.Println("Usage / Commands:")
	fmt.Println("  gort [config.yaml] <dataset_name> [-v|--verbose]  # Run a single prompt set with optional live verbose output")
	fmt.Println("  gort [config.yaml] all [-v|--verbose]             # Run ALL prompt sets in SQLite prompt_bank sequentially")
	fmt.Println("  gort config.yaml [-v|--verbose]                   # Prompt [y/N] confirmation & synthesize via Attacker LLM")
	fmt.Println("  gort ls [index|dataset_name]                      # List all prompt sets or inspect prompts as a table")
	fmt.Println("  gort plugins [index|plugin_id] [--ai]             # List all 155 plugins or inspect a plugin usage card")
	fmt.Println("  gort strategies [index|strategy_id]               # List all 28 attack strategies or inspect a strategy card")
	fmt.Println("  gort validate [config.yaml]                       # Validate YAML format, Attacker LLM logic & run Auto-Fix")
	fmt.Println("  gort results [index|eval_id] [-v|--verbose]       # List stored runs or inspect full prompt/output/turns")
	fmt.Println("  gort pull | sync | --pull-datasets                # Pull and sync all remote datasets into SQLite & prompts/")
}
