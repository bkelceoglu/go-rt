package provider

import (
	"context"
	"strings"
	"sync"

	"github.com/caveman/go-rt/internal/domain"
)

// SerializedProvider wraps a domain.Provider with a shared mutex so two roles
// (such as Attacker LLM and Judge LLM sharing the same backend) never execute concurrently.
type SerializedProvider struct {
	inner domain.Provider
	mu    *sync.Mutex
}

// WrapSharedSequential wraps attacker and judge in a shared mutex if they share the same model or sequential mode is enabled.
func WrapSharedSequential(attacker, judge domain.Provider, forceSequential bool) (domain.Provider, domain.Provider) {
	if attacker == nil || judge == nil {
		return attacker, judge
	}
	if !forceSequential && !strings.EqualFold(attacker.ID(), judge.ID()) {
		return attacker, judge
	}
	sharedMu := &sync.Mutex{}
	return &SerializedProvider{inner: attacker, mu: sharedMu}, &SerializedProvider{inner: judge, mu: sharedMu}
}

// ID returns the underlying provider identifier.
func (s *SerializedProvider) ID() string {
	return s.inner.ID()
}

// Call acquires the shared lock before invoking the underlying LLM provider.
func (s *SerializedProvider) Call(ctx context.Context, req domain.PromptRequest) (domain.ProviderResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inner.Call(ctx, req)
}
