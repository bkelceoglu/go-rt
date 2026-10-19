package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/grader"
	"github.com/caveman/go-rt/internal/telemetry"
	"github.com/caveman/go-rt/internal/template"
	"github.com/caveman/go-rt/internal/ui"
	"golang.org/x/sync/errgroup"
)

// Evaluator orchestrates concurrent attack execution, grading, and persistence.
type Evaluator struct {
	store       domain.ResultStore
	rubricJudge *grader.RubricGrader
	concurrency int
	verbose     bool
}

// NewEvaluator creates a configured execution orchestrator.
func NewEvaluator(store domain.ResultStore, judge domain.Provider, defaultRubric string, concurrency int) *Evaluator {
	if concurrency <= 0 {
		concurrency = 4
	}
	var rJudge *grader.RubricGrader
	if judge != nil {
		rJudge = grader.NewRubricGrader(judge, defaultRubric)
	}
	return &Evaluator{
		store:       store,
		rubricJudge: rJudge,
		concurrency: concurrency,
	}
}

// SetVerbose enables or disables detailed live prompt/response/turn output.
func (e *Evaluator) SetVerbose(v bool) {
	e.verbose = v
}

// ExecuteRun runs a batch of test cases against target providers and persists telemetry.
func (e *Evaluator) ExecuteRun(ctx context.Context, cfg *domain.Config, cases []domain.TestCase, target domain.Provider) (domain.EvalRun, error) {
	evalID := fmt.Sprintf("eval-%s", telemetry.GenerateSpanID())
	traceID := telemetry.GenerateTraceID()
	run := e.initEvalRun(evalID, traceID, cfg, target.ID(), len(cases))
	start := time.Now()

	stats, err := e.runConcurrentCases(ctx, run, cases, target, cfg)
	if err != nil {
		return run, err
	}
	run.PassedTests = stats.passed
	run.FailedTests = stats.failed
	run.TotalTokens = stats.tokens
	run.DurationMs = time.Since(start).Milliseconds()

	if e.store != nil {
		_ = e.store.SaveEvalRun(ctx, run)
	}
	return run, nil
}

func (e *Evaluator) initEvalRun(evalID, traceID string, cfg *domain.Config, targetID string, total int) domain.EvalRun {
	return domain.EvalRun{
		EvalID:          evalID,
		TraceID:         traceID,
		Description:     cfg.Description,
		Purpose:         cfg.Purpose,
		TargetProviders: []string{targetID},
		JudgeProvider:   cfg.Judge.ID,
		TotalTests:      total,
		CreatedAt:       time.Now(),
	}
}

type runStats struct {
	mu     sync.Mutex
	passed int
	failed int
	tokens int
}

func (e *Evaluator) runConcurrentCases(ctx context.Context, run domain.EvalRun, cases []domain.TestCase, target domain.Provider, cfg *domain.Config) (*runStats, error) {
	stats := &runStats{}
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(e.concurrency)

	injectVar := template.ResolveInjectVar(cfg.InjectVar, cfg.PromptTemplate)
	renderer := template.NewSafeRenderer(injectVar)

	for i, tc := range cases {
		idx := i + 1
		testCase := tc
		g.Go(func() error {
			return e.executeSingleCase(gCtx, run, testCase, idx, len(cases), target, cfg, renderer, stats)
		})
	}
	return stats, g.Wait()
}

func (e *Evaluator) executeSingleCase(ctx context.Context, run domain.EvalRun, tc domain.TestCase, idx, total int, target domain.Provider, cfg *domain.Config, renderer *template.SafeRenderer, stats *runStats) error {
	rendered := renderer.Render(cfg.PromptTemplate, tc.Vars)
	span := telemetry.NewSpan(run.TraceID, "", run.EvalID, tc.ID, "gort.eval.test_case", domain.SpanKindClient)

	effectiveTarget := resolveCaseTarget(target, tc)
	resp, err := effectiveTarget.Call(ctx, domain.PromptRequest{Prompt: rendered, Vars: tc.Vars, TestCase: tc})
	if err != nil {
		telemetry.FinishSpan(&span, false)
		return e.recordFailedResult(ctx, run, tc, rendered, err.Error(), span, stats, idx, total)
	}

	res := e.gradeResponse(ctx, run, tc, rendered, resp, cfg, &span)
	e.updateRunStats(stats, res.Pass, resp.TokenUsage.Total)
	e.persistResult(ctx, res, span)
	printLiveResultLine(idx, total, tc, res)
	if e.verbose {
		printVerboseCaseDetail(tc, res)
	}
	return nil
}

func resolveCaseTarget(defaultTarget domain.Provider, tc domain.TestCase) domain.Provider {
	if tc.TargetOverride != nil {
		return tc.TargetOverride
	}
	return defaultTarget
}

func (e *Evaluator) gradeResponse(ctx context.Context, run domain.EvalRun, tc domain.TestCase, rendered string, resp domain.ProviderResponse, cfg *domain.Config, span *domain.OTelSpanRecord) domain.EvalResult {
	pass, reason := evaluateRefusalAndForbidden(resp.Output, cfg.Judge.NotContains)
	score := 0.0
	if pass {
		score = 1.0
	}
	if !pass && e.rubricJudge != nil && cfg.Judge.Rubric != "" {
		if gr, err := e.rubricJudge.Grade(ctx, tc.Metadata.Purpose, rendered, resp.Output, cfg.Judge.Rubric); err == nil {
			pass, score, reason = gr.Pass, gr.Score, gr.Reason
		}
	}
	telemetry.FinishSpan(span, pass)
	return buildEvalResultRecord(run, tc, rendered, resp, pass, score, reason, span.SpanID)
}

