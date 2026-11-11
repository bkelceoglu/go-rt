package web

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
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
)

// AssessmentStatus captures the live execution state of a background security evaluation.
type AssessmentStatus struct {
	Running      bool      `json:"running"`
	EvalID       string    `json:"evalId"`
	Dataset      string    `json:"dataset"`
	TargetID     string    `json:"targetId"`
	StartedAt    time.Time `json:"startedAt"`
	DurationMs   int64     `json:"durationMs"`
	CurrentCase  int       `json:"currentCase"`
	TotalCases   int       `json:"totalCases"`
	PassedTests  int       `json:"passedTests"`
	FailedTests  int       `json:"failedTests"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
}

// AssessmentManager coordinates background red team assessments and status observation.
type AssessmentManager struct {
	db      *store.SQLiteStore
	workDir string
	mu      sync.Mutex
	wg      sync.WaitGroup
	runGen  uint64
	cancel  context.CancelFunc
	status  AssessmentStatus
}

// NewAssessmentManager creates an execution manager.
func NewAssessmentManager(db *store.SQLiteStore, workDir string) *AssessmentManager {
	return &AssessmentManager{
		db:      db,
		workDir: workDir,
	}
}

// Status returns a snapshot of the current assessment run.
func (m *AssessmentManager) Status() AssessmentStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.status
	if s.Running && !s.StartedAt.IsZero() {
		s.DurationMs = time.Since(s.StartedAt).Milliseconds()
	}
	return s
}

// Start launches an assessment in a background goroutine.
func (m *AssessmentManager) Start(cfg *domain.Config, datasetName string) error {
	ctx, cancel, gen, err := m.beginRunSession(datasetName)
	if err != nil {
		return err
	}
	m.wg.Go(func() {
		defer cancel()
		m.runAssessmentWorkflow(ctx, gen, cfg, datasetName)
	})
	return nil
}

func (m *AssessmentManager) beginRunSession(datasetName string) (context.Context, context.CancelFunc, uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.status.Running {
		return nil, nil, 0, errors.New("an assessment is already actively running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.runGen++
	m.cancel = cancel
	m.status = AssessmentStatus{Running: true, Dataset: datasetName, StartedAt: time.Now()}
	return ctx, cancel, m.runGen, nil
}

// Stop terminates any active assessment.
func (m *AssessmentManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.status.Running || m.cancel == nil {
		return errors.New("no active assessment is running")
	}
	m.cancel()
	m.status.Running = false
	m.status.ErrorMessage = "Assessment canceled by operator"
	return nil
}

// Wait blocks until any running background assessment goroutine exits.
func (m *AssessmentManager) Wait() {
	m.wg.Wait()
}

func (m *AssessmentManager) runAssessmentWorkflow(ctx context.Context, gen uint64, cfg *domain.Config, datasetName string) {
	_ = telemetry.EnableLiveLog(filepath.Join(m.workDir, telemetry.DefaultLogPath))
	cases, err := m.resolveTestCases(ctx, cfg, datasetName)
	if err != nil {
		m.recordFailure(gen, err)
		return
	}
	if err := m.executeEvaluationRounds(ctx, cfg, cases); err != nil {
		m.recordFailure(gen, err)
		return
	}
	m.markComplete(gen)
}

func (m *AssessmentManager) resolveTestCases(ctx context.Context, cfg *domain.Config, datasetName string) ([]domain.TestCase, error) {
	args := resolveDatasetArgs(datasetName, cfg.Datasets)
	if len(args) > 0 {
		return m.loadCasesFromDatasets(ctx, cfg, args)
	}
	return m.synthesizeCasesViaAttacker(ctx, cfg)
}

func resolveDatasetArgs(datasetName string, cfgDatasets []string) []string {
	if datasetName == "all" {
		return []string{"all"}
	}
	if datasetName != "" && datasetName != "config" {
		return []string{datasetName}
	}
	return cfgDatasets
}

func (m *AssessmentManager) loadCasesFromDatasets(ctx context.Context, cfg *domain.Config, args []string) ([]domain.TestCase, error) {
	pDir := m.resolvePromptsDir(cfg.PromptsDir)
	sets, err := dataset.ResolveDatasets(ctx, m.db, pDir, args)
	if err != nil {
		return nil, fmt.Errorf("resolve datasets: %w", err)
	}
	all := []domain.TestCase{}
	injectVar := template.ResolveInjectVar(cfg.InjectVar, cfg.PromptTemplate)
	for _, s := range sets {
		all = append(all, dataset.RecordsToTestCases(s.Records, injectVar)...)
	}
	if len(all) == 0 {
		return nil, errors.New("no test cases found in requested datasets")
	}
	return all, nil
}

func (m *AssessmentManager) resolvePromptsDir(dir string) string {
	dir = cmp.Or(strings.TrimSpace(dir), "prompts")
	if !filepath.IsLocal(dir) {
		dir = filepath.Base(dir)
	}
	return filepath.Join(m.workDir, dir)
}

func (m *AssessmentManager) synthesizeCasesViaAttacker(ctx context.Context, cfg *domain.Config) ([]domain.TestCase, error) {
	attacker := provider.BuildFromSpec(cfg.Attacker)
	if attacker == nil {
		return nil, errors.New("no attacker llm configured for automated prompt synthesis")
	}
	syn := plugin.NewSynthesizer(attacker)
	recs, err := plugin.All(ctx, m.db)
	if err != nil {
		return nil, fmt.Errorf("load plugins catalog: %w", err)
	}
	recMap := make(map[string]store.PluginRecord, len(recs))
	for _, r := range recs {
		recMap[r.ID] = r
	}
	return m.synthesizeForPluginSpecs(ctx, syn, cfg, recMap)
}

func (m *AssessmentManager) synthesizeForPluginSpecs(ctx context.Context, syn *plugin.Synthesizer, cfg *domain.Config, recMap map[string]store.PluginRecord) ([]domain.TestCase, error) {
	all := []domain.TestCase{}
	for _, spec := range cfg.Redteam.Plugins {
		rec, numTests := resolvePluginSynthTarget(spec, cfg.Redteam.NumTestsPerPlugin, recMap)
		if cases, err := syn.SynthesizeBatch(ctx, rec, numTests, cfg.Purpose, cfg.Redteam.Languages); err == nil {
			all = append(all, cases...)
		}
	}
	if len(all) == 0 {
		return nil, errors.New("failed synthesizing test cases via attacker llm")
	}
	return all, nil
}

func resolvePluginSynthTarget(spec domain.PluginSpec, defaultNum int, recMap map[string]store.PluginRecord) (store.PluginRecord, int) {
	rec := recMap[spec.ID]
	if rec.ID == "" {
		rec = store.PluginRecord{ID: spec.ID, Category: "redteam", Severity: spec.Severity}
	}
	numTests := spec.NumTests
	if numTests <= 0 {
		numTests = defaultNum
	}
	return rec, numTests
}

type targetEvalEnv struct {
	cfg       *domain.Config
	evaluator *engine.Evaluator
	attacker  domain.Provider
	exemptMap map[string]bool
	injectVar string
	cases     []domain.TestCase
}

func (m *AssessmentManager) executeEvaluationRounds(ctx context.Context, cfg *domain.Config, cases []domain.TestCase) error {
	env := m.buildTargetEvalEnv(ctx, cfg, cases)
	for _, targetSpec := range cfg.Targets {
		if err := m.evaluateTarget(ctx, env, targetSpec); err != nil {
			return err
		}
	}
	return nil
}

func (m *AssessmentManager) buildTargetEvalEnv(ctx context.Context, cfg *domain.Config, cases []domain.TestCase) targetEvalEnv {
	judge := provider.BuildJudge(cfg.Judge)
	var attacker domain.Provider
	if strings.TrimSpace(cfg.Attacker.ID) != "" {
		attacker = provider.BuildFromSpec(cfg.Attacker)
	}
	attacker, judge = provider.WrapSharedSequential(attacker, judge, cfg.SequentialSharedLLM)
	evaluator := engine.NewEvaluator(m.db, judge, cfg.Judge.Rubric, cfg.MaxConcurrency)
	evaluator.SetVerbose(true)
	return targetEvalEnv{
		cfg:       cfg,
		evaluator: evaluator,
		attacker:  attacker,
		exemptMap: m.loadExemptMap(ctx),
		injectVar: template.ResolveInjectVar(cfg.InjectVar, cfg.PromptTemplate),
		cases:     cases,
	}
}

func (m *AssessmentManager) evaluateTarget(ctx context.Context, env targetEvalEnv, spec domain.ProviderSpec) error {
	target := provider.BuildFromSpec(spec)
	expanded, err := strategy.ExpandForTarget(ctx, env.cases, env.cfg.Strategies, env.exemptMap, env.injectVar, target, env.attacker)
	if err != nil {
		return fmt.Errorf("expand strategies for %s: %w", spec.ID, err)
	}
	m.updateTargetStatus(spec.ID, len(expanded))
	run, err := env.evaluator.ExecuteRun(ctx, env.cfg, expanded, target)
	if err != nil {
		return fmt.Errorf("execute run for %s: %w", spec.ID, err)
	}
	m.recordRunCompletion(run)
	return nil
}

func (m *AssessmentManager) loadExemptMap(ctx context.Context) map[string]bool {
	recs, err := plugin.All(ctx, m.db)
	if err != nil {
		return map[string]bool{}
	}
	out := make(map[string]bool, len(recs))
	for _, r := range recs {
		if r.StrategyExempt {
			out[r.ID] = true
		}
	}
	return out
}

func (m *AssessmentManager) updateTargetStatus(targetID string, total int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.TargetID = targetID
	m.status.TotalCases = total
}

func (m *AssessmentManager) recordRunCompletion(run domain.EvalRun) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.EvalID = run.EvalID
	m.status.PassedTests += run.PassedTests
	m.status.FailedTests += run.FailedTests
	m.status.CurrentCase += run.TotalTests
}

func (m *AssessmentManager) recordFailure(gen uint64, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.runGen != gen || !m.status.Running {
		return
	}
	m.status.Running = false
	m.status.ErrorMessage = err.Error()
	m.status.DurationMs = time.Since(m.status.StartedAt).Milliseconds()
}

func (m *AssessmentManager) markComplete(gen uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.runGen != gen || !m.status.Running {
		return
	}
	m.status.Running = false
	m.status.DurationMs = time.Since(m.status.StartedAt).Milliseconds()
}
