package strategy

import (
	"cmp"
	"context"
	"fmt"
	"strings"

	"github.com/bkelceoglu/go-rt/internal/domain"
	"github.com/bkelceoglu/go-rt/internal/provider/agentic"
)

const (
	StrategyLayer     = "gort:strategy:layer"
	StrategyCrescendo = agentic.StrategyCrescendo
	StrategyGOAT      = agentic.StrategyGOAT
	StrategyHydra     = agentic.StrategyHydra
	StrategyRetry     = agentic.StrategyRetry
)

var (
	_ domain.Strategy = (*MultiTurnStrategy)(nil)
	_ domain.Strategy = (*RetryStrategy)(nil)
	_ domain.Strategy = (*LayerStrategy)(nil)
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
	injectVar = cmp.Or(injectVar, "prompt")
	out := make([]domain.TestCase, 0, len(cases))
	for _, tc := range cases {
		out = append(out, m.wrapCase(tc, injectVar))
	}
	return out, nil
}

func (m *MultiTurnStrategy) wrapCase(tc domain.TestCase, injectVar string) domain.TestCase {
	cloned := tc.Clone()
	cloned.Metadata.OriginalText = cmp.Or(cloned.Metadata.OriginalText, cloned.Vars[injectVar])
	cloned.Metadata.StrategyID = m.id
	cloned.ID = fmt.Sprintf("%s-%s", tc.ID, m.shortName)
	if m.target != nil {
		cloned.TargetOverride = agentic.NewCrescendoDecorator(m.target, m.attacker, m.id, m.maxTurns, m.maxBacktracks)
	}
	return cloned
}

// RetryStrategy attaches a RetryDecorator TargetOverride to each TestCase.
type RetryStrategy struct {
	id         string
	maxRetries int
	target     domain.Provider
	attacker   domain.Provider
}

// NewRetryStrategy constructs an automatic retry-on-refusal strategy.
func NewRetryStrategy(spec domain.StrategySpec, target, attacker domain.Provider) *RetryStrategy {
	norm := normalizeStrategyID(spec.ID)
	retries := spec.MaxBacktracks
	if retries <= 0 {
		retries = 2
	}
	return &RetryStrategy{
		id:         norm,
		maxRetries: retries,
		target:     target,
		attacker:   attacker,
	}
}

// ID returns the canonical strategy identifier.
func (r *RetryStrategy) ID() string {
	return r.id
}

// Apply clones each TestCase and attaches a RetryDecorator to TargetOverride.
func (r *RetryStrategy) Apply(_ context.Context, cases []domain.TestCase, injectVar string) ([]domain.TestCase, error) {
	injectVar = cmp.Or(injectVar, "prompt")
	out := make([]domain.TestCase, 0, len(cases))
	for _, tc := range cases {
		out = append(out, r.wrapCase(tc, injectVar))
	}
	return out, nil
}

func (r *RetryStrategy) wrapCase(tc domain.TestCase, injectVar string) domain.TestCase {
	cloned := tc.Clone()
	cloned.Metadata.OriginalText = cmp.Or(cloned.Metadata.OriginalText, cloned.Vars[injectVar])
	cloned.Metadata.StrategyID = r.id
	cloned.ID = fmt.Sprintf("%s-retry", tc.ID)
	if r.target != nil {
		cloned.TargetOverride = agentic.NewRetryDecorator(r.target, r.attacker, r.maxRetries)
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

var framingConstructors = map[string]func() domain.Strategy{
	StrategyJailbreakTemplates:  func() domain.Strategy { return NewJailbreakTemplateStrategy(nil) },
	StrategyMathPrompt:          func() domain.Strategy { return NewMathPromptStrategy() },
	StrategyPromptInjection:     func() domain.Strategy { return NewPromptInjectionStrategy() },
	StrategyCitation:            func() domain.Strategy { return NewCitationStrategy() },
	StrategyAuthoritativeMarkup: func() domain.Strategy { return NewAuthoritativeMarkupStrategy() },
	StrategyMischievousUser:     func() domain.Strategy { return NewMischievousUserStrategy() },
	StrategyJailbreakComposite:  func() domain.Strategy { return NewJailbreakCompositeStrategy() },
	StrategyJailbreakLikert:     func() domain.Strategy { return NewJailbreakLikertStrategy() },
	StrategyJailbreakTree:       func() domain.Strategy { return NewJailbreakTreeStrategy() },
	StrategyJailbreakMeta:       func() domain.Strategy { return NewJailbreakMetaStrategy() },
	StrategyJailbreakGoblin:     func() domain.Strategy { return NewJailbreakGoblinStrategy() },
	StrategyJailbreak:           func() domain.Strategy { return NewJailbreakMasterStrategy() },
	StrategyBestOfN:             func() domain.Strategy { return NewBestOfNStrategy() },
}

// BuildStrategy resolves a static, template, or layer StrategySpec.
func BuildStrategy(spec domain.StrategySpec) (domain.Strategy, error) {
	return BuildStrategyWithProviders(spec, nil, nil)
}

// BuildStrategyWithProviders resolves any static, template, layer, or multi-turn StrategySpec.
func BuildStrategyWithProviders(spec domain.StrategySpec, target, attacker domain.Provider) (domain.Strategy, error) {
	norm := normalizeStrategyID(spec.ID)
	if ctor, ok := framingConstructors[norm]; ok {
		return ctor(), nil
	}
	switch norm {
	case StrategyRetry:
		return NewRetryStrategy(spec, target, attacker), nil
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

type strategyBatchRunner struct {
	specs     []domain.StrategySpec
	injectVar string
	target    domain.Provider
	attacker  domain.Provider
}

// ExpandForTarget applies configured strategies and binds target/attacker providers for multi-turn strategies.
func ExpandForTarget(ctx context.Context, cases []domain.TestCase, specs []domain.StrategySpec, exempt map[string]bool, injectVar string, target, attacker domain.Provider) ([]domain.TestCase, error) {
	if len(specs) == 0 {
		specs = []domain.StrategySpec{{ID: StrategyBasic}}
	}
	exemptCases, activeCases := splitExemptCases(cases, exempt)
	runner := strategyBatchRunner{specs: specs, injectVar: injectVar, target: target, attacker: attacker}
	expanded, err := runner.run(ctx, activeCases)
	if err != nil {
		return nil, err
	}
	return append(exemptCases, expanded...), nil
}

func splitExemptCases(cases []domain.TestCase, exempt map[string]bool) ([]domain.TestCase, []domain.TestCase) {
	exemptOut := []domain.TestCase{}
	activeOut := []domain.TestCase{}
	for _, tc := range cases {
		cloned := tc.Clone()
		if exempt != nil && exempt[cloned.Metadata.PluginID] {
			cloned.Metadata.StrategyID = cmp.Or(cloned.Metadata.StrategyID, StrategyBasic)
			exemptOut = append(exemptOut, cloned)
			continue
		}
		activeOut = append(activeOut, cloned)
	}
	return exemptOut, activeOut
}

func (r strategyBatchRunner) run(ctx context.Context, activeCases []domain.TestCase) ([]domain.TestCase, error) {
	if len(activeCases) == 0 {
		return []domain.TestCase{}, nil
	}
	all := make([]domain.TestCase, 0, len(activeCases)*len(r.specs))
	for _, spec := range r.specs {
		strat, err := BuildStrategyWithProviders(spec, r.target, r.attacker)
		if err != nil {
			return nil, fmt.Errorf("build strategy %s: %w", spec.ID, err)
		}
		mutated, err := strat.Apply(ctx, activeCases, r.injectVar)
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
