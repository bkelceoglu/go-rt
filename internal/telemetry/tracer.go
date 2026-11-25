package telemetry

import (
	"cmp"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"maps"
	"time"

	"github.com/bkelceoglu/go-rt/internal/domain"
)

// GenerateTraceID generates a standard W3C 16-byte (32-character hex) trace ID.
func GenerateTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%032x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// GenerateSpanID generates a standard W3C 8-byte (16-character hex) span ID.
func GenerateSpanID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// NewSpan creates an initialized OTelSpanRecord with nanosecond start time.
func NewSpan(traceID, parentSpanID, evalID, resultID, name string, kind domain.OTelSpanKind) domain.OTelSpanRecord {
	return domain.OTelSpanRecord{
		TraceID:           cmp.Or(traceID, GenerateTraceID()),
		SpanID:            GenerateSpanID(),
		ParentSpanID:      parentSpanID,
		EvalID:            evalID,
		ResultID:          resultID,
		Name:              name,
		Kind:              kind,
		StartTimeUnixNano: time.Now().UnixNano(),
		StatusCode:        "UNSET",
		Attributes:        make(map[string]any),
	}
}

// FinishSpan sets the end time and status code on an active span.
func FinishSpan(span *domain.OTelSpanRecord, success bool) {
	span.EndTimeUnixNano = time.Now().UnixNano()
	span.StatusCode = "ERROR"
	if success {
		span.StatusCode = "OK"
	}
}

// BuildGenAIAttributes constructs OpenTelemetry GenAI Semantic Convention attributes.
func BuildGenAIAttributes(system, model string, usage domain.TokenUsage, latencyMs int64) map[string]any {
	return map[string]any{
		"gen_ai.system":                    system,
		"gen_ai.request.model":             model,
		"gen_ai.usage.input_tokens":        usage.Prompt,
		"gen_ai.usage.output_tokens":       usage.Completion,
		"gen_ai.usage.total_tokens":        usage.Total,
		"gen_ai.client.operation.duration": latencyMs,
	}
}

// BuildRedteamAttributes constructs standardized GORT redteam attributes.
func BuildRedteamAttributes(pluginID, category, attackType, strategyID string, pass bool, score float64) map[string]any {
	return map[string]any{
		"gort.plugin.id":   pluginID,
		"gort.category":    category,
		"gort.attack_type": attackType,
		"gort.strategy.id": strategyID,
		"gort.eval.pass":   pass,
		"gort.eval.score":  score,
	}
}

// MergeAttributes combines redteam and gen_ai attributes into a single map.
func MergeAttributes(a, b map[string]any) map[string]any {
	merged := make(map[string]any, len(a)+len(b))
	maps.Copy(merged, a)
	maps.Copy(merged, b)
	return merged
}
