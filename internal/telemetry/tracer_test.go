package telemetry

import (
	"encoding/hex"
	"testing"

	"github.com/caveman/go-rt/internal/domain"
)

func TestTraceAndSpanIDGeneration(t *testing.T) {
	t.Parallel()
	traceID := GenerateTraceID()
	if len(traceID) != 32 {
		t.Fatalf("expected 32 hex chars for TraceID, got %d (%s)", len(traceID), traceID)
	}
	if _, err := hex.DecodeString(traceID); err != nil {
		t.Fatalf("TraceID %s is not valid hex: %v", traceID, err)
	}

	spanID := GenerateSpanID()
	if len(spanID) != 16 {
		t.Fatalf("expected 16 hex chars for SpanID, got %d (%s)", len(spanID), spanID)
	}
	if _, err := hex.DecodeString(spanID); err != nil {
		t.Fatalf("SpanID %s is not valid hex: %v", spanID, err)
	}
}

func TestSpanLifecycle(t *testing.T) {
	t.Parallel()
	span := NewSpan("", "", "eval-1", "res-1", "gort.test_case", domain.SpanKindClient)
	if span.TraceID == "" || span.SpanID == "" {
		t.Errorf("expected non-empty trace and span IDs")
	}
	if span.StartTimeUnixNano <= 0 {
		t.Errorf("expected non-zero start time")
	}
	if span.StatusCode != "UNSET" {
		t.Errorf("expected UNSET status, got %s", span.StatusCode)
	}

	FinishSpan(&span, true)
	if span.EndTimeUnixNano < span.StartTimeUnixNano {
		t.Errorf("expected end time >= start time")
	}
	if span.StatusCode != "OK" {
		t.Errorf("expected OK status, got %s", span.StatusCode)
	}
}

func TestSemanticAttributes(t *testing.T) {
	t.Parallel()
	usage := domain.TokenUsage{Prompt: 10, Completion: 5, Total: 15}
	genAI := BuildGenAIAttributes("ollama", "llama3.1:8b", usage, 150)
	if genAI["gen_ai.system"] != "ollama" || genAI["gen_ai.usage.total_tokens"] != 15 {
		t.Errorf("unexpected GenAI attributes: %+v", genAI)
	}

	redteam := BuildRedteamAttributes("gort:plugin:sql-injection", "injection", "Blind SQLi", "gort:strategy:basic", false, 0.0)
	if redteam["gort.eval.pass"] != false || redteam["gort.eval.score"] != 0.0 {
		t.Errorf("unexpected Redteam attributes: %+v", redteam)
	}

	merged := MergeAttributes(genAI, redteam)
	if len(merged) != len(genAI)+len(redteam) {
		t.Errorf("expected merged size %d, got %d", len(genAI)+len(redteam), len(merged))
	}
}
