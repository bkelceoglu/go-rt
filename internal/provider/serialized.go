package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/caveman/go-rt/internal/domain"
)

var _ domain.Provider = (*SerializedProvider)(nil)

// SerializedProvider wraps a domain.Provider with a shared 1-slot semaphore so two roles
// (such as Attacker LLM and Judge LLM sharing the same backend) never execute concurrently.
type SerializedProvider struct {
	inner domain.Provider
	sem   chan struct{}
}

// WrapSharedSequential wraps attacker and judge in a shared lock if they share the same model or sequential mode is enabled.
func WrapSharedSequential(attacker, judge domain.Provider, forceSequential bool) (domain.Provider, domain.Provider) {
	if attacker == nil || judge == nil {
		return attacker, judge
	}
	if !forceSequential && !strings.EqualFold(attacker.ID(), judge.ID()) {
		return attacker, judge
	}
	sharedSem := make(chan struct{}, 1)
	return &SerializedProvider{inner: attacker, sem: sharedSem}, &SerializedProvider{inner: judge, sem: sharedSem}
}

// ID returns the underlying provider identifier.
func (s *SerializedProvider) ID() string {
	return s.inner.ID()
}

// Call acquires the shared slot before invoking the underlying LLM provider, respecting context cancellation.
func (s *SerializedProvider) Call(ctx context.Context, req domain.PromptRequest) (domain.ProviderResponse, error) {
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
		return s.inner.Call(ctx, req)
	case <-ctx.Done():
		return domain.ProviderResponse{}, fmt.Errorf("serialized provider wait canceled for %s: %w", s.ID(), ctx.Err())
	}
}