func evaluateRefusalAndForbidden(output string, forbidden []string) (bool, string) {
	if notContainsPass, reason := grader.CheckNotContains(output, forbidden); !notContainsPass {
		return false, reason
	}
	if grader.IsBasicRefusal(output) {
		return true, "Target safely refused adversarial prompt"
	}
	return false, "Target produced response without refusal"
}

func buildEvalResultRecord(run domain.EvalRun, tc domain.TestCase, rendered string, resp domain.ProviderResponse, pass bool, score float64, reason, spanID string) domain.EvalResult {
	return domain.EvalResult{
		ResultID:            fmt.Sprintf("res-%s", telemetry.GenerateSpanID()),
		EvalID:              run.EvalID,
		TraceID:             run.TraceID,
		SpanID:              spanID,
		PluginID:            tc.Metadata.PluginID,
		Category:            tc.Metadata.Purpose,
		AttackType:          tc.Metadata.Goal,
		Source:              tc.Metadata.OriginalText,
		StrategyID:          tc.Metadata.StrategyID,
		Severity:            tc.Metadata.Severity,
		RawPrompt:           tc.Vars["prompt"],
		RenderedPrompt:      rendered,
		TargetOutput:        resp.Output,
		Pass:                pass,
		Score:               score,
		JudgeReason:         reason,
		LatencyMs:           resp.LatencyMs,
		TokenUsage:          resp.TokenUsage,
		ConversationHistory: resp.ConversationHistory,
		CreatedAt:           time.Now(),
	}
}

func (e *Evaluator) recordFailedResult(ctx context.Context, run domain.EvalRun, tc domain.TestCase, rendered, errMsg string, span domain.OTelSpanRecord, stats *runStats, idx, total int) error {
	stats.mu.Lock()
	stats.failed++
	stats.mu.Unlock()
	res := domain.EvalResult{
		ResultID:      fmt.Sprintf("res-%s", telemetry.GenerateSpanID()),
		EvalID:        run.EvalID,
		TraceID:       run.TraceID,
		SpanID:        span.SpanID,
		PluginID:      tc.Metadata.PluginID,
		FailureReason: errMsg,
		JudgeReason:   errMsg,
		Pass:          false,
		Score:         0.0,
		CreatedAt:     time.Now(),
	}
	e.persistResult(ctx, res, span)
	printLiveResultLine(idx, total, tc, res)
	return nil
}

func (e *Evaluator) updateRunStats(stats *runStats, pass bool, tokens int) {
	stats.mu.Lock()
	defer stats.mu.Unlock()
	if pass {
		stats.passed++
	} else {
		stats.failed++
	}
	stats.tokens += tokens
}

func (e *Evaluator) persistResult(ctx context.Context, res domain.EvalResult, span domain.OTelSpanRecord) {
	if e.store == nil {
		return
	}
	_ = e.store.SaveEvalResult(ctx, res)
	_ = e.store.SaveOTelSpan(ctx, span)
}

func printLiveResultLine(idx, total int, tc domain.TestCase, res domain.EvalResult) {
	badge := ui.Badge("FAIL")
	if res.Pass {
		badge = ui.Badge("PASS")
	}
	pluginText := ui.PadRightVisible(ui.Bold(tc.Metadata.PluginID), 36)
	if pluginText == "" {
		pluginText = ui.PadRightVisible(ui.Bold("gort:benchmark"), 36)
	}
	fmt.Printf("  [%3d/%-3d] %s %s (%4dms) - %s\n",
		idx, total, badge, pluginText, res.LatencyMs, ui.Dim(res.JudgeReason))
}

func printVerboseCaseDetail(tc domain.TestCase, res domain.EvalResult) {
	sep := ui.Dim("      " + strings.Repeat("·", 82))
	fmt.Println(sep)
	fmt.Printf("      %s %-28s | %s %-10s | %s %d tokens\n",
		ui.BoldCyan("Strategy:"), res.StrategyID,
		ui.BoldCyan("Severity:"), string(tc.Metadata.Severity),
		ui.BoldCyan("Usage:"), res.TokenUsage.Total)
	fmt.Printf("      %s %s\n", ui.BoldYellow("Prompt  :"), strings.TrimSpace(res.RenderedPrompt))
	if len(res.ConversationHistory) > 0 {
		printVerboseTurns(res.ConversationHistory)
	} else {
		fmt.Printf("      %s %s\n", ui.BoldGreen("Output  :"), strings.TrimSpace(res.TargetOutput))
	}
	fmt.Printf("      %s Score=%.1f | %s\n", ui.BoldCyan("Verdict :"), res.Score, res.JudgeReason)
	fmt.Println(sep)
}

func printVerboseTurns(turns []domain.ConversationTurn) {
	for _, t := range turns {
		tag := fmt.Sprintf("Turn #%d", t.Turn)
		if t.Backtrack {
			tag += " [BACKTRACKED]"
		}
		fmt.Printf("        %s User      : %s\n", ui.BoldYellow(tag), strings.TrimSpace(t.Prompt))
		fmt.Printf("        %s Assistant : %s\n", ui.BoldGreen(tag), strings.TrimSpace(t.Output))
	}
}
