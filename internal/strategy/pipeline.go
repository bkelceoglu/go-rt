package strategy

import (
	"context"
	"fmt"

	"github.com/caveman/go-rt/internal/domain"
)

const StrategyLayer = "gort:strategy:layer"

// LayerStrategy chains multiple child strategies sequentially onto each TestCase.
type LayerStrategy struct {
	steps []domain.Strategy
}

// NewLayerStrategy constructs a sequential multi-step strategy pipeline.
func NewLayerStrategy(stepIDs []string) (*LayerStrategy, error) {
	if len(stepIDs) == 0 {
		stepIDs = []string{StrategyJailbreakTemplates, StrategyBase64}
	}
	steps := make([]domain.Strategy, 0, len(stepIDs))
	for _, id := range stepIDs {
		s, err := BuildStrategy(domain.StrategySpec{ID: id})
		if err != nil {
			return nil, err
		}
		steps = append(steps, s)
	}
	return &LayerStrategy{steps: steps}, nil
}

// ID returns the canonical layer strategy identifier.
func (l *LayerStrategy) ID() string {
	return StrategyLayer
}

// Apply runs each child strategy in sequence while preserving OriginalText.
func (l *LayerStrategy) Apply(ctx context.Context, cases []domain.TestCase, injectVar string) ([]domain.TestCase, error) {
	current := cloneCasesSlice(cases)
	var err error
	for _, step := range l.steps {
		current, err = step.Apply(ctx, current, injectVar)
		if err != nil {
			return nil, err
		}
	}
	for i := range current {
		current[i].Metadata.StrategyID = StrategyLayer
	}
	return current, nil
}

// BuildStrategy resolves a StrategySpec into a concrete domain.Strategy implementation.
func BuildStrategy(spec domain.StrategySpec) (domain.Strategy, error) {
	norm := normalizeStrategyID(spec.ID)
	switch norm {
	case StrategyJailbreakTemplates:
		return NewJailbreakTemplateStrategy(nil), nil
	case StrategyLayer:
		return NewLayerStrategy(spec.Steps)
	default:
		return NewStaticStrategy(norm)
	}
}

// ExpandWithStrategies applies configured strategies to non-exempt cases and preserves exempt cases once.
func ExpandWithStrategies(ctx context.Context, cases []domain.TestCase, specs []domain.StrategySpec, exempt map[string]bool, injectVar string) ([]domain.TestCase, error) {
	if len(specs) == 0 {
		specs = []domain.StrategySpec{{ID: StrategyBasic}}
	}
	exemptCases, activeCases := splitExemptCases(cases, exempt)
	expanded, err := runStrategySpecs(ctx, activeCases, specs, injectVar)
	if err != nil {
		return nil, err
	}
	return append(exemptCases, expanded...), nil
}

func splitExemptCases(cases []domain.TestCase, exempt map[string]bool) ([]domain.TestCase, []domain.TestCase) {
	var exemptOut, activeOut []domain.TestCase
	for _, tc := range cases {
		cloned := tc.Clone()
		if exempt != nil && exempt[cloned.Metadata.PluginID] {
			if cloned.Metadata.StrategyID == "" {
				cloned.Metadata.StrategyID = StrategyBasic
			}
			exemptOut = append(exemptOut, cloned)
			continue
		}
		activeOut = append(activeOut, cloned)
	}
	return exemptOut, activeOut
}

func runStrategySpecs(ctx context.Context, activeCases []domain.TestCase, specs []domain.StrategySpec, injectVar string) ([]domain.TestCase, error) {
	if len(activeCases) == 0 {
		return nil, nil
	}
	var all []domain.TestCase
	for _, spec := range specs {
		strat, err := BuildStrategy(spec)
		if err != nil {
			return nil, fmt.Errorf("build strategy %s: %w", spec.ID, err)
		}
		mutated, err := strat.Apply(ctx, activeCases, injectVar)
		if err != nil {
			return nil, err
		}
		all = append(all, mutated...)
	}
	return all, nil
}

func cloneCasesSlice(cases []domain.TestCase) []domain.TestCase {
	out := make([]domain.TestCase, 0, len(cases))
	for _, tc := range cases {
		out = append(out, tc.Clone())
	}
	return out
}
