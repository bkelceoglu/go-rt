package provider

import (
	"strings"

	"github.com/caveman/go-rt/internal/domain"
)

// BuildFromSpec constructs a concrete Provider (Webhook or OpenAI-compatible) wrapped in RateLimiter.
func BuildFromSpec(spec domain.ProviderSpec) domain.Provider {
	return NewRateLimitedProvider(buildBaseProvider(spec), 0, 0, 4)
}

func buildBaseProvider(spec domain.ProviderSpec) domain.Provider {
	if strings.HasPrefix(spec.ID, "webhook:") || spec.BodyTmpl != "" {
		return NewWebhook(spec, nil)
	}
	return NewOpenAICompatible(spec, nil)
}

// BuildJudge constructs a Judge evaluation provider from JudgeSpec.
func BuildJudge(spec domain.JudgeSpec) domain.Provider {
	if spec.ID == "" {
		return nil
	}
	pSpec := domain.ProviderSpec{
		ID:      spec.ID,
		BaseURL: spec.BaseURL,
		APIKey:  spec.APIKey,
		Temp:    spec.Temp,
		MaxToks: spec.MaxToks,
		Think:   spec.Think,
	}
	return BuildFromSpec(pSpec)
}
