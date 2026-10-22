package web

import (
	"context"
	"fmt"
	"path/filepath"
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
	m.mu.Lock()
	if m.status.Running {
		m.mu.Unlock()
		return fmt.Errorf("an assessment is already actively running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.status = AssessmentStatus{
		Running:   true,
		Dataset:   datasetName,
		StartedAt: time.Now(),
	}
	m.mu.Unlock()

	go m.runAssessmentWorkflow(ctx, cfg, datasetName)
	return nil
}

// Stop terminates any active assessment.
func (m *AssessmentManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.status.Running || m.cancel == nil {
		return fmt.Errorf("no active assessment is running")
	}
	m.cancel()
	m.status.Running = false
	m.status.ErrorMessage = "Assessment canceled by operator"
	return nil
}

func (m *AssessmentManager) runAssessmentWorkflow(ctx context.Context, cfg *domain.Config, datasetName string) {
	_ = telemetry.EnableLiveLog(filepath.Join(m.workDir, telemetry.DefaultLogPath))
	cases, err := m.resolveTestCases(ctx, cfg, datasetName)
	if err != nil {
		m.recordFailure(err)
		return
	}
	if err := m.executeEvaluationRounds(ctx, cfg, cases); err != nil {
		m.recordFailure(err)
		return
	}
	m.markComplete()
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
		return nil, err
	}
	var all []domain.TestCase
	injectVar := template.ResolveInjectVar(cfg.InjectVar, cfg.PromptTemplate)
	for _, s := range sets {
		all = append(all, dataset.RecordsToTestCases(s.Records, injectVar)...)
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("no test cases found in requested datasets")
	}
	return all, nil
}

func (m *AssessmentManager) resolvePromptsDir(dir string) string {
	if dir == "" {
		dir = "prompts"
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(m.workDir, dir)
}

func (m *AssessmentManager) synthesizeCasesViaAttacker(ctx context.Context, cfg *domain.Config) ([]domain.TestCase, error) {
	attacker := provider.BuildFromSpec(cfg.Attacker)
	if attacker == nil {
		return nil, fmt.Errorf("no attacker LLM configured for automated prompt synthesis")
	}
	syn := plugin.NewSynthesizer(attacker)
	recs, err := plugin.All(ctx, m.db)
	if err != nil {
		return nil, err
	}
	recMap := make(map[string]store.PluginRecord, len(recs))
	for _, r := range recs {
		recMap[r.ID] = r
	}
	return m.synthesizeForPluginSpecs(ctx, syn, cfg, recMap)
}

func (m *AssessmentManager) synthesizeForPluginSpecs(ctx context.Context, syn *plugin.Synthesizer, cfg *domain.Config, recMap map[string]store.PluginRecord) ([]domain.TestCase, error) {
	var all []domain.TestCase
	for _, spec := range cfg.Redteam.Plugins {
		rec := recMap[spec.ID]
		if rec.ID == "" {
			rec = store.PluginRecord{ID: spec.ID, Category: "redteam", Severity: spec.Severity}
		}
		numTests := spec.NumTests
		if numTests <= 0 {
			numTests = cfg.Redteam.NumTestsPerPlugin
		}
		cases, err := syn.SynthesizeBatch(ctx, rec, numTests, cfg.Purpose, cfg.Redteam.Languages)
		if err == nil {
			all = append(all, cases...)
		}
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("failed synthesizing test cases via attacker LLM")
	}
	return all, nil
}

func (m *AssessmentManager) executeEvaluationRounds(ctx context.Context, cfg *domain.Config, cases []domain.TestCase) error {
	judge := provider.BuildJudge(cfg.Judge)
	evaluator := engine.NewEvaluator(m.db, judge, cfg.Judge.Rubric, cfg.MaxConcurrency)
	evaluator.SetVerbose(true)
	exemptMap := m.loadExemptMap(ctx)
	injectVar := template.ResolveInjectVar(cfg.InjectVar, cfg.PromptTemplate)

	for _, targetSpec := range cfg.Targets {
		if err := m.evaluateTarget(ctx, cfg, evaluator, targetSpec, exemptMap, injectVar, cases); err != nil {
			return err
		}
	}
	return nil
}

func (m *AssessmentManager) evaluateTarget(ctx context.Context, cfg *domain.Config, evaluator *engine.Evaluator, spec domain.ProviderSpec, exemptMap map[string]bool, injectVar string, cases []domain.TestCase) error {
	target := provider.BuildFromSpec(spec)
	attacker := provider.BuildFromSpec(cfg.Attacker)
	expanded, err := strategy.ExpandForTarget(ctx, cases, cfg.Strategies, exemptMap, injectVar, target, attacker)
	if err != nil {
		return err
	}
	m.updateTargetStatus(spec.ID, len(expanded))
	run, err := evaluator.ExecuteRun(ctx, cfg, expanded, target)
	if err != nil {
		return err
	}
	m.recordRunCompletion(run)
	return nil
}

func (m *AssessmentManager) loadExemptMap(ctx context.Context) map[string]bool {
	recs, _ := plugin.All(ctx, m.db)
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

func (m *AssessmentManager) recordFailure(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.Running = false
	m.status.ErrorMessage = err.Error()
	m.status.DurationMs = time.Since(m.status.StartedAt).Milliseconds()
}

func (m *AssessmentManager) markComplete() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.Running = false
	m.status.DurationMs = time.Since(m.status.StartedAt).Milliseconds()
}
