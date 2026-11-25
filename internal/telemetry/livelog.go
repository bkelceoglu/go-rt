package telemetry

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bkelceoglu/go-rt/internal/domain"
)

// DefaultLogPath is the default append-only live log file for tail -f monitoring.
const DefaultLogPath = ".gort/gort.log"

var (
	logMu         sync.Mutex
	activeLogPath string
)

// EnableLiveLog activates real-time file logging at path and ensures the file exists for tail -f.
func EnableLiveLog(path string) error {
	logMu.Lock()
	defer logMu.Unlock()
	path = cmp.Or(path, DefaultLogPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create live log dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open live log %s: %w", path, err)
	}
	activeLogPath = path
	return f.Close()
}

// AppendLog writes a formatted, timestamped line to the active live log file immediately.
func AppendLog(level, tag, message string) {
	logMu.Lock()
	defer logMu.Unlock()
	if activeLogPath == "" {
		return
	}
	f, err := os.OpenFile(activeLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	ts := time.Now().Format(time.RFC3339)
	_, _ = fmt.Fprintf(f, "%s [%-6s] [%-14s] %s\n", ts, level, tag, sanitizeLogText(message))
}

func sanitizeLogText(s string) string {
	cleaned := strings.ReplaceAll(strings.TrimSpace(s), "\r\n", " \\n ")
	return strings.ReplaceAll(cleaned, "\n", " \\n ")
}

// LogRunStart records the start of an evaluation run in .gort/gort.log.
func LogRunStart(run domain.EvalRun) {
	targets := strings.Join(run.TargetProviders, ",")
	msg := fmt.Sprintf("eval_id=%s trace_id=%s targets=%s judge=%s total_cases=%d desc=%q",
		run.EvalID, run.TraceID, targets, run.JudgeProvider, run.TotalTests, run.Description)
	AppendLog("INFO", "RUN_START", msg)
}

// LogSynthesizedCases records Attacker LLM synthesized prompts in .gort/gort.log.
func LogSynthesizedCases(pluginID string, cases []domain.TestCase) {
	for i, tc := range cases {
		msg := fmt.Sprintf("plugin=%s idx=%d/%d prompt=%q", pluginID, i+1, len(cases), tc.Vars["prompt"])
		AppendLog("SYNTH", pluginID, msg)
	}
}

// LogCaseResult records the complete details of a finished test case in .gort/gort.log.
func LogCaseResult(idx, total int, tc domain.TestCase, res domain.EvalResult) {
	status := "FAIL"
	if res.Pass {
		status = "PASS"
	}
	tag := fmt.Sprintf("CASE %d/%d", idx, total)
	summary := fmt.Sprintf("eval_id=%s plugin=%s strategy=%s score=%.1f latency=%dms tokens=%d",
		res.EvalID, tc.Metadata.PluginID, res.StrategyID, res.Score, res.LatencyMs, res.TokenUsage.Total)
	AppendLog(status, tag, summary)
	AppendLog("DETAIL", tag+" PROMPT", res.RenderedPrompt)
	logCaseOutputOrTurns(tag, res)
	AppendLog("DETAIL", tag+" JUDGE", fmt.Sprintf("pass=%v score=%.1f reason=%q", res.Pass, res.Score, res.JudgeReason))
}

func logCaseOutputOrTurns(tag string, res domain.EvalResult) {
	if len(res.ConversationHistory) == 0 {
		AppendLog("DETAIL", tag+" OUTPUT", res.TargetOutput)
		return
	}
	for _, t := range res.ConversationHistory {
		turnLabel := fmt.Sprintf("%s TURN#%d", tag, t.Turn)
		if t.Backtrack {
			turnLabel += " BACKTRACKED"
		}
		AppendLog("TURN", turnLabel+" USER", t.Prompt)
		AppendLog("TURN", turnLabel+" ASST", t.Output)
	}
}

// LogRunComplete records the completion summary of an evaluation run in .gort/gort.log.
func LogRunComplete(run domain.EvalRun) {
	msg := fmt.Sprintf("eval_id=%s trace_id=%s pass=%d fail=%d total=%d tokens=%d duration_ms=%d",
		run.EvalID, run.TraceID, run.PassedTests, run.FailedTests, run.TotalTests, run.TotalTokens, run.DurationMs)
	AppendLog("INFO", "RUN_COMPLETE", msg)
}
