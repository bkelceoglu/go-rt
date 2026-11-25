package template

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/bkelceoglu/go-rt/internal/domain"
)

// SafeRenderer performs two-phase SSTI-safe variable substitution on prompt templates.
type SafeRenderer struct {
	SkipVars []string
}

// NewSafeRenderer creates a SafeRenderer that protects skipVars (such as injectVar) from secondary evaluation.
func NewSafeRenderer(skipVars ...string) *SafeRenderer {
	return &SafeRenderer{SkipVars: skipVars}
}

// Render substitutes Vars into rawTemplate while guaranteeing SkipVars payloads are injected verbatim.
func (r *SafeRenderer) Render(rawTemplate string, vars domain.Vars) string {
	boundaries := make(map[string]string, len(r.SkipVars))
	stageOne := placeholderRegex.ReplaceAllStringFunc(rawTemplate, func(match string) string {
		return r.replaceFirstStage(match, vars, boundaries)
	})
	return applyBoundaries(stageOne, boundaries)
}

func (r *SafeRenderer) replaceFirstStage(match string, vars domain.Vars, boundaries map[string]string) string {
	sub := placeholderRegex.FindStringSubmatch(match)
	if len(sub) < 2 {
		return match
	}
	val, exists := vars[sub[1]]
	if !exists {
		return match
	}
	if slices.Contains(r.SkipVars, sub[1]) {
		token := fmt.Sprintf("__GORT_INJECT_BOUNDARY_%s__", sub[1])
		boundaries[token] = val
		return token
	}
	return val
}

func applyBoundaries(rendered string, boundaries map[string]string) string {
	result := rendered
	for token, rawPayload := range boundaries {
		result = strings.ReplaceAll(result, token, rawPayload)
	}
	return result
}

// UnpackMultiInput parses a JSON object string and merges its keys into a cloned Vars map.
func UnpackMultiInput(rawJSON string, baseVars domain.Vars) (domain.Vars, error) {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &parsed); err != nil {
		return nil, fmt.Errorf("unpack multi-input json: %w", err)
	}
	out := baseVars.Clone()
	for k, v := range parsed {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out, nil
}
