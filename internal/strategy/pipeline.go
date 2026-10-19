package strategy

import (
	"context"
	"fmt"
	"strings"

	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/provider/agentic"
)

const (
	StrategyLayer     = "gort:strategy:layer"
	StrategyCrescendo = agentic.StrategyCrescendo
	StrategyGOAT      = agentic.StrategyGOAT
	StrategyHydra     = agentic.StrategyHydra
)

// MultiTurnStrategy attaches a Crescendo/GOAT/Hydra TargetOverride decorator to each TestCase.
type MultiTurnStrategy struct {
	id            string
	shortName     string
	maxTurns      int
	maxBacktracks int
	target        domain.Provider
	attacker      domain.Provider
}

// NewMultiTurnStrategy constructs an agentic multi-turn strategy.
func NewMultiTurnStrategy(spec domain.StrategySpec, target, attacker domain.Provider) *MultiTurnStrategy {
	norm := normalizeStrategyID(spec.ID)
	short := strings.TrimPrefix(norm, "gort:strategy:")
	return &MultiTurnStrategy{
		id:            norm,
		shortName:     short,
		maxTurns:      spec.MaxTurns,
		maxBacktracks: spec.MaxBacktracks,
		target:        target,
		attacker:      attacker,
	}
}

// ID returns the canonical multi-turn strategy identifier.
func (m *MultiTurnStrategy) ID() string {
	return m.id
}

// Apply clones each TestCase and attaches a fresh CrescendoDecorator to TargetOverride.
func (m *MultiTurnStrategy) Apply(_ context.Context, cases []domain.TestCase, injectVar string) ([]domain.TestCase, error) {
	if injectVar == "" {
		injectVar = "prompt"
	}
	out := make([]domain.TestCase, 0, len(cases))
	for _, tc := range cases {
		out = append(out, m.wrapCase(tc, injectVar))
	}
	return out, nil
}

func (m *MultiTurnStrategy) wrapCase(tc domain.TestCase, injectVar string) domain.TestCase {
	cloned := tc.Clone()
	if cloned.Metadata.OriginalText == "" {
		cloned.Metadata.OriginalText = cloned.Vars[injectVar]
	}
	cloned.Metadata.StrategyID = m.id
	cloned.ID = fmt.Sprintf("%s-%s", tc.ID, m.shortName)
	if m.target != nil {
		cloned.TargetOverride = agentic.NewCrescendoDecorator(m.target, m.attacker, m.id, m.maxTurns, m.maxBacktracks)
	}
	return cloned
}

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

// BuildStrategy resolves a static, template, or layer StrategySpec.
func BuildStrategy(spec domain.StrategySpec) (domain.Strategy, error) {
	return BuildStrategyWithProviders(spec, nil, nil)
}

// BuildStrategyWithProviders resolves any static, template, layer, or multi-turn StrategySpec.
func BuildStrategyWithProviders(spec domain.StrategySpec, target, attacker domain.Provider) (domain.Strategy, error) {
	norm := normalizeStrategyID(spec.ID)
	switch norm {
	case StrategyJailbreakTemplates:
		return NewJailbreakTemplateStrategy(nil), nil
	case StrategyLayer:
		return NewLayerStrategy(spec.Steps)
	case StrategyCrescendo, StrategyGOAT, StrategyHydra:
		return NewMultiTurnStrategy(spec, target, attacker), nil
	default:
		return NewStaticStrategy(norm)
	}
}

// ExpandWithStrategies applies configured strategies without multi-turn target binding.
func ExpandWithStrategies(ctx context.Context, cases []domain.TestCase, specs []domain.StrategySpec, exempt map[string]bool, injectVar string) ([]domain.TestCase, error) {
	return ExpandForTarget(ctx, cases, specs, exempt, injectVar, nil, nil)
}

// ExpandForTarget applies configured strategies and binds target/attacker providers for multi-turn strategies.
func ExpandForTarget(ctx context.Context, cases []domain.TestCase, specs []domain.StrategySpec, exempt map[string]bool, injectVar string, target, attacker domain.Provider) ([]domain.TestCase, error) {
	if len(specs) == 0 {
		specs = []domain.StrategySpec{{ID: StrategyBasic}}
	}
	exemptCases, activeCases := splitExemptCases(cases, exempt)
	expanded, err := runStrategySpecs(ctx, activeCases, specs, injectVar, target, attacker)
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

func runStrategySpecs(ctx context.Context, activeCases []domain.TestCase, specs []domain.StrategySpec, injectVar string, target, attacker domain.Provider) ([]domain.TestCase, error) {
	if len(activeCases) == 0 {
		return nil, nil
	}
	var all []domain.TestCase
	for _, spec := range specs {
		strat, err := BuildStrategyWithProviders(spec, target, attacker)
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
