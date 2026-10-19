package domain

import (
	"context"
	"maps"
	"slices"
)

// Severity represents the risk impact level of a redteam finding.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Vars holds key-value variables injected into prompt templates.
type Vars map[string]string

// Clone returns a shallow copy of Vars to preserve immutability across strategy mutations.
func (v Vars) Clone() Vars {
	if v == nil {
		return Vars{}
	}
	return maps.Clone(v)
}

// Assertion defines an evaluation check attached to a TestCase.
type Assertion struct {
	Type   string `json:"type"`
	Metric string `json:"metric"`
	Value  string `json:"value,omitempty"`
}

// Metadata stores provenance, plugin/strategy context, and goal tracking for a TestCase.
type Metadata struct {
	PluginID     string            `json:"pluginId"`
	StrategyID   string            `json:"strategyId,omitempty"`
	Purpose      string            `json:"purpose,omitempty"`
	Goal         string            `json:"goal,omitempty"`
	OriginalText string            `json:"originalText,omitempty"`
	Language     string            `json:"language,omitempty"`
	Severity     Severity          `json:"severity"`
	Modifiers    map[string]string `json:"modifiers,omitempty"`
	SessionID    string            `json:"sessionId,omitempty"`
}

// Clone returns a deep copy of Metadata.
func (m Metadata) Clone() Metadata {
	cloned := m
	if m.Modifiers != nil {
		cloned.Modifiers = maps.Clone(m.Modifiers)
	}
	return cloned
}

// TestCase represents a single synthesized attack or evaluation unit.
type TestCase struct {
	ID             string      `json:"id"`
	Vars           Vars        `json:"vars"`
	Assert         []Assertion `json:"assert"`
	Metadata       Metadata    `json:"metadata"`
	TargetOverride Provider    `json:"-"`
}

// Clone returns an isolated copy of TestCase safe for strategy transformations.
func (tc TestCase) Clone() TestCase {
	return TestCase{
		ID:             tc.ID,
		Vars:           tc.Vars.Clone(),
		Assert:         slices.Clone(tc.Assert),
		Metadata:       tc.Metadata.Clone(),
		TargetOverride: tc.TargetOverride,
	}
}

// TokenUsage tracks prompt, completion, and cached token counts.
type TokenUsage struct {
	Prompt     int `json:"prompt"`
	Completion int `json:"completion"`
	Cached     int `json:"cached"`
	Total      int `json:"total"`
	Requests   int `json:"requests"`
}

// PromptRequest encapsulates the rendered prompt and execution context sent to a Provider.
type PromptRequest struct {
	Prompt    string   `json:"prompt"`
	Vars      Vars     `json:"vars,omitempty"`
	SessionID string   `json:"sessionId,omitempty"`
	TestCase  TestCase `json:"-"`
}

// ProviderResponse contains the model or target output, latency, and token metrics.
type ProviderResponse struct {
	Output              string             `json:"output"`
	SessionID           string             `json:"sessionId,omitempty"`
	LatencyMs           int64              `json:"latencyMs"`
	Cached              bool               `json:"cached"`
	TokenUsage          TokenUsage         `json:"tokenUsage"`
	ConversationHistory []ConversationTurn `json:"conversationHistory,omitempty"`
}

// GradingResult represents the pass/fail verdict and score from a Grader.
type GradingResult struct {
	Pass       bool       `json:"pass"`
	Score      float64    `json:"score"`
	Reason     string     `json:"reason"`
	Rubric     string     `json:"rubric,omitempty"`
	TokensUsed TokenUsage `json:"tokensUsed"`
}

// Provider defines the unified interface for LLMs, HTTP targets, and multi-turn strategy decorators.
type Provider interface {
	ID() string
	Call(ctx context.Context, req PromptRequest) (ProviderResponse, error)
}

// PluginConfig holds configuration options passed to a Plugin during synthesis.
type PluginConfig struct {
	NumTests           int               `json:"numTests"`
	InjectVar          string            `json:"injectVar"`
	Purpose            string            `json:"purpose"`
	Language           string            `json:"language,omitempty"`
	MaxCharsPerMessage int               `json:"maxCharsPerMessage,omitempty"`
	Examples           []string          `json:"examples,omitempty"`
	Modifiers          map[string]string `json:"modifiers,omitempty"`
}

// Plugin generates initial seed TestCases for a specific vulnerability category.
type Plugin interface {
	ID() string
	GenerateTests(ctx context.Context, attacker Provider, cfg PluginConfig) ([]TestCase, error)
}

// Strategy mutates seed TestCases or wraps their TargetOverride Provider for multi-turn attacks.
type Strategy interface {
	ID() string
	Apply(ctx context.Context, cases []TestCase, injectVar string) ([]TestCase, error)
}

// GradeRequest provides the full context required to evaluate a target response.
type GradeRequest struct {
	Prompt   string
	Output   string
	TestCase TestCase
	Judge    Provider
}

// Grader evaluates whether a target response resisted or succumbed to an attack.
type Grader interface {
	ID() string
	Grade(ctx context.Context, req GradeRequest) (GradingResult, error)
}
