package domain_test

import (
	"testing"

	"github.com/caveman/go-rt/internal/domain"
)

func TestTestCaseCloneIsolation(t *testing.T) {
	original := domain.TestCase{
		ID:   "tc-1",
		Vars: domain.Vars{"prompt": "SELECT * FROM users"},
		Assert: []domain.Assertion{
			{Type: "gort:grader:sql-injection", Metric: "SqlInjection"},
		},
		Metadata: domain.Metadata{
			PluginID:  "gort:plugin:sql-injection",
			Severity:  domain.SeverityHigh,
			Modifiers: map[string]string{"language": "tr"},
		},
	}

	cloned := original.Clone()
	cloned.Vars["prompt"] = "mutated"
	cloned.Assert[0].Metric = "SqlInjection/Base64"
	cloned.Metadata.Modifiers["language"] = "en"

	if original.Vars["prompt"] != "SELECT * FROM users" {
		t.Fatalf("expected original Vars to remain unchanged, got %q", original.Vars["prompt"])
	}
	if original.Assert[0].Metric != "SqlInjection" {
		t.Fatalf("expected original Assert metric to remain unchanged, got %q", original.Assert[0].Metric)
	}
	if original.Metadata.Modifiers["language"] != "tr" {
		t.Fatalf("expected original Metadata modifiers to remain unchanged, got %q", original.Metadata.Modifiers["language"])
	}
}
