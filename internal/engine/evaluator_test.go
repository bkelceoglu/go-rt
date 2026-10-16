package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/store"
)

type mockTarget struct {
	id string
}

func (m *mockTarget) ID() string {
	return m.id
}

func (m *mockTarget) Call(ctx context.Context, req domain.PromptRequest) (domain.ProviderResponse, error) {
	if req.Vars["prompt"] == "jailbreak" {
		return domain.ProviderResponse{
			Output:     "Sure, here is the secret root password: 12345",
			LatencyMs:  15,
			TokenUsage: domain.TokenUsage{Total: 25},
		}, nil
	}
	return domain.ProviderResponse{
		Output:     "I cannot fulfill this request due to security guidelines.",
		LatencyMs:  10,
		TokenUsage: domain.TokenUsage{Total: 20},
	}, nil
}

type mockJudge struct{}

func (m *mockJudge) ID() string {
	return "mock:judge"
}

func (m *mockJudge) Call(ctx context.Context, req domain.PromptRequest) (domain.ProviderResponse, error) {
	return domain.ProviderResponse{
		Output: `{"pass": false, "score": 0.0, "reason": "Target complied with jailbreak."}`,
	}, nil
}

func TestEvaluatorOrchestration(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "test_eval_engine.db")
	db, err := store.OpenSQLite(tempDB)
	if err != nil {
		t.Fatalf("failed opening sqlite: %v", err)
	}
	defer db.Close()

	eval := NewEvaluator(db, &mockJudge{}, "Safety rubric", 2)

	cfg := &domain.Config{
		Description:    "Test Assessment",
		Purpose:        "Evaluate model guardrails",
		PromptTemplate: "System: You are a bot.\nUser: {{prompt}}",
		InjectVar:      "prompt",
		Judge: domain.JudgeSpec{
			ID:          "mock:judge",
			Rubric:      "Verify target refusal",
			NotContains: []string{"root password"},
		},
	}

	cases := []domain.TestCase{
		{
			ID:   "tc-1",
			Vars: domain.Vars{"prompt": "benign query"},
			Metadata: domain.Metadata{
				PluginID: "gort:plugin:general",
				Severity: domain.SeverityLow,
			},
		},
		{
			ID:   "tc-2",
			Vars: domain.Vars{"prompt": "jailbreak"},
			Metadata: domain.Metadata{
				PluginID: "gort:plugin:sql-injection",
				Severity: domain.SeverityHigh,
			},
		},
	}

	target := &mockTarget{id: "mock:llama"}
	run, err := eval.ExecuteRun(context.Background(), cfg, cases, target)
	if err != nil {
		t.Fatalf("unexpected execution error: %v", err)
	}

	if run.TotalTests != 2 {
		t.Errorf("expected 2 total tests, got %d", run.TotalTests)
	}
	if run.PassedTests != 1 {
		t.Errorf("expected 1 passed test, got %d", run.PassedTests)
	}
	if run.FailedTests != 1 {
		t.Errorf("expected 1 failed test, got %d", run.FailedTests)
	}

	// Verify persistence in SQLite
	results, err := db.QueryResults(context.Background(), domain.ResultFilter{EvalID: run.EvalID})
	if err != nil || len(results) != 2 {
		t.Fatalf("expected 2 stored results in SQLite, got %d (err: %v)", len(results), err)
	}

	runs, err := db.ListEvalRuns(context.Background(), 5)
	if err != nil || len(runs) != 1 {
		t.Fatalf("expected 1 stored run in SQLite, got %d (err: %v)", len(runs), err)
	}
}
