package provider

import (
	"context"
	"fmt"

	"github.com/caveman/go-rt/internal/domain"
	"golang.org/x/time/rate"
)

// RateLimitedProvider wraps any domain.Provider with token-bucket rate limiting and concurrency throttling.
type RateLimitedProvider struct {
	underlying domain.Provider
	limiter    *rate.Limiter
	semaphore  chan struct{}
}

// NewRateLimitedProvider creates a concurrency and RPS-guarded provider decorator.
func NewRateLimitedProvider(underlying domain.Provider, rps float64, burst int, maxConcurrent int) *RateLimitedProvider {
	var lim *rate.Limiter
	if rps > 0 {
		if burst < 1 {
			burst = 1
		}
		lim = rate.NewLimiter(rate.Limit(rps), burst)
	}
	var sem chan struct{}
	if maxConcurrent > 0 {
		sem = make(chan struct{}, maxConcurrent)
	}
	return &RateLimitedProvider{
		underlying: underlying,
		limiter:    lim,
		semaphore:  sem,
	}
}

// ID delegates to the underlying provider identifier.
func (r *RateLimitedProvider) ID() string {
	return r.underlying.ID()
}

// Call acquires rate token, obtains semaphore slot, and invokes the underlying provider.
func (r *RateLimitedProvider) Call(ctx context.Context, req domain.PromptRequest) (domain.ProviderResponse, error) {
	if err := r.waitForRateLimit(ctx); err != nil {
		return domain.ProviderResponse{}, err
	}
	release, err := r.acquireSemaphore(ctx)
	if err != nil {
		return domain.ProviderResponse{}, err
	}
	defer release()
	return r.underlying.Call(ctx, req)
}

func (r *RateLimitedProvider) waitForRateLimit(ctx context.Context) error {
	if r.limiter == nil {
		return nil
	}
	if err := r.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter wait canceled for %s: %w", r.ID(), err)
	}
	return nil
}

func (r *RateLimitedProvider) acquireSemaphore(ctx context.Context) (func(), error) {
	if r.semaphore == nil {
		return func() {}, nil
	}
	select {
	case r.semaphore <- struct{}{}:
		return func() { <-r.semaphore }, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("concurrency throttle wait canceled for %s: %w", r.ID(), ctx.Err())
	}
}
